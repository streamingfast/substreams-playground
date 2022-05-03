package exchange

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "exchange",
	Short:        "bsc exchange tool",
	SilenceUsage: true,
}

func init() {
}
