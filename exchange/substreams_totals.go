package exchange

import (
	"fmt"
	"sort"

	"github.com/streamingfast/substream-pancakeswap/state"
)

type TotalsStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *TotalsStateBuilder) BuildState(pairs PCSPairs, events PCSEvents /* burnEvents, mintEvents */, totals state.SumInt64Setter) error {
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

	for _, el := range all {
		switch ev := el.(type) {
		case *PCSSwap:
			totals.SumInt64(ev.LogOrdinal, fmt.Sprintf("pair:%s:swaps", ev.PairAddress), 1)
		case *PCSBurn:
			totals.SumInt64(ev.LogOrdinal, fmt.Sprintf("pair:%s:burns", ev.PairAddress), 1)
		case *PCSMint:
			totals.SumInt64(ev.LogOrdinal, fmt.Sprintf("pair:%s:mints", ev.PairAddress), 1)
		case PCSPair:
			totals.SumInt64(ev.LogOrdinal, "pairs", 1)
		}
	}

	return nil
}
