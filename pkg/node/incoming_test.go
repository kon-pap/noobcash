package node

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kon-pap/noobcash/pkg/node/backend"
)

func TestAcceptNodesHandler(t *testing.T) {
	t.Run("Send a single node", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			t.Error(err)
		}
		jsNode, err := json.Marshal([]transferNodeTy{
			{
				Hostname: "localhost",
				Port:     "8080",
				PubKey:   backend.PubKeyToPem(&privateKey.PublicKey),
				Id:       1,
			},
		})
		if err != nil {
			log.Fatalln(err)
		}
		// log.Println("Sending node:", string(jsNode))
		body := bytes.NewReader(jsNode)
		req := httptest.NewRequest("POST", "/accept-nodes", body)
		w := httptest.NewRecorder()
		setupNodeHandler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}
		if w.Body.String() != "Registered 1 node(s)" {
			t.Errorf("Expected body %s, got %s", "Registered 1 node(s)", w.Body.String())
		}
	})
}

func TestSubmitBlocksHandler(t *testing.T) {
	t.Run("Send a single block", func(t *testing.T) {
		jsBlock, err := json.Marshal([]*backend.Block{
			backend.NewBlock(1, []byte("test")),
		})
		if err != nil {
			log.Fatalln(err)
		}
		// log.Println("Sending block:", string(jsBlock))
		body := bytes.NewReader(jsBlock)
		req := httptest.NewRequest("POST", "/submit-blocks", body)
		w := httptest.NewRecorder()
		setupNodeHandler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Error("Did not get expected HTTP status code, got", w.Code)
		}
		if w.Body.String() != "Accepted 1 block(s)" {
			t.Errorf("Expected body %s, got %s", "Accepted 1 block(s)", w.Body.String())
		}
	})
}

func TestSubmitTxsHandler(t *testing.T) {
	t.Run("Send a single transaction", func(t *testing.T) {
		newTx, err := myNode.Wallet.CreateTx(1, &myNode.Wallet.PrivKey.PublicKey)
		if err != nil {
			log.Fatalln(err)
		}
		myNode.Wallet.SignTx(newTx)
		jsTx, err := json.Marshal([]*backend.Transaction{newTx})
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("Sending transaction:", string(jsTx))
		body := bytes.NewReader(jsTx)
		req := httptest.NewRequest("POST", "/submit-txs", body)
		w := httptest.NewRecorder()
		setupNodeHandler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Error("Did not get expected HTTP status code, got", w.Code)
		}
		if w.Body.String() != "Accepted 1 transaction(s)" {
			t.Errorf("Expected body %s, got %s", "Accepted 1 transaction(s)", w.Body.String())
		}
	})
}
