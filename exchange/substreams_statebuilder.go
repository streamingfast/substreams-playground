package exchange

import (
	"errors"
	"fmt"
)

type StateBuilder struct {
	readOnly bool
	name     string
	kv       map[string][]byte
	Deltas   []StateDelta
}

func NewStateBuilder(name string) *StateBuilder {
	return &StateBuilder{
		name: name,
		kv:   make(map[string][]byte),
	}
}

type StateDelta struct {
	Op       string // "c"reate, "u"pdate, "d"elete, same as https://nightlies.apache.org/flink/flink-docs-master/docs/connectors/table/formats/debezium/#how-to-use-debezium-format
	Ordinal  uint64 // a sorting key to order deltas, and provide pointers to changes midway
	Key      string
	OldValue []byte
	NewValue []byte
}

var NotFound = errors.New("state key not found")

func (b *StateBuilder) GetFirst(key string) ([]byte, bool) {
	val, found := b.kv[key]
	return val, found
}

func (b *StateBuilder) GetLast(key string) ([]byte, bool) {
	// TODO: FLIP the GetLast and GetFirst, so `GetLast` is the always the fastest, and we
	// rather UNDO the steps until `ord` when we do a GetAt (and undo all when GetFirst)
	// because most of the time, people will want to read the state at the completed block
	// boundary.

	// So upon receiving the deltas, we'll apply them, and consider their reverse values
	// when doing a `GetAt`
	for i := len(b.Deltas) - 1; i >= 0; i-- {
		delta := b.Deltas[i]
		if delta.Key == key {
			switch delta.Op {
			case "d":
				return nil, false
			case "u", "c":
				return delta.NewValue, true
			default:
				// WARN: is that legit? what if some upstream stream is broken? can we trust all those streams?
				panic(fmt.Sprintf("invalid value %q for StateDelta::Op for key %q", delta.Op, delta.Key))
			}
		}
	}
	return b.GetFirst(key)
}

// GetAt returns the key for the state that includes the processing of `ord`.
func (b *StateBuilder) GetAt(ord uint64, key string) ([]byte, bool) {
	for i := len(b.Deltas) - 1; i >= 0; i-- {
		delta := b.Deltas[i]
		if delta.Ordinal > ord {
			continue
		}
		if delta.Key == key {
			switch delta.Op {
			case "d":
				return nil, false
			case "u", "c":
				return delta.NewValue, true
			default:
				// WARN: is that legit? what if some upstream stream is broken? can we trust all those streams?
				panic(fmt.Sprintf("invalid value %q for StateDelta::Op for key %q", delta.Op, delta.Key))
			}
		}
	}
	return b.GetFirst(key)
}
func (b *StateBuilder) Del(ord uint64, key string) {
	if b.readOnly {
		panic("cannot write")
	}
	val, found := b.GetLast(key)
	if found {
		b.Deltas = append(b.Deltas, StateDelta{
			Op:       "d",
			Ordinal:  ord,
			Key:      key,
			OldValue: val,
			NewValue: nil,
		})
	}
}
func (b *StateBuilder) Set(ord uint64, key string, value []byte) {
	if b.readOnly {
		panic("cannot write")
	}

	val, found := b.GetLast(key)
	if found {
		b.Deltas = append(b.Deltas, StateDelta{
			Op:       "u",
			Ordinal:  ord,
			Key:      key,
			OldValue: val,
			NewValue: value,
		})
	} else {
		b.Deltas = append(b.Deltas, StateDelta{
			Op:       "c",
			Ordinal:  ord,
			Key:      key,
			OldValue: nil,
			NewValue: value,
		})
	}
}

func (b *StateBuilder) Flush() {
	for _, delta := range b.Deltas {
		switch delta.Op {
		case "u", "c":
			b.kv[delta.Key] = delta.NewValue
		case "d":
			delete(b.kv, delta.Key)
		}
	}
	b.Deltas = nil
}
