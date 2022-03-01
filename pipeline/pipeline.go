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
	"github.com/streamingfast/substream-pancakeswap/wasm"
	"google.golang.org/protobuf/proto"
)

type Pipeline struct {
	vmType        string // wasm, native
	startBlockNum uint64

	rpcClient       *rpc.Client
	subscriptionHub *subscription.Hub
	rpcCache        *indexer.RPCCache

	intr   *exchange.SubstreamIntrinsics
	stores map[string]*state.Builder

	manifest         *manifest.Manifest
	outputStreamName string

	streamFuncs   []StreamFunc
	nativeOutputs map[string]reflect.Value
	wasmOutputs   map[string][]byte
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
		vmType:           manif.CodeType,
	}
	// pipe.setupSubscriptionHub()
	// pipe.setupPrintPairUpdates()
	return pipe
}

func (p *Pipeline) BuildNative(ioFactory state.IOFactory, forceLoadState bool) error {
	streams, err := p.manifest.Graph.StreamsFor(p.outputStreamName)
	if err != nil {
		return fmt.Errorf("whoops: %w", err)
	}

	nativeStreams := map[string]reflect.Value{
		"pairExtractor":          reflect.ValueOf(&exchange.PairExtractor{SubstreamIntrinsics: p.intr}),
		"pairsState":             reflect.ValueOf(&exchange.PairsStateBuilder{SubstreamIntrinsics: p.intr}),
		"reservesExtractor":      reflect.ValueOf(&exchange.ReservesExtractor{SubstreamIntrinsics: p.intr}),
		"pricesState":            reflect.ValueOf(&exchange.PricesStateBuilder{SubstreamIntrinsics: p.intr}),
		"mintBurnSwapsExtractor": reflect.ValueOf(&exchange.SwapsExtractor{SubstreamIntrinsics: p.intr}),
		"totalsState":            reflect.ValueOf(&exchange.TotalPairsStateBuilder{SubstreamIntrinsics: p.intr}),
		"volumesState":           reflect.ValueOf(&exchange.PCSVolume24hStateBuilder{SubstreamIntrinsics: p.intr}),
	}

	p.stores = map[string]*state.Builder{}
	p.nativeOutputs = map[string]reflect.Value{}

	for _, stream := range streams {
		f, found := nativeStreams[stream.Code.Native]
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
				return nativeMapper(p.nativeOutputs, method, streamName, inputs, debugOutput)
			})
		case "StateBuilder":
			method := f.MethodByName("BuildState")
			if method.IsZero() {
				return fmt.Errorf("BuildState() method not found on %T", f.Interface())
			}

			store := state.New(stream.Name, stream.Output.StoreMergeStrategy, ioFactory)
			if forceLoadState {
				// Use AN ABSOLUTE store, or SQUASH ALL PARTIAL!

				if err := store.Init(p.startBlockNum); err != nil {
					return fmt.Errorf("could not load state for store %s at block num %d: %w", stream.Name, p.startBlockNum, err)
				}
			}
			p.stores[stream.Name] = store
			p.nativeOutputs[stream.Name] = reflect.ValueOf(store)

			fmt.Printf("Adding state builder for stream %q\n", stream.Name)
			p.streamFuncs = append(p.streamFuncs, func() error {
				return nativeStateBuilder(p.nativeOutputs, method, streamName, inputs, debugOutput)
			})

		default:
			return fmt.Errorf("unknown value %q for 'kind' in stream %q", stream.Kind, stream.Name)
		}

	}

	p.vmType = "native"

	return nil
}

