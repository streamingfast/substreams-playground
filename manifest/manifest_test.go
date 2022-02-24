package manifest

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestManifest_YamlUnmarshal(t *testing.T) {
	_, manifest, err := DecodeYamlManifestFromFile("./test/test_manifest.yaml")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(manifest.Streams), 1)
	assert.Equal(t, manifest.GenesisBlock, 6809737)
}

func TestStreamYamlDecode(t *testing.T) {
	type test struct {
		name           string
		rawYamlInput   string
		expectedOutput Stream
	}

	tests := []test{
		{
			name: "basic mapper",
			rawYamlInput: `---
name: pairExtractor
kind: Mapper
code: ./pairExtractor.wasm
inputs:
  - proto:sf.ethereum.types.v1.Block
output:
  type: proto:pcs.types.v1.Pairs`,
			expectedOutput: Stream{
				Name:   "pairExtractor",
				Kind:   "Mapper",
				Code:   "./pairExtractor.wasm",
				Inputs: []string{"proto:sf.ethereum.types.v1.Block"},
				Output: map[string]string{"type": "proto:pcs.types.v1.Pairs"},
			},
		},
		{
			name: "basic store",
			rawYamlInput: `---
name: prices
kind: StateBuilder
code: ./pricesState.wasm
inputs:
  - proto:sf.ethereum.types.v1.Block
  - store:pairs
output:
  storeMergeStrategy: LAST_KEY`,
			expectedOutput: Stream{
				Name:   "prices",
				Kind:   "StateBuilder",
				Code:   "./pricesState.wasm",
				Inputs: []string{"proto:sf.ethereum.types.v1.Block", "store:pairs"},
				Output: map[string]string{"storeMergeStrategy": "LAST_KEY"},
			},
		},
	}

	for _, tt := range tests {
		var tstream Stream
		err := yaml.NewDecoder(strings.NewReader(tt.rawYamlInput)).Decode(&tstream)
		assert.NoError(t, err)
		assert.Equal(t, tt.expectedOutput, tstream)
	}
}

func TestStream_Signature_Basic(t *testing.T) {
	manifest, err := New("./test/test_manifest.yaml")
	assert.NoError(t, err)

	pairExtractorStream := manifest.Graph.streams["pairExtractor"]
	sig, err := pairExtractorStream.Signature(manifest.Graph)
	assert.NoError(t, err)

	sigString := base64.StdEncoding.EncodeToString(sig)
	assert.Equal(t, "4E8LY/jRrRfuzPneS53QBItGhwU=", sigString)
}

func TestStream_Signature_Composed(t *testing.T) {
	manifest, err := New("./test/test_manifest.yaml")
	assert.NoError(t, err)

	pairsStream := manifest.Graph.streams["pairs"]
	sig, err := pairsStream.Signature(manifest.Graph)
	assert.NoError(t, err)

	sigString := base64.StdEncoding.EncodeToString(sig)
	assert.Equal(t, "OAvI+VUy9FU2dWDUNRcZ3KHEoh8=", sigString)
}

func TestStreamLinks_StreamsFor(t *testing.T) {
	streamGraph := &StreamsGraph{
		streams: map[string]Stream{
			"A": {Name: "A"},
			"B": {Name: "B"},
			"C": {Name: "C"},
			"D": {Name: "D"},
			"E": {Name: "E"},
			"F": {Name: "F"},
			"G": {Name: "G"},
			"H": {Name: "H"},
			"I": {Name: "I"},
		},
		links: map[string][]Stream{
			"A": {Stream{Name: "B"}, Stream{Name: "C"}},
			"B": {Stream{Name: "D"}, Stream{Name: "E"}, Stream{Name: "F"}},
			"C": {Stream{Name: "F"}},
			"D": {},
			"E": {},
			"F": {Stream{Name: "G"}, Stream{Name: "H"}},
			"G": {},
			"H": {},
			"I": {Stream{Name: "H"}},
		},
	}

	res, err := streamGraph.StreamsFor("A")
	assert.NoError(t, err)

	order := bytes.NewBuffer(nil)
	for _, l := range res {
		order.WriteString(l.Name)
	}

	assert.Equal(t, "GHDEFBCA", order.String())
}
