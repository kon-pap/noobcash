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
	Mu    sync.Mutex
	queue *list.List
}

func NewTxQueue() *TxQueue {
	return &TxQueue{
		queue: list.New(),
	}
}

func (tq *TxQueue) Enqueue(tx *bck.Transaction) {
	tq.queue.PushBack(tx)
}

func (tq *TxQueue) Dequeue() *bck.Transaction {
	e := tq.queue.Front()
	if e == nil {
		return nil
	}
	tq.queue.Remove(e)
	return e.Value.(*bck.Transaction)
}

func (tq *TxQueue) DequeueMany(n int) []*bck.Transaction {
	txs := make([]*bck.Transaction, n)
	for i := 0; i < n; i++ {
		tx := tq.Dequeue()
		if tx == nil {
			break
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
