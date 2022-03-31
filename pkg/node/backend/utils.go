package backend

import (
	"encoding/hex"
	"encoding/json"
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

type TxOutMap map[string]*TxOut

func (set *TxOutMap) Add(previousTxOut *TxOut) {
	(*set)[previousTxOut.Id] = previousTxOut
}
func (set *TxOutMap) Has(input *TxOut) bool {
	_, ok := (*set)[input.Id]
	return ok
}
func (set *TxOutMap) Remove(input *TxOut) {
	delete(*set, input.Id)
}
func (set *TxOutMap) MarshalJSON() ([]byte, error) {
	tmp := (*map[string]*TxOut)(set)
	return json.Marshal(tmp)
}
func (set *TxOutMap) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*map[string]*TxOut)(set))
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
