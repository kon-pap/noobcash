package backend

import (
	"encoding/hex"
	"fmt"
	"os"
)

func HexEncodeByteSlice(b []byte) string {
	return fmt.Sprintf("%x", b)
}
func HexDecodeByteSlice(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return b
}
