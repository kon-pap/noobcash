package node

import bck "github.com/kon-pap/noobcash/pkg/node/backend"

const ()

type NodeInfo struct {
	// channels for comms
	WInfo *bck.WalletInfo
}

func NewNodeInfo(wInfo *bck.WalletInfo) *NodeInfo {
	return &NodeInfo{
		WInfo: wInfo,
	}
}

//* DRAFT
// for {
//   select {
// 	case newBlock := <- { try to mine a block }:
//   handle newly mined block
// 	case newBlock := <- { wait for incoming block }:
//	 handle newly received block
//   }
// }

/*

 */
