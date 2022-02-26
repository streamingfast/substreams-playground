package wasm

import (
	"fmt"

	"github.com/wasmerio/wasmer-go/wasmer"
)

type Instance struct {
	store      *wasmer.Store
	memory     *wasmer.Memory
	heap       *Heap
	entrypoint *wasmer.Function

	returnValue []byte
	panicError  *PanicError
}

func NewRustInstance(wasmCode []byte, functionName string) (*Instance, error) {
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	instance := &Instance{
		store: store,
	}

	module, err := wasmer.NewModule(instance.store, wasmCode)
	if err != nil {
		return nil, fmt.Errorf("building wasm module:%w", err)
	}

	imports := instance.newImports()
	vmInstance, err := wasmer.NewInstance(module, imports)
	if err != nil {
		return nil, fmt.Errorf("creating instance: %w", err)
	}

	memory, err := vmInstance.Exports.GetMemory("memory")
	if err != nil {
		return nil, fmt.Errorf("unable to get the wasm module memory: %w", err)
	}
	instance.memory = memory
	instance.heap = NewHeap(memory)
	instance.entrypoint, err = vmInstance.Exports.GetRawFunction(functionName)
	if err != nil {
		return nil, fmt.Errorf("getting wasm module function %q: %w", functionName, err)
	}

	// TODO: Ensure that the function has the right amount of parameteres in INPUT

	// heap.allocator, err = instance.Exports.GetFunction("memory.allocate")
	// if err != nil {
	// 	panic(fmt.Errorf("getting memory.allocate func: %w", err))
	// }

	return instance, nil
}

func (i *Instance) newImports() *wasmer.ImportObject {
	imports := wasmer.NewImportObject()
	imports.Register("env", map[string]wasmer.IntoExtern{
		"register_panic": wasmer.NewFunction(
			i.store,
			wasmer.NewFunctionType(
				params(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32),
				returns(),
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				message, err := i.heap.ReadString(args[0].I32(), args[1].I32())
				if err != nil {
					return nil, fmt.Errorf("read message argument: %w", err)
				}

				var filename string
				filenamePtr := args[2].I32()
				if filenamePtr != 0 {
					filename, err = i.heap.ReadString(args[2].I32(), args[3].I32())
					if err != nil {
						return nil, fmt.Errorf("read filename argument: %w", err)
					}
				}

				lineNumber := int(args[4].I32())
				columnNumber := int(args[5].I32())

				i.panicError = &PanicError{message, filename, lineNumber, columnNumber}
				//fmt.Println(i.panicError.Error())

				return nil, i.panicError
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
					return nil, fmt.Errorf("reading string: %w", err)
				}

				fmt.Println(message)

				return nil, nil
			},
		),
		"output": wasmer.NewFunction(
			i.store,
			wasmer.NewFunctionType(
				params(wasmer.I32, wasmer.I32),
				returns(),
			),
			func(args []wasmer.Value) ([]wasmer.Value, error) {
				message, err := i.heap.ReadBytes(args[0].I32(), args[1].I32())
				if err != nil {
					return nil, fmt.Errorf("reading bytes: %w", err)
				}

				i.returnValue = message

				return nil, nil
			},
		),
	})
	return imports
}

func (i *Instance) Execute(block []byte) (out []byte, err error) {
	i.returnValue = nil
	i.panicError = nil

	blockPtr := i.heap.Write(block)
	blockLen := int32(len(block))

	//i.heap.PrintMem()
	_, err = i.entrypoint.Call(blockPtr, blockLen)
	//i.heap.PrintMem()

	return i.returnValue, nil
}
