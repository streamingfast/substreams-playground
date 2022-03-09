package pcs

import (
	"encoding/json"

	imports "github.com/streamingfast/substreams/native-imports"
	"github.com/streamingfast/substreams/state"
)

type PairsStateBuilder struct {
	*imports.Imports
}

func NewPairsStateBuilder(imp *imports.Imports) *PairsStateBuilder { return &PairsStateBuilder{} }

// Process runs linearly, and consume all of the source `pairs` (from
// beginning of history, or as per some parameters to this
// PCSPairsStateBuilder).
//
// The output of Process is a series of StateDeltas, that are computed
// by calls to `Set` and `Del` on the intrinsically exposed `PairsState`.

// input: pbcodec.Block
// output: STATE (path-to-storage, unique ID for storage)
func (p *PairsStateBuilder) Store(pairs PCSPairs, pairsStore *state.Builder) error {
	for _, pair := range pairs {
		jsonContent, err := json.Marshal(pair)
		if err != nil {
			return err
		}

		pairsStore.SetBytes(pair.LogOrdinal, "pair:"+pair.Address, jsonContent)
		pairsStore.Set(pair.LogOrdinal, "tokens:"+generateTokensKey(pair.Token0.Address, pair.Token1.Address), pair.Address)
	}
	return nil
}
