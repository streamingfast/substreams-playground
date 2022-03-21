package pcs

import (
	"fmt"

	pbcodec "github.com/streamingfast/substream-pancakeswap/pb/sf/ethereum/codec/v1"
	"github.com/streamingfast/substreams/state"
)

type PCSVolume24hStateBuilder struct{}

func (p *PCSVolume24hStateBuilder) Store(block *pbcodec.Block, evs *Events, volumes state.SumBigFloatSetter) error {
	timestamp := block.MustTime().Unix()
	dayId := timestamp / 86400
	//prevDayId := dayId - 1
	//dayStartTimestamp := dayId * 86400, downstream can compute it

	if evs == nil {
		return nil
	}
	for _, ev := range evs.Events {
		swap, ok := ev.Type.(*Event_Swap)
		if !ok {
			continue
		}
		if swap.Swap.AmountUsd == "" {
			continue
		}
		amountUSD := strToFloat(swap.Swap.AmountUsd)
		if amountUSD.Cmp(bf()) == 0 {
			continue
		}

		volumes.SumBigFloat(ev.LogOrdinal, fmt.Sprintf("pairs:%d:%s", dayId, ev.PairAddress), amountUSD)
		volumes.SumBigFloat(ev.LogOrdinal, fmt.Sprintf("token:%d:%s", dayId, ev.Token0), amountUSD)
		volumes.SumBigFloat(ev.LogOrdinal, fmt.Sprintf("token:%d:%s", dayId, ev.Token1), amountUSD)
	}

	// volumes.DeletePrefix("pairs:%d", dayId-1)
	// volumes.DeletePrefix("token:%d", dayId-1)

	return nil
}
