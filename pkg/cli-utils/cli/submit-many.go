package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var submitManyCmd = &cobra.Command{
	Use:   "s <path-to-file>",
	Short: "Submit many transactions",
	Long:  "Submit many transactions to the blockchain.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("invalid number of arguments. expected 1, got %d", len(args))
		}
		filepath := args[0]
		file, err := os.Open(filepath)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)

		ip, port, err := getAddress(cmd)
		if err != nil {
			return err
		}
		waitEnable, err := cmd.Flags().GetBool("wait")
		if err != nil {
			return err
		}
		var submit func(string, string) (string, error)
		if waitEnable {
			timeout, err := cmd.Flags().GetInt("timeout")
			if err != nil {
				return err
			}
			submit = createInsistSubmitter(ip, port, timeout)
		} else {
			submit = createSubmitter(ip, port)
		}
		for scanner.Scan() {
			line := scanner.Text()
			recipient, amount := getTxDetails(line)
			reply, err := submit(recipient, amount)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(reply)
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		return nil
	},
}

func createInsistSubmitter(ip string, port, timeout int) func(string, string) (string, error) {
	submit := createSubmitter(ip, port)
	return func(recipient, amount string) (reply string, err error) {
		for reply, err = submit(recipient, amount); err != nil; reply, err = submit(recipient, amount) {
			fmt.Println(err)
			// sleep for 5 seconds
			time.Sleep(time.Duration(timeout) * time.Second)
		}
		return
	}
}

func getTxDetails(line string) (string, string) {
	parts := strings.Split(line, " ")
	return parts[0][2:], parts[1]
}

func init() {
	rootCmd.AddCommand(submitManyCmd)

	submitManyCmd.PersistentFlags().BoolP("wait", "w", false, "retry submitting every transaction until it is accepted")
	submitManyCmd.PersistentFlags().IntP("timeout", "t", 5, "timeout in seconds")
}
