package node

import (
	"crypto"
	"crypto/rsa"

	bck "github.com/kon-pap/noobcash/pkg/node/backend"
)

type Node struct {
	// chain, currBlockId, wallet, ring
	Chain       []*bck.Block
	CurrBlockId int
	Wallet      *bck.Wallet
	Ring        map[string]*NodeInfo
}

var myNode *Node

func NewNode(currBlockId int, bits int) *Node {
	if myNode != nil {
		return myNode // enforces only one node per runtime
	}
	w := bck.NewWallet(bits)
	getInfo := w.GetWalletInfo()
	myNode = &Node{
		Chain:       []*bck.Block{},
		CurrBlockId: currBlockId,
		Wallet:      w,
		Ring: map[string]*NodeInfo{
			getInfo.PubKey: NewNodeInfo(getInfo),
		},
	}
	return myNode
}

//* TRANSACTION
func (n *Node) IsValidSig(tx *bck.Transaction) bool {
	err := rsa.VerifyPKCS1v15(tx.SenderAddress, crypto.SHA256, tx.Id, tx.Signature)
	return err == nil
}
func (n *Node) IsValidTx(tx *bck.Transaction) bool {
	//The validation is consisted of 2 steps
	//Step1: isValidSig
	//Step2: check transaction inputs/outputs
	return n.IsValidSig(tx) && func() bool {
		for txInId := range tx.Inputs {
			if _, ok := n.Wallet.Utxos[string(txInId)]; !ok {
				return false
			}
		}
		return true
	}()
}

/*
func (n *Node) BroadcastTx(tx *bck.Transaction) error {
}
*/

//* BLOCK
/*
func (n *Node) MineBlock(block *bck.Block) error {
}
// currhash is correct && previous_hash is actually the hash of the previous block
func (n *Node) IsValidBlock(block *bck.Block) bool {
}
func (n *Node) ApplyBlock(block *bck.Block) error {
}
func (n *Node) BroadcastBlock(block *bck.Block) error {
}
*/

//* CHAIN
func (n *Node) getLastBlock() *bck.Block {
	if len(n.Chain) == 0 {
		return nil
	}
	return n.Chain[len(n.Chain)-1]
}

/*
func (n *Node) IsValidChain() bool {
}
func (n *Node) ApplyChain(blocks []*bck.Block) error {
}
func (n *Node) ResolveConflict(block *bck.Block) error {
}
*/

//* RING
/*
// 1. Send Wallet pubkey (Connecting will provide the IP and port on its own)
// 2. Receive node id
// 3. Wait for info of all other nodes
func (n *Node) ConnectToBootstrap(ip, port string) error {
}

// Send IP, port, pubkey of all nodes
func (n *Node) BroadcastRingInfo() error {
}
*/
