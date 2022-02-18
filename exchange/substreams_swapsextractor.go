package exchange

import (
	"encoding/json"
	"fmt"
	"math/big"

	eth "github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle-pancakeswap/state"
	"github.com/streamingfast/sparkle/entity"
	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
)

type SwapsExtractor struct {
	*SubstreamIntrinsics
}

func (p *SwapsExtractor) Map(block *pbcodec.Block, pairsState state.Reader, prices state.Reader) (swaps Swaps, err error) {
	for _, trx := range block.TransactionTraces {
		for _, log := range trx.Receipt.Logs {
			// perhaps we can optimize in a small local map, if we
			// found previously in this call, in the State or perhaps
			// we do that in the `GetLast()` stack, optimized
			// heuristics.
			addr := eth.Address(log.Address).Pretty()
			pairCnt, found := pairsState.GetLast("pair:" + addr)
			if !found {
				continue
			}

			ethLog := ssCodecLogToEthLog(log)
			if IsPairSwapEvent(ethLog) {
				ev, err := NewPairSwapEvent(ethLog, block, trx)
				if err != nil {
					return nil, fmt.Errorf("decoding PairSync event: %w", err)
				}

				var pair PCSPair
				if err := json.Unmarshal(pairCnt, &pair); err != nil {
					return nil, err
				}

				logOrdinal := uint64(log.BlockIndex)

				amount0In := intToFloat(ev.Amount0In, pair.Token0.Decimals)
				amount1In := intToFloat(ev.Amount1In, pair.Token1.Decimals)
				amount0Out := intToFloat(ev.Amount0Out, pair.Token0.Decimals)
				amount1Out := intToFloat(ev.Amount1Out, pair.Token1.Decimals)

				amount0Total := entity.FloatAdd(amount0Out, amount0In)
				amount1Total := entity.FloatAdd(amount1Out, amount1In)

				derivedAmountBNB := avgFloats(
					getDerivedPrice(logOrdinal, prices, "bnb", amount0Total.Ptr().Float(), pair.Token0.Address),
					getDerivedPrice(logOrdinal, prices, "bnb", amount1Total.Ptr().Float(), pair.Token1.Address),
				)

				trackedAmountUSD := avgFloats(
					getDerivedPrice(logOrdinal, prices, "usd", amount0Total.Ptr().Float(), pair.Token0.Address),
					getDerivedPrice(logOrdinal, prices, "usd", amount1Total.Ptr().Float(), pair.Token0.Address),
				)

				// populate all those `token:trade_volume`, `token:trade_volume_usd`
				// count TotalTransactions for each token

				//prices.GetAt(logOrdinal, fmt.Sprintf(""))
				_ = amount0Total
				_ = amount1Total

				swap := PCSSwap{
					PairAddress: addr,
					Token0:      pair.Token0.Address,
					Token1:      pair.Token1.Address,
					Transaction: eth.Hash(trx.Hash).Pretty(),

					Amount0In:  amount0In.String(),
					Amount1In:  amount1In.String(),
					Amount0Out: amount0Out.String(),
					Amount1Out: amount1Out.String(),

					AmountBNB: floatToStr(derivedAmountBNB),
					AmountUSD: floatToStr(trackedAmountUSD),
					From:      eth.Address(trx.From).Pretty(),
					To:        ev.To.Pretty(),
					Sender:    ev.Sender.Pretty(),

					LogOrdinal: logOrdinal,
				}

				swaps = append(swaps, swap)
			}
		}
	}
	return
}

func getDerivedPrice(ord uint64, prices state.Reader, derivedToken string, tokenAmount *big.Float, tokenAddr string) *big.Float {
	usdPrice := foundOrZeroFloat(prices.GetAt(ord, fmt.Sprintf("price:%s:usd", tokenAddr, derivedToken)))
	if usdPrice.Cmp(big.NewFloat(0)) == 0 {
		return nil
	}

	return bf().Mul(tokenAmount, usdPrice)
}

func avgFloats(f ...*big.Float) *big.Float {
	sum := big.NewFloat(0)
	var count float64 = 0
	for _, fl := range f {
		if fl == nil {
			continue
		}
		sum = bf().Add(sum, fl)
		count++
	}

	if count == 0 {
		return sum
	}

	return bf().Quo(sum, big.NewFloat(count))
}
