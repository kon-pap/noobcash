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
	Id            int
	Chain         []*bck.Block
	CurrBlockId   int
	Wallet        *bck.Wallet
	IncBlockChn   chan *bck.Block
	MinedBlockChn chan *bck.Block
	Ring          map[string]*NodeInfo
	// TODO: add mutexes to lock necessary resources
	BsNextNodeId *MuInt
}

var myNode *Node

func NewNode(currBlockId, bits int, ip, port string) *Node {
	if myNode != nil {
		return myNode // enforces only one node per runtime
	}
	w := bck.NewWallet(bits)
	myNodeInfo := NewNodeInfo(-1, ip, port, &w.PrivKey.PublicKey)
	myNode = &Node{
		Chain:         []*bck.Block{},
		CurrBlockId:   currBlockId,
		Wallet:        w,
		IncBlockChn:   make(chan *bck.Block, 1),
		MinedBlockChn: make(chan *bck.Block, 1),
		Ring: map[string]*NodeInfo{
			bck.PubKeyToPem(&w.PrivKey.PublicKey): myNodeInfo,
		},
	}
	return myNode
}

func (n *Node) MakeBootstrap() {
	log.Println("Becoming bootstrap...")
	n.Id = 0
	n.BsNextNodeId = &MuInt{
		Value: 1,
	}
}

// Fires a goroutine to listen for incoming or mined blocks
func (n *Node) SelectMinedOrIncomingBlock() {
	go func() {
		for {
			select {
			case minedBlock := <-n.MinedBlockChn:
				//TODO: handle minedBlock
				fmt.Println("Mined block:", minedBlock)
			case incomingBlock := <-n.IncBlockChn:
				//TODO: handle incomingBlock
				fmt.Println("Incoming block:", incomingBlock)
			}
		}
	}()
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

// TODO implement this

/*


 */
func (n *Node) ApplyTx(tx *bck.Transaction) error {
	if !n.IsValidTx(tx) {
		return fmt.Errorf("transaction is not valid")
	}
	stringSenderAddress := bck.PubKeyToPem(tx.SenderAddress)
	stringReceiverAddress := bck.PubKeyToPem(tx.ReceiverAddress)
	stringNodeAddress := bck.PubKeyToPem(&n.Wallet.PrivKey.PublicKey)

	thisIsSender := stringSenderAddress == stringNodeAddress
	thisIsReceiver := stringReceiverAddress == stringNodeAddress

	senderWallet := n.Ring[stringSenderAddress].WInfo

	for txInId := range tx.Inputs {
		previousUtxo := senderWallet.Utxos[string(txInId)]
		senderWallet.Balance -= previousUtxo.Amount
		delete(senderWallet.Utxos, string(txInId))

		// if this wallet is the sender then update the private state as well
		if thisIsSender {
			n.Wallet.Balance -= previousUtxo.Amount
			delete(n.Wallet.Utxos, string(txInId))
		}
	}

	for _, txOut := range tx.Outputs {
		receiverWallet := n.Ring[bck.PubKeyToPem(txOut.Owner)].WInfo
		receiverWallet.Balance += txOut.Amount // increase receiver's balance
		receiverWallet.Utxos[txOut.Id] = txOut // add new txOut to receiver's utxos
		// if this wallet is the receiver then update the private state as well
		if thisIsReceiver {
			n.Wallet.Balance += txOut.Amount
			n.Wallet.Utxos[txOut.Id] = txOut
		}
	}

	return nil

}

/*
func (n *Node) BroadcastTx(tx *bck.Transaction) error {
}
*/

//* BLOCK
// TODO implement this
func (n *Node) MineBlock(block *bck.Block) error {
	return nil
}

// currhash is correct && previous_hash is actually the hash of the previous block
/*
func (n *Node) ApplyBlock(block *bck.Block) error {
}
func (n *Node) BroadcastBlock(block *bck.Block) error {
}
*/
//might need to check if nonce is correct
func (n *Node) IsValidBlock(block *bck.Block) bool {
	lastBlockHash := n.getLastBlock().CurrentHash
	return string(block.CurrentHash) == string(lastBlockHash)
}

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
func (n *Node) ApplyChain(blocks []*bck.Block) error {
	return nil
}

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
