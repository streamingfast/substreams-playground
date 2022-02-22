package exchange

import (
	"fmt"
	"sort"

	"github.com/streamingfast/sparkle-pancakeswap/state"
)

type TotalPairsStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *TotalPairsStateBuilder) BuildState(pairs PCSPairs, events PCSEvents /* burnEvents, mintEvents */, totalPairsStore *state.Builder) error {
	if len(pairs) == 0 && len(events) == 0 {
		return nil
	}

	var all []interface {
		GetOrdinal() uint64
	}
	for _, pair := range pairs {
		all = append(all, pair)
	}
	for _, ev := range events {
		all = append(all, ev)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].GetOrdinal() < all[j].GetOrdinal()
	})

	increment := func(key string, ord uint64) {
		count := foundOrZeroUint64(totalPairsStore.GetLast(key))
		count++
		totalPairsStore.Set(ord, key, fmt.Sprintf("%d", count))
	}

	for _, el := range all {
		switch ev := el.(type) {
		case *PCSSwap:
			increment(fmt.Sprintf("pair:%s:swaps", ev.PairAddress), ev.LogOrdinal)
		case *PCSBurn:
			increment(fmt.Sprintf("pair:%s:burns", ev.PairAddress), ev.LogOrdinal)
		case *PCSMint:
			increment(fmt.Sprintf("pair:%s:mints", ev.PairAddress), ev.LogOrdinal)
		case PCSPair:
			// This should move inside the `StateBuilder::` APIs, as `.AddUint64()` or something
			increment("pairs:count", ev.LogOrdinal)
		}
	}

	return nil
}
