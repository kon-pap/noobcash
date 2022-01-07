package backend

import (
	"time"
)

const blockCapacity int = 10

type Block struct {
	index        int
	timestamp    time.Time
	transactions *[]Transaction
	nonce        string
	currentHash  string
	previousHash string
}

// func createGenesisBlock(n int) Block {

// }
