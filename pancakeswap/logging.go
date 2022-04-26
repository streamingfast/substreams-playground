package pancakeswap

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

func init() {
	zlog, _ = logging.PackageLogger("pancakeswap", "github.com/streamingfast/substreams/pancakeswap")
}
