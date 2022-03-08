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
	//! NOTE: MineBlock will fill the block's hash at the end
	//! Assumption: MineBlock will increment the node.CurrBlockId
	if !n.IsValidTx(tx) {
		return fmt.Errorf("transaction is not valid")
	}
	n.PendingTxs.Enqueue(tx)
	if n.PendingTxs.Len() >= bck.BlockCapacity {
		newBlock := bck.NewBlock(n.CurrBlockId, n.getLastBlock().CurrentHash)
		txs := n.PendingTxs.DequeueMany(bck.BlockCapacity)
		newBlock.AddManyTxs(txs) // error handling not needed here
		go n.MineBlock(newBlock)
	}
	return nil
}

//TODO: Ensure thread-safety
//! note: extra effort was made to facilitate support for multiple receivers per transaction
func (n *Node) ApplyTx(tx *bck.Transaction) error {
	if !n.IsValidTx(tx) {
		return fmt.Errorf("transaction is not valid")
	}
	stringSenderAddress := bck.PubKeyToPem(tx.SenderAddress)
	stringNodeAddress := bck.PubKeyToPem(&n.Wallet.PrivKey.PublicKey)

	senderWallet := n.Ring[stringSenderAddress].WInfo
	thisIsSender := stringSenderAddress == stringNodeAddress

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
		stringReceiverAddress := bck.PubKeyToPem(txOut.Owner)
		receiverWallet := n.Ring[stringReceiverAddress].WInfo

		receiverWallet.Balance += txOut.Amount // increase receiver's balance
		receiverWallet.Utxos[txOut.Id] = txOut // add new txOut to receiver's utxos
		// if this wallet is the receiver then update the private state as well
		if stringReceiverAddress == stringNodeAddress {
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

/*
func (n *Node) ResolveConflict(block *bck.Block) error {
}
*/

//* RING
func (n *Node) ConnectToBootstrap() error {
	sendContent, err := json.Marshal(bootstrapNodeTy{
		Hostname: n.info.Hostname,
		Port:     n.info.Port,
		PubKey:   bck.PubKeyToPem(&n.Wallet.PrivKey.PublicKey),
	})
	if err != nil {
		return err
	}
	sendBody := bytes.NewBuffer(sendContent)
	body, err := GetResponseBody(
		http.DefaultClient.Post(
			fmt.Sprintf("http://%s/bootstrap-node", BootstrapHostname),
			"application/json",
			sendBody,
		),
	)
	if err != nil {
		return err
	}
	n.Id, err = strconv.Atoi(string(body))
	if err != nil {
		return err
	}
	return nil
}

/*
// 1. Send Wallet pubkey
// 2. Receive node id
// 3. Wait for info of all other nodes

// Send IP, port, pubkey of all nodes
func (n *Node) BroadcastRingInfo() error {
}
*/

func (n *Node) Start() error {
	// Start API server
	go n.ServeApiForCli(n.apiport)
	go n.ServeApiForNodes(n.info.Port)

	if !n.IsBootstrap() {
		err := n.ConnectToBootstrap()
		if err != nil {
			return err
		}
	}

	go n.SelectMinedOrIncomingBlock()

	return nil
}
