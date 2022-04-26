package exchange

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/substream-pancakeswap/pancakeswap"
	"github.com/streamingfast/substreams/graph-node/metrics"
	"github.com/streamingfast/substreams/graph-node/storage/postgres"
	"github.com/streamingfast/substreams/runtime"
)

// localCmd represents the command to run pancakeswap exchange substream locally
var localCmd = &cobra.Command{
	Use:          "local [manifest] [module_name]",
	Short:        "Run pancakeswap exchange substream locally",
	RunE:         runLocal,
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true,
}

func init() {
	localCmd.Flags().String("rpc-endpoint", "http://localhost:8546", "RPC endpoint of blockchain node")
	localCmd.Flags().String("state-store-url", "./localdata", "URL of state store")
	localCmd.Flags().String("blocks-store-url", "./localblocks", "URL of blocks store")
	localCmd.Flags().String("rpc-cache-store-url", "./rpc-cache", "URL of rpc cache")
	localCmd.Flags().String("irr-indexes-url", "./localirr", "URL of irreversible blocks")
	localCmd.Flags().Bool("partial", false, "Produce partial stores")
	localCmd.Flags().Uint64("states-save-interval", uint64(10000), "State size")

	rootCmd.AddCommand(localCmd)
}

func runLocal(cmd *cobra.Command, args []string) error {
	err := bstream.ValidateRegistry()
	if err != nil {
		return fmt.Errorf("bstream validate registry %w", err)
	}

	ctx := cmd.Context()

	dsn := mustGetString(cmd, "pg-dsn")
	deployment := mustGetString(cmd, "pg-deployment")
	schema := mustGetString(cmd, "pg-schema")
	transactionsDisabled := mustGetBool(cmd, "pg-disable-transactions")

	subgraphDef := pancakeswap.Definition

	storage, err := postgres.New(zlog, metrics.NewBlockMetrics(), dsn, schema, deployment, subgraphDef, map[string]bool{}, !transactionsDisabled)
	if err != nil {
		return fmt.Errorf("creating postgres store: %w", err)
	}

	err = storage.RegisterEntities()
	if err != nil {
		return fmt.Errorf("store: registaring entities:%w", err)
	}

	loader := pancakeswap.NewLoader(storage, pancakeswap.Definition.Entities)

	config := &runtime.LocalConfig{
		BlocksStoreUrl: mustGetString(cmd, "blocks-store-url"),
		IrrIndexesUrl:  mustGetString(cmd, "irr-indexes-url"),
		StateStoreUrl:  mustGetString(cmd, "state-store-url"),
		RpcEndpoint:    mustGetString(cmd, "rpc-endpoint"),
		RpcCacheUrl:    mustGetString(cmd, "rpc-cache-store-url"),
		PartialMode:    mustGetBool(cmd, "partial"),

		ProtobufBlockType: "sf.ethereum.type.v1.Block",
		Config: &runtime.Config{
			ManifestPath:     args[0],
			OutputStreamName: args[1],
			StartBlock:       uint64(mustGetInt64(cmd, "start-block")),
			StopBlock:        mustGetUint64(cmd, "stop-block"),
			PrintMermaid:     false,

			ReturnHandler:      loader.ReturnHandler,
			StatesSaveInterval: mustGetUint64(cmd, "states-save-interval"),
		},
	}

	err = runtime.LocalRun(ctx, config)

	if err != nil {
		err = fmt.Errorf("run failed: %w", err)
	}

	return err
}
