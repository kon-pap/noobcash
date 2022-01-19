package node

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kon-pap/noobcash/pkg/node/backend"
)

func TestSubmitBlocksHandler(t *testing.T) {
	t.Run("Send a single block", func(t *testing.T) {
		jsBlock, err := json.Marshal([]*backend.Block{
			backend.NewBlock(1, []byte("test")),
		})
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("Sending block:", string(jsBlock))
		body := bytes.NewReader(jsBlock)
		req := httptest.NewRequest("POST", "/submit-blocks", body)
		w := httptest.NewRecorder()
		setupNodeHandler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Error("Did not get expected HTTP status code, got", w.Code)
		}
		if w.Body.String() != "Accepted 1 block(s)" {
			t.Error("Did not get expected greeting, got", w.Body.String())
		}
	})
}
