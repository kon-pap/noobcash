package backend

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

// Defines the capacity of transactions inside a block
var BlockCapacity int // some basic defaults set to fallback on
var TmpBlockCapacity int

var Difficulty int = 1

type Block struct {
	Index        int
	Timestamp    time.Time
	Transactions []*Transaction
	Nonce        string
	CurrentHash  []byte
	PreviousHash []byte
}

func NewBlock(prevHash []byte) *Block {
	return &Block{
		Index:        -1,
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
		CurrentHash:  HexEncodeByteSlice(b.CurrentHash),
		PreviousHash: HexEncodeByteSlice(b.PreviousHash),
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
	b.CurrentHash = HexDecodeByteSlice(blockJson.CurrentHash)
	b.PreviousHash = HexDecodeByteSlice(blockJson.PreviousHash)
	return nil
}

func (b *Block) String() string {
	strBytes, err := (json.Marshal(b))
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return string(strBytes)
}

func (b *Block) AddTx(tx *Transaction) error {
	if b.IsFull() {
		return errors.New("Block is full")
	}
	b.Transactions = append(b.Transactions, tx)
	return nil
}

func (b *Block) AddManyTxs(txs []*Transaction) error {
	if len(b.Transactions)+len(txs) > BlockCapacity {
		return fmt.Errorf("Block can only fit %d more transactions", BlockCapacity-len(b.Transactions))
	}
	b.Transactions = append(b.Transactions, txs...)
	return nil
}

func (b *Block) IsFull() bool {
	return BlockCapacity == len(b.Transactions)
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
		PreviousHash: HexEncodeByteSlice(b.PreviousHash),
	})
}

func (b *Block) ComputeHash() []byte {
	txInfoBytes, err := b.marshalJSONHash()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	byteArray := (sha256.Sum256(txInfoBytes))
	return byteArray[:]
}
func (b *Block) ComputeAndFillHash() {
	b.CurrentHash = b.ComputeHash()
}

func CreateGenesisBlock(n int, pubKey *rsa.PublicKey) *Block {
	initTx := NewGenesisTransaction(pubKey, n*100)
	b := &Block{
		Index:        0,
		Timestamp:    time.Now(),
		Nonce:        "00",
		PreviousHash: []byte("1"),
	}
	if err := b.AddTx(initTx); err != nil {
		log.Println(err)
		return nil
	}
	b.ComputeAndFillHash()
	return b
}

func (b *Block) IsGenesis() bool {
	return b.Nonce == "00" &&
		string(b.PreviousHash) == "1" &&
		len(b.Transactions) == 1 &&
		b.Transactions[0].IsGenesis() &&
		b.Index == 0 &&
		b.Nonce == "00"
}
