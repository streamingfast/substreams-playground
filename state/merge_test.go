package state

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuilder_Merge(t *testing.T) {
	type test struct {
		name          string
		this          *Builder
		thisKV        map[string][]byte
		next          *Builder
		nextKV        map[string][]byte
		expectedError bool
		expectedKV    map[string][]byte
	}

	tests := []test{
		{
			name:          "incompatible merge strategies",
			this:          New("b1", "LAST_KEY", nil),
			next:          New("b2", "SUM_INTS", nil),
			expectedError: true,
		},
		{
			name: "last_key",
			this: New("b1", "LAST_KEY", nil),
			thisKV: map[string][]byte{
				"one": []byte("foo"),
				"two": []byte("bar"),
			},
			next: New("b2", "LAST_KEY", nil),
			nextKV: map[string][]byte{
				"one": []byte("baz"),
				"two": []byte("bar"),
			},
			expectedError: false,
			expectedKV: map[string][]byte{
				"one": []byte("baz"),
				"two": []byte("bar"),
			},
		},
	}

	for _, tt := range tests {
		tt.this.KV = tt.thisKV
		tt.next.KV = tt.nextKV

		err := tt.this.Merge(tt.next)
		if err != nil && !tt.expectedError {
			if !tt.expectedError {
				t.Errorf("got unexpected error in test %s: %w", tt.name, err)
			}
			continue
		}
		assert.Equal(t, tt.expectedKV, tt.this.KV)
	}
}
