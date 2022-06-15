package exchange

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	_ "github.com/streamingfast/sf-ethereum/types"
	"github.com/streamingfast/substream-pancakeswap/pb/pcs/database/v1"
	"github.com/streamingfast/substreams/client"
	"github.com/streamingfast/substreams/manifest"
	pbsubstreams "github.com/streamingfast/substreams/pb/sf/substreams/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

// loadGraphNodeCmd represents the base command
var loadMongoCmd = &cobra.Command{
	Use:          "load-mongo [manifest]",
	Short:        "run pcs sub graph and load a mongo database",
	RunE:         runLoadMongo,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
}

func init() {
	loadMongoCmd.Flags().Int64P("start-block", "s", -1, "Start block for blockchain firehose")
	loadMongoCmd.Flags().Uint64P("stop-block", "t", 0, "Stop block for blockchain firehose")

	loadMongoCmd.Flags().String("firehose-endpoint", "bsc-dev.streamingfast.io:443", "firehose GRPC endpoint")
	loadMongoCmd.Flags().String("substreams-api-key-envvar", "FIREHOSE_API_TOKEN", "name of variable containing firehose authentication token (JWT)")
	loadMongoCmd.Flags().BoolP("insecure", "k", false, "Skip certificate validation on GRPC connection")
	loadMongoCmd.Flags().BoolP("plaintext", "p", false, "Establish GRPC connection in plaintext")

	rootCmd.AddCommand(loadMongoCmd)
}

func runLoadMongo(cmd *cobra.Command, args []string) error {
	err := bstream.ValidateRegistry()
	if err != nil {
		return fmt.Errorf("bstream validate registry %w", err)
	}

	ctx := cmd.Context()

	manifestPath := args[0]
	manifestReader := manifest.NewReader(manifestPath)
	pkg, err := manifestReader.Read()
	if err != nil {
		return fmt.Errorf("read manifest %q: %w", manifestPath, err)
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
		Modules:       pkg.Modules,
		OutputModules: []string{"db_out"},
	}

	stream, err := ssClient.Blocks(ctx, req, callOpts...)
	if err != nil {
		return fmt.Errorf("call sf.substreams.v1.Stream/Blocks: %w", err)
	}

	mongoDB, err := NewMongoDB("mongodb://localhost:27017")
	if err != nil {
		return fmt.Errorf("creating mongo db client")
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
			p := r.Progress
			for _, module := range p.Modules {
				fmt.Println("progress:", module.Name, module.GetProcessedRanges())
			}
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
					databaseChanges := &database.DatabaseChanges{}
					err := proto.Unmarshal(output.GetMapOutput().GetValue(), databaseChanges)
					if err != nil {
						return fmt.Errorf("unmarshalling database changes: %w", err)
					}
					err = applyDatabaseChanges(mongoDB, databaseChanges)
					if err != nil {
						return fmt.Errorf("applying database changes: %w", err)
					}
				}
			}
		}
	}
}

const PCS_DATABASE = "pcs"

type MongoDB struct {
	client *mongo.Client
}

func NewMongoDB(address string) (*MongoDB, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(address))
	if err != nil {
		return nil, err
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &MongoDB{client: client}, nil

}

func (db *MongoDB) SaveEntity(collectionName string, id string, entity map[string]interface{}) error {
	collection := db.client.Database(PCS_DATABASE).Collection(collectionName)
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	_, err := collection.InsertOne(ctx, entity)
	if err != nil {
		return err
	}
	return nil
}

func (db *MongoDB) Update(collectionName string, id string, changes map[string]interface{}) error {

	collection := db.client.Database(PCS_DATABASE).Collection(collectionName)
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	filter := bson.M{"id": id}
	update := bson.M{"$set": changes}
	_, err := collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return err
	}
	return nil
}

func applyDatabaseChanges(db *MongoDB, databaseChanges *database.DatabaseChanges) error {
	for _, change := range databaseChanges.TableChanges {
		id := change.Pk
		switch change.Operation {
		case database.TableChange_UNSET:
		case database.TableChange_CREATE:
			entity := map[string]interface{}{}
			for _, field := range change.Fields {
				//todo: convert value to the right type base on the graphql definition
				entity[field.Name] = field.NewValue
			}
			err := db.SaveEntity(change.Table, id, entity)
			if err != nil {
				return fmt.Errorf("saving entity %s with id %s: %w", change.Table, id, err)
			}
			fmt.Printf("saved entity %s with id %s:\n", change.Table, id)
		case database.TableChange_UPDATE:
			entityChanges := map[string]interface{}{}
			for _, field := range change.Fields {
				//todo: convert value to the right type base on the graphql definition
				entityChanges[field.Name] = field.NewValue
			}
			err := db.Update(change.Table, change.Pk, entityChanges)
			if err != nil {
				return fmt.Errorf("updating entity %s with id %s: %w", change.Table, id, err)
			}
			fmt.Printf("updating entity %s with id %s:\n", change.Table, id)
		case database.TableChange_DELETE:
		}
	}

	return nil
}
