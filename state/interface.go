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

type FirstKeyWriter interface {
	SetIfNotExists(ord uint64, key string, value string)
	SetBytesIfNotExists(ord uint64, key string, value []byte)
}

type Deleter interface {
	// Deletes a range of keys, lexicographically between `lowKey` and `highKey`
	DeleteRange(lowKey, highKey string)
	// Deletes a range of keys, first considering the _value_ of such keys as a _pointerSeparator_-separated list of keys to _also_ delete.
	DeleteRangePointers(lowKey, highKey, pointerSeparator string)
}

// for LAST_KEY, and FIRST_KEY merge strategy, the Writer will simply write the key, with no regard
// to what was there before
type IntegerMaximumWriter interface {
}

type FloatMaximumWriter interface {
}

type IntegerMinimumWriter interface {
}

type FloatMinimumWriter interface {
}

// NOTE: Ça commence à faire beaucoup d'interfaces et de merge strategies
// pkoi ça serait pas le DATA qui porte sa merge strategy? Dépendemment de comment tu set
// la valeur, elle est settée avec un préfixe, toujours, genre:
// is: = int sum
// im: = int min
// iM: = int max
// fs: = float sum
// fm: = float min
// fM: = float max
// kl: = set key, last key wins
// kf: = set key, first key wins (noop if the key is set)
// dr: = delete range key
// dp:SEP: = deletes pointer range
//
// All these prefixes would only apply to the `partial` stores, and we'd need to keep track
// that we don't write to the same key with two different modes, unless we need to start reading
// keys by specifying those prefixes too.

type IntegerDeltaWriter interface {
	AddInt(ord uint64, key string, value *big.Int)
}

type FloatDeltaWriter interface {
	AddFloat(ord uint64, key string, value *big.Float)
}

type Mergeable interface {
	Merge(other *Builder) error
}
