package backend

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
)

type Wallet struct {
	Balance int
	PrivKey *rsa.PrivateKey
	Utxos   map[string]TxOut
}
type WalletInfo struct {
	Balance int              `json:"balance"`
	PubKey  string           `json:"address"`
	Utxos   map[string]TxOut `json:"utxos"`
}

func NewWallet(bits int) *Wallet {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return &Wallet{
		PrivKey: privateKey,
		Utxos:   map[string]TxOut{},
	}
}

func (w *Wallet) GetWalletInfo() *WalletInfo {
	return &WalletInfo{
		Balance: w.Balance,
		PubKey:  PubKeyToPem(&w.PrivKey.PublicKey),
		Utxos:   w.Utxos,
	}
}
func (w *WalletInfo) MarshalJSON() ([]byte, error) {
	type printableWallet struct {
		Balance int     `json:"balance"`
		PubKey  string  `json:"address"`
		Utxos   []TxOut `json:"utxos"`
	}
	txouts := make([]TxOut, len(w.Utxos))
	for _, txout := range w.Utxos {
		txouts = append(txouts, txout)
	}
	return json.Marshal(printableWallet{
		Balance: w.Balance,
		PubKey:  w.PubKey,
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

// func (w *Wallet) SaveWallet(path string) {
// 	fullKeyPath := path
// 	os.MkdirAll(fullKeyPath, 0755)

// 	privateKeyPem := w.PrivKeyToPem()
// 	err := ioutil.WriteFile(fullKeyPath+"/private.pem", []byte(privateKeyPem), 0644)
// 	if err != nil {
// 		fmt.Print(err)
// 		os.Exit(1)
// 	}
// }
// func LoadWallet(path string) *Wallet {
// 	fullKeyPath := path
// 	privateKeyBytes, err := ioutil.ReadFile(fullKeyPath + "/private.pem")
// 	if err != nil {
// 		fmt.Print(err)
// 		os.Exit(1)
// 	}
// 	privateKey := PrivKeyFromPem(string(privateKeyBytes))
// 	return &Wallet{
// 		PrivKey: privateKey,
// 	}
// }

func (w *Wallet) selectUTXOs(targetAmount int) (sum int, txIns []TxIn) {

	return
}

func (w *Wallet) CreateTx(amount int, address *rsa.PublicKey) (*Transaction, error) {
	if amount > w.Balance {
		return nil, fmt.Errorf("tried to send %d but only have %d", amount, w.Balance)
	}
	tx := NewTransaction(&w.PrivKey.PublicKey, address, amount)
	// TODO: coin selection to find utxos to use as TxIns
	// TODO: add TxOut to target address, add TxOut to change address

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

// no need for Balance method, because it is already in the Wallet struct
