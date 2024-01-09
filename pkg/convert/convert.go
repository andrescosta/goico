package convert

func SliceWithFn[T, S any](o []T, fn func(T) S) []S {
	t := make([]S, 0)
	for _, ee := range o {
		t = append(t, fn(ee))
	}
	return t
}
