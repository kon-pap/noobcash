package node

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"

	bck "github.com/kon-pap/noobcash/pkg/node/backend"
)

type NodeInfo struct {
	// channels for comms
	Id       int
	WInfo    *bck.WalletInfo
	Hostname string
	Port     string
}

func NewNodeInfo(id int, hostname, port string, pubKey *rsa.PublicKey) *NodeInfo {
	newNodeInfo := &NodeInfo{
		Id:       id,
		WInfo:    bck.NewWalletInfo(pubKey),
		Hostname: hostname,
		Port:     port,
	}
	return newNodeInfo
}

func (n *Node) SendByteSlice(data []byte, hostname, port string, endpoint endpointTy) (string, error) {
	return GetResponseBody(
		http.Post(
			fmt.Sprintf("http://%s:%s%s", hostname, port, endpoint),
			"application/json",
			bytes.NewBuffer(data),
		),
	)
}

func (n *Node) TrySendByteSlice(data []byte, hostname, port string, endpoint endpointTy) {
	log.Println("Sending to", hostname, port, endpoint)
	reply, err := n.SendByteSlice(data, hostname, port, endpoint)
	if err != nil {
		log.Println("Error sending to", hostname, port, endpoint, err)
		return
	}
	log.Println("Reply:", reply)
}

func (n *Node) TryBroadcastByteSlice(data []byte, endpoint endpointTy) {
	log.Println("Broadcasting to", endpoint)
	for _, node := range n.Ring {
		if node.Id == n.Id {
			continue
		}
		go n.TrySendByteSlice(data, node.Hostname, node.Port, endpoint)
	}
}
