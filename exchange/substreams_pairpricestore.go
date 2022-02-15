package exchange

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/streamingfast/sparkle/entity"
)

type PCSPricesStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *PCSPricesStateBuilder) BuildState(reserveUpdates PCSReserveUpdates, builder *StateBuilder) error {
	for _, update := range reserveUpdates {
		cnt, err := json.Marshal(update)
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}

		builder.Set(update.LogOrdinal, update.PairAddress, cnt)

		if update.PairAddress == USDT_WBNB_PAIR || update.PairAddress == BUSD_WBNB_PAIR {
			newPrice := p.computeUSDPrice(update, builder)
			builder.Set(update.LogOrdinal, "usd", []byte(newPrice.String()))
		}
	}
	return nil
}

func (p *PCSPricesStateBuilder) computeUSDPrice(update PCSReserveUpdate, state *StateBuilder) *big.Float {
	usdtPairData, usdtFound := state.GetAt(update.LogOrdinal, USDT_WBNB_PAIR) // usdt is token0
	busdPairData, busdFound := state.GetAt(update.LogOrdinal, BUSD_WBNB_PAIR) // busd is token1

	var busdPair, usdtPair PCSReserveUpdate

	if busdFound && usdtFound {
		orDie(json.Unmarshal(usdtPairData, &usdtPair))
		orDie(json.Unmarshal(busdPairData, &busdPair))

		busdBNBReserve := strToFloat(busdPair.Reserve0)
		usdtBNBReserve := strToFloat(usdtPair.Reserve1)
		totalLiquidityBNB := entity.FloatAdd(busdBNBReserve, usdtBNBReserve) // skipped `SetPrec(100)` here

		if totalLiquidityBNB.Float().Cmp(bf()) != 0 {
			busdWeight := entity.FloatQuo(busdBNBReserve, totalLiquidityBNB) // skipping `SetPrec(100)` here
			usdtWeight := entity.FloatQuo(usdtBNBReserve, totalLiquidityBNB) // skip `.SetPrec(100)`

			return bf().Add(
				bf().Mul(
					strToFloat(busdPair.Token1Price).Ptr().Float(),
					busdWeight.Ptr().Float(),
				).SetPrec(100),
				bf().Mul(
					strToFloat(usdtPair.Token0Price).Ptr().Float(),
					usdtWeight.Ptr().Float(),
				).SetPrec(100),
			).SetPrec(100)
		} else {
			return big.NewFloat(0)
		}
	} else if busdFound {
		orDie(json.Unmarshal(busdPairData, &busdPair))
		return strToFloat(busdPair.Token1Price).Ptr().Float() // skip `SetPrec(100)` here
	} else if usdtFound {
		orDie(json.Unmarshal(usdtPairData, &usdtPair))
		return strToFloat(usdtPair.Token0Price).Ptr().Float()
	}

	return big.NewFloat(0)
}

func orDie(err error) {
	if err != nil {
		panic("error: " + err.Error())
	}
}
