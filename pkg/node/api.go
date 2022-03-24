package node

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func (n *Node) setupCliHandler() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/balance", n.createGiveBalanceHandler()).Methods("GET")
	r.HandleFunc("/view", n.createGiveLastBlockHandler()).Methods("GET")
	r.HandleFunc("/submit", n.createAcceptAndSubmitTx()).Methods("POST")
	return r
}

// Call only after the node is created
//
// Can be used with 'go' keyword to not block the main thread
func (n *Node) ServeApiForCli(port string) {
	log.Printf("Setting up node server on port %s for incoming requests from CLI", port)
	srv := &http.Server{
		Handler: n.setupCliHandler(),
		Addr:    "0.0.0.0:" + port,
	}
	log.Println("CLI Server exited:", srv.ListenAndServe())
}

func (n *Node) createGiveBalanceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Your balance is: %d", n.Wallet.Balance)
	}
}

func (n *Node) createGiveLastBlockHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lastBlock := n.getLastBlock()
		if lastBlock == nil {
			fmt.Fprintf(w, "No blocks yet")
			return
		}
		blockData, err := json.Marshal(lastBlock)
		if err != nil {
			errMsg := fmt.Sprintf("Internal server error: %s", err.Error())
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
		w.Write(blockData)
	}
}

type reqTx struct {
	Recipient int `json:"recipient"`
	Amount    int `json:"amount"`
}

func (n *Node) createAcceptAndSubmitTx() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tx reqTx
		err := json.NewDecoder(r.Body).Decode(&tx)
		if err != nil {
			errMsg := fmt.Sprintf("Body could not be desirialized: %s", err.Error())
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		var address *rsa.PublicKey
		for _, nInfo := range n.Ring {
			if nInfo.Id == tx.Recipient {
				address = nInfo.WInfo.PubKey
				break
			}
		}
		createdTx, err := n.Wallet.CreateAndSignTx(tx.Amount, address)
		if err != nil {
			errMsg := fmt.Sprintf("Creating transaction error: %s", err.Error())
			log.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		err = n.AcceptTx(createdTx)
		if err != nil {
			errMsg := fmt.Sprintf("Accepting transaction error: %s", err.Error())
			log.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		err = n.BroadcastTx(createdTx)
		if err != nil {
			errMsg := fmt.Sprintf("Broadcasting transaction error: %s", err.Error())
			log.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Submitted transaction to node %d for %d", tx.Recipient, tx.Amount)
	}
}
