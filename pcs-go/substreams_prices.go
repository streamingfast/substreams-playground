package pcs

import (
	"fmt"
	"math/big"

	"github.com/streamingfast/substreams/state"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type DerivedPricesStateBuilder struct{}

func (p *DerivedPricesStateBuilder) Store(reserveUpdates Reserves, pairs state.Reader, reserves state.Reader, derivedPrices *state.Builder) error {
	// TODO: could we get rid of `pairs` as a dependency, by packaging `Token0.Address` directly in the `ReserveUpdate` ?

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

		// sets:
		// * dprice:usd:bnb
		// derived from:
		// * reserve:%s:%s (pair, token)
		//
		// When you set one, you don't have the other, so you need to pass after the `reserve0` and `reserve1` have been set for those two pairs.
		latestUSDPrice := p.computeUSDPrice(update, reserves)
		if update.PairAddress == USDT_WBNB_PAIR || update.PairAddress == BUSD_WBNB_PAIR {
			derivedPrices.Set(update.LogOrdinal, "dprice:usd:bnb", floatToStr(latestUSDPrice))
		}

		// sets:
		// * dprice:%s:bnb (tokenA)  - as contributed by any pair's sync to that token
		// * dprice:%s:usd (tokenA)  - same
		// * dreserve:%s:%s:bnb (pair, token)
		// * dreserve:%s:%s:usd (pair, token)
		// * dreserves:%s:bnb (pair)  - sum of both token's reserves
		// derived from:
		// * price:%s:%s (tokenA, tokenB)
		// * reserve:%s:%s (pair, tokenA)
		usdPriceValid := latestUSDPrice.Cmp(bf()) != 0

		t0DerivedBNBPrice := p.findBnbPricePerToken(update.LogOrdinal, pair.Erc20Token0.Address, pairs, reserves)
		t1DerivedBNBPrice := p.findBnbPricePerToken(update.LogOrdinal, pair.Erc20Token1.Address, pairs, reserves)

		apply := func(derivedBNBPrice *big.Float, tokenAddr string, reserveAmount string) *big.Float {
			if derivedBNBPrice != nil {
				derivedPrices.Set(update.LogOrdinal, fmt.Sprintf("dprice:%s:bnb", tokenAddr), floatToStr(derivedBNBPrice))
				reserveInBNB := bf().Mul(strToFloat(reserveAmount), derivedBNBPrice)
				derivedPrices.Set(update.LogOrdinal, fmt.Sprintf("dreserve:%s:%s:bnb", update.PairAddress, tokenAddr), floatToStr(reserveInBNB))
				if usdPriceValid {
					derivedUSDPrice := bf().Mul(derivedBNBPrice, latestUSDPrice)
					derivedPrices.Set(update.LogOrdinal, fmt.Sprintf("dprice:%s:usd", tokenAddr), floatToStr(derivedUSDPrice))
					reserveInUSD := bf().Mul(reserveInBNB, latestUSDPrice)
					derivedPrices.Set(update.LogOrdinal, fmt.Sprintf("dreserve:%s:%s:usd", update.PairAddress, tokenAddr), floatToStr(reserveInUSD))
				}
				return reserveInBNB
			}
			return bf()
		}
		reserve0BNB := apply(t0DerivedBNBPrice, pair.Erc20Token0.Address, update.Reserve0)
		reserve1BNB := apply(t1DerivedBNBPrice, pair.Erc20Token1.Address, update.Reserve1)
		reservesBNBSum := bf().Add(reserve0BNB, reserve1BNB)
		if reservesBNBSum.Cmp(bf()) != 0 {
			derivedPrices.Set(update.LogOrdinal, fmt.Sprintf("dreserves:%s:bnb", update.PairAddress), floatToStr(reservesBNBSum))
		}
	}
	return nil
}

const (
	BUSD_PRICE_KEY = "price:0xe9e7cea3dedca5984780bafc599bd69add087d56:0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
	USDT_PRICE_KEY = "price:0x55d398326f99059ff775485246999027b3197955:0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
)

