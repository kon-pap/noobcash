package node

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	nodecnt      int
}

func NewNode(currBlockId, bits int, ip, port, apiport string) *Node {
	w := bck.NewWallet(bits)
	newNodeInfo := NewNodeInfo(-1, ip, port, &w.PrivKey.PublicKey)
	newNode := &Node{
		Id:             -1,
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

func (n *Node) MakeBootstrap(nodecnt int) {
	log.Println("Becoming bootstrap...")
	n.Id = 0
	n.Ring[bck.PubKeyToPem(&n.Wallet.PrivKey.PublicKey)].Id = 0
	n.BsNextNodeId = &MuInt{
		Value: 1,
	}
	n.nodecnt = nodecnt
}

func (n *Node) IsBootstrap() bool {
	return n.Id == 0
}

//* TRANSACTION
func (n *Node) IsValidSig(tx *bck.Transaction) bool {
	// Genesis transaction is valid
	// log.Fatalln(bck.HexDecodeByteSlice())
	if tx.SenderAddress == nil {
		return true
	}
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

	n.PendingTxs.Mu.Lock()
	defer n.PendingTxs.Mu.Unlock()

	n.PendingTxs.Enqueue(tx)
	if n.PendingTxs.Len() >= bck.BlockCapacity {
		newBlock := bck.NewBlock(n.CurrBlockId, n.getLastBlock().CurrentHash)
		txs := n.PendingTxs.DequeueMany(bck.BlockCapacity)
		//!NOTE: Lock may be necessary in block,
		//! it's safe for now, blocked by PendingTxs.Mu
		newBlock.AddManyTxs(txs) // error handling unnecessary, newBlock is empty
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
	stringNodeAddress := bck.PubKeyToPem(&n.Wallet.PrivKey.PublicKey)

	// Skip this if tx is the genesis transaction
	if tx.SenderAddress != nil {
		stringSenderAddress := bck.PubKeyToPem(tx.SenderAddress)
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
func (n *Node) IsValidBlock(block *bck.Block) bool {
	// GenesisBlock is valid
	if n.CurrBlockId == 0 && block.Index == 0 {
		return true
	}
	//check if Hash(nonce + Hash(block)) starts with n zeros
	//same process as mining
	dif := strings.Repeat("0", bck.Difficulty)
	nonce := []byte(block.Nonce)

	h := sha256.New()
	h.Write(block.CurrentHash)
	h.Write(nonce)
	lastBlockHash := n.getLastBlock().CurrentHash
	return string(block.CurrentHash) == string(lastBlockHash) && bck.HexEncodeByteSlice(h.Sum(nil))[:bck.Difficulty] == dif
}

// check block validity
func (n *Node) MineBlock(block *bck.Block) {

	// Mined block is sent to be processed
	//find a number which if we hash with block's hash will start with n 0
	//Hash(nonce + Hash(bck)) starts with n zeros
	//The only way is guess nonce and check if it's ok

	log.Println("Mining block", block.Index)
	dif := strings.Repeat("0", bck.Difficulty)

	rand.Seed(time.Now().UnixNano())
	for {
		h := sha256.New()
		h.Write(block.CurrentHash)
		nonce := make([]byte, 32)
		rand.Read(nonce[:])
		h.Write(nonce[:])
		if bck.HexEncodeByteSlice(h.Sum(nil))[:bck.Difficulty] == dif {
			block.Nonce = string(nonce)
			break
		}
	}

	n.MinedBlockChan <- block
}

func (n *Node) ApplyBlock(block *bck.Block) error {
	if !n.IsValidBlock(block) {
		return fmt.Errorf("block is not valid")
	}
	log.Println("Applying new block with", len(block.Transactions), "transactions")
	for _, tx := range block.Transactions {
		if err := n.ApplyTx(tx); err != nil {
			return err
		}
	}
	log.Println("Block successfully applied")
	return nil
}

/*
// currhash is correct && previous_hash is actually the hash of the previous block
// recheck transaction validity
func (n *Node) IsValidBlock(block *bck.Block) bool {
}
func (n *Node) BroadcastBlock(block *bck.Block) error {
}
*/
//might need to check if nonce is correct

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
	n.Ring[bck.PubKeyToPem(&n.Wallet.PrivKey.PublicKey)].Id = n.Id
	if err != nil {
		return err
	}
	return nil
}

// Listens for incoming or mined blocks
//
// Should be called as a goroutine
func (n *Node) SelectMinedOrIncomingBlock() {
	log.Println("Setting up block handler...")
	for {
		select {
		case minedBlock := <-n.MinedBlockChan:
			//TODO: handle minedBlock
			log.Println("Processing mined block...")
			n.ApplyBlock(minedBlock)
		case incomingBlock := <-n.IncBlockChan:
			//TODO: handle incomingBlock
			log.Println("Processing received block...")
			fmt.Println("Incoming block:", incomingBlock)
		}
	}
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

	var jg JobGroup

	if !n.IsBootstrap() {
		log.Println("Connecting to bootstrap...")
		err := n.ConnectToBootstrap()
		if err != nil {
			return fmt.Errorf("expected an integer as id, got '%s'", err)
		}
		log.Println("Assigned id", n.Id)
	} else {
		genBlock := bck.CreateGenesisBlock(n.nodecnt, &n.Wallet.PrivKey.PublicKey)
		if genBlock == nil {
			return fmt.Errorf("genesis block creation failed")
		}
		n.MineBlock(genBlock)
	}

	jg.Add(n.SelectMinedOrIncomingBlock)
	jg.Add(func() { n.ServeApiForCli(n.apiport) })
	jg.Add(func() { n.ServeApiForNodes(n.info.Port) })

	jg.RunAndWait()

	return nil
}
