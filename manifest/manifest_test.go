package manifest

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestManifest_YamlUnmarshal(t *testing.T) {
	_, _, err := DecodeYamlManifestFromFile("./test/test_manifest.yaml")
	assert.NoError(t, err)
}
