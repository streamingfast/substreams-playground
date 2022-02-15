package exchange

import (
	"encoding/json"
	"fmt"
	"math/big"

	eth "github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
)

type ReservesExtractor struct {
	*SubstreamIntrinsics
}

// Map function can take one or more input objects, sync'd by the
// `Block` clock.  Because it depends on a `PairsState`, it needs to
// be run in `Process`, linearly.
func (p *ReservesExtractor) Map(block *pbcodec.Block, pairsState StateReader) (reserves PCSReserveUpdates, err error) {
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
			if IsPairSyncEvent(ethLog) {
				ev, err := NewPairSyncEvent(ethLog, block, trx)
				if err != nil {
					return nil, fmt.Errorf("decoding PairSync event: %w", err)
				}

				var pair PCSPair
				if err := json.Unmarshal(pairCnt, &pair); err != nil {
					return nil, err
				}

				reserves = append(reserves, PCSReserveUpdate{
					PairAddress: eth.Address(log.Address).Pretty(),
					Reserve0:    toFloatString(ev.Reserve0, pair.Token0.Decimals),
					Reserve1:    toFloatString(ev.Reserve1, pair.Token1.Decimals),
					LogOrdinal:  uint64(log.BlockIndex),
				})
			}
		}
	}
	return
}

func toFloatString(in *big.Int, decimals uint32) string {
	bf := entity.ConvertTokenToDecimal(in, int64(decimals))
	return entity.NewFloat(bf).String()
}
