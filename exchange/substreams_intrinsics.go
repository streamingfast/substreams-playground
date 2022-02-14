package exchange

import (
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/sparkle/indexer"
	"github.com/streamingfast/sparkle/subgraph"
)

type SubstreamIntrinsics struct {
	rpcCache      *indexer.RPCCache
	rpcClient     *rpc.Client
	noArchiveMode bool

	currentBlock bstream.BlockRef
}

func NewSubstreamIntrinsics(rpcClient *rpc.Client, rpcCache *indexer.RPCCache, noArchiveMode bool) *SubstreamIntrinsics {
	return &SubstreamIntrinsics{
		rpcClient:     rpcClient,
		rpcCache:      rpcCache,
		noArchiveMode: noArchiveMode,
	}
}
func (s *SubstreamIntrinsics) SetCurrentBlock(ref bstream.BlockRef) {
	s.currentBlock = ref
}

func (i *SubstreamIntrinsics) RPC(calls []*subgraph.RPCCall) ([]*subgraph.RPCResponse, error) {
	return indexer.DoRPCCalls(i.noArchiveMode, i.currentBlock.Num(), i.rpcClient, i.rpcCache, calls)
}
