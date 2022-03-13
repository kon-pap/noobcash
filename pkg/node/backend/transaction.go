package backend

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

type TxOut struct {
	Id            string         `json:"id"`
	TransactionId string         `json:"transactionId"`
	Amount        int            `json:"amount"`
	Owner         *rsa.PublicKey `json:"owner"`
}

func NewTxOut(owner *rsa.PublicKey, amount int) *TxOut {
	return &TxOut{
		Amount: amount,
		Owner:  owner,
	}
}

type txOutJson struct {
	Id            string `json:"id"`
	TransactionId string `json:"transactionId"`
	Amount        int    `json:"amount"`
	Owner         string `json:"owner"`
}

func (txout *TxOut) MarshalJSON() ([]byte, error) {
	return json.Marshal(txOutJson{
		Id:            txout.Id,
		TransactionId: txout.TransactionId,
		Amount:        txout.Amount,
		Owner:         PubKeyToPem(txout.Owner),
	})
}
func (txout *TxOut) UnmarshalJSON(b []byte) error {
	var tmpTxOut txOutJson
	err := json.Unmarshal(b, &tmpTxOut)
	if err != nil {
		return err
	}
	txout.Id = tmpTxOut.Id
	txout.TransactionId = tmpTxOut.TransactionId
	txout.Amount = tmpTxOut.Amount
	txout.Owner = PubKeyFromPem(tmpTxOut.Owner)
	return nil
}

func (txout *TxOut) ComputeAndFillHash() {
	bytes, err := json.Marshal(txout)
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
	txout.Id = HexEncodeByteSlice(h.Sum(nil))
}

type Transaction struct {
	SenderAddress *rsa.PublicKey
	// ReceiverAddress *rsa.PublicKey
	Amount    int
	Id        []byte
	Inputs    InputSetTy
	Outputs   map[string]*TxOut
	Signature []byte
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
	i := 0
	for txInId := range tx.Inputs {
		txIns[i] = string(txInId)
		i++
	}
	i = 0
	for _, txOut := range tx.Outputs {
		txOuts[i] = txOut
		i++
	}
	return json.Marshal(transactionJson{
		Id:            HexEncodeByteSlice(tx.Id),
		SenderAddress: PubKeyToPem(tx.SenderAddress),
		// ReceiverAddress: PubKeyToPem(tx.ReceiverAddress),
		Amount:    tx.Amount,
		Inputs:    txIns,
		Outputs:   txOuts,
		Signature: HexEncodeByteSlice(tx.Signature),
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
	// tx.ReceiverAddress = PubKeyFromPem(txJson.ReceiverAddress)
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

func NewTransaction(from *rsa.PublicKey, amount int) *Transaction {
	return &Transaction{
		SenderAddress: from,
		// ReceiverAddress: to,
		Amount:  amount,
		Inputs:  InputSetTy{},
		Outputs: map[string]*TxOut{},
	}
}

func NewGenesisTransaction(to *rsa.PublicKey, amount int) *Transaction {
	newTx := NewTransaction(nil, amount)
	newTx.Id = []byte("genesis")

	newTxOut := NewTxOut(to, amount)
	newTxOut.Id = HexEncodeByteSlice(newTx.Id)
	newTxOut.ComputeAndFillHash()

	newTx.Outputs[newTxOut.Id] = newTxOut
	newTx.ComputeAndFillHash()
	// newTxOut.TransactionId = HexEncodeByteSlice(newTx.Id)

	return newTx
}

func (tx *Transaction) IsGenesis() bool {
	return tx.SenderAddress == nil
}

func (tx *Transaction) ComputeAndFillHash() {
	txInfoBytes, err := json.Marshal(tx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	byteArray := (sha256.Sum256(txInfoBytes))
	tx.Id = byteArray[:]
	encodedId := HexEncodeByteSlice(tx.Id)
	for _, txOut := range tx.Outputs {
		txOut.TransactionId = encodedId
	}
}
