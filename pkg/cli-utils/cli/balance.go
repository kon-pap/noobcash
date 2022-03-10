package cli

import (
	"fmt"
	"net/http"

	"github.com/kon-pap/noobcash/pkg/node"
	"github.com/spf13/cobra"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "View your balance",
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, port, err := getAddress(cmd)
		if err != nil {
			return err
		}
		body, err := node.GetResponseBody(
			http.DefaultClient.Get(fmt.Sprintf("http://%s:%d/balance", ip, port)),
		)
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(balanceCmd)
}
