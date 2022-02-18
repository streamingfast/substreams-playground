package exchange

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/streamingfast/sparkle-pancakeswap/state"
	"github.com/streamingfast/sparkle/entity"
	"go.uber.org/zap"
)

type PricesStateBuilder struct {
	*SubstreamIntrinsics
}

func (p *PricesStateBuilder) BuildState(reserveUpdates PCSReserveUpdates, pairs state.Reader, prices *state.Builder) error {
	// TODO: could we get rid of `pairs` as a dependency, by packaging `Token0.Address` directly in the `ReserveUpdate` ?

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

		// We should compute the price in here, rather than having that data flow through ReservesUpdates (that Token0Price computation)
		prices.Set(update.LogOrdinal, fmt.Sprintf("price:%s:%s", pair.Token0.Address, pair.Token1.Address), update.Token0Price) // TRIPLE CHECK that the Token0Price really corresponds to Token0 / Token1
		prices.Set(update.LogOrdinal, fmt.Sprintf("price:%s:%s", pair.Token1.Address, pair.Token0.Address), update.Token1Price) // TRIPLE CHECK that the Token1Price really corresponds to Token1 / Token0
		prices.Set(update.LogOrdinal, fmt.Sprintf("reserve0:%s", update.PairAddress), update.Reserve0)
		prices.Set(update.LogOrdinal, fmt.Sprintf("reserve1:%s", update.PairAddress), update.Reserve1)

		// HERE set: "reserve0bnb:%s", and fetch the Reserve0's price in BNB (from price:%s:bnb), and handle things if the price isn't there. DON'T write the key if we can't set a price. This will trickle down the "unset" value to where it belongs downstream.
		// HERE set: "reserve1bnb:%s", and fetch the Reserve1's price in BNB (from price:%s:bnb), and handle things if the price isn't there.
		reserve0BNB := p.setReserveInBNB(update.LogOrdinal, "reserve0", update.PairAddress, pair.Token0.Address, strToFloat(update.Reserve0), prices)
		reserve1BNB := p.setReserveInBNB(update.LogOrdinal, "reserve1", update.PairAddress, pair.Token1.Address, strToFloat(update.Reserve1), prices)

		reservesBNBSum := bf().Add(reserve0BNB, reserve1BNB)
		if reservesBNBSum.Cmp(bf()) != 0 {
			prices.Set(update.LogOrdinal, "reserves_bnb:"+update.PairAddress, floatToStr(reservesBNBSum))
		}

		if update.PairAddress == USDT_WBNB_PAIR || update.PairAddress == BUSD_WBNB_PAIR {
			newPrice := p.computeUSDPrice(update, prices)
			prices.Set(update.LogOrdinal, "price:usd:bnb", floatToStr(newPrice))
			//os.Exit(0)
			//log.Fatalln("stop!")
		}

		latestUSD := foundOrZeroFloat(prices.GetLast("price:usd:bnb"))

		t0DerivedBNBPrice := p.findBnbPricePerToken(update.LogOrdinal, pair.Token0.Address, pairs, prices)
		t1DerivedBNBPrice := p.findBnbPricePerToken(update.LogOrdinal, pair.Token1.Address, pairs, prices)
		t0DerivedUSDPrice := bf().Mul(t0DerivedBNBPrice, latestUSD)
		t1DerivedUSDPrice := bf().Mul(t1DerivedBNBPrice, latestUSD)

		prices.Set(update.LogOrdinal, fmt.Sprintf("price:%s:bnb", pair.Token0.Address), floatToStr(t0DerivedBNBPrice))
		prices.Set(update.LogOrdinal, fmt.Sprintf("price:%s:bnb", pair.Token1.Address), floatToStr(t1DerivedBNBPrice))
		prices.Set(update.LogOrdinal, fmt.Sprintf("price:%s:usd", pair.Token0.Address), floatToStr(t0DerivedUSDPrice))
		prices.Set(update.LogOrdinal, fmt.Sprintf("price:%s:usd", pair.Token1.Address), floatToStr(t1DerivedUSDPrice))
	}
	return nil
}

