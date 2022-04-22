package exchange

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/streamingfast/substream-pancakeswap/pancakeswap"
	"github.com/streamingfast/substreams/graph-node/metrics"
	"github.com/streamingfast/substreams/graph-node/storage/postgres"
	"github.com/streamingfast/substreams/runtime"
)

func init() {
	rootCmd.Flags().Int64P("start-block", "s", -1, "Start block for blockchain firehose")
	rootCmd.Flags().Uint64P("stop-block", "t", 0, "Stop block for blockchain firehose")

	rootCmd.Flags().Bool("local", false, "run with local runtime")
	rootCmd.Flags().Bool("no-return-handler", false, "Avoid printing output for module")

	///local options
	rootCmd.Flags().String("rpc-endpoint", "http://localhost:8546", "RPC endpoint of blockchain node")
	rootCmd.Flags().String("state-store-url", "./localdata", "URL of state store")
	rootCmd.Flags().String("blocks-store-url", "./localblocks", "URL of blocks store")
	rootCmd.Flags().String("rpc-cache-store-url", "./rpc-cache", "URL of blocks store")
	rootCmd.Flags().String("irr-indexes-url", "./localirr", "URL of blocks store")
	rootCmd.Flags().Bool("partial", false, "Produce partial stores")

	///remote options
	rootCmd.Flags().String("firehose-endpoint", "api.streamingfast.io:443", "firehose GRPC endpoint")
	rootCmd.Flags().String("firehose-api-key-envvar", "FIREHOSE_API_KEY", "name of variable containing firehose authentication token (JWT)")
	rootCmd.Flags().BoolP("insecure", "k", false, "Skip certificate validation on GRPC connection")
	rootCmd.Flags().BoolP("plaintext", "p", false, "Establish GRPC connection in plaintext")

	///postgres loader flags
	rootCmd.Flags().String("pg-dsn", "", "dsn for postgres database")
	rootCmd.Flags().String("pg-schema", "", "postgres schema name")
	rootCmd.Flags().Bool("pg-disable-transactions", false, "disable postgres transactions for faster inserts")
	rootCmd.Flags().String("deployment", "", "subgraph deployment name")
}

// remoteCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "exchange [manifest] [module_name]",
	Short:        "Run pancakeswap exchange substream",
	RunE:         runExchange,
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true,
}

func runExchange(cmd *cobra.Command, args []string) error {
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

	config := &runtime.Config{
		ManifestPath:     args[0],
		OutputStreamName: args[1],
		StartBlock:       mustGetUint64(cmd, "start-block"),
		StopBlock:        mustGetUint64(cmd, "stop-block"),
		PrintMermaid:     false,

		ReturnHandler: loader.ReturnHandler,
	}

	localRun := mustGetBool(cmd, "local")

	if localRun {
		config.LocalConfig = &runtime.LocalConfig{
			BlocksStoreUrl: mustGetString(cmd, "blocks-store-url"),
			IrrIndexesUrl:  mustGetString(cmd, "irr-indexes-url"),
			StateStoreUrl:  mustGetString(cmd, "state-store-url"),
			RpcEndpoint:    mustGetString(cmd, "rpc-endpoint"),
			RpcCacheUrl:    mustGetString(cmd, "rpc-cache-store-url"),
			PartialMode:    mustGetBool(cmd, "partial"),

			ProtobufBlockType: "sf.ethereum.type.v1.Block",
		}
		err = runtime.LocalRun(ctx, config)
	} else {
		config.RemoteConfig = &runtime.RemoteConfig{
			FirehoseEndpoint:     mustGetString(cmd, "firehose-endpoint"),
			FirehoseApiKeyEnvVar: mustGetString(cmd, "firehose-api-key-envvar"),
			InsecureMode:         mustGetBool(cmd, "insecure"),
			Plaintext:            mustGetBool(cmd, "plaintext"),
		}
		err = runtime.RemoteRun(ctx, config)
	}

	if err != nil {
		err = fmt.Errorf("run failed: %w", err)
	}

	return err
}
