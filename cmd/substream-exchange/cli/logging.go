package cli

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

func init() {
	zlog = zap.NewNop()
	_ = logging.ApplicationLogger("solgun", "github.com/streamingfast/substream-pancakeswap/cmd/substream-exchange", &zlog,
		logging.WithSwitcherServerAutoStart(),
	)
}
