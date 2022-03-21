package pcs

import (
	"fmt"

	"github.com/streamingfast/substreams/state"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type ReservesStateBuilder struct{}

func (p *ReservesStateBuilder) Store(reserveUpdates *Reserves, pairs state.Reader, reserves state.UpdateKeySetter) error {
	if reserveUpdates == nil {
		return nil
	}
	for _, update := range reserveUpdates.Reserves {
		// TODO: cache those pairs we've already decoded in this `Store` run
		pairData, found := pairs.GetLast("pair:" + update.PairAddress)
		if !found {
			zlog.Warn("pair not found for a reserve update!", zap.String("pair", update.PairAddress))
			continue
		}
		pair := &Pair{}
		if err := proto.Unmarshal(pairData, pair); err != nil {
			return fmt.Errorf("decoding pair: %w", err)
		}
		reserves.Set(update.LogOrdinal, fmt.Sprintf("price:%s:%s", pair.Erc20Token0.Address, pair.Erc20Token1.Address), update.Token0Price) // TRIPLE CHECK that the Token0Price really corresponds to Token0 / Token1
		reserves.Set(update.LogOrdinal, fmt.Sprintf("price:%s:%s", pair.Erc20Token1.Address, pair.Erc20Token0.Address), update.Token1Price) // TRIPLE CHECK that the Token1Price really corresponds to Token1 / Token0

		reserves.Set(update.LogOrdinal, fmt.Sprintf("reserve:%s:%s", update.PairAddress, pair.Erc20Token0.Address), update.Reserve0)
		reserves.Set(update.LogOrdinal, fmt.Sprintf("reserve:%s:%s", update.PairAddress, pair.Erc20Token1.Address), update.Reserve1)
	}
	return nil
}
