package exchange

import (
	"encoding/json"
	"fmt"

	"github.com/streamingfast/substream-pancakeswap/state"
	"go.uber.org/zap"
)

type ReservesStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *ReservesStateBuilder) BuildState(reserveUpdates PCSReserveUpdates, pairs state.Reader, reserves state.UpdateKeySetter) error {
	for _, update := range reserveUpdates {
		// TODO: cache those pairs we've already decoded in this `BuildState` run
		var pair *PCSPair
		pairData, found := pairs.GetLast("pair:" + update.PairAddress)
		if !found {
			zlog.Warn("pair not found for a reserve update!", zap.String("pair", update.PairAddress))
			continue
		}
		if err := json.Unmarshal(pairData, &pair); err != nil {
			return fmt.Errorf("decoding pair: %w", err)
		}
		reserves.Set(update.LogOrdinal, fmt.Sprintf("price:%s:%s", pair.Token0.Address, pair.Token1.Address), update.Token0Price) // TRIPLE CHECK that the Token0Price really corresponds to Token0 / Token1
		reserves.Set(update.LogOrdinal, fmt.Sprintf("price:%s:%s", pair.Token1.Address, pair.Token0.Address), update.Token1Price) // TRIPLE CHECK that the Token1Price really corresponds to Token1 / Token0

		reserves.Set(update.LogOrdinal, fmt.Sprintf("reserve:%s:%s", update.PairAddress, pair.Token0.Address), update.Reserve0)
		reserves.Set(update.LogOrdinal, fmt.Sprintf("reserve:%s:%s", update.PairAddress, pair.Token1.Address), update.Reserve1)
	}
	return nil
}
