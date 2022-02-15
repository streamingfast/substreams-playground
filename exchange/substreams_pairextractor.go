package exchange

import (
	"bytes"
	"fmt"
	"math/big"

	eth "github.com/streamingfast/eth-go"
	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
	"github.com/streamingfast/sparkle/subgraph"
)

type PairExtractor struct {
	*SubstreamIntrinsics

	UseIndexBuilder string // by name,
	Contract        eth.Address
}

// func (p *IndexBuilder) Map(block *pbcodec.Block) (keys IndexableKeys, err error) {
// 	return nil, nil
// }

// inputs: sf.ethereum.v1.codec.Block
// outputs: pancakeswap.v1.PCSPairs  (index on Nil)

// Map function can take one or more input objects, sync'd by the `Block` clock.
func (p *PairExtractor) Map(block *pbcodec.Block) (pairs PCSPairs, err error) {
	for _, trx := range block.TransactionTraces {
		// WARN: this wouldn't catch those contract calls that are nested in sub-Calls
		if !bytes.Equal(trx.To, p.Contract) {
			continue
		}
		for _, log := range trx.Receipt.Logs {
			// fetch the two tokens from the chain like CRAZY
			ethLog := ssCodecLogToEthLog(log)
			evt, err := DecodeEvent(ethLog, block, trx)
			if err != nil {
				return nil, err
			}

			ev, ok := evt.(*FactoryPairCreatedEvent)
			if !ok {
				continue
			}

			erc20Token0, err := p.getToken(ev.Token0)
			if err != nil {
				return nil, err
			}

			erc20Token1, err := p.getToken(ev.Token1)
			if err != nil {
				return nil, err
			}

			ord := uint64(log.BlockIndex)

			pairs = append(pairs, PCSPair{
				Address:               ev.Pair.Pretty(),
				Token0:                *erc20Token0,
				Token1:                *erc20Token1,
				CreationTransactionID: eth.Hash(trx.Hash).Pretty(),

				// FIXME: When we have boundaries, let's sprinkle some in here, for greater precision.
				LogOrdinal:       ord,
				CallStartOrdinal: ord,
				CallEndOrdinal:   ord,
				TrxStartOrdinal:  ord,
				TrxEndOrdinal:    ord,
			})
		}
	}
	return
}

func (p *PairExtractor) getToken(addr eth.Address) (*ERC20Token, error) {
	return &ERC20Token{
		Decimals: 8,
		Name:     "Bitcoin",
		Symbol:   "BSV",
	}, nil
	calls := []*subgraph.RPCCall{
		{
			ToAddr:          addr.Pretty(),
			MethodSignature: "decimals() (uint256)",
		},
		{
			ToAddr:          addr.Pretty(),
			MethodSignature: "name() (string)",
		},
		{
			ToAddr:          addr.Pretty(),
			MethodSignature: "symbol() (string)",
		},
		//		{
		//			ToAddr:          addr.Pretty(),
		//			MethodSignature: "totalSupply() (uint256)",
		//		},
	}

	resps, err := p.RPC(calls)
	if err != nil {
		return nil, fmt.Errorf("rpc call error: %w", err)
	}

	token := &ERC20Token{Address: addr.String()}

	decimalsResponse := resps[0]
	if decimalsResponse.CallError == nil && decimalsResponse.DecodingError == nil {
		token.Decimals = uint32(decimalsResponse.Decoded[0].(*big.Int).Uint64())
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

	return token, nil
}

func ssCodecLogToEthLog(l *pbcodec.Log) *eth.Log {
	return &eth.Log{
		Address:    l.Address,
		Topics:     l.Topics,
		Data:       l.Data,
		Index:      l.Index,
		BlockIndex: l.BlockIndex,
	}
}
