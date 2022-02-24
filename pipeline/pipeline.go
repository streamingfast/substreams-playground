package pipeline

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/sparkle/indexer"
	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
	"github.com/streamingfast/substream-pancakeswap/exchange"
	"github.com/streamingfast/substream-pancakeswap/manifest"
	"github.com/streamingfast/substream-pancakeswap/state"
	"github.com/streamingfast/substream-pancakeswap/subscription"
)

type Pipeline struct {
	startBlockNum uint64

	rpcClient       *rpc.Client
	subscriptionHub *subscription.Hub
	rpcCache        *indexer.RPCCache

	intr   *exchange.SubstreamIntrinsics
	stores map[string]*state.Builder

	manifest         *manifest.Manifest
	outputStreamName string

	streamFuncs   []StreamFunc
	streamOutputs map[string]reflect.Value
}

func New(startBlockNum uint64, rpcClient *rpc.Client, rpcCache *indexer.RPCCache, manif *manifest.Manifest, outputStreamName string) *Pipeline {
	pipe := &Pipeline{
		startBlockNum:    startBlockNum,
		rpcClient:        rpcClient,
		rpcCache:         rpcCache,
		intr:             exchange.NewSubstreamIntrinsics(rpcClient, rpcCache, true),
		stores:           map[string]*state.Builder{},
		manifest:         manif,
		outputStreamName: outputStreamName,
	}
	// pipe.setupSubscriptionHub()
	// pipe.setupPrintPairUpdates()
	return pipe
}

// type Mapper interface {
// 	Map() error
// }
// type StateBuilder interface {
// 	BuildState() error
// }

func (p *Pipeline) Build(ioFactory state.IOFactory, forceLoadState bool) error {
	streams, err := p.manifest.Graph.StreamsFor(p.outputStreamName)
	if err != nil {
		return fmt.Errorf("whoops: %w", err)
	}

	nativeStreams := map[string]reflect.Value{
		"./pairExtractor.wasm":          reflect.ValueOf(&exchange.PairExtractor{SubstreamIntrinsics: p.intr}),
		"./pairsState.wasm":             reflect.ValueOf(&exchange.PairsStateBuilder{SubstreamIntrinsics: p.intr}),
		"./reservesExtractor.wasm":      reflect.ValueOf(&exchange.ReservesExtractor{SubstreamIntrinsics: p.intr}),
		"./pricesState.wasm":            reflect.ValueOf(&exchange.PricesStateBuilder{SubstreamIntrinsics: p.intr}),
		"./mintBurnSwapsExtractor.wasm": reflect.ValueOf(&exchange.SwapsExtractor{SubstreamIntrinsics: p.intr}),
		"./totalsState.wasm":            reflect.ValueOf(&exchange.TotalPairsStateBuilder{SubstreamIntrinsics: p.intr}),
		"./volumesState.wasm":           reflect.ValueOf(&exchange.PCSVolume24hStateBuilder{SubstreamIntrinsics: p.intr}),
	}

	p.stores = map[string]*state.Builder{}
	p.streamOutputs = map[string]reflect.Value{}

	for _, stream := range streams {
		f, found := nativeStreams[stream.Code]
		if !found {
			// TODO: eventually, LOAD the CODE Into WASM boom!
			return fmt.Errorf("native code not found for %q", stream.Code)
		}

		debugOutput := stream.Name == p.outputStreamName
		inputs := []string{}
		for _, in := range stream.Inputs {
			inputs = append(inputs, strings.Split(in, ":")[1])
		}
		streamName := stream.Name // to ensure it's enclosed

		switch stream.Kind {
		case "Mapper":
			method := f.MethodByName("Map")
			if method.IsZero() {
				return fmt.Errorf("Map() method not found on %T", f.Interface())
			}
			fmt.Printf("Adding mapper for stream %q\n", stream.Name)
			p.streamFuncs = append(p.streamFuncs, func() error {
				return mapper(p.streamOutputs, method, streamName, inputs, debugOutput)
			})
		case "StateBuilder":
			method := f.MethodByName("BuildState")
			if method.IsZero() {
				return fmt.Errorf("BuildState() method not found on %T", f.Interface())
			}
			store := state.New(stream.Name, ioFactory)
			if forceLoadState {
				// Use AN ABSOLUTE store, or SQUASH ALL PARTIAL!

				if err := store.Init(p.startBlockNum); err != nil {
					return fmt.Errorf("could not load state for store %s at block num %d: %w", stream.Name, p.startBlockNum, err)
				}
			}
			p.stores[stream.Name] = store
			p.streamOutputs[stream.Name] = reflect.ValueOf(store)

			fmt.Printf("Adding state builder for stream %q\n", stream.Name)
			p.streamFuncs = append(p.streamFuncs, func() error {
				return stateBuilder(p.streamOutputs, method, streamName, inputs, debugOutput)
			})

		default:
			return fmt.Errorf("unknown value %q for 'kind' in stream %q", stream.Kind, stream.Name)
		}

	}

	return nil
}

