package main

import (
	"flag"
	"fmt"

	"github.com/kon-pap/noobcash/pkg/node/backend"
)

func main() {
	isBootstrap := flag.Bool("bootstrap", false, "a bool")
	flag.Parse()

	wallet := backend.NewWallet(1024)
	fmt.Println(wallet)

	if *isBootstrap {
		fmt.Println("This is the bootstrap node (id=0)!")
	}
}
