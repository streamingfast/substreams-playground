package exchange

import (
	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
)

//
// ERC-20 Token Extractor
//
// That would be a strict ERC-20 token extractor, to feed into a StateStream (or stateful stream?)
//
type TokenExtractor struct {
	PickupContractDeployments bool
	PickupTransfers           bool
}

// Map takes inputs, and produces outputs. Many calls to these can be run in parallel, for different
// blocks, as they are stateless.
func (p *TokenExtractor) Map(block *pbcodec.Block) (tokens *ERC20Tokens, err error) {
	for _, trx := range block.TransactionTraces {
		if trx.Status != pbcodec.TransactionTraceStatus_SUCCEEDED {
			// WARN: check that this is the RIGHT thing to do
			continue
		}

		// TODO: each time there's a new contract created, call it with `eth_call` to
		// see if it matches the ERC20 interface, save it value, etc..
	}
	return nil, nil
}
