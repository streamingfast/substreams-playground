package exchange

import (
	"encoding/json"

	"github.com/streamingfast/sparkle-pancakeswap/state"
)

type PairsStateBuilder struct {
	*SubstreamIntrinsics
}

// Process runs linearly, and consume all of the source `pairs` (from
// beginning of history, or as per some parameters to this
// PCSPairsStateBuilder).
//
// The output of Process is a series of StateDeltas, that are computed
// by calls to `Set` and `Del` on the intrinsically exposed `PairsState`.

// input: pbcodec.Block
// output: STATE (path-to-storage, unique ID for storage)
func (p *PairsStateBuilder) BuildState(pairs PCSPairs, pairsStore *state.Builder) error {
	for _, pair := range pairs {
		cnt, err := json.Marshal(pair)
		if err != nil {
			return err
		}

		pairsStore.SetBytes(pair.LogOrdinal, "pair:"+pair.Address, cnt)
		pairsStore.Set(pair.LogOrdinal, "tokens:"+generateTokensKey(pair.Token0.Address, pair.Token1.Address), pair.Address)
	}
	return nil
}
