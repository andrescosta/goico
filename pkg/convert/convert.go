package convert

import (
	"encoding/binary"
	"reflect"
)

func Slices[T, S any](sliceT []T) []S {
	sliceS := make([]S, len(sliceT))
	if len(sliceT) > 0 {
		if !CanConvert[S](sliceT[0]) {
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
func SliceWithFn[T, S any](o []T, fn func(T) S) []S {
	t := make([]S, 0)
	for _, ee := range o {
		t = append(t, fn(ee))
	}
	return t
}

func CanConvert[S any](i interface{}) bool {
	var a S
	t1 := reflect.TypeOf(i)
	t2 := reflect.TypeOf(a)
	return t1.ConvertibleTo(t2)
}
