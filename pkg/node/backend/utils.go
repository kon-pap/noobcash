package backend

import (
	"encoding/hex"
	"fmt"
	"os"
)

func HexEncodeByteSlice(b []byte) string {
	resSlice := hex.EncodeToString(b)
	return resSlice
}
func HexDecodeByteSlice(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return b
}

type InputId string
type InputSetTy map[InputId]struct{}

func (set InputSetTy) Add(inputId string) {
	set[InputId(inputId)] = struct{}{}
}
func (set InputSetTy) Has(inputId string) bool {
	_, ok := set[InputId(inputId)]
	return ok
}
func (set InputSetTy) Remove(inputId string) {
	delete(set, InputId(inputId))
}
