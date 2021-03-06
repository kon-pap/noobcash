package node

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/json"
	"errors"
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
var (
	AverageBlockTime = int64(0)  // nanoseconds
	LastBlockTime    = int64(-1) // nanoseconds
	TotalBlockTimes  = int64(0)

	TxThroughputFlag      = false
	TxThroughputStartTime = time.Now()
	AllTxsDuration        = time.Duration(0)
)

type Node struct {
	Id     int
	Chain  []*bck.Block
	Wallet *bck.Wallet
	Ring   map[string]*NodeInfo

	pendingTxs     *TxQueue
	incBlockChan   chan *bck.Block // send over the block received from the network
	minedBlockChan chan *bck.Block // send over the block mined by this node
	stopMiningChan chan struct{}   // send over the block received to stop mining and handle leftover transactions

	semaCurrentlyMining    *semaphore.Weighted // semaphore supports TryAcquire()
	semaCurrentlyMiningInc *semaphore.Weighted
	muChainLock            sync.Mutex // locks when altering the chain
	muRingLock             sync.Mutex // locks to grab the specific nodeInfo locks without risking deadlocks

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

		semaCurrentlyMining:    semaphore.NewWeighted(1),
		semaCurrentlyMiningInc: semaphore.NewWeighted(1),

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
		for _, txIn := range tx.Inputs {
			if !senderNode.WInfo.Utxos.Has(txIn) {
				log.Println("Wallet", senderNode.Id, "does not have UTXO", txIn.Id)
				return false
			}
		}
		return true
	}()
}
func (n *Node) AcceptTx(tx *bck.Transaction) error {
	if !n.IsValidTx(tx) {
		log.Println("AcceptTx: Invalid transaction")
		return fmt.Errorf("transaction is not valid")
	}
	if !TxThroughputFlag {
		TxThroughputFlag = true
		TxThroughputStartTime = time.Now()
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

		for _, txIn := range tx.Inputs {
			previousUtxo := senderWalletInfo.Utxos[txIn.Id]
			senderWalletInfo.Balance -= previousUtxo.Amount
			senderWalletInfo.Utxos.Remove(txIn)
		}
		// TODO(ORF): Finalize removal by removing the UTXOS from the .Wallet.UTXOS as well.
		// TODO(ORF): and remove them from the reserved UTXOS too.
	}

	for _, txOut := range tx.Outputs {
		receiverAddress := bck.PubKeyToPem(txOut.Owner)
		receiverWalletInfo := n.Ring[receiverAddress].WInfo

		receiverWalletInfo.Balance += txOut.Amount // increase receiver's balance
		receiverWalletInfo.Utxos.Add(txOut)        // add new UTXO to receiver's UTXOs
		// if this wallet is the receiver then update the private state as well
		if receiverAddress == nodeAddress {
			n.Wallet.Balance += txOut.Amount
			n.Wallet.Utxos.Add(txOut)
		}
	}

	AllTxsDuration = time.Since(TxThroughputStartTime)
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

var (
	chainErr         = errors.New("block is not valid for the chain")
	incorrectMineErr = errors.New("block hash does not fulfill the difficulty requirement")
	incorrectHashErr = errors.New("block hash does not equal the provided hash")
)

//* BLOCK
func (n *Node) IsValidBlock(block *bck.Block) (err error) {
	// GenesisBlock is valid
	if block.IsGenesis() {
		return
	}
	dif := strings.Repeat("0", bck.Difficulty)
	lastBlockHash := bck.HexEncodeByteSlice(n.getLastBlock().CurrentHash)
	thisBlockHash := bck.HexEncodeByteSlice(block.CurrentHash)
	thisBlockPreviousHash := bck.HexEncodeByteSlice(block.PreviousHash)

	if lastBlockHash != thisBlockPreviousHash {
		err = chainErr
	} else if !strings.HasPrefix(thisBlockHash, dif) {
		err = incorrectMineErr
	} else if bck.HexEncodeByteSlice(block.ComputeHash()) != thisBlockHash {
		err = incorrectHashErr
	}
	return
}

func (n *Node) CancelNotAppliedBlock(block *bck.Block) {
	n.pendingTxs.EnqueueMany(block.Transactions)
}

func (n *Node) fixBlockTime(start time.Time) {
	timediff := int64(time.Since(start))
	LastBlockTime = timediff
	TotalBlockTimes += timediff
	AverageBlockTime = TotalBlockTimes / int64(len(n.Chain))
}

// Should be usually called as a goroutine.
func (n *Node) MineBlock(block *bck.Block) {
	//*DONE(PAP): use two locks, one shared with checkTxQueue and one shared with SelectMinedOrIncoming
	if block.IsGenesis() {
		panic("Node.MineBlock() called on genesis block")
	}
	log.Println("Mining block")

	start := time.Now()

	//!NOTE Block till you get the lock that allows you to mine
	//!NOTE No other block is being mined
	//!NOTE No new block is being created. Possible if CheckTxQueueForMining enough pendingTxs &
	//!NOTE for a random reason gets the lock first
	n.semaCurrentlyMining.Acquire(context.Background(), 1)
	defer n.semaCurrentlyMining.Release(1)

	//!NOTE semaCurrentlyMiningInc can only be taken by incomingBlock that gets applied
	//!NOTE In this case the already mined block has higher priority so we cancel mining
	if !n.semaCurrentlyMiningInc.TryAcquire(1) {
		n.CancelNotAppliedBlock(block)
		return
	}
	defer n.semaCurrentlyMiningInc.Release(1)

	dif := strings.Repeat("0", bck.Difficulty)

	rand.Seed(time.Now().UnixNano())
	nonce := make([]byte, 32)

	for {
		//*DONE(ORF): Stop mining if a block is received
		//!NOTE(ORF): May need some more care
		select {
		case <-n.stopMiningChan:
			log.Println("Stopping mining...")
			n.CancelNotAppliedBlock(block)
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
	n.fixBlockTime(start)
	n.minedBlockChan <- block
}

//*DONE(ORF): This should extend the n.Chain appropriately
func (n *Node) ApplyBlock(block *bck.Block) error {
	log.Println("Applying new block with", len(block.Transactions), "transactions")

	n.muChainLock.Lock()
	defer n.muChainLock.Unlock()

	if err := n.IsValidBlock(block); err != nil {
		return err
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

func (n *Node) getNodeInfoById(id int) *NodeInfo {
	for _, nInfo := range n.Ring {
		if nInfo.Id == id {
			return nInfo
		}
	}
	panic("Node.getNodeInfoById() called with invalid id")
}

func (n *Node) RevertTx(tx *bck.Transaction) {
	log.Println("Reverting transaction...")
	n.muRingLock.Lock()
	UnlockTxParticipants := n.LockTxParticipants(tx)
	defer UnlockTxParticipants()
	n.muRingLock.Unlock()

	if tx.SenderAddress != nil {
		senderAddress := bck.PubKeyToPem(tx.SenderAddress)
		senderWalletInfo := n.Ring[senderAddress].WInfo

		for _, txIn := range tx.Inputs {
			senderWalletInfo.Balance += txIn.Amount
			senderWalletInfo.Utxos.Add(txIn)
		}
	}
	for _, txOut := range tx.Outputs {
		receiverAddress := bck.PubKeyToPem(txOut.Owner)
		receiverWalletInfo := n.Ring[receiverAddress].WInfo

		// TODO(ORF): This invariant (I think) is already certain to be true
		if !receiverWalletInfo.Utxos.Has(txOut) {
			panic("RevertTx: tried to remove utxo that did not exist in wallet info")
		}
		receiverWalletInfo.Utxos.Remove(txOut)
		receiverWalletInfo.Balance -= txOut.Amount

		if receiverAddress == bck.PubKeyToPem(&n.Wallet.PrivKey.PublicKey) {
			// TODO(ORF): Remove the UTXOS from the reserved UTXOs map, and now this invariant should also be true here
			if !n.Wallet.Utxos.Has(txOut) {
				panic("RevertTx: tried to remove utxo that did not exist in wallet")
			}
			n.Wallet.Utxos.Remove(txOut)
			n.Wallet.Balance -= txOut.Amount
		}
	}
	return
}

func (n *Node) RevertBlock() {
	log.Println("Reverting block...")
	if len(n.Chain) == 0 {
		return
	}
	blockToRemove := n.getLastBlock()
	log.Println("Trying to revert", len(blockToRemove.Transactions), "transactions")
	for _, tx := range blockToRemove.Transactions {
		n.RevertTx(tx)
	}
	n.Chain = n.Chain[:len(n.Chain)-1]
	n.pendingTxs.EnqueueMany(blockToRemove.Transactions)
}

var (
	deeperConflictErr = errors.New("deeper conflict")
)

//*DONE: use RemoveCompletedTxsFromQueue (somewhere)
//*DONE(ORF): Endpoint for requesting n blocks (possibly whole chain)
//*DONE(ORF): Endpoint for requesting chain size
func (n *Node) ResolveConflict() error {
	log.Println("Resolving Conflict...")
	max_len := len(n.Chain)
	max_id := n.Id
	// TODO: Grab locks

	responses, _ := n.BroadcastByteSlice([]byte{}, chainLengthEndpoint)

	//?DEBUG
	log.Println(responses)

	for id, res := range responses {
		if id == n.Id {
			continue
		}

		len, err := strconv.Atoi(string(res))
		if err != nil {
			return err
		}

		if len > max_len {
			max_len = len
			max_id = id
		} else if len == max_len && id > max_id {
			max_id = id

		}
	}
	if max_id == n.Id {
		return nil
	}

	resolverNodeInfo := n.getNodeInfoById(max_id)

	//?DEBUG
	log.Println("Resolving conflict. Current chainlen:", len(n.Chain), "resolver chainlen:", max_len)

	endpoint := fmt.Sprintf("/chain-tail/%d", max_len-len(n.Chain)+1)
	res, err := n.SendByteSlice([]byte{}, resolverNodeInfo.Hostname, resolverNodeInfo.Port, endpointTy(endpoint))
	if err != nil {
		return err
	}

	var blocks []*bck.Block
	err = json.Unmarshal([]byte(res), &blocks)
	if err != nil {
		return err
	}

	//?DEBUG
	log.Println("Resolving conflict. Received", len(blocks), "blocks")

	//*DONE: Replace last Block of the chain and replace it with blocks
	//! Note: It assumes that the block before the one we remove is correct.
	//! Note: Esentially assume max 1 block branches
	n.RevertBlock()
	for _, block := range blocks {
		//!NOTE: other way would be to `n.incBlockChan <- block`
		if err := n.ApplyBlock(block); err != nil && err == chainErr {
			return deeperConflictErr
		}
		n.RemoveCompletedTxsFromQueue(block)
		//!NOTE(ORF): This loop SHOULD be able to fail only at the first iteration.
	}
	return nil
}

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
	chain := len(n.Chain)
	wait := 0
	for range ticker.C {
		wait++
		//* DONE(BIL): or split txouts during transaction creation
		//split bigger amounts in smaller i.e 100 -> 20, 20, 20, 20, 10, 5, 5
		//* DONE(BIL): or both
		if !n.semaCurrentlyMining.TryAcquire(1) {
			continue
		}
		if txs := n.pendingTxs.DequeueMany(bck.TmpBlockCapacity); txs != nil {
			newBlock := bck.NewBlock(n.getLastBlock().CurrentHash)
			newBlock.AddManyTxs(txs) // error handling unnecessary, newBlock is empty
			n.semaCurrentlyMining.Release(1)
			go n.MineBlock(newBlock)
			wait = 0
			continue
		} else if wait > 3 && n.pendingTxs.Len() != 0 {
			//* DONE(BIL): Either gradually decrease the required number of txs
			if bck.TmpBlockCapacity > 1 {
				bck.TmpBlockCapacity--
				log.Println("Temporarily decreasing block capacity to: ", bck.TmpBlockCapacity)
			}
			if len(n.Chain) > chain {
				bck.TmpBlockCapacity = bck.BlockCapacity //reset the block capacity if one or more blocks applied
				chain = len(n.Chain)
			}
			wait = 0
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

func (n *Node) tryApplyBlockOrResolve(block *bck.Block) {
	if err := n.ApplyBlock(block); err != nil && err == chainErr {
		for conflictDepth := 1; n.ResolveConflict() != nil; conflictDepth++ {
			log.Println("Deeper conflict found, retrying with depth:", conflictDepth)
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
			log.Println("Processing mined block...")
			// n.semaCurrentlyMining.Acquire(context.Background(), 1)
			n.tryApplyBlockOrResolve(minedBlock)
			n.BroadcastBlock(minedBlock)
		case incomingBlock := <-n.incBlockChan:
			log.Println("Processing received block...")
			if !n.semaCurrentlyMiningInc.TryAcquire(1) { // means it was mining
				n.stopMiningChan <- struct{}{}
				n.semaCurrentlyMiningInc.Acquire(context.Background(), 1) // block until mining has stopped
			}
			//!NOTE(ORF): Here one way or another we hold the semaphore
			n.tryApplyBlockOrResolve(incomingBlock)
			n.RemoveCompletedTxsFromQueue(incomingBlock)
			n.semaCurrentlyMiningInc.Release(1)
			// }
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
	for id, reply := range replies {
		if id == n.Id {
			continue
		}
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
	// previousCapacity := bck.TmpBlockCapacity
	bck.TmpBlockCapacity = 1

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
	bck.TmpBlockCapacity = bck.BlockCapacity

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
