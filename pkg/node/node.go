package node

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	bck "github.com/kon-pap/noobcash/pkg/node/backend"
)

//TODO: implement logic for filling up blocks not in the Chain slice
//TODO: implement logic for chain branches

var BootstrapHostname string

type Node struct {
	Id             int
	Chain          []*bck.Block
	CurrBlockId    int
	Wallet         *bck.Wallet
	IncBlockChan   chan *bck.Block
	MinedBlockChan chan *bck.Block
	Ring           map[string]*NodeInfo
	PendingTxs     *TxQueue

	info    *NodeInfo
	apiport string

	// Only used by bootstrap
	BsNextNodeId *MuInt
}

func NewNode(currBlockId, bits int, ip, port, apiport string) *Node {
	w := bck.NewWallet(bits)
	newNodeInfo := NewNodeInfo(-1, ip, port, &w.PrivKey.PublicKey)
	newNode := &Node{
		Chain:          []*bck.Block{},
		CurrBlockId:    currBlockId,
		Wallet:         w,
		IncBlockChan:   make(chan *bck.Block, 1),
		MinedBlockChan: make(chan *bck.Block, 1),
		PendingTxs:     NewTxQueue(),
		info:           newNodeInfo,
		apiport:        apiport,
		Ring: map[string]*NodeInfo{
			bck.PubKeyToPem(&w.PrivKey.PublicKey): newNodeInfo,
		},
	}
	return newNode
}

func (n *Node) MakeBootstrap() {
	log.Println("Becoming bootstrap...")
	n.Id = 0
	n.BsNextNodeId = &MuInt{
		Value: 1,
	}
}

// Listens for incoming or mined blocks
//
// Should be called as a goroutine
func (n *Node) SelectMinedOrIncomingBlock() {
	for {
		select {
		case minedBlock := <-n.MinedBlockChan:
			//TODO: handle minedBlock
			fmt.Println("Mined block:", minedBlock)
		case incomingBlock := <-n.IncBlockChan:
			//TODO: handle incomingBlock
			fmt.Println("Incoming block:", incomingBlock)
		}
	}
}

func (n *Node) IsBootstrap() bool {
	return n.Id == 0
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
	}
	return nil
}

/*
func (n *Node) BroadcastTx(tx *bck.Transaction) error {
}
*/

//* BLOCK
// check block validity
func (n *Node) MineBlock(block *bck.Block) {
}

/*
// currhash is correct && previous_hash is actually the hash of the previous block
// recheck transaction validity
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
*/

/*
func (n *Node) ResolveConflict(block *bck.Block) error {
}
*/

//* RING
/*
// 1. Send Wallet pubkey
// 2. Receive node id
// 3. Wait for info of all other nodes
func (n *Node) ConnectToBootstrap() error {
}

// Send IP, port, pubkey of all nodes
func (n *Node) BroadcastRingInfo() error {
}
*/
