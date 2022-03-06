package node

import (
	"fmt"

	"github.com/kon-pap/noobcash/pkg/node"
	"github.com/kon-pap/noobcash/pkg/node/backend"
	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Run Bootstrap node",
	RunE: func(cmd *cobra.Command, args []string) error {
		nodecnt, _ := cmd.Flags().GetInt("nodecnt")
		newNode := node.NewNode(0, 1024)
		genBlock := backend.CreateGenesisBlock(nodecnt, &newNode.Wallet.PrivKey.PublicKey)
		fmt.Println(genBlock)

		port, _ := cmd.Flags().GetString("port")
		node.ServeApiForCli(port)
		//TODO: startup node and handle incoming nodes
		return nil
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}
