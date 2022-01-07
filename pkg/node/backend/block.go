package backend

import (
	"time"
)

const blockCapacity int = 10

type Block struct {
	index        int
	timestamp    time.Time
	transactions []*Transaction
	nonce        string
	currentHash  string
	previousHash string
}

func NewBlock() *Block {
	return &Block{}
}

func (b *Block) AddTx(tx *Transaction) {
	b.transactions = append(b.transactions, tx)
}

// func createGenesisBlock(n int) Block {
// }
