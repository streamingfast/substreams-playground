package pcs

import (
	"fmt"
	"sort"

	"github.com/streamingfast/substreams/state"
)

type TotalsStateBuilder struct {
}

func (p *TotalsStateBuilder) BuildState(pairs *Pairs, events *Events /* burnEvents, mintEvents */, totals state.SumInt64Setter) error {
	if (pairs == nil || len(pairs.Pairs) == 0) && (events == nil || len(events.Events) == 0) {
		return nil
	}

	var all []interface {
		GetOrdinal() uint64
	}
	if pairs != nil {
		for _, pair := range pairs.Pairs {
			all = append(all, pair)
		}
	}
	if events != nil {
		for _, ev := range events.Events {
			all = append(all, ev)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].GetOrdinal() < all[j].GetOrdinal()
	})

	for _, el := range all {
		switch ev := el.(type) {
		case *Event:
			switch ev.Type.(type) {
			case *Event_Swap:
				totals.SumInt64(ev.LogOrdinal, fmt.Sprintf("pair:%s:swaps", ev.PairAddress), 1)
			case *Event_Burn:
				totals.SumInt64(ev.LogOrdinal, fmt.Sprintf("pair:%s:burns", ev.PairAddress), 1)
			case *Event_Mint:
				totals.SumInt64(ev.LogOrdinal, fmt.Sprintf("pair:%s:mints", ev.PairAddress), 1)
			}
		case *Pair:
			totals.SumInt64(ev.LogOrdinal, "pairs", 1)
		}
	}

	return nil
}
