package state

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/merger/bundle"
)

type Builder struct {
	Name string

	bundler *bundle.Bundler
	io      StateIO

	// KV     map[string][]byte // KV is the state, and assumes all Deltas were already applied to it.
	Deltas        []StateDelta // Deltas are always deltas for the given block.
	KV            map[string]Value
	mergeStrategy string
	lastOrdinal   uint64
}

type Value struct {
	// "is" = int sum
	// "im" = int min
	// "iM" = int max
	// "fs" = float sum
	// "fm" = float min
	// "fM" = float max
	// "kl" = set key, last key wins
	// "kf" = set key, first key wins (noop if the key is set)
	// "dr" = delete range key
	// "D[sep-character]" = deletes pointer range
	KeyType string // eventually something better
	Value   []byte
}

func (v Value) String() string {
	return string(v.Value)
}

func New(name string, mergeStrategy string, ioFactory IOFactory) *Builder {
	b := &Builder{
		Name:          name,
		KV:            make(map[string]Value),
		bundler:       nil,
		mergeStrategy: mergeStrategy,
	}
	if ioFactory != nil {
		b.io = ioFactory.New(name)
	}
	return b
}

func (b *Builder) Print() {
	if len(b.Deltas) == 0 {
		return
	}
	fmt.Printf("State deltas for %q\n", b.Name)
	for _, delta := range b.Deltas {
		b.PrintDelta(&delta)
	}
}

func (b *Builder) PrintDelta(delta *StateDelta) {
	fmt.Printf("  %s (o=%d, t=%s) KEY: %q\n", strings.ToUpper(delta.Op), delta.Ordinal, delta.KeyType, delta.Key)
	fmt.Printf("    OLD: %s\n", string(delta.OldValue))
	fmt.Printf("    NEW: %s\n", string(delta.NewValue))
}

func (b *Builder) Init(startBlockNum uint64) error {
	relativeKvStartBlock := (startBlockNum / 100) * 100

	if err := b.ReadState(context.TODO(), relativeKvStartBlock); err != nil {
		return err
	}

	//var deltas []*bundle.OneBlockFile
	//
	//// walk from last kv checkpoint to current start block
	//err := b.io.WalkDeltas(context.TODO(), relativeKvStartBlock+1, startBlockNum-1, func(obf *bundle.OneBlockFile) error {
	//	deltas = append(deltas, obf)
	//	return nil
	//})
	//if err != nil {
	//	return err
	//}
	//
	//sort.Slice(deltas, func(i, j int) bool {
	//	return deltas[i].Num < deltas[j].Num
	//})
	//
	//for _, delta := range deltas {
	//	data, err := b.io.ReadDelta(context.TODO(), delta)
	//	if err != nil {
	//		return err
	//	}
	//	err = json.Unmarshal(data, &b.Deltas)
	//	if err != nil {
	//		return fmt.Errorf("unmarshalling delta for %s at block %d: %w", b.Name, relativeKvStartBlock, err)
	//	}
	//	b.Flush()
	//}

	return nil
}

type StateDelta struct {
	Op       string // "c"reate, "u"pdate, "d"elete, same as https://nightlies.apache.org/flink/flink-docs-master/docs/connectors/table/formats/debezium/#how-to-use-debezium-format
	Ordinal  uint64 // a sorting key to order deltas, and provide pointers to changes midway
	Key      string
	KeyType  string
	OldValue []byte
	NewValue []byte
}

var NotFound = errors.New("state key not found")

func (b *Builder) GetFirst(key string) (Value, bool) {
	for _, delta := range b.Deltas {
		if delta.Key == key {
			switch delta.Op {
			case "d", "u":
				return Value{Value: delta.OldValue, KeyType: delta.KeyType}, true
			case "c":
				return Value{}, false
			default:
				// WARN: is that legit? what if some upstream stream is broken? can we trust all those streams?
				panic(fmt.Sprintf("invalid value %q for StateDelta::Op for key %q", delta.Op, delta.Key))
			}
		}
	}
	return b.GetLast(key)
}

func (b *Builder) GetLast(key string) (Value, bool) {
	val, found := b.KV[key]
	return val, found
}

// GetAt returns the key for the state that includes the processing of `ord`.
func (b *Builder) GetAt(ord uint64, key string) (out Value, found bool) {
	out, found = b.GetLast(key)

	for i := len(b.Deltas) - 1; i >= 0; i-- {
		delta := b.Deltas[i]
		if delta.Ordinal <= ord {
			break
		}
		if delta.Key == key {
			switch delta.Op {
			case "d", "u":
				out = Value{Value: delta.OldValue, KeyType: delta.KeyType}
				found = true
			case "c":
				out = Value{}
				found = false
			default:
				// WARN: is that legit? what if some upstream stream is broken? can we trust all those streams?
				panic(fmt.Sprintf("invalid value %q for StateDelta::Op for key %q", delta.Op, delta.Key))
			}
		}
	}
	return
}

func (b *Builder) Del(ord uint64, key string) {
	b.bumpOrdinal(ord)

	val, found := b.GetLast(key)
	if found {
		delta := &StateDelta{
			Op:       "d",
			Ordinal:  ord,
			Key:      key,
			KeyType:  val.KeyType, // maybe we don't care about the key type for a delete?
			OldValue: val.Value,
			NewValue: nil,
		}
		b.applyDelta(delta)
		b.Deltas = append(b.Deltas, *delta)
	}
}

