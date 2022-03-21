package node

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	bck "github.com/kon-pap/noobcash/pkg/node/backend"
	"golang.org/x/sync/semaphore"
)

const checkTxCountIntervalSeconds = 5

var BootstrapHostname string

type Node struct {
	Id     int
	Chain  []*bck.Block
	Wallet *bck.Wallet
	Ring   map[string]*NodeInfo

	pendingTxs     *TxQueue
	incBlockChan   chan *bck.Block // send over the block received from the network
	minedBlockChan chan *bck.Block // send over the block mined by this node
	stopMiningChan chan struct{}   // send over the block received to stop mining and handle leftover transactions

	semaCurrentlyMining *semaphore.Weighted // semaphore supports TryAcquire()
	muChainLock         sync.Mutex          // locks when altering the chain
	muRingLock          sync.Mutex          // locks to grab the specific nodeInfo locks without risking deadlocks

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
		Id:     -1,
		Chain:  []*bck.Block{},
		Wallet: w,

		pendingTxs:     NewTxQueue(),
		incBlockChan:   make(chan *bck.Block, 1),
		minedBlockChan: make(chan *bck.Block, 1),
		stopMiningChan: make(chan struct{}, 1),

		semaCurrentlyMining: semaphore.NewWeighted(1),

		info:    newNodeInfo,
		apiport: apiport,

		Ring: map[string]*NodeInfo{
			bck.PubKeyToPem(&w.PrivKey.PublicKey): newNodeInfo,
		},
	}
	return newNode
}

