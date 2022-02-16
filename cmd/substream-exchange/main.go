package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/firehose"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/sparkle-pancakeswap/exchange"
	"github.com/streamingfast/sparkle/indexer"
	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
)

func main() {
	fmt.Println(os.Args)
	localBlockPath := "./localblocks"
	if len(os.Args) == 2 {
		localBlockPath = os.Args[1]
	}
	// Start piping blocks from a Firehose instance

	// these, taken from `index.go` in `sparkle/cli` stuff

	rpcEndpoint := os.Getenv("BSC_ENDPOINT")
	if rpcEndpoint == "" {
		rpcEndpoint = "http://localhost:8546" //  kc port-forward sub-pancake4-exchange-lucid-koschei-59686b7cc6-k49jk 8546:10.0.1.19:8546
	}

	//blocksStore, err := dstore.NewDBinStore("gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1")
	blocksStore, err := dstore.NewDBinStore(localBlockPath)
	if err != nil {
		log.Fatalln("error setting up blocks store:", err)
	}

	irrStore, err := dstore.NewStore("./localirr", "", "", false)
	if err != nil {
		log.Fatalln("error setting up blocks store:", err)
	}

	//const startBlock = 6810700 // 6809737
	const startBlock = 6810700
	pipe := setupPipeline(rpcEndpoint, startBlock)

	hose := firehose.New([]dstore.Store{blocksStore}, startBlock, pipe,
		firehose.WithForkableSteps(bstream.StepIrreversible),
		firehose.WithIrreversibleBlocksIndex(irrStore, true, []uint64{1000, 100}),
	)

	if err := hose.Run(context.Background()); err != nil {
		log.Fatalln("running the firehose:", err)
	}
	time.Sleep(5 * time.Second)
}

func setupPipeline(rpcEndpoint string, startBlockNum uint64) bstream.Handler {

	httpClient := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true, // don't reuse connections
		},
		Timeout: 3 * time.Second,
	}
	rpcClient := rpc.NewClient(rpcEndpoint, rpc.WithHttpClient(httpClient))
	subgraphDef := exchange.Definition

	rpcCacheStore, err := dstore.NewStore("./rpc-cache", "", "", false)
	if err != nil {
		log.Fatalln("setting up store for rpc-cache:", err)
	}

	rpcCache := indexer.NewCache(rpcCacheStore, rpcCacheStore, 0, 999)
	rpcCache.Load(context.Background())

	intr := exchange.NewSubstreamIntrinsics(rpcClient, rpcCache, true)
	_ = subgraphDef

	pairsStore := exchange.NewStateBuilder("pairs")
	//pairsStore.Init(startBlockNum, "/Users/cbillett/t/substream-data")

	totalPairsStore := exchange.NewStateBuilder("total_pairs")
	//totalPairsStore.Init(startBlockNum, "/Users/cbillett/t/substream-data")

	pairsPriceStore := exchange.NewStateBuilder("pairs_price")
	volume24hStore := exchange.NewStateBuilder("volume24h")

	pairExtractor := &exchange.PairExtractor{SubstreamIntrinsics: intr, Contract: eth.Address(exchange.FactoryAddressBytes)}
	pcsPairsStateBuilder := &exchange.PCSPairsStateBuilder{SubstreamIntrinsics: intr}
	pcsTotalPairsStateBuilder := &exchange.PCSTotalPairsStateBuilder{SubstreamIntrinsics: intr}
	pcsPricesStateBuilder := &exchange.PCSPricesStateBuilder{SubstreamIntrinsics: intr}
	reservesExtractor := &exchange.ReservesExtractor{SubstreamIntrinsics: intr}
	swapsExtractor := &exchange.SwapsExtractor{SubstreamIntrinsics: intr}
	volume24hStateBuilder := &exchange.PCSVolume24hStateBuilder{SubstreamIntrinsics: intr}

	return bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {

		// TODO: eventually, handle the `undo` signals.
		//  NOTE: The RUNTIME will handle the undo signals. It'll have all it needs.
		if block.Number >= startBlockNum+10000 {
			return io.EOF
		}

		blk := block.ToProtocol().(*pbcodec.Block)
		intr.SetCurrentBlock(blk)

		fmt.Println("-------------------------------------------------------------------")
		fmt.Println("BLOCK", blk.Num(), blk.ID())

		pairs, err := pairExtractor.Map(blk)
		if err != nil {
			return fmt.Errorf("extracting pairs: %w", err)
		}

		//pairs.Print()

		if err := pcsPairsStateBuilder.BuildState(pairs, pairsStore); err != nil {
			return fmt.Errorf("processing pair cache: %w", err)
		}
		//pairsStore.PrintDeltas()

		if err := pcsTotalPairsStateBuilder.BuildState(pairs, totalPairsStore); err != nil {
			return fmt.Errorf("processing total pairs: %w", err)
		}
		//totalPairsStore.PrintDeltas()

		reserveUpdates, err := reservesExtractor.Map(blk, pairsStore)
		if err != nil {
			return fmt.Errorf("processing reserves extractor: %w", err)
		}
		reserveUpdates.Print()

		if err := pcsPricesStateBuilder.BuildState(reserveUpdates, pairsPriceStore); err != nil {
			return fmt.Errorf("pairs price building: %w", err)
		}
		pairsPriceStore.PrintDeltas()

		swaps,  err := swapsExtractor.Map(blk, pairsStore)
		if err != nil {
			return fmt.Errorf("swaps extractor: %w", err)
		}


		if err := volume24hStateBuilder.BuildState(blk, swapUpdates, volume24hStore); err != nil {
			return fmt.Errorf("volume24 builder: %w", err)
		}

		volume24hStore.PrintDeltas()

		// Build a new "ReserveFilter{Pairs: []}"
		// followed by a AvgPriceStateBuilder
		// The idea is to replace: https://github.com/streamingfast/substream-pancakeswap/blob/master/exchange/handle_pair_sync_event.go#L249 into a stream.

		// Prep for next block
		pairsStore.Flush()
		totalPairsStore.Flush()
		pairsPriceStore.Flush()
		volume24hStore.Flush()

		if block.Number%100 == 0 {
			rpcCache.Save(context.Background())
		}

		// MARK INDEX:
		// if len(pairs) != 0 || len(reserveUpdates) != 0 {
		// 	indexer.MarkBlock(block) // each 100 blocks y'écrit whatever
		// }

		return nil
	})
}
