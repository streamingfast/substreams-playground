package exchange

//
//import (
//	"context"
//	"fmt"
//	"github.com/streamingfast/merger/bundle"
//	"io"
//	"io/fs"
//	"io/ioutil"
//	"os"
//	"path/filepath"
//	"strings"
//	"sync"
//)
//
//type MergedFile struct{}
//
//func GetDeltaFileName(name string, blockNum uint64) string {
//	return fmt.Sprintf("%d-%s.delta", blockNum, name)
//}
//
//func GetStateFileName(name string, blockNum uint64) string {
//	return fmt.Sprintf("%d-%s.kv", blockNum, name)
//}
//
//func GetDeltaMergedFileName(name string, blockNum uint64) string {
//	return fmt.Sprintf("%d-%s.merged.delta", blockNum, name)
//}
//
//type StateBuilderIO interface {
//	WriteDelta(ctx context.Context, content []byte, blockNum uint64) error
//	ReadDelta(ctx context.Context, file *bundle.OneBlockFile) ([]byte, error)
//	DeleteDelta(ctx context.Context, file *bundle.OneBlockFile) error
//
//	WalkDeltas(ctx context.Context, startBlockNumber uint64, f func(filename string) error) error
//	MergeDeltas(ctx context.Context, lowerBlockBoundary uint64, files []*bundle.OneBlockFile) error
//
//	ReadMergedDeltas(ctx context.Context, filename string) ([][]byte, error)
//
//	WriteState(ctx context.Context, content []byte, blockNum uint64) error
//	ReadState(ctx context.Context, blockNum uint64) ([]byte, error)
//}
//
//type DiskStateIO struct {
//	name       string
//	dataFolder string
//
//	mu sync.RWMutex
//}
//
//func (d *DiskStateIO) WriteDelta(ctx context.Context, content []byte, blockNum uint64) error {
//	err := ioutil.WriteFile(filepath.Join(d.dataFolder, GetDeltaFileName(d.name, blockNum)), content, os.ModePerm)
//	if err != nil {
//		return fmt.Errorf("writing %s delta at block %d: %w", d.name, blockNum, err)
//	}
//
//	return nil
//}
//
//func (d *DiskStateIO) ReadDelta(ctx context.Context, file *bundle.OneBlockFile) (data []byte, err error) {
//	for filename := range file.Filenames { // will try to get MemoizeData from any of those files
//		path := filepath.Join(d.dataFolder, filename)
//		if _, err = os.Stat(path); err != nil {
//			err = fmt.Errorf("file %s does not exist", path)
//			continue
//		}
//
//		select {
//		case <-ctx.Done():
//			return nil, ctx.Err()
//		default:
//		}
//
//		data, err = ioutil.ReadFile(filepath.Join(d.dataFolder, filename))
//		if err != nil {
//			continue
//		}
//
//	}
//
//	return data, err
//}
//
//func (d *DiskStateIO) ReadMergedDeltas(ctx context.Context, mergedFile string) ([][]byte, error) {
//
//}
//
//func (d *DiskStateIO) DeleteDelta(ctx context.Context, file *bundle.OneBlockFile) error {
//	var err error
//	for filename := range file.Filenames {
//		path := filepath.Join(d.dataFolder, filename)
//		if _, err = os.Stat(path); err != nil {
//			err = fmt.Errorf("file %s does not exist", path)
//			continue
//		}
//
//		select {
//		case <-ctx.Done():
//			return ctx.Err()
//		default:
//		}
//
//		err = os.Remove(path)
//		if err != nil {
//			return err
//		}
//	}
//
//	return err
//}
//
//func (d *DiskStateIO) WalkDeltas(ctx context.Context, startBlockNumber uint64, f func(filename string) error) error {
//	d.mu.RLock()
//	defer d.mu.RUnlock()
//
//	deltaFileBlockNum := func(filepath string) uint64 {
//		return 0
//	}
//
//	filepath.WalkDir(d.dataFolder, func(path string, d fs.DirEntry, err error) error {
//		if d.IsDir() {
//			return nil
//		}
//
//		if !strings.HasSuffix(path, "delta") {
//			return nil
//		}
//
//		block := deltaFileBlockNum(path)
//		if block < startBlockNumber {
//			return nil
//		}
//
//		err = f(path)
//		if err != nil {
//			return err
//		}
//
//		return nil
//	})
//}
//
//func (d *DiskStateIO) MergeDeltas(ctx context.Context, lowerBlockBoundary uint64, files []*bundle.OneBlockFile) error {
//	d.mu.Lock()
//	defer d.mu.Unlock()
//
//	bundleReader := bundle.NewBundleReader(ctx, files, d.ReadDelta)
//
//	path := filepath.Join(d.dataFolder, GetDeltaMergedFileName(d.name, lowerBlockBoundary))
//	bundleWriter, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
//	if err != nil {
//		return fmt.Errorf("opening bundle file %s: %w", path, err)
//	}
//	defer bundleWriter.Close()
//
//	_, err = io.Copy(bundleWriter, bundleReader)
//	if err != nil {
//		return fmt.Errorf("copying data from bundleReader at block boundary %d: %w", lowerBlockBoundary, err)
//	}
//
//	return nil
//}
//
//func (d *DiskStateIO) WriteState(ctx context.Context, content []byte, blockNum uint64) error {
//	err := ioutil.WriteFile(filepath.Join(d.dataFolder, GetStateFileName(d.name, blockNum)), content, os.ModePerm)
//	if err != nil {
//		return fmt.Errorf("writing %s kv at block %d: %w", d.name, blockNum, err)
//	}
//
//	return nil
//}
//
//func (d *DiskStateIO) ReadState(ctx context.Context, blockNumber uint64) ([]byte, error) {
//	relativeStartBlock := (blockNumber / 100) * 100
//	path := filepath.Join(d.dataFolder, GetStateFileName(d.name, relativeStartBlock))
//	if _, err := os.Stat(path); err != nil {
//		return nil, fmt.Errorf("file %s does not exist: %w", path, err)
//	}
//
//	data, err := ioutil.ReadFile(path)
//	if err != nil {
//		return nil, fmt.Errorf("reading file %s: %w", path, err)
//	}
//
//	return data, nil
//}
//
//type NoopStateIO struct {
//}
//
//func (n *NoopStateIO) WriteDelta(ctx context.Context, content []byte, blockNum uint64) error {
//	return nil
//}
//
//func (n *NoopStateIO) ReadDelta(ctx context.Context, file *bundle.OneBlockFile) ([]byte, error) {
//	return nil, nil
//}
//
//func (n *NoopStateIO) DeleteDelta(ctx context.Context, file *bundle.OneBlockFile) error {
//	return nil
//}
//
//func (n *NoopStateIO) WalkDeltas(ctx context.Context, startBlockNumber uint64, f func(filename string) error) error {
//	return nil
//}
//
//func (n *NoopStateIO) MergeDeltas(ctx context.Context, lowerBlockBoundary uint64, files []*bundle.OneBlockFile) error {
//	return nil
//}
//
//func (n *NoopStateIO) WriteState(ctx context.Context, content []byte, blockNum uint64) error {
//	return nil
//}
//
//func (n *NoopStateIO) ReadState(ctx context.Context, blockNum uint64) ([]byte, error) {
//	return nil, nil
//}
