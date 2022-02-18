package exchange

import (
	"fmt"

	"github.com/streamingfast/sparkle-pancakeswap/state"

	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
)

type PCSVolume24hStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *PCSVolume24hStateBuilder) BuildState(block *pbcodec.Block, swaps Swaps, volume24hStore *state.Builder) error {
	timestamp := block.MustTime().Unix()
	dayId := timestamp / 86400
	//prevDayId := dayId - 1
	//dayStartTimestamp := dayId * 86400, downstream can compute it

	for _, swap := range swaps {
		dayPairId := fmt.Sprintf("%s-%d", swap.PairAddress, dayId)

		volume := foundOrZeroFloat(volume24hStore.GetAt(swap.LogOrdinal, dayPairId))
		amountUSD := strToFloat(swap.AmountUSD).Ptr().Float().SetPrec(100)
		newVolume := bf().Add(volume, amountUSD).SetPrec(100)

		volume24hStore.Set(swap.LogOrdinal, dayPairId, floatToStr(newVolume))
		// volume24hStore.SetExpireBlock(dayPairId, block.Number + 1000)
		// "_db:REV_BLOCK_NUM:key" -> ""
		// "_dt:REV_TIMESTAMP:key" -> ""
		// volume24hStore.SetExpireSeconds(dayPairId, 86400)

		// timestamp := block.Timestamp()
		// ttl := timestamp.Add(-2 * 86400 * time.Second)
		// volume24hStore.Set(swap.LogOrdinal, "delete-key-%d", dayId fmt.Sprintf("%s %s", ttl, dayPairId))
	}

	// Each 3 days, we clean-up all the keys
	// deleteme-at-[...] computed based on something
	// for _, deleteKey := range volume24hStore.Prefix("delete-key") {
	// }
	return nil
}
