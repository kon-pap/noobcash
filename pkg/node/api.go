package node

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Call only after the node is created
//
// Can be used with 'go' keyword to not block the main thread
func ServeApi(port string) {
	http.HandleFunc("/balance", giveBalance)
	http.HandleFunc("/view", giveLastBlock)
	http.HandleFunc("/submit", acceptAndSubmitTx)
	http.ListenAndServe(":"+port, nil)
}

func giveBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprintf(w, "Your balance is: %d", myNode.Wallet.Balance)
}

func giveLastBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
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
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
	var tx reqTx
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&tx)
	if err != nil {
		errMsg := fmt.Sprintf("Internal server error: %s", err.Error())
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Submitting transaction to %s for %d", tx.Recipient, tx.Amount)
	// myNode.SubmitTx(tx.Recipient, tx.Amount)
}
