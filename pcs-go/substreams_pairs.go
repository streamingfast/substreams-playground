package pcs

import (
	imports "github.com/streamingfast/substreams/native-imports"
	"github.com/streamingfast/substreams/state"
	"google.golang.org/protobuf/proto"
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
func (p *PairsStateBuilder) Store(pairs Pairs, pairsStore *state.Builder) error {
	for _, pair := range pairs.Pairs {
		cnt, err := proto.Marshal(pair)
		if err != nil {
			return err
		}

		pairsStore.SetBytes(pair.LogOrdinal, "pair:"+pair.Address, cnt)
		pairsStore.Set(pair.LogOrdinal, "tokens:"+generateTokensKey(pair.Erc20Token0.Address, pair.Erc20Token1.Address), pair.Address)
	}
	return nil
}
