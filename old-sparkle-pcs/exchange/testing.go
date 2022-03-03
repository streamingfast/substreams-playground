package exchange

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type blockRef struct {
	id        string
	num       uint64
	timestamp time.Time
}

func (b blockRef) ID() string {
	return b.id
}

func (b blockRef) Number() uint64 {
	return b.num
}

func (b blockRef) Timestamp() time.Time {
	return b.timestamp
}

func TestEvents(t *testing.T, s *Subgraph, events []interface{}) {
	t.Helper()

	for _, event := range events {
		if err := s.HandleEvent(event); err != nil {
			require.NoError(t, err)
		}
	}
}
