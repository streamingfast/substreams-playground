package storage

import (
	"context"
	graphnode "github.com/streamingfast/substream-pancakeswap/graph-node"
	"time"
)

type Store interface {
	BatchSave(ctx context.Context, blockNum uint64, blockHash string, blockTime time.Time, updates map[string]map[string]graphnode.Entity, cursor string) (err error)
	Load(ctx context.Context, id string, entity graphnode.Entity, blockNum uint64) error
	LoadAllDistinct(ctx context.Context, model graphnode.Entity, blockNum uint64) ([]graphnode.Entity, error)

	LoadCursor(ctx context.Context) (string, error)

	CleanDataAtBlock(ctx context.Context, blockNum uint64) error
	CleanUpFork(ctx context.Context, newHeadBlock uint64) error

	Close() error
}
