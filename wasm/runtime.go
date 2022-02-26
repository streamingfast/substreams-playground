package wasm

import (
	"fmt"
	"io/ioutil"

	"github.com/wasmerio/wasmer-go/wasmer"
)

type Instance struct {
	store      *wasmer.Store
	memory     *wasmer.Memory
	heap       *Heap
	entrypoint *wasmer.Function
}

func NewRustInstance(wasmFile string, functionName string) (*Instance, error) {
	wasmBytes, err := ioutil.ReadFile(wasmFile)
	if err != nil {
		return nil, fmt.Errorf("unable to load wasm file %q: %w", wasmFile, err)
	}

	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	instance := &Instance{
		store: store,
	}

	module, err := wasmer.NewModule(instance.store, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to compile wasm file %q: %w", wasmFile, err)
	}

	imports := instance.newImports()
	vmInstance, err := wasmer.NewInstance(module, imports)
	if err != nil {
		return nil, fmt.Errorf("unable to get wasm module instance from %q: %w", wasmFile, err)
	}

	memory, err := vmInstance.Exports.GetMemory("memory")
	if err != nil {
		return nil, fmt.Errorf("unable to get the wasm module memory: %w", err)
	}
	instance.memory = memory
	instance.heap = NewHeap(memory)
	instance.entrypoint, err = vmInstance.Exports.GetRawFunction(functionName)
	if err != nil {
		return nil, fmt.Errorf("unable to get wasm module function %q from %q: %w", functionName, wasmFile, err)
	}

	// heap.allocator, err = instance.Exports.GetFunction("memory.allocate")
	// if err != nil {
	// 	panic(fmt.Errorf("getting memory.allocate func: %w", err))
	// }

	return instance, nil
}

func (i *Instance) newImports() *wasmer.ImportObject {
	imports := wasmer.NewImportObject()
	imports.Register("env", map[string]wasmer.IntoExtern{
		"abort": wasmer.NewFunction(
			i.store,
			wasmer.NewFunctionType(
				params(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32),
				returns(),
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				message, err := i.heap.ReadString(args[0].I32(), 0) // FIXME
				if err != nil {
					return nil, fmt.Errorf("read message argument: %w", err)
				}

				filename, err := i.heap.ReadString(args[1].I32(), 0) // FIXME
				if err != nil {
					return nil, fmt.Errorf("read filename argument: %w", err)
				}

				lineNumber := int(args[2].I32())
				columnNumber := int(args[3].I32())

				fmt.Println("ABORTING", message, filename)

				return nil, &abortError{message, filename, lineNumber, columnNumber}
			},
		),
		"println": wasmer.NewFunction(
			i.store,
			wasmer.NewFunctionType(
				params(wasmer.I32, wasmer.I32),
				returns(),
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				message, err := i.heap.ReadString(args[0].I32(), args[1].I32())
				if err != nil {
					return nil, fmt.Errorf("read message argument: %w", err)
				}

				fmt.Println(message)

				return nil, nil
			},
		),
	})
	return imports
}

func (i *Instance) Execute(block []byte) (out interface{}, err error) {
	params := []interface{}{}

	blockPtr := i.heap.Write(block)
	blockLen := int32(len(block))

	params = append(params, blockPtr, blockLen)

	fmt.Println("PARAMS", params)
	//i.heap.PrintMem()
	out, err = i.entrypoint.Call(params...)
	//i.heap.PrintMem()

	return
	//return toGoValue(out, returnType, i.env)
}
