package wasm

import (
	"fmt"
	"testing"

	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

//go:generate ./build-examples.sh

func TestRustInstance(t *testing.T) {
	instance, err := NewRustInstance("./example-block/pkg/example_block_bg.wasm", "map")
	require.NoError(t, err)

	block := &pbcodec.Block{Ver: 1, Number: 1234, Hash: []byte{0x01, 0x02, 0x03, 0x04}}
	blockBytes, err := proto.Marshal(block)
	require.NoError(t, err)

	out, err := instance.Execute(blockBytes)
	if err != nil {
		fmt.Printf("error here: %T, %v\n", err, err)
	}

	require.NoError(t, err)
	fmt.Println("MAMA", out)

	// data, err := ret.ReadD21ata(env)
	// require.NoError(t, err)
	// fmt.Println("received data as string:", string(data))
	// data2, err := ret2.ReadData(env)
	// require.NoError(t, err)
	// fmt.Println("received data2 as string:", string(data2))
}
