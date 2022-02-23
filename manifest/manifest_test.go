package manifest

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

func TestManifest_YamlUnmarshal(t *testing.T) {
	_, manifest, err := DecodeYamlManifestFromFile("./test/test_manifest.yaml")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(manifest.Streams), 1)
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

func TestStream_Signature(t *testing.T) {
	manifest, err := NewManifest("./test/test_manifest.yaml")
	assert.NoError(t, err)

	pairExtractorStream := manifest.Streams[0]
	sig, err := pairExtractorStream.Signature()
	assert.NoError(t, err)

	sigString := base64.StdEncoding.EncodeToString(sig)
	assert.Equal(t, "ejl836KNBOKIo0QLsV44i0Qh7hg=", sigString)
}
