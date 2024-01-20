package protoutil

import (
	"reflect"

	"google.golang.org/protobuf/proto"
)

func Slices[S any](sliceT []proto.Message) []S {
	sliceS := make([]S, len(sliceT))
	if len(sliceT) > 0 {
		if !canConvert[S](sliceT[0]) {
			return nil
		}
	}
	for ix, e := range sliceT {
		var i any = e
		v := i.(S)
		sliceS[ix] = v
	}
	return sliceS
}

func canConvert[S any](i any) bool {
	var a S
	t1 := reflect.TypeOf(i)
	t2 := reflect.TypeOf(a)
	return t1.ConvertibleTo(t2)
}
