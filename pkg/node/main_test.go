package node

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.Println("Doing initial setup")
	NewNode(0, 1024)
	os.Exit(m.Run())
}
