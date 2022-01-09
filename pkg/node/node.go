package node

import bck "github.com/kon-pap/noobcash/pkg/node/backend"

type Node struct {
	// chain, currBlockId, wallet, ring
	Chain       []bck.Block
	CurrBlockId int
	Wallet      *bck.Wallet
	Ring        []byte
}

func NewNode(currBlockId int, bits int) *Node {
	w := bck.NewWallet(bits)
	return &Node{
		CurrBlockId: currBlockId,
		Wallet:      w,
	}
}

/*
func (n *Node) IsValidSig(tx bck.Transaction) bool {
}
func (n *Node) IsValidTx(tx bck.Transaction) bool {
}
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
