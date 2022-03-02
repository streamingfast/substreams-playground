package state

import "math/big"

type Reader interface {
	GetFirst(key string) ([]byte, bool)
	GetLast(key string) ([]byte, bool)
	GetAt(ord uint64, key string) ([]byte, bool)
}

type Writer interface {
	Set(ord uint64, key string, value string)
	SetBytes(ord uint64, key string, value []byte)
}

// for LAST_KEY, and FIRST_KEY merge strategy, the Writer will simply write the key, with no regard
// to what was there before

type IntegerDeltaWriter interface {
	AddInt(ord uint64, key string, value *big.Int)
}

type FloatDeltaWriter interface {
	AddFloat(ord uint64, key string, value *big.Float)
}

type Mergeable interface {
	Merge(other *Builder) error
}
