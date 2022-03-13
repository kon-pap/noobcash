package backend

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"sort"
)

type Wallet struct {
	Balance int
	PrivKey *rsa.PrivateKey
	Utxos   map[string]*TxOut
}
type WalletInfo struct {
	Balance int
	PubKey  *rsa.PublicKey
	Utxos   map[string]*TxOut
}

func NewWallet(bits int) *Wallet {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return &Wallet{
		PrivKey: privateKey,
		Utxos:   map[string]*TxOut{},
	}
}

func NewWalletInfo(pubKey *rsa.PublicKey) *WalletInfo {
	return &WalletInfo{
		PubKey: pubKey,
		Utxos:  map[string]*TxOut{},
	}
}

func (w *Wallet) GetWalletInfo() *WalletInfo {
	return &WalletInfo{
		Balance: w.Balance,
		PubKey:  &w.PrivKey.PublicKey,
		Utxos:   w.Utxos,
	}
}
func (w *WalletInfo) MarshalJSON() ([]byte, error) {
	type printableWallet struct {
		Balance int      `json:"balance"`
		PubKey  string   `json:"address"`
		Utxos   []*TxOut `json:"utxos"`
	}
	txouts := make([]*TxOut, len(w.Utxos))
	i := 0
	for _, txout := range w.Utxos {
		txouts[i] = txout
		i++
	}
	return json.Marshal(printableWallet{
		Balance: w.Balance,
		PubKey:  PubKeyToPem(w.PubKey),
		Utxos:   txouts,
	})
}

////
// Serialization and deserialization
////
func (w *Wallet) MarshalJSON() ([]byte, error) {
	// TODO: Implement using Marshaler  of utxo
	return json.Marshal(w.GetWalletInfo())
}

func (w *Wallet) String() string {
	strBytes, err := (json.Marshal(w))
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return string(strBytes)
}

func (w *WalletInfo) String() string {
	strBytes, err := (json.Marshal(w))
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return string(strBytes)
}

func PrivKeyToPem(privKey *rsa.PrivateKey) string {
	if privKey == nil {
		return "0"
	}
	privKeyBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privKeyBlock := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	}
	return string(pem.EncodeToMemory(&privKeyBlock))
}
func PrivKeyFromPem(s string) *rsa.PrivateKey {
	if s == "0" {
		return nil
	}
	block, _ := pem.Decode([]byte(s))
	if block == nil {
		fmt.Println("Failed to decode PEM block containing the key")
		os.Exit(1)
	}
	if block.Type != "RSA PRIVATE KEY" {
		fmt.Println("RSA private key is of the wrong type", block.Type)
		os.Exit(1)
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return key
}
func PubKeyToPem(pubKey *rsa.PublicKey) string {
	if pubKey == nil {
		return "0"
	}
	publicKeyBytes := x509.MarshalPKCS1PublicKey(pubKey)
	publicKeyBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	return string(pem.EncodeToMemory(publicKeyBlock))
}
func PubKeyFromPem(s string) *rsa.PublicKey {
	if s == "0" {
		return nil
	}
	block, _ := pem.Decode([]byte(s))
	if block == nil {
		fmt.Println("Failed to decode PEM block containing the key")
		os.Exit(1)
	}
	if block.Type != "RSA PUBLIC KEY" {
		fmt.Println("RSA public key is of the wrong type", block.Type)
		os.Exit(1)
	}
	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return key
}

type utxoTmp struct {
	key string
	val *TxOut
}
type utxoTmpListTy []utxoTmp

func (t utxoTmpListTy) Less(i, j int) bool { return t[i].val.Amount > t[j].val.Amount }
func (t utxoTmpListTy) Len() int           { return len(t) }
func (t utxoTmpListTy) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

// func (w *Wallet) selectUTXOsRandomImprove(targetAmount int) (sum int, txIns []*TxOut) {}

// Chooses utxos from the wallet that are sufficient to pay the amount,
// removes them from the utxos map, and returns them along with their sum.
func (w *Wallet) selectUTXOsLargestFirst(targetAmount int) (sum int, previousTxOuts []*TxOut, err error) {
	tmp := make(utxoTmpListTy, len(w.Utxos))
	i := 0
	for k, v := range w.Utxos {
		tmp[i] = utxoTmp{k, v}
		i++
	}
	sort.Sort(tmp)
	for _, v := range tmp {
		if sum >= targetAmount {
			break
		}
		sum += v.val.Amount
		previousTxOuts = append(previousTxOuts, v.val)
	}
	if sum < targetAmount {
		err = errors.New("not enough money")
	} else {
		for _, chosen := range previousTxOuts {
			delete(w.Utxos, chosen.Id)
		}
	}
	return
}

func (w *Wallet) CreateTx(amount int, address *rsa.PublicKey) (*Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("tried to create transaction for %d", amount)
	}
	if amount > w.Balance {
		return nil, fmt.Errorf("tried to create transaction for %d but only have %d", amount, w.Balance)
	}
	tx := NewTransaction(&w.PrivKey.PublicKey, amount)

	sum, previousTxOuts, err := w.selectUTXOsLargestFirst(amount)
	if err != nil {
		return nil, err
	}
	for _, txOut := range previousTxOuts {
		tx.Inputs.Add(txOut.Id)
	}

	targetTxOut := NewTxOut(address, amount)
	targetTxOut.ComputeAndFillHash()
	tx.Outputs[targetTxOut.Id] = targetTxOut

	changeAmount := sum - amount
	if changeAmount > 0 { // if change exists
		changeTxOut := NewTxOut(&w.PrivKey.PublicKey, changeAmount)
		changeTxOut.ComputeAndFillHash()
		tx.Outputs[changeTxOut.Id] = changeTxOut
	}

	tx.ComputeAndFillHash()
	return tx, nil
}

type TxTargetTy struct {
	Address *rsa.PublicKey
	Amount  int
}

func (w *Wallet) CreateMultiTargetTx(targets ...TxTargetTy) (*Transaction, error) {
	var totalAmount int
	for _, target := range targets {
		totalAmount += target.Amount
	}
	if totalAmount <= 0 {
		return nil, fmt.Errorf("tried to create transaction for %d", totalAmount)
	}
	if totalAmount > w.Balance {
		return nil, fmt.Errorf("tried to create transaction for %d but only have %d", totalAmount, w.Balance)
	}
	tx := NewTransaction(&w.PrivKey.PublicKey, totalAmount)
	sum, previousTxOuts, err := w.selectUTXOsLargestFirst(totalAmount)
	if err != nil {
		return nil, err
	}
	for _, txOut := range previousTxOuts {
		tx.Inputs.Add(txOut.Id)
	}
	changeAmount := sum - totalAmount
	for _, target := range targets {
		targetTxOut := NewTxOut(target.Address, target.Amount)
		targetTxOut.ComputeAndFillHash()
		tx.Outputs[targetTxOut.Id] = targetTxOut
	}
	if changeAmount > 0 {
		changeTxOut := NewTxOut(&w.PrivKey.PublicKey, changeAmount)
		changeTxOut.ComputeAndFillHash()
		tx.Outputs[changeTxOut.Id] = changeTxOut
	}
	tx.ComputeAndFillHash()
	return tx, nil
}

func (w *Wallet) SignTx(tx *Transaction) error {
	signature, err := rsa.SignPKCS1v15(rand.Reader, w.PrivKey, crypto.SHA256, tx.Id)
	if err != nil {
		return err
	}
	tx.Signature = signature
	return nil
}
