package exchange

import (
	"fmt"
	"sort"

	"github.com/streamingfast/sparkle-pancakeswap/state"
)

type PCSTotalPairsStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *PCSTotalPairsStateBuilder) BuildState(pairs PCSPairs, swapEvents Swaps /* burnEvents, mintEvents */, totalPairsStore *state.Builder) error {
	if len(pairs) == 0 && len(swapEvents) == 0 {
		return nil
	}

	var all []interface {
		GetOrdinal() uint64
	}
	for _, pair := range pairs {
		all = append(all, pair)
	}
	for _, swap := range swapEvents {
		all = append(all, swap)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].GetOrdinal() < all[j].GetOrdinal()
	})

	for _, el := range all {
		switch ev := el.(type) {
		case PCSSwap:
			key := fmt.Sprintf("pair:%s:swaps", ev.PairAddress)
			count := foundOrZeroUint64(totalPairsStore.GetLast(key))
			count++
			totalPairsStore.Set(ev.LogOrdinal, key, fmt.Sprintf("%d", count))
		case PCSPair:
			// This should move inside the `StateBuilder::` APIs, as `.AddUint64()` or something
			count := foundOrZeroUint64(totalPairsStore.GetLast("pairs:count"))
			count++
			totalPairsStore.Set(ev.LogOrdinal, "pairs:count", fmt.Sprintf("%d", count))
		}
	}

	return nil
}
