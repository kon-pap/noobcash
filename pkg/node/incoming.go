package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	bck "github.com/kon-pap/noobcash/pkg/node/backend"
)

type endpointTy string

const (
	acceptNodesEndpoint   = endpointTy("/accept-nodes")
	submitBlocksEndpoint  = endpointTy("/submit-blocks")
	submitTxsEndpoint     = endpointTy("/submit-txs")
	bootstrapNodeEndpoint = endpointTy("/bootstrap-node")
)

func (n *Node) setupNodeHandler() *mux.Router {
	r := mux.NewRouter()
	// Accepts a list of fellow nodes in its ring
	r.HandleFunc(string(acceptNodesEndpoint), n.createAcceptNodesHandler()).Methods("POST")
	// Accepts a list of blocks to try and apply to the chain
	r.HandleFunc(string(submitBlocksEndpoint), n.createSubmitBlocksHandler()).Methods("POST")
	// Accepts a list of transactions to try and insert into blocks
	r.HandleFunc(string(submitTxsEndpoint), n.createSubmitTxsHandler()).Methods("POST")
	if n.IsBootstrap() { // only bootstrap node can register new nodes
		r.HandleFunc(string(bootstrapNodeEndpoint), n.createBootstrapNodeHandler()).Methods("POST")
	}
	return r
}

// Call only after the node is created
//
// Can be used with 'go' keyword to not block the main thread
func (n *Node) ServeApiForNodes(port string) {
	log.Printf("Setting up node server on port %s for incoming requests from nodes", port)
	srv := &http.Server{
		Handler: n.setupNodeHandler(),
		Addr:    "0.0.0.0:" + port,
	}
	log.Println("Node server exitied:", srv.ListenAndServe())
}

type transferNodeTy struct {
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
	PubKey   string `json:"pubKey"`
	Id       int    `json:"id"`
}

func (n *Node) createAcceptNodesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var nodes []transferNodeTy
		err := json.NewDecoder(r.Body).Decode(&nodes)
		if err != nil || len(nodes) == 0 {
			errMsg := fmt.Sprintf("Body could not be desirialized: %s", err.Error())
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		// this step is needed cause json Decode may fail silently
		firstNodeTmp := nodes[0]
		if firstNodeTmp.PubKey == "" {
			log.Println("Received node without public key:", firstNodeTmp)
			http.Error(w, "Received node without public key", http.StatusBadRequest)
			return
		}
		regCnt := 0
		for _, currNode := range nodes {
			if _, ok := n.Ring[currNode.PubKey]; ok {
				log.Println("Incoming node pubkey is already registered:", currNode.Id)
				continue
			}
			// add it to the ring
			log.Println("Adding node with id:", currNode.Id, "to ring")
			n.Ring[currNode.PubKey] = NewNodeInfo(
				currNode.Id,
				currNode.Hostname,
				currNode.Port,
				bck.PubKeyFromPem(currNode.PubKey),
			)
			regCnt++
		}
		fmt.Fprintf(w, "Registered %d node(s)", regCnt)
	}
}
func (n *Node) createSubmitBlocksHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var blocks []*bck.Block
		err := json.NewDecoder(r.Body).Decode(&blocks)
		if err != nil {
			errMsg := fmt.Sprintf("Body could not be desirialized: %s", err.Error())
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		for _, currBlock := range blocks {
			n.IncBlockChan <- currBlock
		}
		fmt.Fprintf(w, "Accepted %d block(s)", len(blocks))
	}
}

func (n *Node) createSubmitTxsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var txs []*bck.Transaction
		err := json.NewDecoder(r.Body).Decode(&txs)
		if err != nil {
			errMsg := fmt.Sprintf("Body could not be desirialized: %s", err.Error())
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		for _, currTx := range txs {
			err := n.AcceptTx(currTx)
			if err != nil {
				log.Println("Error accepting tx:", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		fmt.Fprintf(w, "Accepted %d transaction(s)", len(txs))
	}
}

type bootstrapNodeTy struct {
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
	PubKey   string `json:"pubKey"`
}

func (n *Node) createBootstrapNodeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var node *bootstrapNodeTy
		err := json.NewDecoder(r.Body).Decode(&node)
		if err != nil {
			errMsg := fmt.Sprintf("Body could not be deserialized: %s", err.Error())
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		if node.PubKey == "" {
			errMsg := "Received node without public key"
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		n.BsNextNodeId.Mu.Lock()
		log.Println("Bootstraping node with id:", n.BsNextNodeId.Value)
		n.Ring[node.PubKey] = NewNodeInfo(
			n.BsNextNodeId.Value,
			node.Hostname,
			node.Port,
			bck.PubKeyFromPem(node.PubKey),
		)
		n.BsNextNodeId.Value++
		n.BsNextNodeId.Mu.Unlock()

		fmt.Fprintf(w, "%d", n.Ring[node.PubKey].Id)
	}

}
