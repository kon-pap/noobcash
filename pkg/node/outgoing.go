package node

import (
	"crypto/rsa"

	bck "github.com/kon-pap/noobcash/pkg/node/backend"
)

// Enum of types of messages
const ()

type NodeInfo struct {
	// channels for comms
	Id       int
	WInfo    *bck.WalletInfo
	hostname string
	port     string
}

func NewNodeInfo(id int, hostname, port string, pubKey *rsa.PublicKey) *NodeInfo {
	newNodeInfo := &NodeInfo{
		WInfo:    bck.NewWalletInfo(pubKey),
		hostname: hostname,
		port:     port,
	}
	return newNodeInfo
}

// To be called by Bootstrap node after all nodes are registered to him
func (n *NodeInfo) SendNodeInfos() {
}

/*
 */