func (b *Builder) bumpOrdinal(ord uint64) {
	if b.lastOrdinal > ord {
		panic("cannot Set or Del a value on a state.Builder with an ordinal lower than the previous")
	}
	b.lastOrdinal = ord
}

func (b *Builder) AddInt(ord uint64, key string, value *big.Int) {
	sum := new(big.Int)
	val, found := b.GetAt(ord, key)
	if !found {
		sum = value
	} else {
		prev, _ := new(big.Int).SetString(string(val.Value), 10)
		if prev == nil {
			sum = value
		} else {
			sum.Add(prev, value)
		}
	}
	b.set(ord, key, "is", []byte(sum.String()))
}

func (b *Builder) AddFloat(ord uint64, key string, value *big.Float) {
	sum := new(big.Float)
	val, found := b.GetAt(ord, key)
	if !found {
		sum = value
	} else {
		prev, _, err := big.ParseFloat(string(val.Value), 10, 100, big.ToNearestEven)
		if prev == nil || err != nil {
			sum = value
		} else {
			sum.Add(prev, value)
		}
	}
	b.set(ord, key, "fs", []byte(sum.Text('g', -1)))
}

func (b *Builder) SetBytes(ord uint64, key string, value []byte) {
	b.set(ord, key, "kl", value)
}
func (b *Builder) Set(ord uint64, key string, value string) {
	b.set(ord, key, "kl", []byte(value))
}

func (b *Builder) set(ord uint64, key string, keyType string, value []byte) {
	b.bumpOrdinal(ord)

	val, found := b.GetLast(key)
	if found && keyType != val.KeyType {
		panic(fmt.Sprintf("key %q cannot change aggregation method", key))
	}

	var delta *StateDelta
	if found {
		//Uncomment when finished debugging:
		if bytes.Compare(value, val.Value) == 0 {
			return
		}
		delta = &StateDelta{
			Op:       "u",
			Ordinal:  ord,
			Key:      key,
			KeyType:  keyType,
			OldValue: val.Value,
			NewValue: value,
		}
	} else {
		delta = &StateDelta{
			Op:       "c",
			Ordinal:  ord,
			Key:      key,
			KeyType:  keyType,
			OldValue: nil,
			NewValue: value,
		}
	}
	b.applyDelta(delta)
	b.Deltas = append(b.Deltas, *delta)
}

func (b *Builder) applyDelta(delta *StateDelta) {
	switch delta.Op {
	case "u", "c":
		b.KV[delta.Key] = Value{
			KeyType: delta.KeyType,
			Value:   delta.NewValue,
		}
	case "d":
		delete(b.KV, delta.Key)
	}

}

func (b *Builder) Flush() {
	for _, delta := range b.Deltas {
		b.applyDelta(&delta)
	}
	b.Deltas = nil
	b.lastOrdinal = 0
}

func (b *Builder) StoreBlock(ctx context.Context, block *bstream.Block) error {
	blockNumber := block.Number

	if b.bundler == nil {
		exclusiveHighestBlockLimit := ((blockNumber / 100) * 100) + 100
		b.bundler = bundle.NewBundler(100, exclusiveHighestBlockLimit)
	}

	bundleCompleted, highestBlockLimit := b.bundler.BundleCompleted()
	if bundleCompleted {
		files := b.bundler.ToBundle(highestBlockLimit)

		//todo: currently no-op.
		err := b.io.MergeDeltas(ctx, b.bundler.BundleInclusiveLowerBlock(), files)
		if err != nil {
			return err
		}

		b.bundler.Commit(highestBlockLimit)
		b.bundler.Purge(func(oneBlockFilesToDelete []*bundle.OneBlockFile) {
			for _, file := range oneBlockFilesToDelete {
				//todo: currently no-op.
				_ = b.io.DeleteDelta(ctx, file)
			}
		})

		if err := b.WriteState(ctx, block); err != nil {
			return err
		}
	}

	obf := mustBlockToOneBlockFile(b.Name, block)

	//content, _ := json.MarshalIndent(b.Deltas, "", "  ")
	//err := b.io.WriteDelta(ctx, content, obf)
	//if err != nil {
	//	return fmt.Errorf("writing %s delta at block %d: %w", b.Name, blockNumber, err)
	//}

	b.bundler.AddOneBlockFile(obf)

	return nil
}

func (b *Builder) ReadState(ctx context.Context, startBlockNum uint64) error {
	data, err := b.io.ReadState(ctx, startBlockNum)
	if err != nil {
		return err
	}

	kv := map[string]Value{}

	if err = json.Unmarshal(data, &kv); err != nil {
		return fmt.Errorf("unmarshalling kv for %s at block %d: %w", b.Name, startBlockNum, err)
	}

	b.KV = kv

	fmt.Printf("loading KV from disk for %q: %d entries\n", b.Name, len(b.KV))

	return nil
}

func (b *Builder) WriteState(ctx context.Context, block *bstream.Block) error {
	// kv := stringMap(b.KV) // FOR READABILITY ON DISK

	content, err := json.MarshalIndent(b.KV, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal kv state: %w", err)
	}

	if err = b.io.WriteState(ctx, content, block); err != nil {
		return fmt.Errorf("writing %s kv at block %d: %w", b.Name, block.Num(), err)
	}

	return nil
}

func stringMap(in map[string][]byte) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = string(v)
	}
	return out
}

func byteMap(in map[string]string) map[string][]byte {
	out := map[string][]byte{}
	for k, v := range in {
		out[k] = []byte(v)
	}
	return out
}
