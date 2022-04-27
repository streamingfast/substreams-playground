package exchange

import (
	"fmt"
	"io"
	"os"

	"github.com/streamingfast/bstream"
	_ "github.com/streamingfast/sf-ethereum/types"
	"github.com/streamingfast/substream-pancakeswap/pancakeswap"
	"github.com/streamingfast/substreams/client"
	"github.com/streamingfast/substreams/graph-node/metrics"
	"github.com/streamingfast/substreams/graph-node/storage/postgres"
	"github.com/streamingfast/substreams/manifest"
	pbsubstreams "github.com/streamingfast/substreams/pb/sf/substreams/v1"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:          "exchange [manifest]",
	Short:        "Run pancakeswap exchange substream",
	RunE:         runRoot,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
}

func init() {
	rootCmd.Flags().Int64P("start-block", "s", -1, "Start block for blockchain firehose")
	rootCmd.Flags().Uint64P("stop-block", "t", 0, "Stop block for blockchain firehose")
	rootCmd.Flags().Bool("no-return-handler", false, "Avoid printing output for module")

	rootCmd.Flags().String("firehose-endpoint", "api.streamingfast.io:443", "firehose GRPC endpoint")
	rootCmd.Flags().String("substreams-api-key-envvar", "FIREHOSE_API_KEY", "name of variable containing firehose authentication token (JWT)")
	rootCmd.Flags().BoolP("insecure", "k", false, "Skip certificate validation on GRPC connection")
	rootCmd.Flags().BoolP("plaintext", "p", false, "Establish GRPC connection in plaintext")

	///postgres loader flags
	rootCmd.Flags().String("pg-dsn", "", "dsn for postgres database")
	rootCmd.Flags().String("pg-schema", "", "postgres schema name")
	rootCmd.Flags().Bool("pg-disable-transactions", false, "disable postgres transactions for faster inserts")
	rootCmd.Flags().String("pg-deployment", "", "subgraph deployment name")
}

func runRoot(cmd *cobra.Command, args []string) error {
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

	manifestPath := args[0]
	manif, err := manifest.New(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest %q: %w", manifestPath, err)
	}

	manif.PrintMermaid()

	manifProto, err := manif.ToProto()
	if err != nil {
		return fmt.Errorf("parse manifest to proto%q: %w", manifestPath, err)
	}

	ssClient, callOpts, err := client.NewSubstreamsClient(
		mustGetString(cmd, "firehose-endpoint"),
		os.Getenv(mustGetString(cmd, "substreams-api-key-envvar")),
		mustGetBool(cmd, "insecure"),
		mustGetBool(cmd, "plaintext"),
	)
	if err != nil {
		return fmt.Errorf("substreams client setup: %w", err)
	}

	req := &pbsubstreams.Request{
		StartBlockNum: mustGetInt64(cmd, "start-block"),
		StopBlockNum:  mustGetUint64(cmd, "stop-block"),
		ForkSteps:     []pbsubstreams.ForkStep{pbsubstreams.ForkStep_STEP_IRREVERSIBLE},
		Manifest:      manifProto,
		OutputModules: []string{"db_out", "pairs", "totals"},
	}

	stream, err := ssClient.Blocks(ctx, req, callOpts...)
	if err != nil {
		return fmt.Errorf("call sf.substreams.v1.Stream/Blocks: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch r := resp.Message.(type) {
		case *pbsubstreams.Response_Progress:
			_ = r.Progress
		case *pbsubstreams.Response_SnapshotData:
			_ = r.SnapshotData
		case *pbsubstreams.Response_SnapshotComplete:
			_ = r.SnapshotComplete
		case *pbsubstreams.Response_Data:

			for _, output := range r.Data.Outputs {
				for _, log := range output.Logs {
					fmt.Println("LOG: ", log)
				}
				if output.Name == "db_out" {
					if err := loader.ReturnHandler(output.GetMapOutput().GetValue(), r.Data.Step, r.Data.Cursor, r.Data.Clock); err != nil {
						fmt.Printf("RETURN HANDLER ERROR: %s\n", err)
					}
				}
			}
		}
	}
}
