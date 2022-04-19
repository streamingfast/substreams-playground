package main

import (
	"context"
	"fmt"

	_ "github.com/streamingfast/sf-ethereum/codec"
	"github.com/streamingfast/substreams/graph-node/metrics"
	"github.com/streamingfast/substreams/graph-node/storage/postgres"
	"github.com/streamingfast/substreams/runtime"
)

func main() {
	ctx := context.Background()

	dsn := "postgresql://graph:secureme@localhost:5432/graph?enable_incremental_sort=off&sslmode=disable"
	deployment := "deployment.1"
	schema := "toto"
	withTransactions := true

	subgraphDef := Definition
	storage, err := postgres.New(zlog, metrics.NewBlockMetrics(), dsn, schema, deployment, subgraphDef, map[string]bool{}, withTransactions)
	if err != nil {
		panic(fmt.Errorf("creating postgres store: %w", err))
	}
	err = storage.RegisterEntities()
	if err != nil {
		panic(fmt.Errorf("store: registaring entities:%w", err))
	}
	loader := NewLoader(storage, Definition.Entities)

	cfg := &runtime.LocalConfig{
		ManifestPath:      "/Users/cbillett/devel/sf/substream-playground/wasm_substreams_manifest.yaml",
		OutputStreamName:  "db_out",
		BlocksStoreUrl:    "/Users/cbillett/devel/sf/substream-playground/localblocks",
		StateStoreUrl:     "/Users/cbillett/devel/sf/substream-playground/localdata",
		IrrIndexesUrl:     "/Users/cbillett/devel/sf/substream-playground/localirr",
		ProtobufBlockType: "sf.ethereum.type.v1.Block",
		StartBlock:        6_800_000,
		StopBlock:         6_900_000,
		RpcEndpoint:       "https://summer-bitter-snow.bsc.quiknode.pro/b8359f47150c2079bc571dc5d107506043d191fd/",
		PrintMermaid:      false,
		RpcCacheUrl:       "gs://dfuseio-global-blocks-us/eth-bsc-mainnet/rpc-cache",
		PartialMode:       false,
		ReturnHandler:     loader.ReturnHandler,
	}

	err = runtime.LocalRun(ctx, cfg)
	if err != nil {
		panic(err)
	}
}
