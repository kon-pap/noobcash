package node

import (
	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Run Bootstrap node",
	RunE: func(cmd *cobra.Command, args []string) error {
		newNode, closeFunc := setupNode(cmd)
		defer closeFunc()

		nodecnt, _ := cmd.Flags().GetInt("nodecnt")
		newNode.MakeBootstrap(nodecnt)

		if err := newNode.Start(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	bootstrapCmd.PersistentFlags().Int("nodecnt", 5, "Number of nodes to bootstrap for")

	rootCmd.AddCommand(bootstrapCmd)
}
