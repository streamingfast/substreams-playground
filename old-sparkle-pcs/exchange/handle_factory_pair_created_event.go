package exchange

import (
	"fmt"
	"math/big"

	eth "github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
	"github.com/streamingfast/sparkle/subgraph"
)

func (s *Subgraph) newPair(pairAddress, token0Address, token1Address eth.Address, factory *PancakeFactory) (*Pair, error) {
	pair := NewPair(pairAddress.Pretty())

	token0, err := s.getToken(token0Address, factory)
	if err != nil {
		return nil, err
	}

	token1, err := s.getToken(token1Address, factory)
	if err != nil {
		return nil, err
	}

	if err := s.Save(token0); err != nil {
		return nil, err
	}

	if err := s.Save(token1); err != nil {
		return nil, err
	}

	pair.Token0 = token0.ID
	pair.Token1 = token1.ID
	pair.Block = entity.NewIntFromLiteralUnsigned(s.Block().Number())
	pair.Timestamp = entity.NewIntFromLiteral(s.Block().Timestamp().Unix())
	pair.Name = fmt.Sprintf("%s-%s", token0.Symbol, token1.Symbol)

	return pair, nil
}

/*
	token0 = NewToken(ev.Token0.Pretty())
	token0.Name = tm.Name
	token0.Symbol = tm.Symbol
	token0.Decimals = IL(int64(tm.Decimals))

*/

func (s *Subgraph) getToken(tokenAddress eth.Address, factory *PancakeFactory) (*Token, error) {
	if tokenAddress == nil {
		return nil, nil
	}

	token := NewToken(tokenAddress.Pretty())
	err := s.Load(token)
	if err != nil {
		return nil, err
	}

	if token.Exists() {
		return token, nil
	}

	err = s.Save(factory)
	if err != nil {
		return nil, err
	}

	calls := []*subgraph.RPCCall{
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "decimals() (uint256)",
		},
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "name() (string)",
		},
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "symbol() (string)",
		},
		//		{
		//			ToAddr:          tokenAddress.Pretty(),
		//			MethodSignature: "totalSupply() (uint256)",
		//		},
	}

	resps, err := s.RPC(calls)
	if err != nil {
		return nil, fmt.Errorf("rpc call error: %w", err)
	}

	decimalsResponse := resps[0]
	if decimalsResponse.CallError == nil && decimalsResponse.DecodingError == nil {
		token.Decimals = IL(decimalsResponse.Decoded[0].(*big.Int).Int64())
	}

	nameResponse := resps[1]
	if nameResponse.CallError == nil && nameResponse.DecodingError == nil {
		token.Name = nameResponse.Decoded[0].(string)
	} else {
		token.Name = "unknown"
	}

	symbolResponse := resps[2]
	if symbolResponse.CallError == nil && symbolResponse.DecodingError == nil {
		token.Symbol = symbolResponse.Decoded[0].(string)
	} else {
		token.Symbol = "unknown"
	}

	//	totalSupplyResponse := resps[3]
	//	if totalSupplyResponse.CallError == nil && totalSupplyResponse.DecodingError == nil {
	//		token.TotalSupply = IL(totalSupplyResponse.Decoded[0].(*big.Int).Int64())
	//	}

	token.DerivedBNB = FL(0).Ptr()
	token.DerivedUSD = FL(0).Ptr()

	if err := s.Save(token); err != nil {
		return nil, fmt.Errorf("saving token: %w", err)
	}

	return token, nil
}

func (s *Subgraph) HandleFactoryPairCreatedEvent(ev *FactoryPairCreatedEvent) error {
	factory := NewPancakeFactory(FactoryAddress)
	err := s.Load(factory)
	if err != nil {
		return err
	}

	if !factory.Exists() {
		bundle := NewBundle("1")
		if err := s.Save(bundle); err != nil {
			return err
		}
	}

	pair, err := s.newPair(ev.Pair, ev.Token0, ev.Token1, factory)
	if err != nil {
		return err
	}

	factory.TotalPairs = entity.IntAdd(factory.TotalPairs, IL(1))
	if err := s.Save(factory); err != nil {
		return err
	}

	err = s.Save(pair)
	if err != nil {
		return fmt.Errorf("saving pair: %w", err)
	}

	err = s.CreatePairTemplateWithTokens(ev.Pair, ev.Token0, ev.Token1)
	if err != nil {
		return fmt.Errorf("creating pair template: %w", err)
	}

	return nil
}
