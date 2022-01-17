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
	Ring        map[string]*bck.WalletInfo
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
		Ring:        map[string]*bck.WalletInfo{getInfo.PubKey: getInfo},
	}
	return myNode
}
func (n *Node) getLastBlock() *bck.Block {
	if len(n.Chain) == 0 {
		return nil
	}
	return n.Chain[len(n.Chain)-1]
}

func (n *Node) IsValidSig(tx bck.Transaction) bool {

	err := rsa.VerifyPKCS1v15(tx.SenderAddress, crypto.SHA256, tx.Id, tx.Signature)
	return err == nil
}
func (n *Node) IsValidTx(tx bck.Transaction) bool {
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
func (n *Node) MineBlock(block *bck.Block) error {
}
func (n *Node) BroadcastBlock(block *bck.Block) error {
}
func (n *Node) IsValidBlock(block *bck.Block) bool {
}
func (n *Node) IsValidChain() bool {
}
func (n *Node) ResolveConflict(block *bck.Block) error {
}*/
