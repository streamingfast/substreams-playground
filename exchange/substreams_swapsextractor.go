package exchange

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	eth "github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle-pancakeswap/state"
	"github.com/streamingfast/sparkle/entity"
	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
)

type SwapsExtractor struct {
	*SubstreamIntrinsics
}

func (p *SwapsExtractor) Map(block *pbcodec.Block, pairs state.Reader, prices state.Reader) (out []interface{}, err error) {
	for _, trx := range block.TransactionTraces {
		trxID := eth.Hash(trx.Hash).Pretty()
		for _, call := range trx.Calls {
			if call.StateReverted {
				continue
			}
			if len(call.Logs) == 0 {
				continue
			}

			addr := eth.Address(call.Address).Pretty()
			pairCnt, found := pairs.GetLast("pair:" + addr)
			if !found {
				continue
			}

			var pair *PCSPair
			if err := json.Unmarshal(pairCnt, &pair); err != nil {
				return nil, err
			}

			var events []interface{}
			for _, log := range call.Logs {
				ethLog := ssCodecLogToEthLog(log)
				event, err := DecodeEvent(ethLog, block, trx)
				if err != nil {
					return nil, fmt.Errorf("parsing event: %w", err)
				}

				events = append(events, event)
			}

			// Match the different patterns
			fmt.Printf("CALL %d on pair %q: ", call.Index, addr)
			for _, ev := range events {
				fmt.Printf("%s ", strings.Replace(strings.Replace(strings.Split(fmt.Sprintf("%T", ev), ".")[1], "Pair", "", -1), "Event", "", -1))
			}
			fmt.Printf("\n")

			_ = pairCnt

			// First pattern:
			// last = Mint, 4 logs (includes the handling of the first optional Transfer)
			// implies: Transfer Transfer Sync Mint
			var newOutput interface{}
			var err error
			if len(events) == 4 {
				evMint, okMint := events[3].(*PairMintEvent)
				evBurn, _ := events[3].(*PairBurnEvent)
				evSync := events[2].(*PairSyncEvent)
				evTr2 := events[1].(*PairTransferEvent)
				evTr1 := events[0].(*PairTransferEvent)
				if okMint {
					newOutput, err = p.processMint(prices, pair, evTr1, evTr2, evSync, evMint)
				} else {
					newOutput, err = p.processBurn(prices, pair, evTr1, evTr2, evSync, evBurn)
				}
			} else if len(events) == 3 {
				evMint, okMint := events[2].(*PairMintEvent)
				evBurn, _ := events[2].(*PairBurnEvent)
				evSync := events[1].(*PairSyncEvent)
				evTr2 := events[0].(*PairTransferEvent)
				if okMint {
					newOutput, err = p.processMint(prices, pair, nil, evTr2, evSync, evMint)
				} else {
					newOutput, err = p.processBurn(prices, pair, nil, evTr2, evSync, evBurn)
				}
			} else if len(events) == 2 {
				evSwap, okSwap := events[1].(*PairSwapEvent)
				if okSwap {
					evSync := events[0].(*PairSyncEvent)
					newOutput, err = p.processSwap(prices, pair, evSync, evSwap)
				} else {
					fmt.Println("HUh? what's that?")
				}
			} else if len(events) == 1 {
				continue
				// if evTransfer, ok := events[0].(*PairTransferEvent); ok {
				// 	newOutput = p.processTransfer(prices, evTransfer)
				// } else if evApproval, ok := events[0].(*PairApprovalEvent); ok {
				// 	newOutput = p.processApproval(prices, evApproval)
				// }
			}

			if err != nil {
				return nil, fmt.Errorf("process pair call: %w", err)
			}
			if newOutput != nil {
				out = append(out, newOutput)
			}

			// Second pattern:
			// last = Mint, 3 logs (not the optional Transfer)
			// implies: Transfer Sync Mint

			// Third & fourth patterns:
			// as as Mint, but replaced by "Burn" instead

			// Fifth pattern:
			// Sync Swap - a swap :)

			// Sixth pattern:
			// Transfer only

			// Seventh pattern:
			// Approval only
		}

		// continue

		// for _, log := range trx.Receipt.Logs {
		// 	// perhaps we can optimize in a small local map, if we
		// 	// found previously in this call, in the State or perhaps
		// 	// we do that in the `GetLast()` stack, optimized
		// 	// heuristics.
		// 	addr := eth.Address(log.Address).Pretty()

		// 	ethLog := ssCodecLogToEthLog(log)
		// 	event, err := DecodeEvent(ethLog, block, trx)
		// 	if err != nil {
		// 		return nil, fmt.Errorf("parsing event: %w", err)
		// 	}

		// 	var pairCnt []byte

		// 	switch ev := event.(type) {
		// 	case *PairSwapEvent:

		// 		swaps = append(swaps, swap)
		// 		fmt.Println("SWAP", trxID, ev)
		// 	case *PairBurnEvent:
		// 		fmt.Println("BURN", trxID, ev)
		// 	case *PairMintEvent:
		// 		fmt.Println("MINT", trxID, ev)
		// 	case *PairTransferEvent:
		// 		fmt.Println("XFER", trxID, ev)
		// 	case *PairSyncEvent:
		// 		fmt.Println("SYNC", trxID, ev)

		// 	default:
		// 		fmt.Printf("Another event type: %T\n", ev)
		// 	}
		// }
	}
	return
}

func (p *SwapsExtractor) processMint(prices state.Reader, pair *PCSPair, tr1 *PairTransferEvent, tr2 *PairTransferEvent, sync *PairSyncEvent, mint *PairMintEvent) (out *PCSMint, err error) {
	logOrdinal := uint64(log.BlockIndex)
	return nil, nil
}

func (p *SwapsExtractor) processBurn(prices state.Reader, pair *PCSPair, tr1 *PairTransferEvent, tr2 *PairTransferEvent, sync *PairSyncEvent, burn *PairBurnEvent) (out *PCSBurn, err error) {
	logOrdinal := uint64(log.BlockIndex)
	return nil, nil
}

func (p *SwapsExtractor) processSwap(prices state.Reader, pair *PCSPair, sync *PairSyncEvent, swap *PairSwapEvent) (out *PCSSwap, err error) {
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

	swap := PCSSwap{
		PairAddress: addr,
		Token0:      pair.Token0.Address,
		Token1:      pair.Token1.Address,
		Transaction: trxID,

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

	return swap
}

// func (p *SwapsExtractor) processTransfer(prices state.Reader, tr *PairTransferEvent) (swaps Swaps, err error) {
// 	return nil, nil
// }

// func (p *SwapsExtractor) processApproval(prices state.Reader, approval *PairApprovalEvent) (swaps Swaps, err error) {
// 	return nil, nil
// }

func getDerivedPrice(ord uint64, prices state.Reader, derivedToken string, tokenAmount *big.Float, tokenAddr string) *big.Float {
	usdPrice := foundOrZeroFloat(prices.GetAt(ord, fmt.Sprintf("price:%s:%s", tokenAddr, derivedToken)))
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