func (p *Pipeline) BuildWASM(ioFactory state.IOFactory, forceLoadState bool) error {
	streams, err := p.manifest.Graph.StreamsFor(p.outputStreamName)
	if err != nil {
		return fmt.Errorf("building execution graph: %w", err)
	}

	p.stores = map[string]*state.Builder{}
	p.wasmOutputs = map[string][]byte{}

	for _, stream := range streams {
		debugOutput := stream.Name == p.outputStreamName
		var inputs []wasm.Input
		for _, in := range stream.Inputs {
			t := strings.Split(in, ":") // TODO: check we do have 2 and only 2 parts.
			switch t[0] {
			case "stream":

				inputs = append(inputs, wasm.Input{
					Type: wasm.InputStream,
					Name: t[1],
				})
			case "store":
				inputs = append(inputs, wasm.Input{
					Type:  wasm.InputStore,
					Name:  t[1],
					Store: p.stores[t[1]],
				})
			default:
				return fmt.Errorf("invalid input type %q for stream %q in input %q", t[0], stream.Name, in)
			}
		}
		streamName := stream.Name // to ensure it's enclosed

		mod, err := wasm.NewModule(stream.Code.Content)
		if err != nil {
			return fmt.Errorf("new wasm module: %w", err)
		}

		switch stream.Kind {
		case "Mapper":
			fmt.Printf("Adding mapper for stream %q\n", stream.Name)
			p.streamFuncs = append(p.streamFuncs, func() error {
				return wasmMapper(p.wasmOutputs, mod, stream.Code.Entrypoint, streamName, inputs, debugOutput)
			})
		case "StateBuilder":
			inputs = append(inputs, wasm.Input{
				Type:          wasm.OutputStore,
				Name:          streamName,
				Store:         p.stores[streamName],
				MergeStrategy: stream.Output.StoreMergeStrategy,
			})
			fmt.Printf("Adding state builder for stream %q\n", stream.Name)
			p.streamFuncs = append(p.streamFuncs, func() error {
				return wasmStateBuilder(p.wasmOutputs, mod, stream.Code.Entrypoint, streamName, inputs, debugOutput)
			})

		// case "StateBuilder":
		// 	method := f.MethodByName("BuildState")
		// 	if method.IsZero() {
		// 		return fmt.Errorf("BuildState() method not found on %T", f.Interface())
		// 	}
		// 	store := state.New(stream.Name, ioFactory)
		// 	if forceLoadState {
		// 		// Use AN ABSOLUTE store, or SQUASH ALL PARTIAL!

		// 		if err := store.Init(p.startBlockNum); err != nil {
		// 			return fmt.Errorf("could not load state for store %s at block num %d: %w", stream.Name, p.startBlockNum, err)
		// 		}
		// 	}
		// 	p.stores[stream.Name] = store
		// 	p.wasmOutputs[stream.Name] = reflect.ValueOf(store)

		// 	fmt.Printf("Adding state builder for stream %q\n", stream.Name)
		// 	p.streamFuncs = append(p.streamFuncs, func() error {
		// 		return stateBuilder(p.streamOutputs, method, streamName, inputs, debugOutput)
		// 	})

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
		switch p.vmType {
		case "native":
			p.nativeOutputs["sf.ethereum.types.v1.Block"] = reflect.ValueOf(blk)
		case "wasm/rust-v1":
			// block.Payload.Get() could do the same, but does it go through the same
			// CORRECTIONS of the block, that the BlockDecoder does?
			blkBytes, err := proto.Marshal(blk)
			if err != nil {
				return fmt.Errorf("packing block: %w", err)
			}

			p.wasmOutputs["sf.ethereum.types.v1.Block"] = blkBytes
		default:
			panic("unsupported vmType " + p.vmType)
		}

		fmt.Println("-------------------------------------------------------------------")
		fmt.Printf("BLOCK +%d %d %s\n", blk.Num()-p.startBlockNum, blk.Num(), blk.ID())

		// runtime.LockOSThread()
		// defer runtime.UnlockOSThread()
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

func nativeMapper(vals map[string]reflect.Value, method reflect.Value, name string, inputs []string, printOutputs bool) error {
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

func nativeStateBuilder(vals map[string]reflect.Value, method reflect.Value, name string, inputs []string, printOutputs bool) error {
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

func wasmMapper(vals map[string][]byte, mod *wasm.Module, entrypoint string, name string, inputs []wasm.Input, printOutputs bool) (err error) {
	var vm *wasm.Instance
	if vm, err = wasmCall(vals, mod, entrypoint, name, inputs); err != nil {
		return err
	}

	out := vm.Output()

	vals[name] = out

	if len(out) != 0 && printOutputs {
		fmt.Printf("Stream output %q:\n    %v\n", name, out)
	}

	return nil
}

func wasmStateBuilder(vals map[string][]byte, mod *wasm.Module, entrypoint string, name string, inputs []wasm.Input, printOutputs bool) (err error) {
	var vm *wasm.Instance
	if vm, err = wasmCall(vals, mod, entrypoint, name, inputs); err != nil {
		return err
	}

	if printOutputs {
		vm.PrintDeltas()
	}

	return nil
}

func wasmCall(vals map[string][]byte, mod *wasm.Module, entrypoint string, name string, inputs []wasm.Input) (*wasm.Instance, error) {
	vmInst, err := mod.NewInstance(entrypoint)
	if err != nil {
		return nil, fmt.Errorf("new wasm instance: %w", err)
	}

	for _, input := range inputs {
		if input.Type == wasm.InputStream {
			input.StreamData = vals[input.Name]
		}
	}

	if err = vmInst.Execute(inputs); err != nil {
		return nil, fmt.Errorf("stream %s: wasm execution failed: %w", name, err)
	}

	return vmInst, err
}
