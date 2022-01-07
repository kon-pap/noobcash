package backend

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
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

func (w Wallet) SaveWallet(path string) {
	fullKeyPath := path
	os.MkdirAll(fullKeyPath, 0755)

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(w.PrivKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privatePemFile, err := os.Create(fullKeyPath + "/private.pem")
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	err = pem.Encode(privatePemFile, privateKeyBlock)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	publicKeyBytes := x509.MarshalPKCS1PublicKey(w.PubKey)
	publicKeyBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicPemFile, err := os.Create(fullKeyPath + "/public.pem")
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	err = pem.Encode(publicPemFile, publicKeyBlock)
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
	privateKeyBlock, _ := pem.Decode(privateKeyBytes)
	if privateKeyBlock.Type != "RSA PRIVATE KEY" {
		fmt.Println("RSA private key is of the wrong type", privateKeyBlock.Type)
		os.Exit(1)
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	publicKeyBytes, err := ioutil.ReadFile(fullKeyPath + "/public.pem")
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	publicKeyBlock, _ := pem.Decode(publicKeyBytes)
	if publicKeyBlock.Type != "RSA PUBLIC KEY" {
		fmt.Println("RSA public key is of the wrong type", publicKeyBlock.Type)
		os.Exit(1)
	}
	publicKey, err := x509.ParsePKCS1PublicKey(publicKeyBlock.Bytes)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
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
