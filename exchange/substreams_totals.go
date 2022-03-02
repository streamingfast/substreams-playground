package exchange

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/streamingfast/substream-pancakeswap/state"
)

type TotalsStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *TotalsStateBuilder) BuildState(pairs PCSPairs, events PCSEvents /* burnEvents, mintEvents */, totals state.IntegerDeltaWriter) error {
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

	one := big.NewInt(1)

	for _, el := range all {
		switch ev := el.(type) {
		case *PCSSwap:
			totals.AddInt(ev.LogOrdinal, fmt.Sprintf("pair:%s:swaps", ev.PairAddress), one)
		case *PCSBurn:
			totals.AddInt(ev.LogOrdinal, fmt.Sprintf("pair:%s:burns", ev.PairAddress), one)
		case *PCSMint:
			totals.AddInt(ev.LogOrdinal, fmt.Sprintf("pair:%s:mints", ev.PairAddress), one)
		case PCSPair:
			totals.AddInt(ev.LogOrdinal, "pairs", one)
		}
	}

	return nil
}
