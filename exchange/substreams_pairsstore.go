package exchange

import (
	"encoding/json"

	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
)

type PCSPairsStore struct {
	*SubstreamIntrinsics

	PairsStore *StateBuilder
}

// Process runs linearly, and consume all of the source `pairs` (from
// beginning of history, or as per some parameters to this
// PCSPairsStore).
//
// The output of Process is a series of StateDeltas, that are computed
// by calls to `Set` and `Del` on the intrinsically exposed `PairsState`.


// input: pbcodec.Block
// output: STATE (path-to-storage, unique ID for storage)
func (p *PCSPairsStore) Process(block *pbcodec.Block, pairs PCSPairs) error {
	for _, pair := range pairs {
		cnt, err := json.Marshal(pair)
		if err != nil {
			return err
		}

		p.PairsStore.Set(pair.LogOrdinal, pair.Address, cnt)
	}
	return nil
}
