package cli

import (
	"fmt"
	"net/http"

	"github.com/kon-pap/noobcash/pkg/node"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View the last block",
	Long:  `View the last block in the blockchain.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, port, err := getAddress(cmd)
		if err != nil {
			return err
		}
		body, err := node.GetResponseBody(
			http.DefaultClient.Get(fmt.Sprintf("http://%s:%d/view", ip, port)),
		)
		if err != nil {
			return err
		}
		fmt.Println(body)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
}
