package node

import (
	"fmt"
	"os"

	"github.com/kon-pap/noobcash/pkg/node"
	"github.com/kon-pap/noobcash/pkg/node/backend"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "Noobcash node",
	Long: `Noobcash is a peer-to-peer blockchain network supporting basic payments.
Class project for the course "Distributed Systems" at the National Technical University of Athens`,
	RunE: func(cmd *cobra.Command, args []string) error {
		isBootstrap, _ := cmd.Flags().GetBool("bootstrap")
		port, _ := cmd.Flags().GetString("port")
		nodecnt, _ := cmd.Flags().GetInt("nodecnt")
		// wallet := backend.NewWallet(1024)
		newNode := node.NewNode(0, 1024)
		if isBootstrap {
			genBlock := backend.CreateGenesisBlock(nodecnt, &newNode.Wallet.PrivKey.PublicKey)
			fmt.Println(genBlock)
		}
		fmt.Println("Starting http api server on port", port)
		node.ServeApi(port)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolP("bootstrap", "b", false, "Controls whether current node is bootstrap node or not")
	rootCmd.PersistentFlags().StringP("port", "p", "9090", "Port to serve http api on")
	rootCmd.PersistentFlags().IntP("nodecnt", "c", 5, "Number of nodes to bootstrap for")
}
