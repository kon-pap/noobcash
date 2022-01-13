package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "A command line interface for Noobcash",
	Long: `Noobcash is a peer-to-peer blockchain network supporting basic payments.
Class project for the course "Distributed Systems" at the National Technical University of Athens`,
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
}
