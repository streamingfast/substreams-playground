package exchange

import (
	"encoding/json"
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
	dayStartTimestamp := dayId * 86400

	for _, swap := range swaps {
		dayPairId := fmt.Sprintf("%s-%d", swap.PairAddress, dayId)

		prevVolData, found := volume24hStore.GetAt(swap.LogOrdinal, dayPairId)
		var volume *VolumeAggregate
		if !found {
			volume = &VolumeAggregate{
				Pair:      swap.PairAddress,
				Date:      dayStartTimestamp,
				VolumeUSD: 0,
			}
		} else {
			if err := json.Unmarshal(prevVolData, volume); err != nil {
				return fmt.Errorf("unmarshal prev vol: %w", err)
			}
		}

		amountUSD, _ := strToFloat(swap.AmountUSD).Ptr().Float().Float64()
		volume.VolumeUSD += amountUSD

		volumeData, err := json.Marshal(volume)
		if err != nil {
			return fmt.Errorf("volume marshal: %w", err)
		}

		volume24hStore.Set(swap.LogOrdinal, dayPairId, volumeData)

		// timestamp := block.Timestamp()
		// ttl := timestamp.Add(-2 * 86400 * time.Second)
		// volume24hStore.Set(swap.LogOrdinal, "delete-key", fmt.Sprintf("%s %s", ttl, dayPairId))
	}

	// Each 3 days, we clean-up all the keys
	// deleteme-at-[...] computed based on something
	// for _, deleteKey := range volume24hStore.Prefix("delete-key") {
	// }
	return nil
}
