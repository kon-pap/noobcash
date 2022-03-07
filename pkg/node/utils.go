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

type TxQueue struct {
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
	var txs []*bck.Transaction
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
