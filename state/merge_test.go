package state

import "testing"

func TestBuilder_Merge(t *testing.T) {
	type test struct {
		name          string
		this          *Builder
		next          *Builder
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
	}

	for _, tt := range tests {
		err := tt.this.Merge(tt.next)
		if err != nil && !tt.expectedError {
			t.Errorf("got unexpected error in test %s: %w", tt.name, err)
			continue
		}
	}
}
