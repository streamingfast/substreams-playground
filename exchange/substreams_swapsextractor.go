package exchange

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/streamingfast/sparkle-pancakeswap/state"

	eth "github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
)

type SwapsExtractor struct {
	*SubstreamIntrinsics
}

func (p *SwapsExtractor) Map(block *pbcodec.Block, pairsState state.Reader, pricesState state.Reader) (swaps Swaps, err error) {
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

				token0Price := foundOrZeroFloat(pricesState.GetAt(logOrdinal, fmt.Sprintf("price:%s:bnb", pair.Token0.Address)))
				token1Price := foundOrZeroFloat(pricesState.GetAt(logOrdinal, fmt.Sprintf("price:%s:bnb", pair.Token1.Address)))

				derivedAmountBNB := bf().Quo(
					bf().Add(
						bf().Mul(token0Price, amount0Total.Ptr().Float()),
						bf().Mul(token1Price, amount1Total.Ptr().Float()),
					),
					big.NewFloat(2),
				)

				var amountUSD string

				usdPriceData, found := pricesState.GetAt(logOrdinal, "price:usd:bnb")
				if found {
					usdPrice := bytesToFloat(usdPriceData).Ptr().Float()
					// TODO: revise this, that's not really what the Swap does

					amountUSD = floatToStr(bf().Mul(derivedAmountBNB, usdPrice))
				}

				//prices.GetAt(logOrdinal, fmt.Sprintf(""))
				// TODO: DO SOMETHING HERE! It's always 123.. quite the shortcut :)
				_ = amount0Total
				_ = amount1Total

				swap := PCSSwap{
					PairAddress: addr,
					// Token0: pair.Token0.Address,
					// Token1: pair.Token1.Address,
					Transaction: eth.Hash(trx.Hash).Pretty(),
					Amount0In:   amount0In.String(),
					Amount1In:   amount1In.String(),
					Amount0Out:  amount0Out.String(),
					Amount1Out:  amount1Out.String(),

					AmountUSD: amountUSD,

					LogOrdinal: logOrdinal,
				}

				swaps = append(swaps, swap)
			}
		}
	}
	return
}
