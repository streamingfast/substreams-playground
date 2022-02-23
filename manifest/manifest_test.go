package manifest

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestManifest_YamlUnmarshal(t *testing.T) {
	_, _, err := DecodeYamlManifestFromFile("./test/test_manifest.yaml")
	assert.NoError(t, err)
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
