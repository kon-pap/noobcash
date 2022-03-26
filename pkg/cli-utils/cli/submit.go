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
		submit := createSubmitter(ip, port)
		reply, err := submit(args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Println(reply)

		return nil
	},
}

func createSubmitter(ip string, port int) func(string, string) (string, error) {
	return func(recipient, amount string) (string, error) {
		transactionJson := bytes.NewBuffer([]byte(`{"recipient":` + recipient + `,"amount":` + amount + `}`))
		body, err := node.GetResponseBody(
			http.Post(fmt.Sprintf("http://%s:%d/submit", ip, port), "application/json", transactionJson),
		)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
}

func init() {
	rootCmd.AddCommand(submitCmd)
}
