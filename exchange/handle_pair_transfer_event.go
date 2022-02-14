package exchange

import (
	"fmt"

	"math/big"

	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
	"go.uber.org/zap"
)

func (s *Subgraph) HandlePairTransferEvent(ev *PairTransferEvent) error {
	if s.StepBelow(3) {
		return nil
	}

	s.Log.Debug("handling transfer event",
		zap.Uint64("block_num", s.Block().Number()),
		zap.String("trx_Trace", ev.Transaction.Hash.Pretty()),
		zap.Reflect("event", ev),
	)

	// Initial liquidity.
	if ev.To.Pretty() == ZeroAddress && (ev.Value.Cmp(big.NewInt(1000)) == 0) {
		return nil
	}

	// get pair and load contract
	pair := NewPair(ev.LogAddress.Pretty())
	if err := s.Load(pair); err != nil {
		return fmt.Errorf("loading pair id %s: %w", ev.LogAddress.Pretty(), err)
	}

	// liquidity token amount being transferred
	value := entity.ConvertTokenToDecimal(ev.Value, 18)

	// get or create transaction
	trx := NewTransaction(ev.Transaction.Hash.Pretty())
	if err := s.Load(trx); err != nil {
		return err
	}

	if !trx.Exists() {
		block := s.Block()

		trx.Timestamp = IL(block.Timestamp().Unix())
		trx.Block = IL(int64(block.Number()))
	}

	// mints
	if ev.From.Pretty() == ZeroAddress {
		pair.TotalSupply = F(bf().Add(pair.TotalSupply.Float(), value))
		if err := s.Save(pair); err != nil {
			return fmt.Errorf("saving pair %s: %w", pair.ID, err)
		}

		var completed bool
		if len(trx.Mints) != 0 {
			var err error
			if completed, err = s.isCompleteMint(trx.Mints[len(trx.Mints)-1]); err != nil {
				return err
			}
		}

		s.Log.Debug("mint count", zap.Int("n", len(trx.Mints)))

		if len(trx.Mints) == 0 || completed { // create new mint if no mints so far or if last one is done already
			mint := NewMint(fmt.Sprintf("%s-%d", ev.Transaction.Hash.Pretty(), len(trx.Mints)))
			mint.Transaction = trx.ID
			mint.Pair = pair.ID
			mint.Token0 = pair.Token0
			mint.Token1 = pair.Token1
			mint.To = ev.To.Pretty()
			mint.Liquidity = F(value)
			mint.Timestamp = I(trx.Timestamp.Int())
			if err := s.Save(mint); err != nil {
				return fmt.Errorf("saving new mint: %w", err)
			}
			s.Log.Debug("mint things - transfer", zap.String("to", eth.Address(mint.To).Pretty()))

			trx.Mints = append(trx.Mints, mint.ID)
			if err := s.Save(trx); err != nil {
				return fmt.Errorf("saving trx: %w", err)
			}
		}
	}

	// case where direct send first on BNB withdrawals
	if ev.To.Pretty() == pair.ID {
		burn := NewBurn(fmt.Sprintf("%s-%d", ev.Transaction.Hash.Pretty(), len(trx.Burns)))
		burn.Transaction = trx.ID
		burn.Pair = pair.ID
		burn.Token0 = pair.Token0
		burn.Token1 = pair.Token1
		burn.Liquidity = F(value)
		burn.Timestamp = I(trx.Timestamp.Int())
		to := ev.To.Pretty()
		burn.To = &to
		sender := ev.From.Pretty()
		burn.Sender = &sender
		burn.NeedsComplete = true
		if err := s.Save(burn); err != nil {
			return fmt.Errorf("saving burn: %w", err)
		}

		trx.Burns = append(trx.Burns, burn.ID)
		if err := s.Save(trx); err != nil {
			return fmt.Errorf("saving trx: %w", err)
		}
	}

	// burn
	if ev.To.Pretty() == ZeroAddress && ev.From.Pretty() == pair.ID {
		pair.TotalSupply = F(bf().Sub(pair.TotalSupply.Float(), value))
		if err := s.Save(pair); err != nil {
			return err
		}

		var burn *Burn
		if len(trx.Burns) > 0 {
			currentBurn := NewBurn(trx.Burns[len(trx.Burns)-1])
			if err := s.Load(currentBurn); err != nil {
				return err
			}

			if currentBurn.NeedsComplete {
				burn = currentBurn
			} else {
				burn = NewBurn(fmt.Sprintf("%s-%d", ev.Transaction.Hash.Pretty(), len(trx.Burns)))
				burn.Transaction = trx.ID
				burn.NeedsComplete = false
				burn.Pair = pair.ID
				burn.Token0 = pair.Token0
				burn.Token1 = pair.Token1
				burn.Liquidity = F(value)
				burn.Timestamp = I(trx.Timestamp.Int())
			}
		} else {
			burn = NewBurn(fmt.Sprintf("%s-%d", ev.Transaction.Hash.Pretty(), len(trx.Burns)))
			burn.Transaction = trx.ID
			burn.NeedsComplete = false
			burn.Pair = pair.ID
			burn.Token0 = pair.Token0
			burn.Token1 = pair.Token1
			burn.Liquidity = F(value)
			burn.Timestamp = I(trx.Timestamp.Int())
		}

		var completed bool
		if len(trx.Mints) != 0 {
			var err error
			if completed, err = s.isCompleteMint(trx.Mints[len(trx.Mints)-1]); err != nil {
				return err
			}
		}

		if len(trx.Mints) != 0 && !completed {
			mint := NewMint(trx.Mints[len(trx.Mints)-1])
			if err := s.Load(mint); err != nil {
				return err
			}

			burn.FeeTo = &mint.To
			burn.FeeLiquidity = mint.Liquidity.Ptr()
			// remove the logical mint
			if err := s.Remove(mint); err != nil {
				return err
			}
			// update the transaction

			trx.Mints = trx.Mints[:len(trx.Mints)-1]
			if err := s.Save(trx); err != nil {
				return err
			}
		}

		if err := s.Save(burn); err != nil {
			return err
		}

		if burn.NeedsComplete {
			trx.Burns[len(trx.Burns)-1] = burn.ID
		} else {
			trx.Burns = append(trx.Burns, burn.ID)
		}

		if err := s.Save(trx); err != nil {
			return err
		}
	}

	if err := s.Save(trx); err != nil {
		return err
	}

	return nil
}

func (s *Subgraph) isCompleteMint(mintId string) (bool, error) {
	mint := NewMint(mintId)
	err := s.Load(mint)
	if err != nil {
		return false, err
	}
	senderStr := ""
	var completed bool
	if mint.Sender != nil {
		senderStr = *mint.Sender
		completed = true
	}
	s.Log.Debug("checking if mint is completed", zap.String("mint_id", mintId), zap.Bool("completed", completed), zap.String("sender", senderStr))

	//   return MintEvent.load(mintId).sender !== null
	return completed, nil
}
