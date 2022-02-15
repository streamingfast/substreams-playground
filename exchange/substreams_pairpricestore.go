package exchange

import (
	"encoding/json"
	"fmt"
)

type PCSReservesStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *PCSReservesStateBuilder) BuildState(reserveUpdates PCSReserveUpdates, builder *StateBuilder) error {

	for _, update := range reserveUpdates {
		cnt, err := json.Marshal(update)
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}

		builder.Set(update.LogOrdinal, update.PairAddress, cnt)


		if update.PairAddress == USDT_WBNB_PAIR {
			builder.Set(update.LogOrdinal, "usd", cnt)
		}


	}
	return nil
}
