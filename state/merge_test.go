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
			this:          New("b1", MergeStrategyLastKey, nil),
			next:          New("b2", MergeStrategySumInts, nil),
			expectedError: true,
		},
		{
			name: "last_key",
			this: New("b1", MergeStrategyLastKey, nil),
			thisKV: map[string][]byte{
				"one": []byte("foo"),
				"two": []byte("bar"),
			},
			next: New("b2", MergeStrategyLastKey, nil),
			nextKV: map[string][]byte{
				"one":   []byte("baz"),
				"three": []byte("lol"),
			},
			expectedError: false,
			expectedKV: map[string][]byte{
				"one":   []byte("baz"),
				"two":   []byte("bar"),
				"three": []byte("lol"),
			},
		},
		{
			name: "sum_ints",
			this: New("b1", MergeStrategySumInts, nil),
			thisKV: map[string][]byte{
				"one": []byte("1"),
				"two": []byte("2"),
			},
			next: New("b2", MergeStrategySumInts, nil),
			nextKV: map[string][]byte{
				"one":   []byte("1"),
				"three": []byte("3"),
			},
			expectedError: false,
			expectedKV: map[string][]byte{
				"one":   []byte("2"),
				"two":   []byte("2"),
				"three": []byte("3"),
			},
		},
		{
			name: "sum_floats",
			this: New("b1", MergeStrategySumFloats, nil),
			thisKV: map[string][]byte{
				"one": []byte("1.0"),
				"two": []byte("2.0"),
			},
			next: New("b2", MergeStrategySumFloats, nil),
			nextKV: map[string][]byte{
				"one":   []byte("1.0"),
				"three": []byte("3.0"),
			},
			expectedError: false,
			expectedKV: map[string][]byte{
				"one":   []byte("2.0"),
				"two":   []byte("2.0"),
				"three": []byte("3.0"),
			},
		},
		{
			name: "min_int",
			this: New("b1", MergeStrategyMinInt, nil),
			thisKV: map[string][]byte{
				"one": []byte("1"),
				"two": []byte("2"),
			},
			next: New("b2", MergeStrategyMinInt, nil),
			nextKV: map[string][]byte{
				"one":   []byte("2"),
				"three": []byte("3"),
			},
			expectedError: false,
			expectedKV: map[string][]byte{
				"one":   []byte("1"),
				"two":   []byte("2"),
				"three": []byte("3"),
			},
		},
		{
			name: "min_float",
			this: New("b1", MergeStrategyMinFloat, nil),
			thisKV: map[string][]byte{
				"one": []byte("1.0"),
				"two": []byte("2.0"),
			},
			next: New("b2", MergeStrategyMinFloat, nil),
			nextKV: map[string][]byte{
				"one":   []byte("2.0"),
				"three": []byte("3.0"),
			},
			expectedError: false,
			expectedKV: map[string][]byte{
				"one":   []byte("1.0"),
				"two":   []byte("2.0"),
				"three": []byte("3.0"),
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

		for k, v := range tt.this.KV {
			if tt.this.mergeStrategy == MergeStrategySumFloats || tt.this.mergeStrategy == MergeStrategyMinFloat {
				actual, _ := foundOrZeroFloat(v, true).Float64()
				expected, _ := foundOrZeroFloat(tt.expectedKV[k], true).Float64()
				assert.InDelta(t, actual, expected, 0.01)
			} else {
				assert.Equal(t, v, tt.expectedKV[k])
			}
		}

		for k, v := range tt.expectedKV {
			if tt.this.mergeStrategy == MergeStrategySumFloats || tt.this.mergeStrategy == MergeStrategyMinFloat {
				actual, _ := foundOrZeroFloat(v, true).Float64()
				expected, _ := foundOrZeroFloat(tt.this.KV[k], true).Float64()
				assert.InDelta(t, actual, expected, 0.01)
			} else {
				assert.Equal(t, v, tt.this.KV[k])
			}
		}
	}
}