func (p *DerivedPricesStateBuilder) computeUSDPrice(update *Reserve, reserves state.Reader) *big.Float {
	// SAME PROBLEM of READING from the state store you're building.
	busdBNBReserve := foundOrZeroFloat(reserves.GetAt(update.LogOrdinal, fmt.Sprintf("reserve:%s:%s", BUSD_WBNB_PAIR, WBNB_ADDRESS)))
	usdtBNBReserve := foundOrZeroFloat(reserves.GetAt(update.LogOrdinal, fmt.Sprintf("reserve:%s:%s", USDT_WBNB_PAIR, WBNB_ADDRESS)))
	totalLiquidityBNB := bf().Add(busdBNBReserve, usdtBNBReserve).SetPrec(100)

	zero := bf()

	if totalLiquidityBNB.Cmp(zero) == 0 {
		return big.NewFloat(0)
	}

	if busdBNBReserve.Cmp(zero) == 0 {
		return foundOrZeroFloat(reserves.GetAt(update.LogOrdinal, USDT_PRICE_KEY))
	} else if usdtBNBReserve.Cmp(zero) == 0 {
		return foundOrZeroFloat(reserves.GetAt(update.LogOrdinal, BUSD_PRICE_KEY))
	}

	// both found, average out

	busdWeight := bf().Quo(busdBNBReserve, totalLiquidityBNB).SetPrec(100)
	usdtWeight := bf().Quo(usdtBNBReserve, totalLiquidityBNB).SetPrec(100)

	busdPrice := foundOrZeroFloat(reserves.GetAt(update.LogOrdinal, BUSD_PRICE_KEY))
	usdtPrice := foundOrZeroFloat(reserves.GetAt(update.LogOrdinal, USDT_PRICE_KEY))

	return bf().Add(
		bf().Mul(busdPrice, busdWeight).SetPrec(100),
		bf().Mul(usdtPrice, usdtWeight).SetPrec(100),
	).SetPrec(100)
}

// findBnbPricePerToken provides a derived price multiplier from this token to BNB, transiting through trusted pairs.
func (p *DerivedPricesStateBuilder) findBnbPricePerToken(logOrdinal uint64, tinyTokenAddr string, pairs state.Reader, reserves state.Reader) *big.Float {
	if tinyTokenAddr == WBNB_ADDRESS {
		return big.NewFloat(1) // BNB price of a BNB is always 1
	}

	directToBNBPrice, found := reserves.GetLast(fmt.Sprintf("price:%s:%s", WBNB_ADDRESS, tinyTokenAddr)) // FIXME: ensure order is right
	if found {
		return bytesToFloat(directToBNBPrice)
	}

	// loop all whitelist for a matching pair
	for _, majorToken := range []string{
		"0xe9e7cea3dedca5984780bafc599bd69add087d56", // BUSD
		"0x55d398326f99059ff775485246999027b3197955", // USDT
		"0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", // USDC
		"0x23396cf899ca06c4472205fc903bdb4de249d6fc", // UST
		"0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c", // BTCB
		"0x2170ed0880ac9a755fd29b2688956bd959f933f8", // WETH
	} {
		tinyToMajorPair, found := pairs.GetAt(logOrdinal, "tokens:"+generateTokensKey(tinyTokenAddr, majorToken))
		if !found {
			continue
		}

		majorToBNBPrice, found := reserves.GetAt(logOrdinal, fmt.Sprintf("price:%s:%s", majorToken, WBNB_ADDRESS)) // FIXME: make sure order is right
		if !found {
			continue
		}

		tinyToMajorPrice, found := reserves.GetAt(logOrdinal, fmt.Sprintf("price:%s:%s", tinyTokenAddr, majorToken)) // FIXME: make sure order is right
		if !found {
			continue
		}

		// Check if we have sufficient reserves in those
		majorReserve, found := reserves.GetAt(logOrdinal, fmt.Sprintf("reserve:%s:%s", string(tinyToMajorPair), majorToken))
		if !found {
			continue
		}

		majorToBNBPriceFloat := bytesToFloat(majorToBNBPrice)
		bnbReserveInMajorPair := bf().Mul(majorToBNBPriceFloat, bytesToFloat(majorReserve))
		// We're checking for half of it, because `reserves_bnb` would have both sides in it.
		// We could very well check the other reserve's BNB value, would be a bit more heavy, but we can do it.
		if bnbReserveInMajorPair.Cmp(big.NewFloat(5)) <= 0 {
			// Not enough liquidity
			continue
		}

		return bf().Mul(bytesToFloat(tinyToMajorPrice), majorToBNBPriceFloat)
	}

	return nil
}
