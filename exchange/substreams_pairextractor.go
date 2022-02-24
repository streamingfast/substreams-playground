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
}

// inputs: sf.ethereum.v1.codec.Block
// outputs: pancakeswap.v1.PCSPairs  (index on Nil)

// AS: U32(pointer to block begin), U32(block length), U32(readable_store_index), U32(writable_store_index), U32(), U32(poitner to start where I'll read 4 bytes for length), U32(pointer to start where I'll return value), bytes du pointer de block

// func map(block []byte, pairs_store_index uint32) (pairs []byte) {
// 	proto.Unmarshal(block, &myblock)
// 	env.store_get_last(pairs_store_index, "key")

// 	return proto.Marshal(pairs)
// }

// Map function can take one or more input objects, sync'd by the `Block` clock.
func (p *PairExtractor) Map(block *pbcodec.Block) (pairs PCSPairs, err error) {
	for _, trx := range block.TransactionTraces {
		// WARN: this wouldn't catch those contract calls that are nested in sub-Calls
		if !bytes.Equal(trx.To, FactoryAddressBytes) {
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
				BlockNum:              block.Number,

				LogOrdinal: ord,
			})
		}
	}
	return
}

func (p *PairExtractor) getToken(addr eth.Address) (*ERC20Token, error) {
	// return &ERC20Token{
	// 	Address:  addr.Pretty(),
	// 	Decimals: 8,
	// 	Name:     "Bitcoin",
	// 	Symbol:   "BSV",
	// }, nil
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

	token := &ERC20Token{Address: addr.Pretty()}

	decimalsResponse := resps[0]
	if decimalsResponse.CallError == nil && decimalsResponse.DecodingError == nil {
		token.Decimals = int64(decimalsResponse.Decoded[0].(*big.Int).Uint64())
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
