package cli

import (
	"fmt"
	"net/http"

	"github.com/kon-pap/noobcash/pkg/node"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Get stats",
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, port, err := getAddress(cmd)
		if err != nil {
			return err
		}
		body, err := node.GetResponseBody(
			http.DefaultClient.Get(fmt.Sprintf("http://%s:%d/view/stats", ip, port)),
		)
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
