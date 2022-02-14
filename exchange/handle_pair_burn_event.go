package exchange

import (
	"github.com/streamingfast/sparkle/entity"
)

func (s *Subgraph) HandlePairBurnEvent(ev *PairBurnEvent) error {
	if s.StepBelow(3) {
		return nil
	}

	trx := NewTransaction(ev.Transaction.Hash.Pretty())
	if err := s.Load(trx); err != nil {
		return err
	}

	// safety check
	if !trx.Exists() {
		return nil
	}

	burn := NewBurn(trx.Burns[len(trx.Burns)-1])
	if err := s.Load(burn); err != nil {
		return err
	}

	pair := NewPair(ev.LogAddress.Pretty())
	if err := s.Load(pair); err != nil {
		return err
	}

	pancake := NewPancakeFactory(FactoryAddress)
	if err := s.Load(pancake); err != nil {
		return err
	}

	token0 := NewToken(pair.Token0)
	if err := s.Load(token0); err != nil {
		return err
	}
	token1 := NewToken(pair.Token1)
	if err := s.Load(token1); err != nil {
		return err
	}

	token0Amount := entity.ConvertTokenToDecimal(ev.Amount0, token0.Decimals.Int().Int64())
	token1Amount := entity.ConvertTokenToDecimal(ev.Amount1, token1.Decimals.Int().Int64())

	token0.TotalTransactions = entity.IntAdd(token0.TotalTransactions, IL(1))
	token1.TotalTransactions = entity.IntAdd(token1.TotalTransactions, IL(1))

	bundle := NewBundle("1")
	if err := s.Load(bundle); err != nil {
		return err
	}
	amountTotalUSD := bf().Mul(
		bf().Add(
			bf().Mul(token1.DerivedBNB.Float(), token1Amount),
			bf().Mul(token0.DerivedBNB.Float(), token0Amount),
		),
		bundle.BnbPrice.Float(),
	)

	pair.TotalTransactions = entity.IntAdd(pair.TotalTransactions, IL(1))
	pancake.TotalTransactions = entity.IntAdd(pancake.TotalTransactions, IL(1))

	// save entities
	if err := s.Save(token0); err != nil {
		return err
	}
	if err := s.Save(token1); err != nil {
		return err
	}
	if err := s.Save(pair); err != nil {
		return err
	}
	if err := s.Save(pancake); err != nil {
		return err
	}

	// burn.Sender = ev.Sender.Bytes()
	burn.Amount0 = F(token0Amount).Ptr()
	burn.Amount1 = F(token1Amount).Ptr()
	burn.LogIndex = IL(int64(ev.LogIndex)).Ptr()
	burn.AmountUSD = F(amountTotalUSD).Ptr()

	if err := s.Save(burn); err != nil {
		return err
	}

	// // update day entities
	if _, err := s.UpdatePairDayData(ev.LogAddress); err != nil {
		return err
	}

	if _, err := s.UpdatePairHourData(ev.LogAddress); err != nil {
		return err
	}

	if _, err := s.UpdatePancakeDayData(); err != nil {
		return err
	}

	if _, err := s.UpdateTokenDayData(ev.LogAddress, token0, bundle); err != nil {
		return err
	}
	if _, err := s.UpdateTokenDayData(ev.LogAddress, token1, bundle); err != nil {
		return err
	}

	return nil
}
