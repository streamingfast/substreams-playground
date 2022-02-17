package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/streamingfast/bstream"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/streamingfast/merger/bundle"
)

type Builder struct {
	name string

	bundler *bundle.Bundler
	io      StateIO

	KV     map[string][]byte // KV is the state, and assumes all Deltas were already applied to it.
	Deltas []StateDelta      // Deltas are always deltas for the given block.
}

func NewStateBuilder(name string) *Builder {
	return &Builder{
		name:    name,
		KV:      make(map[string][]byte),
		bundler: nil,
		io:      &NoopStateIO{},
	}
}

func (b *Builder) PrintDeltas() {
	if len(b.Deltas) == 0 {
		return
	}
	fmt.Println("State deltas for", b.name)
	for _, delta := range b.Deltas {
		fmt.Printf("  %s (%d) KEY: %q\n", strings.ToUpper(delta.Op), delta.Ordinal, delta.Key)
		fmt.Printf("    OLD: %s\n", string(delta.OldValue))
		fmt.Printf("    NEW: %s\n", string(delta.NewValue))
	}
}

func (b *Builder) Init(startBlockNum uint64, dataFolder string) error {
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

func (b *Builder) GetFirst(key string) ([]byte, bool) {
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

func (b *Builder) GetLast(key string) ([]byte, bool) {
	val, found := b.KV[key]
	return val, found

}

// GetAt returns the key for the state that includes the processing of `ord`.
func (b *Builder) GetAt(ord uint64, key string) (out []byte, found bool) {
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
func (b *Builder) Del(ord uint64, key string) {
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
func (b *Builder) Set(ord uint64, key string, value []byte) {
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

func (b *Builder) applyDelta(delta *StateDelta) {
	switch delta.Op {
	case "u", "c":
		b.KV[delta.Key] = delta.NewValue
	case "d":
		delete(b.KV, delta.Key)
	}

}

func (b *Builder) Flush() {
	for _, delta := range b.Deltas {
		b.applyDelta(&delta)
	}
	b.Deltas = nil
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
		err := b.io.MergeDeltas(ctx, b.bundler.BundleInclusiveLowerBlock(), files)
		if err != nil {
			return err
		}

		b.bundler.Commit(highestBlockLimit)
		b.bundler.Purge(func(oneBlockFilesToDelete []*bundle.OneBlockFile) {
			for _, file := range oneBlockFilesToDelete {
				_ = b.io.DeleteDelta(ctx, file)
			}
		})

		content, _ := json.MarshalIndent(b.KV, "", "  ")
		err = b.io.WriteState(ctx, content, blockNumber)
		if err != nil {
			return fmt.Errorf("writing %s kv at block %d: %w", b.name, blockNumber, err)
		}
	}

	content, _ := json.MarshalIndent(b.Deltas, "", "  ")
	err := b.io.WriteDelta(ctx, content, blockNumber)
	if err != nil {
		return fmt.Errorf("writing %s delta at block %d: %w", b.name, blockNumber, err)
	}

	obf := b.mustBlockToOneBlockFile(block)
	b.bundler.AddOneBlockFile(obf)

	b.Flush()
	return nil
}

func (b *Builder) mustBlockToOneBlockFile(block *bstream.Block) *bundle.OneBlockFile {
	getUint64Pointer := func(n uint64) *uint64 {
		var ptr *uint64
		*ptr = n
		return ptr
	}

	filename := GetDeltaFileName(b.name, block.Num())

	return &bundle.OneBlockFile{
		CanonicalName: filename,
		Filenames: map[string]struct{}{
			filename: {},
		},
		BlockTime:   time.Time{},
		Num:         block.Num(),
		InnerLibNum: getUint64Pointer(block.LibNum),
	}
}

func GetDeltaFileName(name string, blockNum uint64) string {
	return fmt.Sprintf("%d-%s.delta", blockNum, name)
}

func GetStateFileName(name string, blockNum uint64) string {
	return fmt.Sprintf("%d-%s.kv", blockNum, name)
}

func GetDeltaMergedFileName(name string, blockNum uint64) string {
	return fmt.Sprintf("%d-%s.deltas", blockNum, name)
}

type StateIO interface {
	WriteDelta(ctx context.Context, content []byte, blockNum uint64) error
	ReadDelta(ctx context.Context, into []byte, file *bundle.OneBlockFile) error
	DeleteDelta(ctx context.Context, file *bundle.OneBlockFile) error

	WalkDeltas(ctx context.Context) ([]*bundle.OneBlockFile, error)
	MergeDeltas(ctx context.Context, lowerBlockBoundary uint64, files []*bundle.OneBlockFile) error

	WriteState(ctx context.Context, content []byte, blockNum uint64) error
	ReadState(ctx context.Context, into []byte, blockNum uint64) error
}

type DiskStateIO struct {
	name       string
	dataFolder string
}

func (d *DiskStateIO) WriteDelta(ctx context.Context, content []byte, blockNum uint64) error {
	err := ioutil.WriteFile(filepath.Join(d.dataFolder, GetDeltaFileName(d.name, blockNum)), content, os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing %s delta at block %d: %w", d.name, blockNum, err)
	}

	return nil
}

func (d *DiskStateIO) ReadDelta(ctx context.Context, into []byte, file *bundle.OneBlockFile) error {
	//TODO implement me
	panic("implement me")
}

func (d *DiskStateIO) DeleteDelta(ctx context.Context, file *bundle.OneBlockFile) error {
	err := os.Remove(filepath.Join(d.dataFolder, file.CanonicalName))
	return err
}

func (d *DiskStateIO) WalkDeltas(ctx context.Context) ([]*bundle.OneBlockFile, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DiskStateIO) MergeDeltas(ctx context.Context, lowerBlockBoundary uint64, files []*bundle.OneBlockFile) error {
	bundleReader := bundle.NewBundleReader(ctx, files, func(ctx context.Context, oneBlockFile *bundle.OneBlockFile) (data []byte, err error) {
		err = d.ReadDelta(ctx, data, oneBlockFile)
		return
	})

	path := filepath.Join(d.dataFolder, GetDeltaMergedFileName(d.name, lowerBlockBoundary))
	bundleWriter, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("opening bundle file %s: %w", path, err)
	}

	_, err = io.Copy(bundleWriter, bundleReader)
	if err != nil {
		return fmt.Errorf("copying data from bundleReader at block boundary %d: %w", lowerBlockBoundary, err)
	}

	return nil
}

func (d *DiskStateIO) WriteState(ctx context.Context, content []byte, blockNum uint64) error {
	err := ioutil.WriteFile(filepath.Join(d.dataFolder, GetStateFileName(d.name, blockNum)), content, os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing %s kv at block %d: %w", d.name, blockNum, err)
	}

	return nil
}

func (d *DiskStateIO) ReadState(ctx context.Context, into []byte, blockNumber uint64) error {
	//TODO implement me
	panic("implement me")
}

type NoopStateIO struct {
}

func (n *NoopStateIO) WriteDelta(ctx context.Context, content []byte, blockNum uint64) error {
	return nil
}

func (n *NoopStateIO) ReadDelta(ctx context.Context, into []byte, file *bundle.OneBlockFile) error {
	return nil
}

func (n *NoopStateIO) DeleteDelta(ctx context.Context, file *bundle.OneBlockFile) error {
	return nil
}

func (n *NoopStateIO) WalkDeltas(ctx context.Context) ([]*bundle.OneBlockFile, error) {
	return nil, nil
}

func (n *NoopStateIO) MergeDeltas(ctx context.Context, lowerBlockBoundary uint64, files []*bundle.OneBlockFile) error {
	return nil
}

func (n *NoopStateIO) WriteState(ctx context.Context, content []byte, blockNum uint64) error {
	return nil
}

func (n *NoopStateIO) ReadState(ctx context.Context, into []byte, blockNum uint64) error {
	return nil
}
