package exchange

import (
	"github.com/spf13/cobra"
	_ "github.com/streamingfast/substream-pancakeswap/cli/exchange/graphnode"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "exchange",
	Short:        "bsc exchange tool",
	SilenceUsage: true,
}

func init() {
}
