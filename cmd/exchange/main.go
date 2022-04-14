package main

import (
	"context"
	"fmt"

	"github.com/streamingfast/substreams/graph-node/metrics"
	"github.com/streamingfast/substreams/graph-node/storage/postgres"
	"github.com/streamingfast/substreams/runtime"
)

func main() {
	ctx := context.Background()

	var dsn string
	var deployment string
	var schema string
	var withTransactions bool

	subgraphDef := Definition
	storage, err := postgres.New(zlog, metrics.NewBlockMetrics(), dsn, schema, deployment, subgraphDef, map[string]bool{}, withTransactions)
	if err != nil {
		panic(fmt.Errorf("creating postgres store: %w", err))
	}
	loader := NewLoader(storage, Definition.Entities)

	cfg := &runtime.LocalConfig{
		ReturnHandler: loader.ReturnHandler,
	}

	err = runtime.LocalRun(ctx, cfg)
	if err != nil {
		panic(err)
	}
}
