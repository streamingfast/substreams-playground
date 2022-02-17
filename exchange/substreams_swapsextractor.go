package exchange

import (
	"encoding/json"
	"fmt"
	"github.com/streamingfast/sparkle-pancakeswap/state"
	"math/big"

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
			pairCnt, found := pairsState.GetLast(addr)
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

				var usdPrice *big.Float
				usdPriceData, found := pricesState.GetAt(logOrdinal, "usd")
				if !found {
					usdPrice = bf()
				}

				_ = amount0Total
				_ = amount1Total
				_ = usdPrice
				_ = usdPriceData
				amountUSD := "123"

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
