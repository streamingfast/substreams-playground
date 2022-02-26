package wasm

import (
	"fmt"

	"github.com/wasmerio/wasmer-go/wasmer"
)

type Heap struct {
	memory          *wasmer.Memory
	allocator       wasmer.NativeFunction
	nextPtrLocation int32
	freeSpace       uint
}

func NewHeap(memory *wasmer.Memory) *Heap {
	if len(memory.Data()) != int(memory.DataSize()) {
		panic("ALSKDJ")
	}
	return &Heap{
		memory:    memory,
		freeSpace: memory.DataSize(), // double check, is that the FREE memory or the total allocated memory?
	}
}

func (h *Heap) Write(bytes []byte) int32 {
	size := len(bytes)

	if uint(size) > h.freeSpace {
		fmt.Println("memory grown")
		numberOfPages := (uint(size) / wasmer.WasmPageSize) + 1
		grown := h.memory.Grow(wasmer.Pages(numberOfPages))
		if !grown {
			panic("couldn't grow memory")
		}
		h.freeSpace += (wasmer.WasmPageSize * numberOfPages)
	}

	ptr := h.nextPtrLocation

	memoryData := h.memory.Data()
	copy(memoryData[ptr:], bytes)

	h.nextPtrLocation += int32(size)
	h.freeSpace -= uint(size)

	return ptr
}

func (h *Heap) ReadString(offset int32, length int32) (string, error) {
	bytes, err := h.ReadBytes(offset, length)
	if err != nil {
		return "", fmt.Errorf("read bytes: %w", err)
	}
	return string(bytes), nil
}

func (h *Heap) ReadBytes(offset int32, length int32) ([]byte, error) {
	bytes := h.memory.Data()
	if offset < 0 {
		return nil, fmt.Errorf("offset %d env must be positive", offset)
	}

	if offset >= int32(len(bytes)) {
		return nil, fmt.Errorf("offset %d env out of memory bounds ending at %d env", offset, len(bytes))
	}

	end := offset + length
	if end > int32(len(bytes)) {
		return nil, fmt.Errorf("end %d env out of memory bounds ending at %d env", end, len(bytes))
	}

	return bytes[offset : offset+length], nil
}

func (h *Heap) PrintMem() {
	data := h.memory.Data()
	for i, datum := range data {
		if i > 1024 {
			if datum == 0 {
				continue
			}
		}
		fmt.Print(datum, ", ")
	}
	fmt.Print("\n")
}
