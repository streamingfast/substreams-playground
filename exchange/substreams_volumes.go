package exchange

import (
	"fmt"

	"github.com/streamingfast/substream-pancakeswap/state"

	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
)

type PCSVolume24hStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *PCSVolume24hStateBuilder) BuildState(block *pbcodec.Block, evs PCSEvents, volumes state.SumBigFloatSetter) error {
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

		// Get("day") // "12312" == currentDayId

		volumes.SumBigFloat(swap.LogOrdinal, fmt.Sprintf("pairs:%s:%d", swap.PairAddress, dayId), amountUSD)
		volumes.SumBigFloat(swap.LogOrdinal, fmt.Sprintf("token:%s:%d", swap.Token0, dayId), amountUSD)
		volumes.SumBigFloat(swap.LogOrdinal, fmt.Sprintf("token:%s:%d", swap.Token1, dayId), amountUSD)

		// volume24hStore.SetExpireBlock(dayPairId, block.Number + 1000)
		// "_db:fffffffee:REV_BLOCK_NUM:recevier:%s%:%d" -> ""
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
	//
	// WARN: if we automatically merge the files, the Deletions risk
	// not being deterministic (although we could still do proper clean-up
	// and keep memory low, and things would continue working).
	// * Maybe we want to think of a deterministic way to do clean-up
	//   based on some conventions, or with fixed _:delete keys or something
	//   or some concepts of TTL, that the mergeStrategy could honor
	//
	// volume24hStore.DelPrefix(prefix)
	// volume24hStore.DelPrefixPointers(prefix, keySeparator) // reads the key, and deletes keys that are stored in the value, with a `keySeparator`
	// volume24hStore.DelScan(lowKey, highKey)
	//
	// TO EASE in stores merging, we could STORE the DelPrefix, or range
	// deletions that happen in a PARTIAL store, as special keys like:
	// _:del_range:value1:value2 => ""
	// this way we could apply it to the previously squashed store, and clean
	// up keys that would have been cleaned-up had we been linear.
	// Once they are applied, we can delete them from the "absolute" store.
	return nil
}
