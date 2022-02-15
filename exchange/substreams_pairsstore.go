package exchange

import (
	"encoding/json"
)

type PCSPairsStateBuilder struct {
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
func (p *PCSPairsStateBuilder) Process(pairs PCSPairs, pairsStore *StateBuilder) error {
	for _, pair := range pairs {
		cnt, err := json.Marshal(pair)
		if err != nil {
			return err
		}

		pairsStore.Set(pair.LogOrdinal, pair.Address, cnt)
		pairsStore.Set(pair.LogOrdinal, generateTokensKey(pair.Token0.Address, pair.Token1.Address), []byte(pair.Address))
	}
	return nil
}
