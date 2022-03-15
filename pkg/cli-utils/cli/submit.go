package cli

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kon-pap/noobcash/pkg/node"
	"github.com/spf13/cobra"
)

var submitCmd = &cobra.Command{
	Use:   "t <recipient_address> <amount>",
	Short: "Submit a transaction",
	Long:  `Submit a transaction to the blockchain.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("invalid number of arguments. expected 2, got %d", len(args))
		}
		_, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		_, err = strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		ip, port, err := getAddress(cmd)
		if err != nil {
			return err
		}
		transactionJson := bytes.NewBuffer([]byte(`{"recipient":` + args[0] + `,"amount":` + args[1] + `}`))
		body, err := node.GetResponseBody(
			http.Post(fmt.Sprintf("http://%s:%d/submit", ip, port), "application/json", transactionJson),
		)
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(submitCmd)
}
