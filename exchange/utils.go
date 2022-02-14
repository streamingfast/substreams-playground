package exchange

import (
	"math/big"
)

func getTrackedVolumeUSD(bundle *Bundle, tokenAmount0 *big.Float, token0 *Token, tokenAmount1 *big.Float, token1 *Token) *big.Float {
	price0 := bf().Mul(token0.DerivedBNB.Float(), bundle.BnbPrice.Float())
	price1 := bf().Mul(token1.DerivedBNB.Float(), bundle.BnbPrice.Float())

	token0Whitelisted := isWhitelistedAddress(token0.ID)
	token1Whitelisted := isWhitelistedAddress(token1.ID)

	// both are whitelist tokens, take average of both amounts
	if token0Whitelisted && token1Whitelisted {
		sum := bf().Add(
			bf().Mul(tokenAmount0, price0),
			bf().Mul(tokenAmount1, price1),
		)
		avg := bf().Quo(sum, big.NewFloat(2.0))
		return avg
	}

	if token0Whitelisted && !token1Whitelisted {
		// take full value of the whitelisted token amount
		return bf().Mul(tokenAmount0, price0)
	}

	if !token0Whitelisted && token1Whitelisted {
		// take full value of the whitelisted token amount
		return bf().Mul(tokenAmount1, price1)
	}

	// neither token is on white list, tracked volume is 0
	return big.NewFloat(0)
}

func getTrackedLiquidityUSD(bundle *Bundle, tokenAmount0 *big.Float, token0 *Token, tokenAmount1 *big.Float, token1 *Token) *big.Float {
	price0 := bf().Mul(token0.DerivedBNB.Float().SetPrec(100), bundle.BnbPrice.Float().SetPrec(100)).SetPrec(100)
	price1 := bf().Mul(token1.DerivedBNB.Float().SetPrec(100), bundle.BnbPrice.Float().SetPrec(100)).SetPrec(100)

	token0Whitelisted := isWhitelistedAddress(token0.ID)
	token1Whitelisted := isWhitelistedAddress(token1.ID)

	// both are whitelist tokens, take average of both amounts
	if token0Whitelisted && token1Whitelisted {
		return bf().Add(
			bf().Mul(tokenAmount0, price0).SetPrec(100),
			bf().Mul(tokenAmount1, price1).SetPrec(100),
		).SetPrec(100)
	}

	floatTwo := big.NewFloat(2)
	if token0Whitelisted && !token1Whitelisted {
		// take double value of the whitelisted token amount
		return bf().Mul(
			bf().Mul(tokenAmount0, price0).SetPrec(100),
			floatTwo,
		).SetPrec(100)
	}

	if !token0Whitelisted && token1Whitelisted {
		// take double value of the whitelisted token amount
		return bf().Mul(
			bf().Mul(tokenAmount1, price1).SetPrec(100),
			floatTwo,
		).SetPrec(100)
	}

	// neither token is on white list, tracked volume is 0
	return big.NewFloat(0)
}

func generateTokensKey(token0, token1 string) string {
	if token0 > token1 {
		return token1 + token0
	}
	return token0 + token1
}

// whitelist is a slice because we need to respect the order when using it in certain location, so
// we must not converted to a map[string]bool directly unless there is a strict ordering way to list them.
var whitelist = []string{
	"0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c", // WBNB
	"0xe9e7cea3dedca5984780bafc599bd69add087d56", // BUSD
	"0x55d398326f99059ff775485246999027b3197955", // USDT
	"0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", // USDC
	"0x23396cf899ca06c4472205fc903bdb4de249d6fc", // UST
	"0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c", // BTCB
	"0x2170ed0880ac9a755fd29b2688956bd959f933f8", // WETH
}

var whitelistCacheMap = map[string]bool{}

func isWhitelistedAddress(address string) bool {
	if _, ok := whitelistCacheMap[address]; ok {
		return true
	}

	for _, addr := range whitelist {
		if addr != address {
			continue
		}

		whitelistCacheMap[address] = true
		return true
	}

	return false
}
