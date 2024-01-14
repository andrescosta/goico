package collection

type void struct{}

type Set[T comparable] struct {
	theset map[T]void
}

func NewSet[T comparable](values ...T) *Set[T] {
	return NewSetFn(values, func(v T) T { return v })
}

func NewSetFn[T comparable, Y any](values []Y, fn func(Y) T) *Set[T] {
	s := &Set[T]{
		theset: make(map[T]void),
	}
	for _, v := range values {
		s.Add(fn(v))
	}
	return s
}

func (s Set[T]) Add(t T) {
	s.theset[t] = void{}
}

func (s Set[T]) Has(t T) (ok bool) {
	_, ok = s.theset[t]
	return
}

func (s Set[T]) Delete(t T) (ok bool) {
	r := s.Has(t)
	delete(s.theset, t)
	return r
}

func (s Set[T]) Size() int {
	return len(s.theset)
}

func (s Set[T]) Values() []T {
	vals := make([]T, s.Size())
	var i int
	s.Range(func(t T) bool {
		vals[i] = t
		i++
		return true
	})
	return vals
}

func (s Set[T]) Range(fn func(T) bool) {
	for k := range s.theset {
		if !fn(k) {
			return
		}
	}
}
