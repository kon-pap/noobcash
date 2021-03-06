package node

import (
	"container/list"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	bck "github.com/kon-pap/noobcash/pkg/node/backend"
)

// Integer wrapped in a mutex.
type MuInt struct {
	Mu    sync.Mutex
	Value int
}

// Doubly linked list used as transaction queue
// wraps a mutex to facilitate multi-threaded access
type TxQueue struct {
	mu    sync.Mutex
	queue *list.List
}

func NewTxQueue() *TxQueue {
	return &TxQueue{
		queue: list.New(),
	}
}

func (tq *TxQueue) enqueue(tx *bck.Transaction) {
	tq.queue.PushBack(tx)
}

// Thread-safe enqueue
func (tq *TxQueue) Enqueue(tx *bck.Transaction) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	tq.enqueue(tx)
}

// Thread-safe EnqueueMany
//
// Extra effort made to lock for minimum possible time
func (tq *TxQueue) EnqueueMany(txs []*bck.Transaction) {
	extensionList := list.New()
	for _, tx := range txs {
		extensionList.PushBack(tx)
	}

	tq.mu.Lock()
	defer tq.mu.Unlock()
	//TODO(ORF): Consider PushFrontList to give higher priority to the re-inserted txs
	tq.queue.PushBackList(extensionList)
}

func (tq *TxQueue) dequeue() *bck.Transaction {
	e := tq.queue.Front()
	if e == nil {
		return nil
	}
	tq.queue.Remove(e)
	return e.Value.(*bck.Transaction)
}

// Thread-safe dequeue
//
// returns nil if queue is empty
func (tq *TxQueue) Dequeue() *bck.Transaction {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	return tq.dequeue()
}

// Thread-safe DequeueMany
//
// returns nil if queueLen < n
func (tq *TxQueue) DequeueMany(n int) []*bck.Transaction {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.queue.Len() < n {
		return nil
	}
	txs := make([]*bck.Transaction, 0, n)
	for i := 0; i < n; i++ {
		tx := tq.dequeue()
		if tx == nil {
			panic("TxQueue.DequeueMany: tx == nil, but queue.Len() >= n")
		}
		txs = append(txs, tx)
	}
	return txs
}

func (tq *TxQueue) DequeueManyByValue(txIdsToLookFor stringSet) int {
	var queueElemsToRemove []*list.Element

	tq.mu.Lock()
	defer tq.mu.Unlock()
	for e := tq.queue.Front(); e != nil; e = e.Next() {
		tx := e.Value.(*bck.Transaction)
		if txIdsToLookFor.ContainsByteSlice(tx.Id) {
			queueElemsToRemove = append(queueElemsToRemove, e)
		}
	}
	removeCnt := len(queueElemsToRemove)
	for _, elem := range queueElemsToRemove {
		tq.queue.Remove(elem)
	}
	return removeCnt
}

func (tq *TxQueue) Len() int {
	return tq.queue.Len()
}

// Helper func that extracts the complete body from the result of an
// http client call
func GetResponseBody(resp *http.Response, err error) (string, error) {
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error %d, with body '%s'", resp.StatusCode, string(body))
	}
	return string(body), nil
}

type Job func()

// Utility struct to setup, start, and wait for jobs to finish
type JobGroup struct {
	wg   sync.WaitGroup
	jobs []Job
}

func (jg *JobGroup) Add(job Job) {
	jg.jobs = append(jg.jobs, job)
	jg.wg.Add(1)
}
func (jg *JobGroup) Run() {
	for _, job := range jg.jobs {
		go job()
	}
}

func (jg *JobGroup) RunAndWait() {
	for _, job := range jg.jobs {
		go func(job Job) {
			defer jg.wg.Done()
			job()
		}(job)
	}
	jg.wg.Wait()
}

type stringSet map[string]struct{}

func (ss stringSet) Add(s string) {
	ss[s] = struct{}{}
}
func (ss stringSet) Contains(s string) bool {
	_, ok := ss[s]
	return ok
}

func (ss stringSet) AddByteSlice(b []byte) {
	ss[bck.HexEncodeByteSlice(b)] = struct{}{}
}
func (ss stringSet) ContainsByteSlice(b []byte) bool {
	_, ok := ss[bck.HexEncodeByteSlice(b)]
	return ok
}
