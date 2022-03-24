package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "A command line interface for Noobcash",
	Long: `Noobcash is a peer-to-peer blockchain network supporting basic payments.
Class project for the course "Distributed Systems" at the National Technical University of Athens`,
}

func getAddress(cmd *cobra.Command) (ip string, port int, err error) {
	address, err := cmd.Flags().GetString("address")
	if err != nil {
		return
	}
	addrPort := strings.Split(address, ":")
	ip = addrPort[0]
	port, err = strconv.Atoi(addrPort[1])
	if err != nil {
		return
	}
	return
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// persistent global app flags here
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringP("address", "a", "localhost:9090", "server address of noobcash node api to query")

	balanceCmd.SilenceUsage = true
	submitCmd.SilenceUsage = true
	viewCmd.SilenceUsage = true
	submitManyCmd.SilenceUsage = true

	balanceCmd.SilenceErrors = true
	submitCmd.SilenceErrors = true
	viewCmd.SilenceErrors = true
	submitManyCmd.SilenceErrors = true
}
