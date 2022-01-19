package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func setupCliHandler() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/balance", giveBalance).Methods("GET")
	r.HandleFunc("/view", giveLastBlock).Methods("GET")
	r.HandleFunc("/submit", acceptAndSubmitTx).Methods("POST")
	return r
}

// Call only after the node is created
//
// Can be used with 'go' keyword to not block the main thread
func ServeApiForCli(port string) {
	log.Printf("Setting up node server on port %s for incoming requests from CLI", port)
	srv := &http.Server{
		Handler: setupCliHandler(),
		Addr:    "0.0.0.0:" + port,
	}
	log.Println("CLI Server exited:", srv.ListenAndServe())
}

func giveBalance(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Your balance is: %d", myNode.Wallet.Balance)
}

func giveLastBlock(w http.ResponseWriter, r *http.Request) {
	lastBlock := myNode.getLastBlock()
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

type reqTx struct {
	Recipient string `json:"recipient"`
	Amount    int    `json:"amount"`
}

func acceptAndSubmitTx(w http.ResponseWriter, r *http.Request) {
	var tx reqTx
	err := json.NewDecoder(r.Body).Decode(&tx)
	if err != nil {
		errMsg := fmt.Sprintf("Body could not be desirialized: %s", err.Error())
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "Submitting transaction to %s for %d", tx.Recipient, tx.Amount)
	// myNode.SubmitTx(tx.Recipient, tx.Amount)
}
