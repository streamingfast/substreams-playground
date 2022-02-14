package main

import (
	"github.com/streamingfast/sparkle/cli"
	"github.com/streamingfast/sparkle/subgraph"
	"github.com/streamingfast/sparkle-pancakeswap/exchange"
)

func main() {
	subgraph.MainSubgraphDef = exchange.Definition
	cli.Execute()
}
