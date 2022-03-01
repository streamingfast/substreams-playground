package state

import (
	"fmt"
	"math/big"
	"strconv"
)

func (b *Builder) Merge(next *Builder) error {
	if b.mergeStrategy != next.mergeStrategy {
		return fmt.Errorf("incompatible merge strategies. strategy %s cannot be merged with strategy %s", b.mergeStrategy, next.mergeStrategy)
	}

	switch b.mergeStrategy {
	case "LAST_KEY":
		if next.lastOrdinal < b.lastOrdinal {
			return nil
		}

		for k, v := range next.KV {
			b.SetBytes(next.lastOrdinal, k, v)
		}
	case "SUM_INTS":
		for k, v := range next.KV {
			v0 := foundOrZeroUint64(b.GetLast(k))
			v1 := foundOrZeroUint64(v, true)
			v_sum := v0 + v1
			b.Set(next.lastOrdinal, k, fmt.Sprintf("%d", v_sum))
		}
	case "SUM_FLOATS":
		for k, v := range next.KV {
			v0 := foundOrZeroFloat(b.GetLast(k))
			v1 := foundOrZeroFloat(v, true)
			v_sum := bf().Add(v0, v1).SetPrec(100)
			b.Set(next.lastOrdinal, k, floatToStr(v_sum))
		}
	default:
		return fmt.Errorf("unsupported merge strategy %s", b.mergeStrategy)
	}

	b.bundler = nil

	return nil
}

func foundOrZeroUint64(in []byte, found bool) uint64 {
	if !found {
		return 0
	}
	val, err := strconv.ParseInt(string(in), 10, 64)
	if err != nil {
		return 0
	}
	return uint64(val)
}

func foundOrZeroFloat(in []byte, found bool) *big.Float {
	if !found {
		return bf()
	}
	return bytesToFloat(in)
}

func strToFloat(in string) *big.Float {
	newFloat, _, err := big.ParseFloat(in, 10, 100, big.ToNearestEven)
	if err != nil {
		panic(fmt.Sprintf("cannot load float %q: %s", in, err))
	}
	return newFloat.SetPrec(100)
}

func bytesToFloat(in []byte) *big.Float {
	return strToFloat(string(in))
}

func floatToStr(f *big.Float) string {
	return f.Text('g', -1)
}

func floatToBytes(f *big.Float) []byte {
	return []byte(floatToStr(f))
}

var bf = func() *big.Float { return new(big.Float).SetPrec(100) }
