package exchange

import (
	_ "github.com/streamingfast/sf-ethereum/types"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().Int64P("start-block", "s", -1, "Start block for blockchain firehose")
	rootCmd.PersistentFlags().Uint64P("stop-block", "t", 0, "Stop block for blockchain firehose")
	rootCmd.PersistentFlags().Bool("no-return-handler", false, "Avoid printing output for module")

	///postgres loader flags
	rootCmd.PersistentFlags().String("pg-dsn", "", "dsn for postgres database")
	rootCmd.PersistentFlags().String("pg-schema", "", "postgres schema name")
	rootCmd.PersistentFlags().Bool("pg-disable-transactions", false, "disable postgres transactions for faster inserts")
	rootCmd.PersistentFlags().String("pg-deployment", "", "subgraph deployment name")
}

// remoteCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "exchange [manifest] [module_name]",
	Short:        "Run pancakeswap exchange substream",
	SilenceUsage: true,
}
