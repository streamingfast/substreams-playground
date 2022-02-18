package exchange

import (
	"fmt"
	"math/big"

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
		if swap.AmountUSD == "" {
			continue
		}
		amountUSD := strToFloat(swap.AmountUSD).Ptr().Float().SetPrec(100)
		if amountUSD.Cmp(bf()) == 0 {
			continue
		}

		increment(volume24hStore, fmt.Sprintf("pair:%s:%d", swap.PairAddress, dayId), swap.LogOrdinal, amountUSD)
		increment(volume24hStore, fmt.Sprintf("token:%s:%d", swap.Token0, dayId), swap.LogOrdinal, amountUSD)
		increment(volume24hStore, fmt.Sprintf("token:%s:%d", swap.Token1, dayId), swap.LogOrdinal, amountUSD)
		increment(volume24hStore, fmt.Sprintf("sender:%s:%d", swap.Sender, dayId), swap.LogOrdinal, amountUSD)
		increment(volume24hStore, fmt.Sprintf("receiver:%s:%d", swap.To, dayId), swap.LogOrdinal, amountUSD)

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

func increment(store *state.Builder, key string, ord uint64, amount *big.Float) {
	volume := foundOrZeroFloat(store.GetAt(ord, key))
	newVolume := bf().Add(volume, amount).SetPrec(100)
	store.Set(ord, key, floatToStr(newVolume))
}
