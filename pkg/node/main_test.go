package node

import (
	"log"
	"os"
	"testing"
)

var testNode *Node

func TestMain(m *testing.M) {
	log.Println("Setting up test environment...")
	testNode = NewNode(0, 1024, "localhost", "7070", "8080")
	testNode.MakeBootstrap()
	os.Exit(m.Run())
}
