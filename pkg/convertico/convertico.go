package convertico

import (
	"encoding/binary"

	"github.com/andrescosta/goico/pkg/reflectico"
)

func SliceWithSlice[T, S any](sliceT []T) []S {
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

func SliceWithFunc[T, Y any](d []T, g func(T) Y) []Y {
	r := make([]Y, 0)
	for _, ee := range d {
		r = append(r, g(ee))
	}
	return r
}

func SliceWithFuncName[T, Y any](name string, d []T, g func(string, T) Y) []Y {
	r := make([]Y, 0)
	for _, ee := range d {
		r = append(r, g(name, ee))
	}
	return r
}
