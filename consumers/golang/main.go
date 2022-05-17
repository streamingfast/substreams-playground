package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/streamingfast/substreams/client"
	"github.com/streamingfast/substreams/decode"
	"github.com/streamingfast/substreams/manifest"
	pbsubstreams "github.com/streamingfast/substreams/pb/sf/substreams/v1"
)

func main() {
	pkg, err := manifest.New("https://github.com/streamingfast/substreams-playground/releases/download/v0.5.0/pcs-v0.5.0.spkg")
	errCheck("reading manifest", err)

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
		Modules:      pkg.Modules,
		OutputModules: []string{"block_to_tokens"},
	}

	stream, err := ssClient.Blocks(context.Background(), req, callOpts...)
	errCheck("creating stream", err)

	returnHandler := decode.NewPrintReturnHandler(pkg, []string{"block_to_tokens"}, true)

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
				if err := returnHandler(r.Data, nil); err != nil {
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
