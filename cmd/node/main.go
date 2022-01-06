package main

import (
	"fmt"

	"github.com/kon-pap/noobcash/pkg/env"
	"github.com/kon-pap/noobcash/pkg/node/backend"
)

const dotenvPath string = "./config/node.env"

func main() {
	env.Import(dotenvPath)
	walletPath := env.Get("WALLET_PATH")

	wallet := backend.LoadWallet(walletPath, "first")
	fmt.Println(wallet.PrivKey.D.Bytes())

	// fmt.Println(env.Get("BOOTSTRAP_NODE_IP"))
}
