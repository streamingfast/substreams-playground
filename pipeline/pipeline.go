package pipeline

import (
	"context"
	"fmt"
	"io"
	"reflect"
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

	streamFuncs := map[string]reflect.Value{
		"pairExtractor":          reflect.ValueOf(&exchange.PairExtractor{SubstreamIntrinsics: p.intr}),
		"pairs":                  reflect.ValueOf(&exchange.PairsStateBuilder{SubstreamIntrinsics: p.intr}),
		"totals":                 reflect.ValueOf(&exchange.TotalPairsStateBuilder{SubstreamIntrinsics: p.intr}),
		"prices":                 reflect.ValueOf(&exchange.PricesStateBuilder{SubstreamIntrinsics: p.intr}),
		"reservesExtractor":      reflect.ValueOf(&exchange.ReservesExtractor{SubstreamIntrinsics: p.intr}),
		"mintBurnSwapsExtractor": reflect.ValueOf(&exchange.SwapsExtractor{SubstreamIntrinsics: p.intr}),
		"volume24h":              reflect.ValueOf(&exchange.PCSVolume24hStateBuilder{SubstreamIntrinsics: p.intr}),
	}

	vals := map[string]reflect.Value{}
	for storeName, store := range p.stores {
		vals[storeName] = reflect.ValueOf(store)
	}

	return bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) (err error) {
		// defer func() {
		// 	if r := recover(); r != nil {
		// 		err = fmt.Errorf("panic: %w", r)
		// 	}
		// }()

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

		p.intr.SetCurrentBlock(block)

		blk := block.ToProtocol().(*pbcodec.Block)
		vals["Block"] = reflect.ValueOf(blk)

		fmt.Println("-------------------------------------------------------------------")
		fmt.Printf("BLOCK +%d %d %s\n", blk.Num()-p.startBlockNum, blk.Num(), blk.ID())

		mapper(vals, streamFuncs, "pairExtractor", []string{"Block"}, true)
		stateBuilder(vals, streamFuncs, "pairs", []string{"pairExtractor"}, true)
		mapper(vals, streamFuncs, "reservesExtractor", []string{"Block", "pairs"}, true)
		stateBuilder(vals, streamFuncs, "prices", []string{"reservesExtractor", "pairs"}, true)
		mapper(vals, streamFuncs, "mintBurnSwapsExtractor", []string{"Block", "pairs", "prices"}, true)
		stateBuilder(vals, streamFuncs, "totals", []string{"pairExtractor", "mintBurnSwapsExtractor"}, true)
		stateBuilder(vals, streamFuncs, "volume24h", []string{"Block", "mintBurnSwapsExtractor"}, true)

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

type Printer interface {
	Print()
}

func printer(in interface{}) {
	if p, ok := in.(Printer); ok {
		p.Print()
	}
}

func mapper(vals map[string]reflect.Value, streams map[string]reflect.Value, name string, inputs []string, printOutputs bool) {
	var inputVals []reflect.Value
	for _, in := range inputs {
		inputVals = append(inputVals, vals[in])
	}
	out := streams[name].MethodByName("Map").Call(inputVals)
	if len(out) != 2 {
		panic("invalid number of outputs for call on Map ethod")
	}
	vals[name] = out[0]

	if err, ok := out[1].Interface().(error); ok && err != nil {
		panic(fmt.Errorf("stream %s: %w", name, err))
	}
}

func stateBuilder(vals map[string]reflect.Value, streams map[string]reflect.Value, name string, inputs []string, printOutputs bool) {
	var inputVals []reflect.Value
	for _, in := range inputs {
		inputVals = append(inputVals, vals[in])
	}
	inputVals = append(inputVals, vals[name])

	// TODO: we can cache the `Method` retrieved on the stream.
	out := streams[name].MethodByName("BuildState").Call(inputVals)
	if len(out) != 1 {
		panic("invalid number of outputs for call on BuildState method")
	}
	if err, ok := out[0].Interface().(error); ok && err != nil {
		panic(fmt.Errorf("stream %s: %w", name, err))
	}
}
