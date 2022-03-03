package main

import (
	"reflect"

	_ "github.com/streamingfast/substream-pancakeswap/codec"

	pcs "github.com/streamingfast/substream-pancakeswap/pcs-go"
	"github.com/streamingfast/substreams/cli"
	imports "github.com/streamingfast/substreams/native-imports"
	"github.com/streamingfast/substreams/registry"
)

func main() {
	registry.Register("pcs_pair_extractor", func(imp *imports.Imports) reflect.Value {
		return reflect.ValueOf(&pcs.PairExtractor{Imports: imp})
	})
	registry.Register("pcs_pairs_state_builder", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(pcs.PairsStateBuilder{}) })
	registry.Register("pcs_reserves_extractor", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.ReservesExtractor{}) })
	registry.Register("pcs_reserves_state_builder", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.ReservesStateBuilder{}) })
	registry.Register("pcs_derived_prices_state_builder", func(imp *imports.Imports) reflect.Value {
		return reflect.ValueOf(&pcs.DerivedPricesStateBuilder{})
	})
	registry.Register("pcs_mint_burn_swaps_extractor", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.SwapsExtractor{}) })
	registry.Register("pcs_totals_state_builder", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.TotalsStateBuilder{}) })
	registry.Register("pcs_volumes_state_builder", func(imp *imports.Imports) reflect.Value {
		return reflect.ValueOf(&pcs.PCSVolume24hStateBuilder{})
	})

	cli.ProtobufBlockType = "sf.ethereum.codec.v1.Block"

	cli.Main()
}
