package backend

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

type Wallet struct {
	id      string
	PrivKey *rsa.PrivateKey
	PubKey  rsa.PublicKey
}

func CreateWallet(id string, bits int) Wallet {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	publicKey := privateKey.PublicKey
	return Wallet{
		id:      id,
		PrivKey: privateKey,
		PubKey:  publicKey,
	}
}

func (w Wallet) WritePEM(path string) {
	fullKeyPath := path + "/" + w.id
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
	publicKeyBytes := x509.MarshalPKCS1PublicKey(&w.PubKey)
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
