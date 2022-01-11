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

func NewNode(currBlockId int, bits int) *Node {
	w := bck.NewWallet(bits)
	getInfo := w.GetWalletInfo()
	return &Node{
		CurrBlockId: currBlockId,
		Wallet:      w,
		Ring:        map[string]*bck.WalletInfo{getInfo.PubKey: getInfo},
	}
}

func (n *Node) IsValidSig(tx bck.Transaction) bool {

	err := rsa.VerifyPKCS1v15(tx.SenderAddress, crypto.SHA256, tx.Id, tx.Signature)
	return err == nil
}
func (n *Node) IsValidTx(tx bck.Transaction) bool {
	//The validation is consisted of 2 steps
	//Step1: isValidSig
	//Step2: check transaction inputs/outputs
	isValidSig := n.IsValidSig(tx) //Step1
	txInputs := tx.Inputs
	for _, i := range txInputs {
		if _, ok := n.Wallet.Utxos[i.PreviousOutputId]; !ok {
			return false
		}
	}
	return isValidSig
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
}
*/
