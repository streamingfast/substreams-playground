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
	registry.Register("pcs_pair_extractor", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.PairExtractor{Imports: imp}) })               // done
	registry.Register("pcs_pairs_state_builder", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.PairsStateBuilder{}) })                  // done
	registry.Register("pcs_reserves_extractor", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.ReservesExtractor{}) })                   // todo
	registry.Register("pcs_reserves_state_builder", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.ReservesStateBuilder{}) })            // todo
	registry.Register("pcs_derived_prices_state_builder", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.DerivedPricesStateBuilder{}) }) // todo
	registry.Register("pcs_mint_burn_swaps_extractor", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.SwapsExtractor{}) })               // todo
	registry.Register("pcs_totals_state_builder", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.TotalsStateBuilder{}) })                // todo
	registry.Register("pcs_volumes_state_builder", func(imp *imports.Imports) reflect.Value { return reflect.ValueOf(&pcs.PCSVolume24hStateBuilder{}) })         // todo

	cli.ProtobufBlockType = "sf.ethereum.type.v1.Block"

	cli.Main()
}
