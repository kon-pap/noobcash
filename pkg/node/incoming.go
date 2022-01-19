package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	bck "github.com/kon-pap/noobcash/pkg/node/backend"
)

func setupNodeHandler() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/register-nodes", registerNodesHandler).Methods("POST")
	r.HandleFunc("/submit-blocks", submitBlocksHandler).Methods("POST")
	r.HandleFunc("/submit-txs", submitTxsHandler).Methods("POST")
	return r
}

// Call only after the node is created
//
// Can be used with 'go' keyword to not block the main thread
func ServeApiForNodes(port string) {
	log.Printf("Setting up node server on port %s for incoming requests from nodes", port)
	srv := &http.Server{
		Handler: setupNodeHandler(),
		Addr:    "0.0.0.0:" + port,
	}
	log.Println("Node server exitied:", srv.ListenAndServe())
}

func registerNodesHandler(w http.ResponseWriter, r *http.Request) {
	var nodes []struct {
		Hostname string `json:"hostname"`
		Port     string `json:"port"`
		PubKey   string `json:"pubKey"`
		Id       int    `json:"id"`
	}
	err := json.NewDecoder(r.Body).Decode(&nodes)
	if err != nil || len(nodes) == 0 {
		errMsg := fmt.Sprintf("Body could not be desirialized: %s", err.Error())
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	regCnt := 0
	for _, currNode := range nodes {
		if currNode.PubKey == "" {
			log.Println("Received node without public key:", currNode)
			http.Error(w, "Received node without public key", http.StatusBadRequest)
			return
		}
		if _, ok := myNode.Ring[currNode.PubKey]; ok {
			log.Println("Registration attempted on already registered node with incoming id:", currNode.Id)
			continue
		}
		// add it to the ring
		log.Println("Adding node with id:", currNode.Id, "to ring")
		myNode.Ring[currNode.PubKey] = NewNodeInfo(
			currNode.Id,
			currNode.Hostname,
			currNode.Port,
			bck.PubKeyFromPem(currNode.PubKey),
		)
		regCnt++
	}
	fmt.Fprintf(w, "Registered %d node(s)", regCnt)
}

func submitBlocksHandler(w http.ResponseWriter, r *http.Request) {
	var blocks []*bck.Block
	err := json.NewDecoder(r.Body).Decode(&blocks)
	if err != nil {
		errMsg := fmt.Sprintf("Body could not be desirialized: %s", err.Error())
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	myNode.ApplyChain(blocks)
	fmt.Fprintf(w, "Accepted %d block(s)", len(blocks))
}

func submitTxsHandler(w http.ResponseWriter, r *http.Request) {
	var txs []*bck.Transaction
	err := json.NewDecoder(r.Body).Decode(&txs)
	if err != nil {
		errMsg := fmt.Sprintf("Body could not be desirialized: %s", err.Error())
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	for _, currTx := range txs {
		if !myNode.IsValidTx(currTx) {
			log.Println("Received invalid transaction:", currTx)

			continue
		}
		myNode.getLastBlock().AddTx(currTx)
		if myNode.getLastBlock().IsFull() {
			myNode.MineBlock(myNode.getLastBlock()) // TODO: make this async ?
			myNode.Chain = append(myNode.Chain, bck.NewBlock(
				len(myNode.Chain),
				myNode.getLastBlock().CurrentHash,
			))
		}
	}
}
