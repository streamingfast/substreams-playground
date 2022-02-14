package exchange

import (
	"fmt"
	"math/big"

	"github.com/streamingfast/sparkle/entity"
)

func (s *Subgraph) HandlePairSwapEvent(ev *PairSwapEvent) error {
	if s.StepBelow(4) {
		return nil
	}

	pair := NewPair(ev.LogAddress.Pretty())
	err := s.Load(pair)
	if err != nil {
		return fmt.Errorf("loading pair: %w", err)
	}

	token0 := NewToken(pair.Token0)
	err = s.Load(token0)
	if err != nil {
		return fmt.Errorf("loading initialToken 0: %w", err)
	}

	token1 := NewToken(pair.Token1)
	err = s.Load(token1)
	if err != nil {
		return fmt.Errorf("loading initialToken 1: %w", err)
	}

	amount0In := entity.ConvertTokenToDecimal(ev.Amount0In, token0.Decimals.Int().Int64())
	amount1In := entity.ConvertTokenToDecimal(ev.Amount1In, token1.Decimals.Int().Int64())
	amount0Out := entity.ConvertTokenToDecimal(ev.Amount0Out, token0.Decimals.Int().Int64())
	amount1Out := entity.ConvertTokenToDecimal(ev.Amount1Out, token1.Decimals.Int().Int64())

	// totals for volume updateTradeVolumes
	amount0Total := bf().Add(amount0Out, amount0In)
	amount1Total := bf().Add(amount1Out, amount1In)

	//// BNB/USD prices
	bundle := NewBundle("1")
	err = s.Load(bundle)
	if err != nil {
		return fmt.Errorf("loading bundle: %w", err)
	}

	// get total amounts of derived USD and BNB for tracking
	derivedAmountBNB := bf().Quo(
		bf().Add(
			bf().Mul(token1.DerivedBNB.Float(), amount1Total),
			bf().Mul(token0.DerivedBNB.Float(), amount0Total),
		),
		big.NewFloat(2),
	)

	derivedAmountUSD := bf().Mul(derivedAmountBNB, bundle.BnbPrice.Float())

	// only accounts for volume through white listed tokens
	trackedAmountUSD := getTrackedVolumeUSD(bundle, amount0Total, token0, amount1Total, token1)

	//let trackedAmountBNB: BigDecimal
	var trackedAmountBNB *big.Float
	if bundle.BnbPrice.Float().Cmp(big.NewFloat(0)) == 0 {
		trackedAmountBNB = big.NewFloat(0)
	} else {
		trackedAmountBNB = bf().Quo(trackedAmountUSD, bundle.BnbPrice.Float())
	}

	// @ steps 3 trade  volume is realtive per shard
	// @ steps 4 is where you should sqaush and it becomes absolute and that where you can save eneities

	// update token0 global volume and initialToken liquidity stats

	token0.TradeVolume = entity.FloatAdd(token0.TradeVolume, F(bf().Add(amount0In, amount0Out)))
	token0.TradeVolumeUSD = entity.FloatAdd(token0.TradeVolumeUSD, F(trackedAmountUSD))
	token0.UntrackedVolumeUSD = entity.FloatAdd(token0.UntrackedVolumeUSD, F(derivedAmountUSD))

	// update token1 global volume and initialToken liquidity stats
	token1.TradeVolume = entity.FloatAdd(token1.TradeVolume, F(bf().Add(amount1In, amount1Out)))
	token1.TradeVolumeUSD = entity.FloatAdd(token1.TradeVolumeUSD, F(trackedAmountUSD))
	token1.UntrackedVolumeUSD = entity.FloatAdd(token0.UntrackedVolumeUSD, F(derivedAmountUSD))

	// update txn counts
	token0.TotalTransactions = entity.IntAdd(token0.TotalTransactions, IL(1))
	token1.TotalTransactions = entity.IntAdd(token1.TotalTransactions, IL(1))

	// update pair volume data, use tracked amount if we have it as its probably more accurate
	pair.VolumeUSD = entity.FloatAdd(pair.VolumeUSD, F(trackedAmountUSD))
	pair.VolumeToken0 = entity.FloatAdd(pair.VolumeToken0, F(amount0Total))
	pair.VolumeToken1 = entity.FloatAdd(pair.VolumeToken1, F(amount1Total))
	pair.UntrackedVolumeUSD = entity.FloatAdd(pair.UntrackedVolumeUSD, F(derivedAmountUSD))

	pair.TotalTransactions = entity.IntAdd(pair.TotalTransactions, IL(1))
	if err := s.Save(pair); err != nil {
		return fmt.Errorf("saving pair: %w", err)
	}

	// update global values, only used tracked amounts for volume
	pancake := NewPancakeFactory(FactoryAddress)
	err = s.Load(pancake)
	if err != nil {
		return fmt.Errorf("loading pancake factory: %w", err)
	}

	pancake.TotalVolumeUSD = entity.FloatAdd(pancake.TotalVolumeUSD, F(trackedAmountUSD))
	pancake.TotalVolumeBNB = entity.FloatAdd(pancake.TotalVolumeBNB, F(trackedAmountBNB))
	pancake.UntrackedVolumeUSD = entity.FloatAdd(pancake.UntrackedVolumeUSD, F(derivedAmountUSD))

	pancake.TotalTransactions = entity.IntAdd(pancake.TotalTransactions, IL(1))
	// save entities

	if err := s.Save(token0); err != nil {
		return fmt.Errorf("saving initialToken 0: %w", err)
	}

	if err := s.Save(token1); err != nil {
		return fmt.Errorf("saving initialToken 1: %w", err)
	}

	if err := s.Save(pancake); err != nil {
		return fmt.Errorf("saving pancake: %w", err)
	}

	transaction := NewTransaction(ev.Transaction.Hash.Pretty())
	err = s.Load(transaction)
	if err != nil {
		return fmt.Errorf("loading transaction: %w", err)
	}

	if !transaction.Exists() {
		block := s.Block()

		transaction.Block = IL(int64(block.Number()))
		transaction.Timestamp = IL(block.Timestamp().Unix())
	}

	swap := NewSwap(fmt.Sprintf("%s-%d", transaction.ID, len(transaction.Swaps)))

	// update swap event
	swap.Transaction = transaction.ID
	swap.Pair = pair.ID
	swap.Token0 = pair.Token0
	swap.Token1 = pair.Token1
	swap.Timestamp = transaction.Timestamp
	swap.Sender = ev.Sender.Pretty()
	swap.Amount0In = F(amount0In)
	swap.Amount1In = F(amount1In)
	swap.Amount0Out = F(amount0Out)
	swap.Amount1Out = F(amount1Out)
	swap.To = ev.To.Pretty()
	swap.From = ev.Transaction.From.Pretty()
	swap.LogIndex = IL(int64(ev.LogIndex)).Ptr()

	// use the tracked amount if we have it
	if trackedAmountUSD.Cmp(big.NewFloat(0)) == 0 {
		swap.AmountUSD = F(derivedAmountUSD)
	} else {
		swap.AmountUSD = F(trackedAmountUSD)
	}

	if err := s.Save(swap); err != nil {
		return fmt.Errorf("saving swap: %w", err)
	}

	transaction.Swaps = append(transaction.Swaps, swap.ID)

	if err := s.Save(transaction); err != nil {
		return fmt.Errorf("saving transaction: %w", err)
	}

	pairDayData, err := s.UpdatePairDayData(ev.LogAddress)
	if err != nil {
		return fmt.Errorf("updating pair day data: %w", err)
	}

	pairHourData, err := s.UpdatePairHourData(ev.LogAddress)
	if err != nil {
		return fmt.Errorf("updating pair hour data: %w", err)
	}

	pancakeDayData, err := s.UpdatePancakeDayData()
	if err != nil {
		return fmt.Errorf("update pancake day data: %w", err)
	}

	token0DayData, err := s.UpdateTokenDayData(ev.LogAddress, token0, bundle)
	if err != nil {
		return fmt.Errorf("update token0 day data: %w", err)
	}

	token1DayData, err := s.UpdateTokenDayData(ev.LogAddress, token1, bundle)
	if err != nil {
		return fmt.Errorf("udpate token1 day data: %w", err)
	}

	pancakeDayData.DailyVolumeUSD = entity.FloatAdd(pancakeDayData.DailyVolumeUSD, F(trackedAmountUSD))
	pancakeDayData.DailyVolumeBNB = entity.FloatAdd(pancakeDayData.DailyVolumeBNB, F(trackedAmountBNB))
	pancakeDayData.DailyVolumeUntracked = entity.FloatAdd(pancakeDayData.DailyVolumeUntracked, F(derivedAmountUSD))

	err = s.Save(pancakeDayData)
	if err != nil {
		return err
	}

	pairDayData.DailyVolumeToken0 = entity.FloatAdd(pairDayData.DailyVolumeToken0, F(amount0Total))
	pairDayData.DailyVolumeToken1 = entity.FloatAdd(pairDayData.DailyVolumeToken1, F(amount1Total))
	pairDayData.DailyVolumeUSD = entity.FloatAdd(pairDayData.DailyVolumeUSD, F(trackedAmountUSD))
	err = s.Save(pairDayData)
	if err != nil {
		return err
	}

	pairHourData.HourlyVolumeToken0 = entity.FloatAdd(pairHourData.HourlyVolumeToken0, F(amount0Total))
	pairHourData.HourlyVolumeToken1 = entity.FloatAdd(pairHourData.HourlyVolumeToken1, F(amount1Total))
	pairHourData.HourlyVolumeUSD = entity.FloatAdd(pairHourData.HourlyVolumeUSD, F(trackedAmountUSD))
	err = s.Save(pairHourData)
	if err != nil {
		return err
	}

	token0DayData.DailyVolumeToken = entity.FloatAdd(token0DayData.DailyVolumeToken, F(amount0Total))
	token0DayData.DailyVolumeBNB = entity.FloatAdd(token0DayData.DailyVolumeBNB, F(bf().Mul(amount0Total, token0.DerivedBNB.Float())))
	token0DayData.DailyVolumeUSD = entity.FloatAdd(token0DayData.DailyVolumeUSD, F(bf().Mul(bf().Mul(amount0Total, token0.DerivedBNB.Float()), bundle.BnbPrice.Float())))
	err = s.Save(token0DayData)
	if err != nil {
		return err
	}

	token1DayData.DailyVolumeToken = entity.FloatAdd(token1DayData.DailyVolumeToken, F(amount1Total))
	token1DayData.DailyVolumeBNB = entity.FloatAdd(token1DayData.DailyVolumeBNB, F(bf().Mul(amount1Total, token1.DerivedBNB.Float())))
	token1DayData.DailyVolumeUSD = entity.FloatAdd(token1DayData.DailyVolumeUSD, F(bf().Mul(bf().Mul(amount1Total, token1.DerivedBNB.Float()), bundle.BnbPrice.Float())))
	err = s.Save(token1DayData)
	if err != nil {
		return err
	}

	return nil
}
