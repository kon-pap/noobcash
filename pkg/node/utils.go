package node

import "sync"

type MuInt struct {
	Mu    sync.Mutex
	Value int
}
