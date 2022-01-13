package node

import (
	"fmt"
	"os"

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
		wallet := backend.NewWallet(1024)
		// fmt.Println(wallet)
		if isBootstrap {
			// fmt.Println("This is the bootstrap node (id=0)!")
			genBlock := backend.CreateGenesisBlock(100, &wallet.PrivKey.PublicKey)
			fmt.Println(genBlock)
		}

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
	rootCmd.Flags().BoolP("bootstrap", "b", false, "Controls whether current node is bootstrap node or not")
}
