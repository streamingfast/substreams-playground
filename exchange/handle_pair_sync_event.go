package exchange

import (
	"fmt"
	"math/big"

	"github.com/streamingfast/sparkle/entity"
	"go.uber.org/zap"
)

func (s *Subgraph) HandlePairSyncEvent(event *PairSyncEvent) error {
	if s.StepBelow(2) {
		return nil
	}

	pair := NewPair(event.LogAddress.Pretty())
	if err := s.Load(pair); err != nil {
		return fmt.Errorf("loading pair: %s :%w", event.LogAddress.Pretty(), err)
	}

	if !pair.Exists() {
		return fmt.Errorf("could not find pair %s", event.LogAddress.Pretty())
	}

	token0 := NewToken(pair.Token0)
	if err := s.Load(token0); err != nil {
		return fmt.Errorf("loading token 0: %s of pair: %s:%w", pair.Token0, event.LogAddress.Pretty(), err)
	}

	token1 := NewToken(pair.Token1)
	if err := s.Load(token1); err != nil {
		return fmt.Errorf("loading token 1: %s of pair: %s :%w", pair.Token1, event.LogAddress.Pretty(), err)
	}

	pancake := NewPancakeFactory(FactoryAddress)
	if err := s.Load(pancake); err != nil {
		return err
	}

	s.Log.Debug("handler sync pre dump",
		zap.Reflect("token0", token0),
		zap.Reflect("token1", token1),
		zap.Reflect("pancake", pancake),
		zap.Reflect("pair", pair),
	)

	// reset factory liquidity by subtracting only tracked liquidity
	pancake.TotalLiquidityBNB = F(bf().Sub(
		pancake.TotalLiquidityBNB.Float(),
		pair.TrackedReserveBNB.Float(),
	))

	s.Log.Debug("removed tracked reserved BNB", zap.Stringer("value", pancake.TotalLiquidityBNB.Float()))

	token0.TotalLiquidity = F(bf().Sub(token0.TotalLiquidity.Float(), pair.Reserve0.Float()))
	token1.TotalLiquidity = F(bf().Sub(token1.TotalLiquidity.Float(), pair.Reserve1.Float()))

	pair.Reserve0 = F(entity.ConvertTokenToDecimal(event.Reserve0, token0.Decimals.Int().Int64()))
	pair.Reserve1 = F(entity.ConvertTokenToDecimal(event.Reserve1, token1.Decimals.Int().Int64()))

	if pair.Reserve1.Float().Cmp(bf()) != 0 {
		pair.Token0Price = F(bf().Quo(pair.Reserve0.Float(), pair.Reserve1.Float()))
	} else {
		pair.Token0Price = FL(0)
	}

	if pair.Reserve0.Float().Cmp(bf()) != 0 {
		pair.Token1Price = F(bf().Quo(pair.Reserve1.Float(), pair.Reserve0.Float()))
	} else {
		pair.Token1Price = FL(0)
	}

	zlog.Debug("set token prices",
		zap.Stringer("pair.token_0_price", pair.Token0Price),
		zap.Stringer("pair.token_1_price", pair.Token1Price),
	)

	// We need to compute the BNB price *before* we save the pair (code just below)
	// the reason for this, is that we don't want the reserver that are set above to affect
	// the calcualtion of the BNB price (this was taken from the typsecript code)
	bnbPrice, err := s.GetBnbPriceInUSD()
	if err != nil {
		return err
	}

	if s.StepBelow(3) {
		// In parralel reproc, we are ending here if step is below 3, as such, we need to save the pair right away
		s.Log.Debug("updated pair", zap.Reflect("pair", pair))
		if err := s.Save(pair); err != nil {
			return err
		}

		return nil
	}

	bundle := NewBundle("1")
	if err := s.Load(bundle); err != nil {
		return err
	}

	prevBnbPrice := bundle.BnbPrice
	bundle.BnbPrice = F(bnbPrice)
	if err := s.Save(bundle); err != nil {
		return err
	}
	s.Log.Debug("updated bundle price", zap.Reflect("bundle", bundle), zap.Any("prev_bnb_price", prevBnbPrice), zap.Uint64("block_number", event.Block.Number), zap.Stringer("transaction_id", event.Transaction.Hash))

	t0DerivedBNB, err := s.FindBnbPerToken(token0.ID)
	if err != nil {
		return err
	}

	zlog.Debug("calculated derived BNB price for token0", zap.String("value", t0DerivedBNB.Text('g', -1)))

	token0.DerivedBNB = F(t0DerivedBNB).Ptr()
	token0.DerivedUSD = F(bf().Mul(t0DerivedBNB, bnbPrice)).Ptr()
	if err := s.Save(token0); err != nil {
		return err
	}

	t1DerivedBNB, err := s.FindBnbPerToken(token1.ID)
	if err != nil {
		return err
	}

	zlog.Debug("calculated derived BNB price for token1", zap.String("value", t1DerivedBNB.Text('g', -1)))

	token1.DerivedBNB = F(t1DerivedBNB).Ptr()
	token1.DerivedUSD = F(bf().Mul(t1DerivedBNB, bnbPrice)).Ptr()
	if err := s.Save(token1); err != nil {
		return err
	}

	s.Log.Debug("new token prices",
		zap.Stringer("token0", token0.DerivedBNB.Float()),
		zap.Stringer("token1", token1.DerivedBNB.Float()),
	)

	// get tracked liquidity - will be 0 if neither is in whitelist
	trackedLiquidityBNB := big.NewFloat(0)
	if bnbPrice.Cmp(bf()) != 0 {
		tr := getTrackedLiquidityUSD(bundle, pair.Reserve0.Float(), token0, pair.Reserve1.Float(), token1)
		trackedLiquidityBNB = bf().Quo(
			tr,
			bnbPrice,
		)
	}

	s.Log.Debug("new tracked liquidity bnb in the pair",
		zap.String("value", trackedLiquidityBNB.Text('g', -1)),
	)

	// use derived amounts within pair
	pair.TrackedReserveBNB = F(trackedLiquidityBNB)

	pair.ReserveBNB = F(bf().Add(
		bf().Mul(
			pair.Reserve0.Float(),
			t0DerivedBNB,
		),
		bf().Mul(
			pair.Reserve1.Float(),
			t1DerivedBNB,
		),
	))

	pair.ReserveUSD = F(bf().Mul(
		pair.ReserveBNB.Float(),
		bnbPrice,
	))

	// use tracked amounts globally

	pancake.TotalLiquidityBNB = entity.FloatAdd(pancake.TotalLiquidityBNB, F(trackedLiquidityBNB))
	pancake.TotalLiquidityUSD = F(bf().Mul(
		pancake.TotalLiquidityBNB.Float(),
		bnbPrice,
	))

	token0.TotalLiquidity = entity.FloatAdd(token0.TotalLiquidity, pair.Reserve0)
	token1.TotalLiquidity = entity.FloatAdd(token1.TotalLiquidity, pair.Reserve1)

	// save entities
	if err := s.Save(pair); err != nil {
		return err
	}

	if err := s.Save(pancake); err != nil {
		return err
	}

	if err := s.Save(token0); err != nil {
		return err
	}

	if err := s.Save(token1); err != nil {
		return err
	}

	return nil
}

