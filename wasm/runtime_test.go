package wasm

import (
	"fmt"
	"io/ioutil"
	"testing"

	pbcodec "github.com/streamingfast/sparkle/pb/sf/ethereum/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

//go:generate ./build-examples.sh

func TestRustInstance(t *testing.T) {
	wasmCode, err := ioutil.ReadFile("./example-block/pkg/example_block_bg.wasm")
	require.NoError(t, err)

	instance, err := NewRustInstance(wasmCode, "map")
	require.NoError(t, err, "filename: example_block_bg.wasm")

	block := &pbcodec.Block{
		Ver:    1,
		Number: 1234,
		Hash:   []byte{0x01, 0x02, 0x03, 0x04},
		Header: &pbcodec.BlockHeader{
			ParentHash: []byte{0x00, 0x01, 0x02, 0x03},
		},
		TransactionTraces: []*pbcodec.TransactionTrace{
			{Hash: []byte{0x03, 0x03, 0x03, 0x03}},
			{Hash: []byte{0x04, 0x04, 0x04, 0x04}},
		},
	}
	blockBytes, err := proto.Marshal(block)
	require.NoError(t, err)

	retVal, err := instance.Execute(blockBytes)
	if err != nil {
		fmt.Printf("error here: %T, %v\n", err, err)
	}
	require.NoError(t, err)

	expect, err := proto.Marshal(block.Header)
	require.NoError(t, err)

	assert.Equal(t, retVal, expect)
}
