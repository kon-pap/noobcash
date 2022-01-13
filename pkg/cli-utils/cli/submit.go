package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var submitCmd = &cobra.Command{
	Use:   "t <recipient_address> <amount>",
	Short: "Submit a transaction",
	Long:  `Submit a transaction to the blockchain.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("invalid number of arguments. expected 2, got %d", len(args))
		}
		// TODO: implement
		return nil
	},
}

func init() {
	rootCmd.AddCommand(submitCmd)
}
