package backend

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
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
	Id              string  `json:"id"`
	SenderAddress   string  `json:"senderAddress"`
	ReceiverAddress string  `json:"receiverAddress"`
	Amount          int     `json:"amount"`
	Inputs          []TxIn  `json:"inputs"`
	Outputs         []TxOut `json:"outputs"`
	Signature       string  `json:"signature"`
}

func (tx *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(transactionJson{
		Id:              HexEncodeByteSlice(tx.Id),
		SenderAddress:   PubKeyToPem(tx.SenderAddress),
		ReceiverAddress: PubKeyToPem(tx.ReceiverAddress),
		Amount:          tx.Amount,
		Inputs:          tx.Inputs,
		Outputs:         tx.Outputs,
		Signature:       HexEncodeByteSlice(tx.Signature),
	})
}
func (tx *Transaction) UnmarshalJSON(b []byte) error {
	var txJson transactionJson
	err := json.Unmarshal(b, &txJson)
	if err != nil {
		return err
	}
	tx.Id = HexDecodeByteSlice(txJson.Id)
	tx.SenderAddress = PubKeyFromPem(txJson.SenderAddress)
	tx.ReceiverAddress = PubKeyFromPem(txJson.ReceiverAddress)
	tx.Amount = txJson.Amount
	tx.Inputs = txJson.Inputs
	tx.Outputs = txJson.Outputs
	tx.Signature = HexDecodeByteSlice(txJson.Signature)
	return nil
}
func (tx *Transaction) String() string {
	strBytes, err := json.Marshal(tx)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return string(strBytes)
}

func NewTransaction(from, to *rsa.PublicKey, amount int) *Transaction {
	return &Transaction{
		SenderAddress:   from,
		ReceiverAddress: to,
		Amount:          amount,
		Inputs:          []TxIn{},
		Outputs:         []TxOut{},
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
func HexEncodeByteSlice(b []byte) string {
	return fmt.Sprintf("%x", b)
}
func HexDecodeByteSlice(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return b
}