// Initialize any fields required for the node to act as bootstrap
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
	if err != nil {
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

func (n *Node) ApplyTx(tx *bck.Transaction) error {
	if !n.IsValidTx(tx) {
		return fmt.Errorf("transaction is not valid")
	}
	var nodeAddress string = bck.PubKeyToPem(&n.Wallet.PrivKey.PublicKey)
	var senderAddress string
	var senderWalletInfo *bck.WalletInfo

	//*DONE(ORF): Ensure thread-safety
	//!NOTE(ORF): Could be useless to lock here, since chain lock will most likely always block this beforehand
	n.muRingLock.Lock()
	UnlockTxParticipants := n.LockTxParticipants(tx)
	defer UnlockTxParticipants()
	n.muRingLock.Unlock()

	// Skip this if tx is the genesis transaction
	if tx.SenderAddress != nil {
		senderAddress = bck.PubKeyToPem(tx.SenderAddress)
		senderWalletInfo = n.Ring[senderAddress].WInfo

		for txInId := range tx.Inputs {
			previousUtxo := senderWalletInfo.Utxos[string(txInId)]
			senderWalletInfo.Balance -= previousUtxo.Amount
			delete(senderWalletInfo.Utxos, string(txInId))
		}
	}

	for _, txOut := range tx.Outputs {
		receiverAddress := bck.PubKeyToPem(txOut.Owner)
		receiverWalletInfo := n.Ring[receiverAddress].WInfo

		receiverWalletInfo.Balance += txOut.Amount // increase receiver's balance
		receiverWalletInfo.Utxos[txOut.Id] = txOut // add new txOut to receiver's utxos
		// if this wallet is the receiver then update the private state as well
		if receiverAddress == nodeAddress {
			n.Wallet.Balance += txOut.Amount
			n.Wallet.Utxos[txOut.Id] = txOut
		}
	}

	return nil

}

func (n *Node) LockTxParticipants(tx *bck.Transaction) func() {
	myLockedPubKeys := make(stringSet)
	if tx.SenderAddress != nil {
		senderAddress := bck.PubKeyToPem(tx.SenderAddress)
		n.Ring[senderAddress].Mu.Lock()
		myLockedPubKeys.Add(senderAddress)
	}
	for _, txOut := range tx.Outputs {
		receiverAddress := bck.PubKeyToPem(txOut.Owner)
		if myLockedPubKeys.Contains(receiverAddress) {
			continue
		}
		n.Ring[receiverAddress].Mu.Lock()
		myLockedPubKeys.Add(receiverAddress)
	}
	return func() {
		for pubKey := range myLockedPubKeys {
			n.Ring[pubKey].Mu.Unlock()
		}
	}
}

//*DONE(BILL)
func (n *Node) BroadcastTx(tx *bck.Transaction) error {
	txInSlice := []*bck.Transaction{tx}
	dataInJSON, err := json.Marshal(txInSlice)
	if err != nil {
		return err
	}
	n.TryBroadcastByteSlice(dataInJSON, submitTxsEndpoint)
	return nil
}

//* BLOCK
func (n *Node) IsValidBlock(block *bck.Block) bool {
	// GenesisBlock is valid
	if block.IsGenesis() {
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

func (n *Node) CancelBlock(block *bck.Block) {
	n.pendingTxs.EnqueueMany(block.Transactions)
}

// Should be usually called as a goroutine.
func (n *Node) MineBlock(block *bck.Block) {
	// TODO(PAP): use two locks, one shared with checkTxQueue and one shared with SelectMinedOrIncoming
	if block.IsGenesis() {
		panic("Node.MineBlock() called on genesis block")
	}
	log.Println("Mining block")

	//!NOTE(ORF): Only other possible holder is the block receiver, in which case we should reset mining
	if !n.semaCurrentlyMining.TryAcquire(1) {
		n.CancelBlock(block)
		return
	}
	defer n.semaCurrentlyMining.Release(1)

	dif := strings.Repeat("0", bck.Difficulty)

	rand.Seed(time.Now().UnixNano())
	nonce := make([]byte, 32)

	for {
		//*DONE(ORF): Stop mining if a block is received
		//!NOTE(ORF): May need some more care
		select {
		case <-n.stopMiningChan:
			log.Println("Stopping mining...")
			n.CancelBlock(block)
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

//*DONE(ORF): This should extend the n.Chain appropriately
func (n *Node) ApplyBlock(block *bck.Block) error {
	log.Println("Applying new block with", len(block.Transactions), "transactions")

	n.muChainLock.Lock()
	defer n.muChainLock.Unlock()

	if !n.IsValidBlock(block) {
		return fmt.Errorf("block is not valid")
	}
	for _, tx := range block.Transactions {
		if err := n.ApplyTx(tx); err != nil {
			return err
		}
	}
	n.Chain = append(n.Chain, block)
	block.Index = len(n.Chain) // len will already be inceremented by 1

	log.Println("Block successfully applied")
	return nil
}

//*DONE(BILL)
func (n *Node) BroadcastBlock(block *bck.Block) error {
	tmpBlock := []*bck.Block{block}
	blockInJson, err := json.Marshal(tmpBlock)
	if err != nil {
		return err
	}
	n.TryBroadcastByteSlice(blockInJson, submitBlocksEndpoint)
	return nil
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
//TODO: use RemoveCompletedTxsFromQueue (somewhere)

//TODO(ORF): Endpoint for requesting n blocks (possibly whole chain)
//TODO(ORF): Endpoint for requesting chain size
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
	ticker := time.NewTicker(time.Second * checkTxCountIntervalSeconds)
	capacity := bck.BlockCapacity
	chain := len(n.Chain)
	for range ticker.C {
		// TODO(BIL): Either gradually decrease the required number of txs
		if capacity > 0 {
			capacity--
		} else {
			log.Printf("Tried to decrease capacity below zero")
			//maybe this is an edge case and will never have such a problem
			//if there is a constant problem here we must increase checkTxCountIntervalSeconds
		}
		if len(n.Chain) > chain {
			capacity = bck.BlockCapacity //reset the block capacity if one or more blocks applied
		}
		// TODO(BIL): or split txouts during transaction creation
		//split bigger amounts in smaller i.e 100 -> 20, 20, 20, 20, 10, 5, 5
		// TODO(BIL): or both
		if !n.semaCurrentlyMining.TryAcquire(1) {
			continue
		}
		if txs := n.pendingTxs.DequeueMany(bck.BlockCapacity); txs != nil {
			newBlock := bck.NewBlock(n.getLastBlock().CurrentHash)
			newBlock.AddManyTxs(txs) // error handling unnecessary, newBlock is empty
			n.semaCurrentlyMining.Release(1)
			go n.MineBlock(newBlock)
			continue
		}
		n.semaCurrentlyMining.Release(1)
	}
}

// Restore any txs from almostAcceptedBlock that are not in incomingBlock
func (n *Node) RemoveCompletedTxsFromQueue(incomingBlock *bck.Block) {
	txIdsToLookFor := make(stringSet)
	for _, tx := range incomingBlock.Transactions {
		txIdsToLookFor.AddByteSlice(tx.Id)
	}
	n.pendingTxs.DequeueManyByValue(txIdsToLookFor)
}

// Listens for incoming or mined blocks
//
// Should be called as a goroutine
func (n *Node) SelectMinedOrIncomingBlock() {
	log.Println("Setting up block handler...")
	for {
		select {
		case minedBlock := <-n.minedBlockChan:
			log.Println("Processing mined block...")
			n.ApplyBlock(minedBlock)
			n.BroadcastBlock(minedBlock)
		case incomingBlock := <-n.incBlockChan:
			log.Println("Processing received block...")
			if !n.IsValidBlock(incomingBlock) {
				//!NOTE: handle conflict in applyBlock
			} else {
				if !n.semaCurrentlyMining.TryAcquire(1) { // means it was mining
					n.stopMiningChan <- struct{}{}
					n.semaCurrentlyMining.Acquire(context.Background(), 1) // block until mining has stopped
				}
				//!NOTE(ORF): Here one way or another we hold the semaphore
				n.ApplyBlock(incomingBlock)
				n.RemoveCompletedTxsFromQueue(incomingBlock)
				n.semaCurrentlyMining.Release(1)
			}
		}
	}
}

func (n *Node) BroadcastRingInfo() error {
	var nodes []transferNodeTy
	for pubKey, nInfo := range n.Ring {
		nodes = append(nodes, transferNodeTy{
			Hostname: nInfo.Hostname,
			Port:     nInfo.Port,
			PubKey:   pubKey,
			Id:       nInfo.Id,
		})
	}
	sendContent, err := json.Marshal(nodes)
	if err != nil {
		return err
	}
	replies, err := n.BroadcastByteSlice(sendContent, acceptNodesEndpoint)
	if err != nil {
		return err
	}
	for _, reply := range replies {
		regCnt, err := strconv.Atoi(strings.Split(reply, " ")[1])
		if err != nil {
			return err
		}
		if regCnt != n.nodecnt-1 {
			log.Printf("Fellow node registered  %d nodes, but should have registered %d ", regCnt, n.nodecnt-1)
		} else {
			log.Println("Fellow node replied:", reply)
		}
	}

	return nil
}

func (n *Node) DoInitialBootstrapActions() {
	//*DONE(PAP): Wait for N nodes, broadcast ring info, wait for N responses,
	//*DONE(PAP): send genesis block and money spreading block(s)
	//!NOTE: Normally mined blocks will be broadcast automatically in the future
	//!NOTE:   so the money-spreading block may need to be "accepted" after genesis broadcast
	//* DONE(PAP): Money-spreading block can be submitted normaly
	log.Printf("Starting setup process for %d nodes\n", n.nodecnt)

	// Broadcast Ring info
	err := n.BroadcastRingInfo()
	if err != nil {
		log.Println("Error broadcasting ring info:", err)
		os.Exit(1)
	}
	log.Println("Ring broadcasted successfully")

	genBlock := bck.CreateGenesisBlock(n.nodecnt, &n.Wallet.PrivKey.PublicKey)
	if genBlock == nil {
		log.Println("Error creating genesis block")
		os.Exit(1)
	}
	n.ApplyBlock(genBlock)
	log.Println("Genesis is in the chain")

	n.BroadcastBlock(genBlock)
	log.Println("Genesis is broadcasted")

	// Setting block capacity to 1
	//! Works because we don't check block capacity in isValidBlock
	previousCapacity := bck.BlockCapacity
	bck.BlockCapacity = 1

	targets := make([]*bck.TxTargetTy, 0, n.nodecnt-1)
	for _, nInfo := range n.Ring {
		if nInfo.Id == n.Id {
			continue
		}
		targets = append(targets, &bck.TxTargetTy{
			Address: nInfo.WInfo.PubKey,
			Amount:  100,
		})
	}
	tx, err := n.Wallet.CreateAndSignMultiTargetTx(targets...)
	if err != nil {
		log.Println("Error creating/signing transaction:", err)
		os.Exit(1)
	}
	err = n.AcceptTx(tx)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	// poll until the money-spreading block is on-chain
	for len(n.Chain) == 1 {
		time.Sleep(time.Second)
	}
	log.Println("Created initial transactions and added to the chain")
	// Resetting block capacity
	bck.BlockCapacity = previousCapacity

	log.Println("Reset block capacity. Game on!")
}

func (n *Node) ConnectToBootstrapJob() {
	log.Println("Connecting to bootstrap...")
	err := n.ConnectToBootstrap()
	if err != nil {
		log.Fatalf("expected an integer as id, got '%s'", err)
	}
	log.Println("Assigned id", n.Id)
}

func (n *Node) Start() error {

	var jg JobGroup

	jg.Add(func() { n.ServeApiForCli(n.apiport) })
	jg.Add(func() { n.ServeApiForNodes(n.info.Port) })
	jg.Add(n.SelectMinedOrIncomingBlock)
	jg.Add(n.CheckTxQueueForMining)

	if !n.IsBootstrap() {
		jg.Add(n.ConnectToBootstrapJob)
	}

	jg.RunAndWait()

	return nil
}
