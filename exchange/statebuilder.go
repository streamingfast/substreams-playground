package exchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

type StateReader interface {
	GetFirst(key string) ([]byte, bool)
	GetLast(key string) ([]byte, bool)
	GetAt(ord uint64, key string) ([]byte, bool)
}

type StateBuilder struct {
	name string

	KV     map[string][]byte // KV is the state, and assumes all Deltas were already applied to it.
	Deltas []StateDelta      // Deltas are always deltas for the given block.
}

func NewStateBuilder(name string) *StateBuilder {
	return &StateBuilder{
		name: name,
		KV:   make(map[string][]byte),
	}
}

func (b *StateBuilder) Init(startBlockNum uint64, dataFolder string) error {
	relativeKvStartBlock := (startBlockNum / 100) * 100
	kvTotalPairFile := fmt.Sprintf("%s/%d-%s.kv", dataFolder, relativeKvStartBlock, b.name)
	if _, err := os.Stat(kvTotalPairFile); err == nil {
		data, _ := ioutil.ReadFile(kvTotalPairFile)
		err = json.Unmarshal(data, &b.KV)
		if err != nil {
			return fmt.Errorf("unmarshalling kv for %s at block %d: %w", b.name, relativeKvStartBlock, err)
		}
	}

	for i := relativeKvStartBlock; i < startBlockNum; i++ {
		deltaFile := fmt.Sprintf("%s/%d-%s.delta", dataFolder, i, b.name)
		if _, err := os.Stat(deltaFile); err == nil {
			data, _ := ioutil.ReadFile(deltaFile)
			err = json.Unmarshal(data, &b.Deltas)
			if err != nil {
				return fmt.Errorf("unmarshalling delta for %s at block %d: %s", b.name, i, err)
			}
			b.Flush()
		}
	}
	return nil
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
	for _, delta := range b.Deltas {
		if delta.Key == key {
			switch delta.Op {
			case "d", "u":
				return delta.OldValue, true
			case "c":
				return nil, false
			default:
				// WARN: is that legit? what if some upstream stream is broken? can we trust all those streams?
				panic(fmt.Sprintf("invalid value %q for StateDelta::Op for key %q", delta.Op, delta.Key))
			}
		}
	}
	return b.GetLast(key)
}

func (b *StateBuilder) GetLast(key string) ([]byte, bool) {
	val, found := b.KV[key]
	return val, found

}

// GetAt returns the key for the state that includes the processing of `ord`.
func (b *StateBuilder) GetAt(ord uint64, key string) (out []byte, found bool) {
	out, found = b.GetLast(key)

	for i := len(b.Deltas) - 1; i >= 0; i-- {
		delta := b.Deltas[i]
		if delta.Ordinal <= ord {
			break
		}
		if delta.Key == key {
			switch delta.Op {
			case "d", "u":
				out = delta.OldValue
				found = true
			case "c":
				out = nil
				found = false
			default:
				// WARN: is that legit? what if some upstream stream is broken? can we trust all those streams?
				panic(fmt.Sprintf("invalid value %q for StateDelta::Op for key %q", delta.Op, delta.Key))
			}
		}
	}
	return
}
func (b *StateBuilder) Del(ord uint64, key string) {
	val, found := b.GetLast(key)
	if found {
		delta := &StateDelta{
			Op:       "d",
			Ordinal:  ord,
			Key:      key,
			OldValue: val,
			NewValue: nil,
		}
		b.applyDelta(delta)
		b.Deltas = append(b.Deltas, *delta)
	}
}
func (b *StateBuilder) Set(ord uint64, key string, value []byte) {
	val, found := b.GetLast(key)
	var delta *StateDelta
	if found {
		delta = &StateDelta{
			Op:       "u",
			Ordinal:  ord,
			Key:      key,
			OldValue: val,
			NewValue: value,
		}
	} else {
		delta = &StateDelta{
			Op:       "c",
			Ordinal:  ord,
			Key:      key,
			OldValue: nil,
			NewValue: value,
		}
	}
	b.applyDelta(delta)
	b.Deltas = append(b.Deltas, *delta)
}

func (b *StateBuilder) applyDelta(delta *StateDelta) {
	switch delta.Op {
	case "u", "c":
		b.KV[delta.Key] = delta.NewValue
	case "d":
		delete(b.KV, delta.Key)
	}

}

func (b *StateBuilder) Flush() {
	for _, delta := range b.Deltas {
		b.applyDelta(&delta)
	}
	b.Deltas = nil
}

func (b *StateBuilder) StoreAndFlush(blockNumber uint64, dataFolder string) error {

	// if b.BundleCompleted() {
	//cnt, _ := json.MarshalIndent(b.KV, "", "  ")
	//err := ioutil.WriteFile(fmt.Sprintf("%s/%d-%s.kv", dataFolder, blockNumber, b.name), cnt, os.ModePerm)
	//if err != nil {
	//	return fmt.Errorf("writing %s kv at block %d: %w", b.name, blockNumber, err)
	//}

	//todo: maybe delta to merge = b.toBundle()

	//b.bundle.commit()
	//b.bundle.purge()
	//}

	if blockNumber%100 == 0 {

		cnt, _ := json.MarshalIndent(b.KV, "", "  ")
		err := ioutil.WriteFile(fmt.Sprintf("%s/%d-%s.kv", dataFolder, blockNumber, b.name), cnt, os.ModePerm)
		if err != nil {
			return fmt.Errorf("writing %s kv at block %d: %w", b.name, blockNumber, err)
		}
	}

	cnt, _ := json.MarshalIndent(b.Deltas, "", "  ")
	err := ioutil.WriteFile(fmt.Sprintf("%s/%d-%s.delta", dataFolder, blockNumber, b.name), cnt, os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing %s delta at block %d: %w", b.name, blockNumber, err)
	}
	b.Flush()
	return nil
}
