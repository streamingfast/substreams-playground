package exchange

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/streamingfast/sparkle/entity"
	"go.uber.org/zap"
)

type PCSPricesStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *PCSPricesStateBuilder) BuildState(reserveUpdates PCSReserveUpdates, pairs StateReader, builder *StateBuilder) error {
	// TODO: could we get rid of `pairs` as a dependency, by packaging `Token0.Address` directly in the `ReserveUpdate` ?

	for _, update := range reserveUpdates {
		// TODO: cache those pairs we've already decoded in this `BuildState` run
		var pair *PCSPair
		pairData, found := pairs.GetLast(update.PairAddress)
		if !found {
			zlog.Warn("pair not found for a reserve update!", zap.String("pair", update.PairAddress))
			continue
		}
		if err := json.Unmarshal(pairData, &pair); err != nil {
			return fmt.Errorf("decoding pair: %w", err)
		}

		// We should compute the price in here, rather than having that data flow through ReservesUpdates (that Token0Price computation)
		builder.Set(update.LogOrdinal, fmt.Sprintf("price:%s:%s", pair.Token0.Address, pair.Token1.Address), []byte(update.Token0Price)) // TRIPLE CHECK that the Token0Price really corresponds to Token0 / Token1
		builder.Set(update.LogOrdinal, fmt.Sprintf("price:%s:%s", pair.Token1.Address, pair.Token0.Address), []byte(update.Token1Price)) // TRIPLE CHECK that the Token1Price really corresponds to Token1 / Token0
		builder.Set(update.LogOrdinal, fmt.Sprintf("reserve0:%s", pair.Address), []byte(update.Reserve0))
		builder.Set(update.LogOrdinal, fmt.Sprintf("reserve1:%s", pair.Address), []byte(update.Reserve1))

		// HERE set: "reserve0bnb:%s", and fetch the Reserve0's price in BNB (from price:%s:bnb), and handle things if the price isn't there. DON'T write the key if we can't set a price. This will trickle down the "unset" value to where it belongs downstream.
		// HERE set: "reserve1bnb:%s", and fetch the Reserve1's price in BNB (from price:%s:bnb), and handle things if the price isn't there.

		if update.PairAddress == USDT_WBNB_PAIR || update.PairAddress == BUSD_WBNB_PAIR {
			newPrice := p.computeUSDPrice(update, builder)
			builder.Set(update.LogOrdinal, "price:usd:bnb", []byte(newPrice.String()))
		}

		var latestUSD *big.Float
		latestUSDData, found := builder.GetLast("price:usd:bnb")
		if !found {
			latestUSD = bf()
		} else {
			latestUSD = strToFloat(string(latestUSDData)).Ptr().Float()
		}

		t0DerivedBNB := p.findBnbPricePerToken(update.LogOrdinal, pair.Token0.Address)
		t1DerivedBNB := p.findBnbPricePerToken(update.LogOrdinal, pair.Token1.Address)
		t0DerivedUSD := bf().Mul(t0DerivedBNB, latestUSD)
		t1DerivedUSD := bf().Mul(t1DerivedBNB, latestUSD)

		builder.Set(update.LogOrdinal, fmt.Sprintf("price:%s:bnb", pair.Token0.Address), []byte(floatToStr(t0DerivedBNB)))
		builder.Set(update.LogOrdinal, fmt.Sprintf("price:%s:bnb", pair.Token1.Address), []byte(floatToStr(t1DerivedBNB)))
		builder.Set(update.LogOrdinal, fmt.Sprintf("price:%s:usd", pair.Token0.Address), []byte(floatToStr(t0DerivedUSD)))
		builder.Set(update.LogOrdinal, fmt.Sprintf("price:%s:usd", pair.Token1.Address), []byte(floatToStr(t1DerivedUSD)))
	}
	return nil
}

func (p *PCSPricesStateBuilder) findBnbPricePerToken(logOrdinal uint64, tokenAddr string) *big.Float {
	if tokenAddr == WBNB_ADDRESS {
		return big.NewFloat(1) // BNB price of a BNB is always 1
	}

	// loop all whitelist for a matching pair
	for _, otherToken := range whitelist {
		_ = otherToken
		// 		// pairAddress, err := GetAt(logOrdinal, generateTokensKey(tokenAddress, otherToken))
		// // if err == NotFound {
		// //     continue // check for other..
		// // }
		// pairAddress := s.getPairAddressForTokens(tokenAddress, otherToken)
		// if pairAddress == "" {
		// 	s.Log.Debug("pair not found for tokens", zap.String("left", tokenAddress), zap.String("right", otherToken))
		// 	continue
		// }

		// pair := NewPair(pairAddress)
		// if err := s.Load(pair); err != nil {
		// 	return nil, err
		// }

		// // get pair WBNB + pair.PairAddress, get its pair, and its price?!

		// PROBLEM: WHO COMPUTES RESERVEBNB?!? Isn't that looping on your own head?
		// It requires that we have processed the reserves, and marked them as BNB
		// Handle things if that ReserveBNB key isn't present.

		// if pair.Token0 == tokenAddress && pair.ReserveBNB.Float().Cmp(MINIMUM_LIQUIDITY_THRESHOLD_BNB) > 0 {
		// 	token1 := NewToken(pair.Token1)
		// 	if err := s.Load(token1); err != nil {
		// 		return nil, err
		// 	}
		// 	return bf().Mul(pair.Token1Price.Float(), token1.DerivedBNB.Float()), nil
		// }
		// if pair.Token1 == tokenAddress && pair.ReserveBNB.Float().Cmp(MINIMUM_LIQUIDITY_THRESHOLD_BNB) > 0 {
		// 	token0 := NewToken(pair.Token0)
		// 	if err := s.Load(token0); err != nil {
		// 		return nil, err
		// 	}
		// 	return bf().Mul(pair.Token0Price.Float(), token0.DerivedBNB.Float()), nil
		// }
	}

	return bf()

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
