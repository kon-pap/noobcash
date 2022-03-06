package node

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.Println("Setting up test environment...")
	NewNode(0, 1024, "localhost", "7070")
	myNode.MakeBootstrap()
	os.Exit(m.Run())
}
