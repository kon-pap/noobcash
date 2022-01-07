package backend

import "crypto/rsa"

type Transaction struct {
	senderAddress   *rsa.PublicKey
	receiverAddress *rsa.PublicKey
	amount          int
	id              string
	inputs          *[]TransactionInput
	outputs         *[]TransactionOutput
	signature       string
}

type TransactionInput struct {
	previousOutputId string
}

type TransactionOutput struct {
	id            string
	transactionId string
	amount        int
}
