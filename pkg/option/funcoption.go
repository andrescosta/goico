package option

type FuncOption[T any] struct {
	f func(T)
}

//lint:ignore U1000 Ignore unused (likely a bug in the lint)
func (fdo *FuncOption[T]) Apply(do T) {
	fdo.f(do)
}

func NewFuncOption[T any](f func(T)) *FuncOption[T] {
	return &FuncOption[T]{
		f: f,
	}
}
