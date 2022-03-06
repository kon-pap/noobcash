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
		nodeport, _ := cmd.Flags().GetString("nodeport")
		ip, _ := cmd.Flags().GetString("ip")
		newNode := node.NewNode(0, 1024, ip, nodeport)
		genBlock := backend.CreateGenesisBlock(nodecnt, &newNode.Wallet.PrivKey.PublicKey)
		fmt.Println(genBlock)

		apiport, _ := cmd.Flags().GetString("apiport")
		node.ServeApiForCli(apiport)
		//TODO: startup node and handle incoming nodes
		return nil
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}
