package exchange

import (
	"testing"

	"github.com/streamingfast/sparkle-pancakeswap/state"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriceStore(t *testing.T) {
	b := &PricesStateBuilder{SubstreamIntrinsics: nil}

	pairs := state.New("pairs", nil)
	prices := state.New("prices", nil)
	updates := []PCSReserveUpdate{}
	require.NoError(t, b.BuildState(PCSReserveUpdates(updates), pairs, prices))

	assert.Equal(t, byteMap(map[string]string{}), prices.KV)
}
