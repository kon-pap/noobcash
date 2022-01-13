package cli

import (
	"github.com/spf13/cobra"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "View your balance",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: implement
		return nil
	},
}

func init() {
	rootCmd.AddCommand(balanceCmd)
}
