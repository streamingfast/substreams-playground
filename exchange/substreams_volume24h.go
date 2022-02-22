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

func (p *PCSVolume24hStateBuilder) BuildState(block *pbcodec.Block, evs PCSEvents, volume24hStore *state.Builder) error {
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
	return nil
}

func increment(store *state.Builder, key string, ord uint64, amount *big.Float) {
	volume := foundOrZeroFloat(store.GetAt(ord, key))
	newVolume := bf().Add(volume, amount).SetPrec(100)
	store.Set(ord, key, floatToStr(newVolume))
}
