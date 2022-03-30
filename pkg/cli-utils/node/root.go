package node

import (
	"fmt"
	"io"
	"log"
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
		newNode, closeFunc := setupNode(cmd)
		defer closeFunc()

		if err := newNode.Start(); err != nil {
			return err
		}
		return nil
	},
}

func saveLogs(fileId string) func() {
	logfile := "./logs" + fileId + ".txt"
	f, _ := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)

	out := os.Stderr
	mw := io.MultiWriter(out, f)

	r, w, _ := os.Pipe()

	// os.Stdout = w
	os.Stderr = w

	log.SetOutput(mw)
	exit := make(chan bool)

	go func() {
		_, _ = io.Copy(mw, r)
		exit <- true
	}()

	return func() {
		_ = w.Close()
		<-exit
		_ = f.Close()
	}
}

func getNodeApiHostDetails(cmd *cobra.Command) (string, string) {
	hostname, _ := cmd.Flags().GetString("hostname")
	x := strings.Split(hostname, ":")
	return x[0], x[1]
}

// Get cli flags create and set up a new node
func setupNode(cmd *cobra.Command) (*node.Node, func()) {
	ip, nodeport := getNodeApiHostDetails(cmd)
	apiport, _ := cmd.Flags().GetString("apiport")
	newNode := node.NewNode(0, 1024, ip, nodeport, apiport)
	node.BootstrapHostname, _ = cmd.Flags().GetString("bootstrap")
	backend.BlockCapacity, _ = cmd.Flags().GetInt("capacity")
	backend.TmpBlockCapacity = backend.BlockCapacity
	backend.Difficulty, _ = cmd.Flags().GetInt("difficulty")

	return newNode, saveLogs(apiport)
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
	rootCmd.PersistentFlags().IntP("capacity", "c", 10, "Transaction capacity of a block")
	rootCmd.PersistentFlags().IntP("difficulty", "d", 1, "Difficulty of mining a block")
}
