package node

import (
	"bytes"
	"crypto"
	"crypto/rsa"
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

const checkTxCountIntervalMilliSeconds = 5

var BootstrapHostname string

type Node struct {
	Id          int
	Chain       []*bck.Block
	CurrBlockId int
	Wallet      *bck.Wallet
	Ring        map[string]*NodeInfo

	pendingTxs     *TxQueue
	incBlockChan   chan *bck.Block // send over the block received from the network
	minedBlockChan chan *bck.Block // send over the block mined by this node
	stopMiningChan chan *bck.Block // send over the block received to stop mining and handle leftover transactions

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
		Id:          -1,
		Chain:       []*bck.Block{},
		CurrBlockId: currBlockId,
		Wallet:      w,

		pendingTxs:     NewTxQueue(),
		incBlockChan:   make(chan *bck.Block),
		minedBlockChan: make(chan *bck.Block),
		stopMiningChan: make(chan *bck.Block, 1),

		info:    newNodeInfo,
		apiport: apiport,

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
	if !n.IsValidTx(tx) {
		return fmt.Errorf("transaction is not valid")
	}
	n.pendingTxs.Enqueue(tx)
	return nil
}

//TODO(ORF): Ensure thread-safety
//! note: extra effort was made to facilitate support for multiple receivers per transaction
//! note: checking if enough txs exist, could be done by a goroutine every some time
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
//TODO(BILL)
func (n *Node) BroadcastTx(tx *bck.Transaction) error {
}
*/

//* BLOCK
func (n *Node) IsValidBlock(block *bck.Block) bool {
	// GenesisBlock is valid
	if n.CurrBlockId == 0 && block.Index == 0 {
		return true
	}
	dif := strings.Repeat("0", bck.Difficulty)
	lastBlockHash := bck.HexEncodeByteSlice(n.getLastBlock().CurrentHash)
	thisBlockHash := bck.HexEncodeByteSlice(block.CurrentHash)
	thisBlockPreviousHash := bck.HexEncodeByteSlice(block.PreviousHash)

	return lastBlockHash == thisBlockPreviousHash && // this block's previous block is our current last block
		strings.HasPrefix(thisBlockHash, dif) && // this block's hash starts with the required number of zeros
		bck.HexEncodeByteSlice(block.ComputeHash()) == thisBlockHash // this block's hash is correct
}

func (n *Node) HandleStopMining(incomingBlock, almostMinedBlock *bck.Block) {
	//TODO(ORF): Compare incomingBlock's and almostMinedBlock's transactions, and
	//TODO(ORF): and call enqueueMany for any that were not in incomingBlock
}

func (n *Node) MineBlock(block *bck.Block) {
	//*DONE(ORF): CHANGE this to insert the nonce in the block and hash it again
	log.Println("Mining block", block.Index)
	dif := strings.Repeat("0", bck.Difficulty)

	rand.Seed(time.Now().UnixNano())
	nonce := make([]byte, 32)

	for {
		//*DONE(ORF): Stop mining if a block is received
		select {
		case incomingBlock := <-n.stopMiningChan:
			log.Println("Stopping mining...")
			n.HandleStopMining(incomingBlock, block)
			return
		default: // used to prevent blocking
		}
		rand.Read(nonce[:])
		block.Nonce = bck.HexEncodeByteSlice(nonce)
		block.ComputeAndFillHash()
		if strings.HasPrefix(bck.HexEncodeByteSlice(block.CurrentHash), dif) {
			break
		}
	}
	n.minedBlockChan <- block
}

//TODO(ORF): This should extend the n.Chain appropriately
//TODO(ORF): And update n.CurrBlockId
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
//TODO(BILL)
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
//TODO: Throw away transactions that were submitted in the incoming block.
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

func (n *Node) CheckTxQueueForMining() {
	// return ticker if we need to stop the job sometime (make this into a jobfactory)
	ticker := time.NewTicker(time.Millisecond * checkTxCountIntervalMilliSeconds)
	for range ticker.C {
		if txs := n.pendingTxs.DequeueMany(bck.BlockCapacity); txs != nil {
			newBlock := bck.NewBlock(n.CurrBlockId, n.getLastBlock().CurrentHash)
			newBlock.AddManyTxs(txs) // error handling unnecessary, newBlock is empty
			go n.MineBlock(newBlock)
		}
	}
}

// Listens for incoming or mined blocks
//
// Should be called as a goroutine
func (n *Node) SelectMinedOrIncomingBlock() {
	log.Println("Setting up block handler...")
	for {
		select {
		case minedBlock := <-n.minedBlockChan:
			//TODO: handle minedBlock
			log.Println("Processing mined block...")
			n.ApplyBlock(minedBlock)
		case incomingBlock := <-n.incBlockChan:
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
//TODO(BILL)
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
		//TODO(PAP): Wait for N nodes, broadcast ring info, wait for N responses,
		//TODO(PAP): send genesis block and money spreading block
	} else {
		genBlock := bck.CreateGenesisBlock(n.nodecnt, &n.Wallet.PrivKey.PublicKey)
		if genBlock == nil {
			return fmt.Errorf("genesis block creation failed")
		}
		go n.MineBlock(genBlock)
	}

	jg.Add(func() { n.ServeApiForCli(n.apiport) })
	jg.Add(func() { n.ServeApiForNodes(n.info.Port) })
	jg.Add(n.SelectMinedOrIncomingBlock)
	//*DONE(ORF): Add a job to check for enough txs for a new block
	jg.Add(n.CheckTxQueueForMining)

	jg.RunAndWait()

	return nil
}
