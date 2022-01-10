package backend

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// Defines the capacity of transactions inside a block
const blockCapacity int = 10

type Block struct {
	Index        int
	Timestamp    time.Time
	Transactions []*Transaction
	Nonce        string
	CurrentHash  string
	PreviousHash string
}

func NewBlock(index int, prevHash string) *Block {
	return &Block{
		Index:        index,
		Timestamp:    time.Now(),
		PreviousHash: prevHash,
	}
}

// This type will be used to send block to other nodes
type blockJson struct {
	Timestamp    time.Time      `json:"createdTimestamp"`
	Transactions []*Transaction `json:"transactions"`
	Nonce        string         `json:"nonce"`
	CurrentHash  string         `json:"currentHash"`
	PreviousHash string         `json:"previousHash"`
}

func (b *Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(blockJson{
		Timestamp:    b.Timestamp,
		Transactions: b.Transactions,
		Nonce:        b.Nonce,
		CurrentHash:  b.CurrentHash,
		PreviousHash: b.PreviousHash,
	})
}

func (b *Block) UnmarshalJSON(data []byte) error {
	var blockJson blockJson
	err := json.Unmarshal(data, &blockJson)
	if err != nil {
		return err
	}
	b.Timestamp = blockJson.Timestamp
	b.Transactions = blockJson.Transactions
	b.Nonce = blockJson.Nonce
	b.CurrentHash = blockJson.CurrentHash
	b.PreviousHash = blockJson.PreviousHash
	return nil
}

func (b *Block) String() string {
	strBytes, err := (json.Marshal(b))
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return fmt.Sprintf("Block id: %d, %s", b.Index, string(strBytes))
}

func (b *Block) AddTx(tx *Transaction) error {
	if blockCapacity == len(b.Transactions) {
		return errors.New("Block is full")
	}
	b.Transactions = append(b.Transactions, tx)
	return nil
}

func (b *Block) UpdateNonce(nonce string) {
	b.Nonce = nonce
}

// This type will be used to create the currentHash of the block
type blockJsonHash struct {
	Timestamp    time.Time
	Nonce        string
	PreviousHash string
}

func (b *Block) marshalJSONHash() ([]byte, error) {
	return json.Marshal(blockJsonHash{
		Timestamp:    b.Timestamp,
		Nonce:        b.Nonce,
		PreviousHash: b.PreviousHash,
	})
}

func (b *Block) ComputeAndFillHash() {
	txInfoBytes, err := b.marshalJSONHash()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	byteArray := (sha256.Sum256(txInfoBytes))
	b.CurrentHash = string(byteArray[:])
}