// findBnbPricePerToken provides a derived price multiplier from this token to BNB, transiting through trusted pairs.
func (p *PricesStateBuilder) findBnbPricePerToken(logOrdinal uint64, tokenAddr string, pairs state.Reader, prices state.Reader) *big.Float {
	if tokenAddr == WBNB_ADDRESS {
		return big.NewFloat(1) // BNB price of a BNB is always 1
	}

	// loop all whitelist for a matching pair
	for _, otherToken := range whitelist {
		pairAddr, found := pairs.GetAt(logOrdinal, "tokens:"+generateTokensKey(tokenAddr, otherToken))
		if !found {
			zlog.Debug("pair not found for tokens", zap.String("left", tokenAddr), zap.String("right", otherToken))
			continue
		}

		var pair *PCSPair
		pairData, _ := pairs.GetAt(logOrdinal, "pair:"+string(pairAddr))
		json.Unmarshal(pairData, pair)

		_ = otherToken

		val, found := prices.GetLast(fmt.Sprintf("reserves_bnb:%s", pairAddr))
		if !found {
			continue
		}
		if bytesToFloat(val).Ptr().Float().Cmp(MINIMUM_LIQUIDITY_THRESHOLD_BNB) <= 0 {
			continue
		}

		val1, found := prices.GetLast(fmt.Sprintf("price:%s:bnb", otherToken))
		if !found {
			continue
		}
		val2, found := prices.GetLast(fmt.Sprintf("price:%s:%s", tokenAddr, otherToken))
		if !found {
			continue
		}

		return entity.FloatMul(bytesToFloat(val1), bytesToFloat(val2)).Ptr().Float()

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

func (p *PricesStateBuilder) setReserveInBNB(ord uint64, reserveName string, pairAddr string, tokenAddr string, reserveAmount entity.Float, prices *state.Builder) (out *big.Float) {
	zero := bf()
	val, found := prices.GetLast(fmt.Sprintf("price:%s:bnb", tokenAddr))
	if !found {
		return zero
	}

	bnbPrice := strToFloat(string(val))
	bnbAmount := entity.FloatMul(bnbPrice, reserveAmount)

	out = bnbAmount.Ptr().Float()

	if out.Cmp(zero) != 0 {
		prices.Set(ord, fmt.Sprintf("%sbnb:%s", reserveName, pairAddr), floatToStr(out))
	}

	return out
}

const (
	BUSD_PRICE_KEY = "price:0xe9e7cea3dedca5984780bafc599bd69add087d56:0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
	USDT_PRICE_KEY = "price:0x55d398326f99059ff775485246999027b3197955:0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
)

func (p *PricesStateBuilder) computeUSDPrice(update PCSReserveUpdate, prices *state.Builder) *big.Float {
	busdBNBReserve := foundOrZeroFloat(prices.GetAt(update.LogOrdinal, fmt.Sprintf("reserve0:%s", BUSD_WBNB_PAIR)))
	usdtBNBReserve := foundOrZeroFloat(prices.GetAt(update.LogOrdinal, fmt.Sprintf("reserve1:%s", USDT_WBNB_PAIR)))
	totalLiquidityBNB := bf().Add(busdBNBReserve, usdtBNBReserve).SetPrec(100)

	zero := bf()

	if totalLiquidityBNB.Cmp(zero) == 0 {
		return big.NewFloat(0)
	}

	if busdBNBReserve.Cmp(zero) == 0 {
		fmt.Println("only usdt found")
		return foundOrZeroFloat(prices.GetAt(update.LogOrdinal, USDT_PRICE_KEY))
	} else if usdtBNBReserve.Cmp(zero) == 0 {
		fmt.Println("only busd found")
		return foundOrZeroFloat(prices.GetAt(update.LogOrdinal, BUSD_PRICE_KEY))
	}

	fmt.Println("both found")

	busdWeight := bf().Quo(busdBNBReserve, totalLiquidityBNB).SetPrec(100)
	usdtWeight := bf().Quo(usdtBNBReserve, totalLiquidityBNB).SetPrec(100)

	busdPrice := foundOrZeroFloat(prices.GetAt(update.LogOrdinal, BUSD_PRICE_KEY))
	usdtPrice := foundOrZeroFloat(prices.GetAt(update.LogOrdinal, USDT_PRICE_KEY))

	return bf().Add(
		bf().Mul(busdPrice, busdWeight).SetPrec(100),
		bf().Mul(usdtPrice, usdtWeight).SetPrec(100),
	).SetPrec(100)
}
