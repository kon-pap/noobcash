package backend

import "crypto/rsa"

type TxIn struct {
	previousOutputId string
}

// func NewTxIn() TxIn {
// }

type TxOut struct {
	id            string
	transactionId string
	amount        int
}

// func NewTxOut() TxOut {
// }

type Transaction struct {
	senderAddress   *rsa.PublicKey
	receiverAddress *rsa.PublicKey
	amount          int
	id              string
	inputs          []TxIn
	outputs         []TxOut
	signature       string
}

// func NewTransaction() *Transaction {
// }
