package node

import (
	"crypto"
	"crypto/rsa"
	"fmt"
	"log"

	bck "github.com/kon-pap/noobcash/pkg/node/backend"
)

type Node struct {
	// chain, currBlockId, wallet, ring
	Id          int
	Chain       []*bck.Block
	CurrBlockId int
	Wallet      *bck.Wallet
	Ring        map[string]*NodeInfo
	// TODO: add mutexes to lock necessary resources
}

var myNode *Node

func NewNode(currBlockId int, bits int) *Node {
	if myNode != nil {
		return myNode // enforces only one node per runtime
	}
	w := bck.NewWallet(bits)
	walletInfo := w.GetWalletInfo()
	myNodeInfo := NewNodeInfo(-1, "", "", walletInfo.PubKey)
	myNode = &Node{
		Chain:       []*bck.Block{},
		CurrBlockId: currBlockId,
		Wallet:      w,
		Ring: map[string]*NodeInfo{
			bck.PubKeyToPem(walletInfo.PubKey): myNodeInfo,
		},
	}
	return myNode
}

//* TRANSACTION
func (n *Node) IsValidSig(tx *bck.Transaction) bool {
	err := rsa.VerifyPKCS1v15(tx.SenderAddress, crypto.SHA256, tx.Id, tx.Signature)
	if err == nil {
		log.Println("Signature validation failed")
	}
	return err == nil
}
func (n *Node) IsValidTx(tx *bck.Transaction) bool {
	//The validation is consisted of 2 steps
	//Step1: isValidSig
	//Step2: check transaction inputs/outputs
	return n.IsValidSig(tx) && func() bool {
		senderNode := n.Ring[bck.PubKeyToPem(tx.SenderAddress)]
		for txInId := range tx.Inputs {
			if _, ok := senderNode.WInfo.Utxos[string(txInId)]; !ok {
				log.Println("Wallet", senderNode.Id, "does not have UTXO", txInId)
				return false
			}
		}
		return true
	}()
}
func (n *Node) AcceptTx(tx *bck.Transaction) error {
	if !n.IsValidTx(tx) {
		return fmt.Errorf("transaction is not valid")
	}
	if n.getLastBlock() == nil {
		return fmt.Errorf("internal error: no block in chain")
	}
	n.getLastBlock().AddTx(tx)
	if n.getLastBlock().IsFull() {
		//! NOTE: MineBlock will fill the block's hash at the end
		//! Assumption: MineBlock will increment the node.CurrBlockId
		n.MineBlock(n.getLastBlock())
		n.Chain = append(n.Chain, bck.NewBlock(
			len(n.Chain),
			n.getLastBlock().CurrentHash,
		))
	}
	return nil
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
