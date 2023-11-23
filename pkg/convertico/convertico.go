package convertico

func ConvertSlices[T, S any](sliceT []T) []S {
	sliceS := make([]S, len(sliceT))
	for _, e := range sliceT {
		var i interface{} = e
		val := i.(S)
		sliceS = append(sliceS, val)
	}
	return sliceS
}
