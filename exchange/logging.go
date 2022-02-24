package exchange

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

func init() {
	zlog, _ = logging.PackageLogger("exchange", "github.com/streamingfast/substream-pancakeswap/exchange")
}
