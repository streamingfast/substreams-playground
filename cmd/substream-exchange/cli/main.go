package cli

import (
	"go.uber.org/zap"
)

func Main() {
	/// all flags here

	// run cmd
	err := rootCmd.Execute()
	if err != nil {
		zlog.Error("running cmd", zap.Error(err))
	}
}
