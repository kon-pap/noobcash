package backend

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
)

type TxIn struct {
	PreviousOutputId string
}

// func NewTxIn() TxIn {
// }

type TxOut struct {
	Id            string
	TransactionId string
	Amount        int
	Owner         *rsa.PublicKey `json:"-"`
}

// func NewTxOut() TxOut {
// }

type Transaction struct {
	SenderAddress   *rsa.PublicKey
	ReceiverAddress *rsa.PublicKey
	Amount          int
	Id              []byte
	Inputs          []TxIn
	Outputs         []TxOut
	Signature       []byte
}
type transactionJson struct {
	SenderAddress   string
	ReceiverAddress string
	Amount          int
	Inputs          []TxIn
	Outputs         []TxOut
}

func (tx *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(transactionJson{
		SenderAddress:   PubKeyToPem(tx.SenderAddress),
		ReceiverAddress: PubKeyToPem(tx.ReceiverAddress),
		Amount:          tx.Amount,
		Inputs:          tx.Inputs,
		Outputs:         tx.Outputs,
	})
}

func NewTransaction(from, to *rsa.PublicKey, amount int, inputs []TxIn, outputs []TxOut) *Transaction {
	return &Transaction{
		SenderAddress:   from,
		ReceiverAddress: to,
		Amount:          amount,
		Inputs:          inputs,
		Outputs:         outputs,
	}
}

func (tx *Transaction) ComputeAndFillHash() {
	txInfoBytes, err := json.Marshal(tx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	byteArray := (sha256.Sum256(txInfoBytes))
	tx.Id = byteArray[:]
}
func (tx *Transaction) GetHashStr() string {
	return fmt.Sprintf("%x", tx.Id)
}
