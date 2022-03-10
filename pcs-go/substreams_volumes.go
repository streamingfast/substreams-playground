package pcs

import (
	"fmt"

	pbcodec "github.com/streamingfast/substream-pancakeswap/pb/sf/ethereum/codec/v1"
	"github.com/streamingfast/substreams/state"
)

type PCSVolume24hStateBuilder struct{}

func (p *PCSVolume24hStateBuilder) Store(block *pbcodec.Block, evs PCSEvents, volumes state.SumBigFloatSetter) error {
	timestamp := block.MustTime().Unix()
	dayId := timestamp / 86400
	//prevDayId := dayId - 1
	//dayStartTimestamp := dayId * 86400, downstream can compute it

	for _, ev := range evs {
		swap, ok := ev.(*PCSSwap)
		if !ok {
			continue
		}
		if swap.AmountUSD == "" {
			continue
		}
		amountUSD := strToFloat(swap.AmountUSD)
		if amountUSD.Cmp(bf()) == 0 {
			continue
		}

		volumes.SumBigFloat(swap.LogOrdinal, fmt.Sprintf("pairs:%d:%s", dayId, swap.PairAddress), amountUSD)
		volumes.SumBigFloat(swap.LogOrdinal, fmt.Sprintf("token:%d:%s", dayId, swap.Token0), amountUSD)
		volumes.SumBigFloat(swap.LogOrdinal, fmt.Sprintf("token:%d:%s", dayId, swap.Token1), amountUSD)
	}

	// volumes.DeletePrefix("pairs:%d", dayId-1)
	// volumes.DeletePrefix("token:%d", dayId-1)

	return nil
}
