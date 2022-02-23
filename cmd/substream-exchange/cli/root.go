package cli

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/firehose"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/sparkle-pancakeswap/pipeline"
	"github.com/streamingfast/sparkle-pancakeswap/state"
	"github.com/streamingfast/sparkle/indexer"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "substream-pancakeswap",
	Short: "A PancakeSwap substream",
	RunE:  runRoot,
}

func runRoot(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var blockCount uint64 = 1000
	if len(args) > 0 {
		val, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid block count %s", args[1])
		}
		blockCount = uint64(val)
	}

	startBlockNum := viper.GetInt64("start-block")
	forceLoadState := false
	if startBlockNum > genesisBlock {
		forceLoadState = true
	}

	localBlocksPath := viper.GetString("blocks-store-url")
	blocksStore, err := dstore.NewDBinStore(localBlocksPath)
	if err != nil {
		return fmt.Errorf("setting up blocks store: %w", err)
	}

	irrIndexesPath := viper.GetString("irr-indexes-url")
	irrStore, err := dstore.NewStore(irrIndexesPath, "", "", false)
	if err != nil {
		return fmt.Errorf("setting up irr blocks store: %w", err)
	}

	rpcCacheStore, err := dstore.NewStore("./rpc-cache", "", "", false)
	if err != nil {
		return fmt.Errorf("setting up store for rpc-cache: %w", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true, // don't reuse connections
		},
		Timeout: 3 * time.Second,
	}

	rpcEndpoint := viper.GetString("rpc-endpoint")
	rpcClient := rpc.NewClient(rpcEndpoint, rpc.WithHttpClient(httpClient))
	rpcCache := indexer.NewCache(rpcCacheStore, rpcCacheStore, 0, 999)
	rpcCache.Load(ctx)

	stateStorePath := viper.GetString("state-store-url")
	stateStore, err := dstore.NewStore(stateStorePath, "", "", false)
	if err != nil {
		return fmt.Errorf("setting up store for data: %w", err)
	}

	ioFactory := state.NewStoreStateIOFactory(stateStore)
	stores := map[string]*state.Builder{}
	for _, storeName := range []string{"pairs", "total_pairs", "prices", "volume24h"} {
		s := state.New(storeName, ioFactory)
		stores[storeName] = s
	}

	if forceLoadState {
		// Use AN ABSOLUTE store, or SQUASH ALL PARTIAL!
		err := loadStateFromDisk(stores, uint64(startBlockNum))
		if err != nil {
			return err
		}
	}

	pipe := pipeline.New(uint64(startBlockNum), rpcClient, rpcCache, stores)

	handler := pipe.HandlerFactory(blockCount)

	hose := firehose.New([]dstore.Store{blocksStore}, startBlockNum, handler,
		firehose.WithForkableSteps(bstream.StepIrreversible),
		firehose.WithIrreversibleBlocksIndex(irrStore, true, []uint64{10000, 1000, 100}),
	)

	if err := hose.Run(context.Background()); err != nil {
		return fmt.Errorf("running the firehose: %w", err)
	}
	time.Sleep(5 * time.Second)

	return nil
}

func loadStateFromDisk(stores map[string]*state.Builder, startBlockNum uint64) error {
	for storeName, store := range stores {
		if err := store.Init(startBlockNum); err != nil {
			return fmt.Errorf("could not load state for store %s at block num %d: %w", storeName, startBlockNum, err)
		}
	}
	return nil
}
