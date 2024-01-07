package convert

import (
	"reflect"
)

func Slices[T, S any](sliceT []T) []S {
	sliceS := make([]S, len(sliceT))
	if len(sliceT) > 0 {
		if !canConvert[S](sliceT[0]) {
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

func SliceWithFn[T, S any](o []T, fn func(T) S) []S {
	t := make([]S, 0)
	for _, ee := range o {
		t = append(t, fn(ee))
	}
	return t
}

func canConvert[S any](i interface{}) bool {
	var a S
	t1 := reflect.TypeOf(i)
	t2 := reflect.TypeOf(a)
	return t1.ConvertibleTo(t2)
}
