package main

import (
	"github.com/kon-pap/noobcash/pkg/env"
	"github.com/kon-pap/noobcash/pkg/node/backend"
)

const dotenvPath string = "./config/node.env"

func main() {
	env.Import(dotenvPath)
	walletPath := env.Get("WALLET_PATH")

	wallet := backend.CreateWallet("first", 1024)
	wallet.WritePEM(walletPath)

	// fmt.Println(env.Get("BOOTSTRAP_NODE_IP"))
}
