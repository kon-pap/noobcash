package node

import (
	"container/list"
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

// Thread-safe enqueue
func (tq *TxQueue) Enqueue(tx *bck.Transaction) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tq.queue.PushBack(tx)
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
	tq.queue.PushBackList(extensionList)
}

// Thread-safe dequeue
//
// returns nil if queue is empty
func (tq *TxQueue) Dequeue() *bck.Transaction {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	e := tq.queue.Front()
	if e == nil {
		return nil
	}
	tq.queue.Remove(e)
	return e.Value.(*bck.Transaction)
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
	txs := make([]*bck.Transaction, n)
	for i := 0; i < n; i++ {
		tx := tq.Dequeue()
		if tx == nil {
			panic("TxQueue.DequeueMany: tx == nil, but queue.Len() >= n")
		}
		txs = append(txs, tx)
	}
	return txs
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
