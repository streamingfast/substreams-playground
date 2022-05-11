package exchange

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

func init() {
	zlog, _ = logging.ApplicationLogger("exchange", "github.com/streamingfast/substreams-playground/exchange",
		logging.WithSwitcherServerAutoStart(),
	)
}
