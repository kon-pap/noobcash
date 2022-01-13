package cli

import (
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View the last block",
	Long:  `View the last block in the blockchain.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: implement
		return nil
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
}
