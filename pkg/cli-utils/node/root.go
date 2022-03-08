package node

import (
	"fmt"
	"os"
	"strings"

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
		newNode := setupNode(cmd)
		// fmt.Println(newNode)
		newNode.Start()
		return nil
	},
}

func getNodeApiHostDetails(cmd *cobra.Command) (string, string) {
	hostname, _ := cmd.Flags().GetString("hostname")
	x := strings.Split(hostname, ":")
	return x[0], x[1]
}

// Get cli flags create and set up a new node
func setupNode(cmd *cobra.Command) *node.Node {
	ip, nodeport := getNodeApiHostDetails(cmd)
	apiport, _ := cmd.Flags().GetString("apiport")
	newNode := node.NewNode(0, 1024, ip, nodeport, apiport)
	node.BootstrapHostname, _ = cmd.Flags().GetString("bootstrap")
	backend.BlockCapacity, _ = cmd.Flags().GetInt("blockcapacity")
	return newNode
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringP("apiport", "p", "9090", "Port to serve http api on")
	rootCmd.PersistentFlags().StringP("hostname", "n", "localhost:7070", "IP on which this node's node-api is available")
	rootCmd.PersistentFlags().StringP("bootstrap", "b", "localhost:7070", "Hostname of the bootstrap node")
	rootCmd.PersistentFlags().IntP("blockcapacity", "c", 10, "Transaction capacity of a block")
}
