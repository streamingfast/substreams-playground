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

// remoteCmd represents the command to run pancakeswap exchange substream remotely
var remoteCmd = &cobra.Command{
	Use:          "remote [manifest] [module_name]",
	Short:        "Run pancakeswap exchange substream remotely",
	RunE:         runRemote,
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true,
}

func init() {
	remoteCmd.Flags().String("firehose-endpoint", "api.streamingfast.io:443", "firehose GRPC endpoint")
	remoteCmd.Flags().String("firehose-api-key-envvar", "FIREHOSE_API_KEY", "name of variable containing firehose authentication token (JWT)")
	remoteCmd.Flags().BoolP("insecure", "k", false, "Skip certificate validation on GRPC connection")
	remoteCmd.Flags().BoolP("plaintext", "p", false, "Establish GRPC connection in plaintext")

	rootCmd.AddCommand(remoteCmd)
}

func runRemote(cmd *cobra.Command, args []string) error {
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

	config := &runtime.RemoteConfig{
		FirehoseEndpoint:     mustGetString(cmd, "firehose-endpoint"),
		FirehoseApiKeyEnvVar: mustGetString(cmd, "firehose-api-key-envvar"),
		InsecureMode:         mustGetBool(cmd, "insecure"),
		Plaintext:            mustGetBool(cmd, "plaintext"),
		Config: &runtime.Config{
			ManifestPath:     args[0],
			OutputStreamName: args[1],
			StartBlock:       uint64(mustGetInt64(cmd, "start-block")),
			StopBlock:        mustGetUint64(cmd, "stop-block"),
			PrintMermaid:     false,

			ReturnHandler: loader.ReturnHandler,
		},
	}

	err = runtime.RemoteRun(ctx, config)

	if err != nil {
		err = fmt.Errorf("run failed: %w", err)
	}

	return err
}