// `stateBuidler` aura 4 modes d'opération:
//   * fetch an absolute snapshot from disk at EXACTLY the point we're starting
//   * fetch a partial snapshot, and fuse with previous snapshots, in which I need local "pairExtractor" building.
//   * connect to a remote firehose (I can cut the upstream dependencies
//   * if resources are available, SCHEDULE on BACKING NODES a parallel processing for that segment
//   * completely roll out LOCALLY the full historic reprocessing BEFORE continuing

type StreamFunc func() error

func (p *Pipeline) HandlerFactory(blockCount uint64) bstream.Handler {
	return bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) (err error) {
		// defer func() {
		// 	if r := recover(); r != nil {
		// 		err = fmt.Errorf("panic: %w", r)
		// 	}
		// }()

		// TODO: eventually, handle the `undo` signals.
		//  NOTE: The RUNTIME will handle the undo signals. It'll have all it needs.
		if block.Number >= p.startBlockNum+blockCount {
			for _, s := range p.stores {
				s.WriteState(context.Background(), block)
			}

			p.rpcCache.Save(context.Background())

			return io.EOF
		}

		p.intr.SetCurrentBlock(block)

		blk := block.ToProtocol().(*pbcodec.Block)
		p.streamOutputs["sf.ethereum.types.v1.Block"] = reflect.ValueOf(blk)

		fmt.Println("-------------------------------------------------------------------")
		fmt.Printf("BLOCK +%d %d %s\n", blk.Num()-p.startBlockNum, blk.Num(), blk.ID())

		for _, streamFunc := range p.streamFuncs {
			if err := streamFunc(); err != nil {
				return err
			}
		}

		// Prep for next block, clean-up all deltas. This ought to be
		// done by the runtime, when doing clean-up between blocks.
		for _, s := range p.stores {
			s.Flush()
		}

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

func mapper(vals map[string]reflect.Value, method reflect.Value, name string, inputs []string, printOutputs bool) error {
	var inputVals []reflect.Value
	for _, in := range inputs {
		inputVals = append(inputVals, vals[in])
	}
	out := method.Call(inputVals)
	if len(out) != 2 {
		return fmt.Errorf("invalid number of outputs for Map call in code for stream %q, should be 2 (data, error)", name)
	}
	vals[name] = out[0]

	p, ok := out[0].Interface().(Printer)
	if ok && printOutputs {
		p.Print()
	}

	if err, ok := out[1].Interface().(error); ok && err != nil {
		return fmt.Errorf("mapper stream %q: %w", name, err)
	}
	return nil
}

func stateBuilder(vals map[string]reflect.Value, method reflect.Value, name string, inputs []string, printOutputs bool) error {
	var inputVals []reflect.Value
	for _, in := range inputs {
		inputVals = append(inputVals, vals[in])
	}
	inputVals = append(inputVals, vals[name])

	// TODO: we can cache the `Method` retrieved on the stream.
	out := method.Call(inputVals)
	if len(out) != 1 {
		return fmt.Errorf("invalid number of outputs for BuildState call in code for stream %q, should be 1 (error)", name)
	}
	p, ok := vals[name].Interface().(Printer)
	if ok && printOutputs {
		p.Print()
	}
	if err, ok := out[0].Interface().(error); ok && err != nil {
		return fmt.Errorf("state builder stream %q: %w", name, err)
	}
	return nil
}