var MINIMUM_LIQUIDITY_THRESHOLD_BNB = big.NewFloat(10)

func (s *Subgraph) FindBnbPerToken(tokenAddress string) (*big.Float, error) {
	if tokenAddress == WBNB_ADDRESS {
		return big.NewFloat(1), nil
	}

	for _, otherToken := range whitelist {
		pairAddress := s.getPairAddressForTokens(tokenAddress, otherToken)
		if pairAddress == "" {
			s.Log.Debug("pair not found for tokens", zap.String("left", tokenAddress), zap.String("right", otherToken))
			continue
		}

		pair := NewPair(pairAddress)
		if err := s.Load(pair); err != nil {
			return nil, err
		}

		if pair.Token0 == tokenAddress && pair.ReserveBNB.Float().Cmp(MINIMUM_LIQUIDITY_THRESHOLD_BNB) > 0 {
			token1 := NewToken(pair.Token1)
			if err := s.Load(token1); err != nil {
				return nil, err
			}
			return bf().Mul(pair.Token1Price.Float(), token1.DerivedBNB.Float()), nil
		}
		if pair.Token1 == tokenAddress && pair.ReserveBNB.Float().Cmp(MINIMUM_LIQUIDITY_THRESHOLD_BNB) > 0 {
			token0 := NewToken(pair.Token0)
			if err := s.Load(token0); err != nil {
				return nil, err
			}
			return bf().Mul(pair.Token0Price.Float(), token0.DerivedBNB.Float()), nil
		}
	}
	return big.NewFloat(0), nil
}

const (
	WBNB_ADDRESS   = "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
	BUSD_WBNB_PAIR = "0x58f876857a02d6762e0101bb5c46a8c1ed44dc16" // created block 589414
	USDT_WBNB_PAIR = "0x16b9a82891338f9ba80e2d6970fdda79d1eb0dae" // created block 648115
)

func (s *Subgraph) GetBnbPriceInUSD() (*big.Float, error) {
	// fetch bnb prices for each stablecoin
	usdtPair := NewPair(USDT_WBNB_PAIR) // usdt is token0
	if err := s.Load(usdtPair); err != nil {
		return nil, err
	}
	busdPair := NewPair(BUSD_WBNB_PAIR) // busd is token1
	if err := s.Load(busdPair); err != nil {
		return nil, err
	}

	if busdPair.Exists() && usdtPair.Exists() {
		totalLiquidityBNB := bf().Add(
			busdPair.Reserve0.Float(),
			usdtPair.Reserve1.Float(),
		).SetPrec(100)

		if totalLiquidityBNB.Cmp(bf()) != 0 {
			busdWeight := bf().Quo(busdPair.Reserve0.Float(), totalLiquidityBNB).SetPrec(100)
			usdtWeight := bf().Quo(usdtPair.Reserve1.Float(), totalLiquidityBNB).SetPrec(100)

			return bf().Add(
				bf().Mul(
					busdPair.Token1Price.Float(),
					busdWeight,
				).SetPrec(100),
				bf().Mul(
					usdtPair.Token0Price.Float(),
					usdtWeight,
				).SetPrec(100),
			).SetPrec(100), nil
		} else {
			return big.NewFloat(0), nil
		}
	} else if busdPair.Exists() {
		return busdPair.Token1Price.Float().SetPrec(100), nil
	} else if usdtPair.Exists() {
		return usdtPair.Token0Price.Float().SetPrec(100), nil
	}

	return big.NewFloat(0), nil
}
