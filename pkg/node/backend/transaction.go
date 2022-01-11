package backend

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

type InputId string
type InputSetTy map[InputId]struct{}

func (set InputSetTy) Add(inputId string) {
	set[InputId(inputId)] = struct{}{}
}
func (set InputSetTy) Has(inputId string) bool {
	_, ok := set[InputId(inputId)]
	return ok
}
func (set InputSetTy) Remove(inputId string) {
	delete(set, InputId(inputId))
}

// func NewTxIn() TxIn {
// }

type TxOut struct {
	Id            string
	TransactionId string
	Amount        int
	Owner         *rsa.PublicKey `json:"-"`
}

func NewTxOut(owner *rsa.PublicKey, amount int) *TxOut {
	return &TxOut{
		Amount: amount,
		Owner:  owner,
	}
}
func (txout *TxOut) ComputeAndFillHash() {
	type txOutJson struct {
		Id            string `json:"id"`
		TransactionId string `json:"transactionId"`
		Amount        int    `json:"amount"`
		Owner         string `json:"owner"`
	}
	bytes, err := json.Marshal(txOutJson{
		Id:            txout.Id,
		TransactionId: txout.TransactionId,
		Amount:        txout.Amount,
		Owner:         PubKeyToPem(txout.Owner),
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	rand.Seed(time.Now().UnixNano())
	h := sha256.New()
	h.Write(bytes)
	b := make([]byte, 32)
	rand.Read(b[:])
	h.Write(b[:])
	txout.Id = string(h.Sum(nil))
}

type Transaction struct {
	SenderAddress   *rsa.PublicKey
	ReceiverAddress *rsa.PublicKey
	Amount          int
	Id              []byte
	Inputs          InputSetTy
	Outputs         map[string]*TxOut
	Signature       []byte
}
type transactionJson struct {
	Id              string   `json:"id"`
	SenderAddress   string   `json:"senderAddress"`
	ReceiverAddress string   `json:"receiverAddress"`
	Amount          int      `json:"amount"`
	Inputs          []string `json:"inputs"`
	Outputs         []*TxOut `json:"outputs"`
	Signature       string   `json:"signature"`
}

func (tx *Transaction) MarshalJSON() ([]byte, error) {
	txIns := make([]string, len(tx.Inputs))
	txOuts := make([]*TxOut, len(tx.Outputs))
	for txInId := range tx.Inputs {
		txIns = append(txIns, string(txInId))
	}
	for _, txOut := range tx.Outputs {
		txOuts = append(txOuts, txOut)
	}
	return json.Marshal(transactionJson{
		Id:              HexEncodeByteSlice(tx.Id),
		SenderAddress:   PubKeyToPem(tx.SenderAddress),
		ReceiverAddress: PubKeyToPem(tx.ReceiverAddress),
		Amount:          tx.Amount,
		Inputs:          txIns,
		Outputs:         txOuts,
		Signature:       HexEncodeByteSlice(tx.Signature),
	})
}
func (tx *Transaction) UnmarshalJSON(b []byte) error {
	var txJson transactionJson
	err := json.Unmarshal(b, &txJson)
	if err != nil {
		return err
	}
	txIns := make(InputSetTy, len(txJson.Inputs))
	txOuts := make(map[string]*TxOut, len(txJson.Outputs))
	for _, txInId := range txJson.Inputs {
		txIns.Add(txInId)
	}
	for _, txOut := range txJson.Outputs {
		txOuts[txOut.Id] = txOut
	}

	tx.Id = HexDecodeByteSlice(txJson.Id)
	tx.SenderAddress = PubKeyFromPem(txJson.SenderAddress)
	tx.ReceiverAddress = PubKeyFromPem(txJson.ReceiverAddress)
	tx.Amount = txJson.Amount
	tx.Inputs = txIns
	tx.Outputs = txOuts
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
		Inputs:          InputSetTy{},
		Outputs:         map[string]*TxOut{},
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
