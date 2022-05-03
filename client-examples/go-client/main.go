package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/streamingfast/substreams/client"
	"github.com/streamingfast/substreams/decode"
	"github.com/streamingfast/substreams/manifest"
	pbsubstreams "github.com/streamingfast/substreams/pb/sf/substreams/v1"
)

func main() {
	manif, err := manifest.New("../../eth-token/substreams-eth-token.yaml")

	errCheck("reading manifest", err)

	manifProto, err := manif.ToProto()
	errCheck("converting manifest to protobuf", err)

	ssClient, callOpts, err := client.NewSubstreamsClient(
		"bsc-dev.streamingfast.io:443",
		os.Getenv("SUBSTREAMS_API_TOKEN"),
		false, false,
	)
	errCheck("creating substreams client", err)

	req := &pbsubstreams.Request{
		StartBlockNum: 100_000,
		StopBlockNum:  200_000,
		ForkSteps:     []pbsubstreams.ForkStep{pbsubstreams.ForkStep_STEP_IRREVERSIBLE},
		Manifest:      manifProto,
		OutputModules: []string{"block_to_tokens"},
	}

	stream, err := ssClient.Blocks(context.Background(), req, callOpts...)
	errCheck("creating stream", err)

	parser := protoparse.Parser{}
	fileDescs, err := parser.ParseFiles("../../eth-token/proto/tokens.proto")
	errCheck("loading proto files", err)

	returnHandler := decode.NewPrintReturnHandler(manif, fileDescs, []string{"block_to_tokens"})

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				os.Exit(0)
			}
			errCheck("receiving response", err)
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
					fmt.Println("Remove log: ", log)
				}
				if err := returnHandler(r.Data); err != nil {
					fmt.Printf("RETURN HANDLER ERROR: %s\n", err)
				}
			}
		}
	}

}

func errCheck(message string, err error) {
	if err != nil {
		fmt.Println(message, ":", err.Error())
		os.Exit(1)
	}
}
