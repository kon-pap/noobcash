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

func SplitAmount(amount int, pieces int) (split []int) {

	if amount < pieces {
		//err = fmt.Errorf("can't split %d in %d pieces ", amount, pieces)
		//!NOTE: Might have to return the amount as is
		split = append(split, amount)
		return
	} else if amount%pieces == 0 {
		for i := 0; i < pieces; i++ {
			split = append(split, amount/pieces)
		}
	} else {
		a := pieces - (amount % pieces)
		b := amount / pieces
		for i := 0; i < pieces; i++ {
			if i >= a {
				split = append(split, b+1)
			} else {
				split = append(split, b)
			}
		}
	}
	return
}

func Splitter(amount int) (split []int) {
	switch {
	case amount < 10:
		split = append(split, amount)
	case amount >= 10 && amount < 20:
		split = SplitAmount(amount, 2)
	case amount >= 20 && amount < 70:
		split = SplitAmount(amount, 3)
	case amount >= 70 && amount < 100:
		split = SplitAmount(amount, 4)
	default:
		split = SplitAmount(amount, 10)

	}
	return

}
