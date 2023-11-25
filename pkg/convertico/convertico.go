package convertico

func ConvertSlices[T, S any](sliceT []T) []S {
	sliceS := make([]S, len(sliceT))
	for ix, e := range sliceT {
		var i interface{} = e
		val := i.(S)
		sliceS[ix] = val
	}
	return sliceS
}
