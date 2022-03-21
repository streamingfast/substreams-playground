package pcs

import (
	"fmt"
	"math/big"
	"strconv"
)

const (
	WBNB_ADDRESS   = "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
	BUSD_WBNB_PAIR = "0x58f876857a02d6762e0101bb5c46a8c1ed44dc16" // created block 6810708
	USDT_WBNB_PAIR = "0x16b9a82891338f9ba80e2d6970fdda79d1eb0dae" // created block 6810780
)

func generateTokensKey(token0, token1 string) string {
	if token0 > token1 {
		return token1 + ":" + token0
	}
	return token0 + ":" + token1
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

func init() {
	for _, addr := range whitelist {
		whitelistCacheMap[addr] = true
	}
}

func isWhitelistedAddress(address string) bool {
	_, ok := whitelistCacheMap[address]
	return ok
}

func byteMap(in map[string]string) map[string][]byte {
	out := map[string][]byte{}
	for k, v := range in {
		out[k] = []byte(v)
	}
	return out
}

func stringMap(in map[string][]byte) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = string(v)
	}
	return out
}

func foundOrZeroFloat(in []byte, found bool) *big.Float {
	if !found {
		return bf()
	}

	return bytesToFloat(in)
}

func foundOrZeroUint64(in []byte, found bool) uint64 {
	if !found {
		return 0
	}
	val, err := strconv.ParseInt(string(in), 10, 64)
	if err != nil {
		return 0
	}
	return uint64(val)
}

func orDie(err error) {
	if err != nil {
		panic("error: " + err.Error())
	}
}

func floatToStr(f *big.Float) string {
	return f.Text('g', -1)
}

func floatToBytes(f *big.Float) []byte {
	return []byte(floatToStr(f))
}

func ConvertTokenToDecimal(amount *big.Int, decimals uint64) *big.Float {
	a := new(big.Float).SetInt(amount).SetPrec(100)
	if decimals == 0 {
		return a
	}

	return a.Quo(a, ExponentToBigFloat(int64(decimals)).SetPrec(100)).SetPrec(100)
}

func ExponentToBigFloat(decimals int64) *big.Float {
	bd := new(big.Float).SetInt64(1)
	ten := new(big.Float).SetInt64(10)
	for i := int64(0); i < decimals; i++ {
		bd = bd.Mul(bd, ten)
	}
	return bd
}

func strToFloat(in string) *big.Float {
	newFloat, _, err := big.ParseFloat(in, 10, 100, big.ToNearestEven)
	if err != nil {
		panic(fmt.Sprintf("cannot load float %q: %s", in, err))
	}
	return newFloat.SetPrec(100)
}

func bytesToFloat(in []byte) *big.Float {
	return strToFloat(string(in))
}
