package backend

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
)

type Wallet struct {
	Balance int
	PrivKey *rsa.PrivateKey
	PubKey  *rsa.PublicKey
}

func NewWallet(id string, bits int) Wallet {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	publicKey := privateKey.PublicKey
	return Wallet{
		PrivKey: privateKey,
		PubKey:  &publicKey,
	}
}
func (w Wallet) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Balance int
		PubKey  string
		PrivKey string
	}{
		Balance: w.Balance,
		PubKey:  w.PubKeyToPem(),
		PrivKey: w.PrivKeyToPem(),
	})
}
func (w *Wallet) UnmarshalJSON(data []byte) error {
	var wallet struct {
		Balance int
		PubKey  string
		PrivKey string
	}
	err := json.Unmarshal(data, &wallet)
	if err != nil {
		return err
	}
	w.Balance = wallet.Balance
	w.PubKey = PubKeyFromPem(wallet.PubKey)
	w.PrivKey = PrivKeyFromPem(wallet.PrivKey)
	return nil
}

func (w Wallet) String() string {
	strBytes, err := (json.Marshal(w))
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return string(strBytes)
}

func (w Wallet) PrivKeyToPem() string {
	privKeyBytes := x509.MarshalPKCS1PrivateKey(w.PrivKey)
	privKeyBlock := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	}
	return string(pem.EncodeToMemory(&privKeyBlock))
}
func PrivKeyFromPem(s string) *rsa.PrivateKey {
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

func (w Wallet) PubKeyToPem() string {
	publicKeyBytes := x509.MarshalPKCS1PublicKey(w.PubKey)
	publicKeyBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	return string(pem.EncodeToMemory(publicKeyBlock))
}
func PubKeyFromPem(s string) *rsa.PublicKey {
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

func (w Wallet) SaveWallet(path string) {
	fullKeyPath := path
	os.MkdirAll(fullKeyPath, 0755)

	privateKeyPem := w.PrivKeyToPem()
	err := ioutil.WriteFile(fullKeyPath+"/private.pem", []byte(privateKeyPem), 0644)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	publicKeyPem := w.PubKeyToPem()
	err = ioutil.WriteFile(fullKeyPath+"/public.pem", []byte(publicKeyPem), 0644)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}

func LoadWallet(path string) Wallet {
	fullKeyPath := path
	privateKeyBytes, err := ioutil.ReadFile(fullKeyPath + "/private.pem")
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	privateKey := PrivKeyFromPem(string(privateKeyBytes))

	publicKeyBytes, err := ioutil.ReadFile(fullKeyPath + "/public.pem")
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	publicKey := PubKeyFromPem(string(publicKeyBytes))

	return Wallet{
		PrivKey: privateKey,
		PubKey:  publicKey,
	}
}

// func (w Wallet) CreateTx(amount int, address *rsa.PublicKey) (Transaction, error) {
// }

// func (w Wallet) SignTx(tx Transaction) error {
// }
// no need for Balance method, because it is already in the Wallet struct
