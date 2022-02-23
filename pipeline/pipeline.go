package pipeline

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/sparkle-pancakeswap/exchange"
	"github.com/streamingfast/sparkle-pancakeswap/state"
	"github.com/streamingfast/sparkle-pancakeswap/subscription"
	"github.com/streamingfast/sparkle/indexer"
	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
	"go.uber.org/zap"
)

type Pipeline struct {
	startBlockNum uint64

	rpcClient       *rpc.Client
	subscriptionHub *subscription.Hub
	rpcCache        *indexer.RPCCache

	intr   *exchange.SubstreamIntrinsics
	stores map[string]*state.Builder
}

func New(startBlockNum uint64, rpcClient *rpc.Client, rpcCache *indexer.RPCCache, stores map[string]*state.Builder) *Pipeline {
	pipe := &Pipeline{
		startBlockNum: startBlockNum,
		rpcClient:     rpcClient,
		rpcCache:      rpcCache,
		intr:          exchange.NewSubstreamIntrinsics(rpcClient, rpcCache, true),
		stores:        stores,
	}
	pipe.setupSubscriptionHub()
	pipe.setupPrintPairUpdates()
	return pipe
}

func (p *Pipeline) setupSubscriptionHub() {
	// TODO: wwwooah, SubscriptionHub has a meaning in the context of bstream,
	// this would be *another* flavor SubscriptionHub? We're talking of a generic Pub/Sub here?
	//
	// Let's discuss the purpose of this thing.
	p.subscriptionHub = subscription.NewHub()

	for storeName := range p.stores {
		if err := p.subscriptionHub.RegisterTopic(storeName); err != nil {
			zlog.Fatal("pair subscriber register topic", zap.Error(err))
		}
	}

}

func (p *Pipeline) setupPrintPairUpdates() {
	pairSubscriber := subscription.NewSubscriber()
	if err := p.subscriptionHub.Subscribe(pairSubscriber, "pairs"); err != nil {
		zlog.Fatal("subscription hub subscribe", zap.Error(err))
	}

	go func() {
		for {
			delta, err := pairSubscriber.Next()
			if err != nil {
				zlog.Fatal("pair subscriber next", zap.Error(err))
			}
			if !strings.HasPrefix(delta.Key, "pair") {
				continue
			}

			p.stores["pairs"].PrintDelta(delta)

		}
	}()
	// End subscription hub
}

// type Mapper interface {
// 	Map() error
// }
// type StateBuilder interface {
// 	BuildState() error
// }

func (p *Pipeline) HandlerFactory(blockCount uint64) bstream.Handler {
	// maps := map[string]Mapper{
	// 	"pairExtractor": &exchange.PairExtractor{SubstreamIntrinsics: p.intr},
	// }
	// states := map[string]StateBuilder{
	// 	"pairs": &exchange.PairsStateBuilder{SubstreamIntrinsics: p.intr},

	// }

	pairExtractor := &exchange.PairExtractor{SubstreamIntrinsics: p.intr}
	pairsStateBuilder := &exchange.PairsStateBuilder{SubstreamIntrinsics: p.intr}
	totalPairsStateBuilder := &exchange.TotalPairsStateBuilder{SubstreamIntrinsics: p.intr}
	pricesStateBuilder := &exchange.PricesStateBuilder{SubstreamIntrinsics: p.intr}
	reservesExtractor := &exchange.ReservesExtractor{SubstreamIntrinsics: p.intr}
	swapsExtractor := &exchange.SwapsExtractor{SubstreamIntrinsics: p.intr}
	volume24hStateBuilder := &exchange.PCSVolume24hStateBuilder{SubstreamIntrinsics: p.intr}

	return bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {

		// TODO: eventually, handle the `undo` signals.
		//  NOTE: The RUNTIME will handle the undo signals. It'll have all it needs.
		if block.Number >= p.startBlockNum+blockCount {
			//
			// FLUSH ALL THE STORES TO DISK
			// PRINT THE BLOCK NUMBER WHERE WE STOP, NEXT TIME START FROM THERE
			//
			for _, s := range p.stores {
				s.WriteState(context.Background(), block)
			}

			p.rpcCache.Save(context.Background())

			return io.EOF
		}

		blk := block.ToProtocol().(*pbcodec.Block)
		p.intr.SetCurrentBlock(blk)

		fmt.Println("-------------------------------------------------------------------")
		fmt.Printf("BLOCK +%d %d %s\n", blk.Num()-p.startBlockNum, blk.Num(), blk.ID())

		pairs, err := pairExtractor.Map(blk)
		if err != nil {
			return fmt.Errorf("extracting pairs: %w", err)
		}
		pairs.Print()

		if err := pairsStateBuilder.BuildState(pairs, p.stores["pairs"]); err != nil {
			return fmt.Errorf("processing pair cache: %w", err)
		}
		p.stores["pairs"].PrintDeltas()

		// subscription hub thing
		err = p.subscriptionHub.BroadcastDeltas("pairs", p.stores["pairs"].Deltas)
		if err != nil {
			return fmt.Errorf("broadcasting deltas for topic [pairs]")
		}
		// END subscription hub

		reserveUpdates, err := reservesExtractor.Map(blk, p.stores["pairs"])
		if err != nil {
			return fmt.Errorf("processing reserves extractor: %w", err)
		}
		reserveUpdates.Print()

		if err := pricesStateBuilder.BuildState(reserveUpdates, p.stores["pairs"], p.stores["prices"]); err != nil {
			return fmt.Errorf("pairs price building: %w", err)
		}
		p.stores["prices"].PrintDeltas()

		swaps, err := swapsExtractor.Map(blk, p.stores["pairs"], p.stores["prices"])
		if err != nil {
			return fmt.Errorf("swaps extractor: %w", err)
		}

		swaps.Print()

		if err := totalPairsStateBuilder.BuildState(pairs, swaps, p.stores["total_pairs"]); err != nil {
			return fmt.Errorf("processing total pairs: %w", err)
		}
		p.stores["total_pairs"].PrintDeltas()

		if err := volume24hStateBuilder.BuildState(blk, swaps, p.stores["volume24h"]); err != nil {
			return fmt.Errorf("volume24 builder: %w", err)
		}

		p.stores["volume24h"].PrintDeltas()

		// for _, s := range p.stores {
		// 	err := s.StoreBlock(context.Background(), block)
		// 	if err != nil {
		// 		return err
		// 	}
		// }

		// Prep for next block, clean-up all deltas. This ought to be
		// done by the runtime, when doing clean-up between blocks.
		for _, s := range p.stores {
			s.Flush()
		}

		// MARK INDEX:
		// if len(pairs) != 0 || len(reserveUpdates) != 0 {
		// 	indexer.MarkBlock(block) // each 100 blocks y'écrit whatever
		// }

		return nil
	})
}
