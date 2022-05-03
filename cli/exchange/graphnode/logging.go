package graphnode

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

func init() {
	zlog, _ = logging.PackageLogger("graphnode-loader", "github.com/streamingfast/substreams-playground/cli/exchange/graphnode")
}
