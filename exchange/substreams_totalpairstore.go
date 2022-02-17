package exchange

import (
	"encoding/json"
	"fmt"
	"github.com/streamingfast/sparkle-pancakeswap/state"
)

type PCSTotalPairsStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *PCSTotalPairsStateBuilder) BuildState(pairs PCSPairs, totalPairsStore *state.Builder) error {
	if len(pairs) == 0 {
		return nil
	}
	count := 0
	lastData, found := totalPairsStore.GetLast("count")
	if found {
		if err := json.Unmarshal(lastData, &count); err != nil {
			return fmt.Errorf("unmarshalling last total pair count: %w", err)
		}
	}
	for _, pair := range pairs {
		count++
		data, err := json.Marshal(count)
		if err != nil {
			return err
		}

		totalPairsStore.Set(pair.LogOrdinal, "count", data)
	}
	return nil
}
