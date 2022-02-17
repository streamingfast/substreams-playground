package state

import (
	"context"
	"fmt"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/merger/bundle"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type IOFactory interface {
	New(name string) StateIO
}

type DiskStateIOFactory struct {
	dataFolder string
}

func NewDiskStateIOFactory(folder string) IOFactory {
	return &DiskStateIOFactory{dataFolder: folder}
}

func (f *DiskStateIOFactory) New(name string) StateIO {
	return &DiskStateIO{
		name:       name,
		dataFolder: f.dataFolder,
	}
}

type StateIO interface {
	WriteDelta(ctx context.Context, content []byte, obf *bundle.OneBlockFile) error
	ReadDelta(ctx context.Context, obf *bundle.OneBlockFile) ([]byte, error)
	DeleteDelta(ctx context.Context, obf *bundle.OneBlockFile) error

	WalkDeltas(ctx context.Context, startBlockNumber uint64, f func(obf *bundle.OneBlockFile) error) error
	MergeDeltas(ctx context.Context, lowerBlockBoundary uint64, files []*bundle.OneBlockFile) error

	WriteState(ctx context.Context, content []byte, block *bstream.Block) error
	ReadState(ctx context.Context, blockNum uint64) ([]byte, error)
}

type DiskStateIO struct {
	name       string
	dataFolder string
}

func (d *DiskStateIO) WriteDelta(ctx context.Context, content []byte, obf *bundle.OneBlockFile) error {
	err := ioutil.WriteFile(filepath.Join(d.dataFolder, GetDeltaFileName(d.name, mustOneBlockFileToBlock(obf))), content, os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing %s delta at block %d: %w", d.name, obf.Num, err)
	}

	return nil
}

func (d *DiskStateIO) ReadDelta(ctx context.Context, obf *bundle.OneBlockFile) (data []byte, err error) {
	for filename := range obf.Filenames { // will try to get MemoizeData from any of those files
		path := filepath.Join(d.dataFolder, filename)
		if _, err = os.Stat(path); err != nil {
			err = fmt.Errorf("file %s does not exist", path)
			continue
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		data, err = ioutil.ReadFile(filepath.Join(d.dataFolder, filename))
		if err != nil {
			continue
		}

	}

	return data, err
}

func (d *DiskStateIO) DeleteDelta(ctx context.Context, obf *bundle.OneBlockFile) error {
	//TODO: this is currently a no-op.  merging and purging of files will be a future optimization
	return nil
}

func (d *DiskStateIO) WalkDeltas(ctx context.Context, startBlockNumber uint64, f func(obf *bundle.OneBlockFile) error) error {
	return filepath.WalkDir(d.dataFolder, func(path string, de fs.DirEntry, err error) error {
		if de.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, "delta") {
			return nil
		}

		pathPrefix := fmt.Sprintf("%s%b", d.dataFolder, filepath.Separator)
		fileName := path
		if strings.HasPrefix(path, pathPrefix) {
			fileName = path[len(pathPrefix):]
		}

		obf := mustParseFileToOneBlockFile(fileName)
		obf.Filenames[path] = struct{}{}

		if obf.Num < startBlockNumber {
			return nil
		}

		err = f(obf)
		if err != nil {
			return err
		}

		return nil
	})
}

func (d *DiskStateIO) MergeDeltas(ctx context.Context, lowerBlockBoundary uint64, files []*bundle.OneBlockFile) error {
	//TODO: this is currently a no-op.  merging and purging of files will be a future optimization
	return nil
}

func (d *DiskStateIO) WriteState(ctx context.Context, content []byte, block *bstream.Block) error {
	err := ioutil.WriteFile(filepath.Join(d.dataFolder, GetStateFileName(d.name, block)), content, os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing %s kv at block %d: %w", d.name, block.Number, err)
	}

	return nil
}

func (d *DiskStateIO) ReadState(ctx context.Context, blockNumber uint64) ([]byte, error) {
	relativeStartBlock := (blockNumber / 100) * 100

	block := &bstream.Block{Number: relativeStartBlock}

	path := filepath.Join(d.dataFolder, GetStateFileName(d.name, block))
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("file %s does not exist: %w", path, err)
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", path, err)
	}

	return data, nil
}

type NoopStateIO struct {
}

func (n *NoopStateIO) WriteDelta(ctx context.Context, content []byte, obf *bundle.OneBlockFile) error {
	return nil
}

func (n *NoopStateIO) ReadDelta(ctx context.Context, obf *bundle.OneBlockFile) ([]byte, error) {
	return nil, nil
}

func (n *NoopStateIO) DeleteDelta(ctx context.Context, obf *bundle.OneBlockFile) error {
	return nil
}

func (n *NoopStateIO) WalkDeltas(ctx context.Context, startBlockNumber uint64, f func(obf *bundle.OneBlockFile) error) error {
	return nil
}

func (n *NoopStateIO) MergeDeltas(ctx context.Context, lowerBlockBoundary uint64, files []*bundle.OneBlockFile) error {
	return nil
}

func (n *NoopStateIO) WriteState(ctx context.Context, content []byte, block *bstream.Block) error {
	return nil
}

func (n *NoopStateIO) ReadState(ctx context.Context, blockNum uint64) ([]byte, error) {
	return nil, nil
}

func GetDeltaFileName(name string, block *bstream.Block) string {
	return fmt.Sprintf("%d-%d-%s-%s-%s.delta", block.Num(), block.LIBNum(), block.ID(), block.PreviousID(), name)
}

func GetStateFileName(name string, block *bstream.Block) string {
	blockNum := block.Num()
	return fmt.Sprintf("%d-%s.kv", blockNum, name)
}

func mustParseFileToOneBlockFile(path string) *bundle.OneBlockFile {
	trimmedPath := strings.TrimSuffix(path, ".delta")
	parts := strings.Split(trimmedPath, "-")
	if len(parts) != 5 {
		panic("invalid path")
	}

	uint64ToPtr := func(num uint64) *uint64 {
		var p *uint64
		*p = num
		return p
	}

	blockId := parts[2]
	blockPrevId := parts[3]
	blockNum, err := strconv.Atoi(parts[0])
	if err != nil {
		panic("invalid block num")
	}
	blockLibNum, err := strconv.Atoi(parts[1])
	if err != nil {
		panic("invalid prev block num")
	}

	return &bundle.OneBlockFile{
		CanonicalName: path,
		ID:            blockId,
		Num:           uint64(blockNum),
		InnerLibNum:   uint64ToPtr(uint64(blockLibNum)),
		PreviousID:    blockPrevId,
	}
}

func mustBlockToOneBlockFile(name string, block *bstream.Block) *bundle.OneBlockFile {
	getUint64Pointer := func(n uint64) *uint64 {
		var ptr *uint64
		*ptr = n
		return ptr
	}

	filename := GetDeltaFileName(name, block)

	return &bundle.OneBlockFile{
		CanonicalName: filename,
		Filenames: map[string]struct{}{
			filename: {},
		},
		ID:          block.ID(),
		PreviousID:  block.PreviousID(),
		BlockTime:   block.Time(),
		Num:         block.Num(),
		InnerLibNum: getUint64Pointer(block.LibNum),
	}
}

func mustOneBlockFileToBlock(obf *bundle.OneBlockFile) *bstream.Block {
	return &bstream.Block{
		Id:         obf.ID,
		Number:     obf.Num,
		PreviousId: obf.PreviousID,
		Timestamp:  obf.BlockTime,
		LibNum:     obf.LibNum(),
	}
}
