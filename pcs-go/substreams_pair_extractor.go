package pcs

import (
	"bytes"
	"fmt"
	"math/big"

	eth "github.com/streamingfast/eth-go"
	pbcodec "github.com/streamingfast/substream-pancakeswap/pb/sf/ethereum/codec/v1"
	imports "github.com/streamingfast/substreams/native-imports"
	pbsubstreams "github.com/streamingfast/substreams/pb/sf/ethereum/substreams/v1"
)

type PairExtractor struct {
	*imports.Imports
}

// Map function can take one or more input objects, sync'd by the `Block` clock.
func (p *PairExtractor) Map(block *pbcodec.Block) (pairs *Pairs, err error) {
	pairs = &Pairs{}
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

			pairs.Pairs = append(pairs.Pairs, &Pair{
				Address:               ev.Pair.Pretty(),
				Erc20Token0:           erc20Token0,
				Erc20Token1:           erc20Token1,
				CreationTransactionId: eth.Hash(trx.Hash).Pretty(),
				BlockNum:              block.Number,

				LogOrdinal: ord,
			})
		}
	}
	if len(pairs.Pairs) == 0 {
		return nil, nil
	}
	return
}

var decimalsMethod = eth.MustNewMethodDef("decimals() (uint256)")
var decimalsMethodSig = decimalsMethod.MethodID()
var nameMethod = eth.MustNewMethodDef("name() (string)")
var nameMethodSig = nameMethod.MethodID()
var symbolMethod = eth.MustNewMethodDef("symbol() (string)")
var symbolMethodSig = symbolMethod.MethodID()

func (p *PairExtractor) getToken(addr eth.Address) (*ERC20Token, error) {
	addrBytes := addr.Bytes()
	calls := &pbsubstreams.RpcCalls{
		Calls: []*pbsubstreams.RpcCall{
			{
				ToAddr:          addrBytes,
				MethodSignature: decimalsMethodSig,
			},
			{
				ToAddr:          addrBytes,
				MethodSignature: nameMethodSig,
			},
			{
				ToAddr:          addrBytes,
				MethodSignature: symbolMethodSig,
			},
		},
	}

	resps := p.RPC(calls)

	token := &ERC20Token{Address: addr.Pretty()}

	decimalsResponse := resps.Responses[0]
	if !decimalsResponse.Failed {
		decoded, err := decimalsMethod.DecodeOutput(decimalsResponse.Raw)
		if err != nil {
			return nil, fmt.Errorf("decoding token decimals() response: %w", err)
		}

		token.Decimals = uint64(decoded[0].(*big.Int).Uint64())
	}

	nameResponse := resps.Responses[1]
	if !nameResponse.Failed {
		decoded, err := nameMethod.DecodeOutput(nameResponse.Raw)
		if err != nil {
			return nil, fmt.Errorf("decoding token name() response: %w", err)
		}
		token.Name = decoded[0].(string)
	} else {
		token.Name = "unknown"
	}

	symbolResponse := resps.Responses[2]
	if !symbolResponse.Failed {
		decoded, err := symbolMethod.DecodeOutput(symbolResponse.Raw)
		if err != nil {
			return nil, fmt.Errorf("decoding token symbol() response: %w", err)
		}
		token.Symbol = decoded[0].(string)
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
