package convertico

import (
	"encoding/binary"

	"github.com/andrescosta/goico/pkg/reflectico"
)

func ConvertSlices[T, S any](sliceT []T) []S {
	sliceS := make([]S, len(sliceT))
	if len(sliceT) > 0 {
		if !reflectico.CanConvert[S](sliceT[0]) {
			return nil
		}
	}
	for ix, e := range sliceT {
		var i interface{} = e
		v := i.(S)
		sliceS[ix] = v
	}
	return sliceS
}

func Itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func Btoi(v []byte) uint64 {
	return binary.BigEndian.Uint64(v)
}
