package exchange

import (
	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
	"go.uber.org/zap"
)

func (s *Subgraph) HandlePairMintEvent(ev *PairMintEvent) error {
	if s.StepBelow(3) {
		return nil
	}

	trx := NewTransaction(ev.Transaction.Hash.Pretty())
	if err := s.Load(trx); err != nil {
		return err
	}

	mint := NewMint(trx.Mints[len(trx.Mints)-1])
	if err := s.Load(mint); err != nil {
		return err
	}
	s.Log.Debug("mint things - mint", zap.String("to", eth.Address(mint.To).Pretty()))

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

	// update txn counts
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

	sender := ev.Sender.Pretty()
	mint.Sender = &sender
	mint.Amount0 = F(token0Amount).Ptr()
	mint.Amount1 = F(token1Amount).Ptr()
	mint.LogIndex = IL(int64(ev.LogIndex)).Ptr()
	mint.AmountUSD = F(amountTotalUSD).Ptr()
	if err := s.Save(mint); err != nil {
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
