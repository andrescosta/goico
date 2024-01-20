package collection_test

import (
	"cmp"
	"slices"
	"testing"

	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/collection"
	"github.com/andrescosta/goico/pkg/test"
)

func TestSetString(t *testing.T) {
	checkVals(t, []string{"val1", "val2", "val3"})
	checkVals(t, []int{88, 77, 66, 55, 90})
}

func TestZero(t *testing.T) {
	set1 := NewSet[string]()
	test.Equals(t, set1.Size(), 0)
}

func TestFunc(t *testing.T) {
	type data struct {
		index int
		name  string
	}
	values := []data{
		{1, "val1"},
		{2, "val2"},
		{3, "val3"},
	}
	set1 := NewSetFn(values, func(v data) int { return v.index })
	test.Equals(t, set1.Size(), 3)
	for _, s := range values {
		if !set1.Has(s.index) {
			t.Errorf("%v not found", s.index)
		}
	}
	set2 := NewSetFn(values, func(v data) string { return v.name })
	test.Equals(t, set2.Size(), 3)
	for _, s := range values {
		if !set2.Has(s.name) {
			t.Errorf("%v not found", s.name)
		}
	}
}

func TestDuplicated(t *testing.T) {
	valuess := []string{"A", "A", "B", "B", "C", "C", "D"}
	set1s := NewSet(valuess...)
	if set1s.Size() != 4 {
		t.Errorf("expected 4 got %d", set1s.Size())
	}
	for _, s := range valuess {
		if !set1s.Has(s) {
			t.Errorf("%v not found", s)
		}
	}
	values2s := set1s.Values()
	if len(values2s) != 4 {
		t.Errorf("expected 4 got %d", set1s.Size())
	}
	valuesi := []int{1, 1, 2, 2, 2, 3, 3, 3, 4, 4}
	set1i := NewSet(valuesi...)
	if set1i.Size() != 4 {
		t.Errorf("expected 4 got %d", set1s.Size())
	}
	for _, s := range valuesi {
		if !set1i.Has(s) {
			t.Errorf("%v not found", s)
		}
	}
	values2i := set1i.Values()
	if len(values2i) != 4 {
		t.Errorf("expected 4 got %d", set1s.Size())
	}
}

func checkVals[T cmp.Ordered](t *testing.T, values []T) {
	set1 := NewSet(values...)
	for _, s := range values {
		if !set1.Has(s) {
			t.Errorf("%v not found", s)
		}
	}
	test.Equals(t, set1.Size(), len(values))
	set1.Delete(values[0])
	if set1.Has(values[0]) {
		t.Errorf("found %v", values[0])
	}
	set1.Add(values[0])
	for _, s := range values {
		if !set1.Has(s) {
			t.Errorf("%v not found after deletion", s)
		}
	}
	values2 := set1.Values()
	slices.Sort(values)
	slices.Sort(values2)
	if !slices.Equal(values, values2) {
		t.Error("slices are different")
	}
	values3 := make([]T, 0)
	set1.Range(func(s T) bool {
		values3 = append(values3, s)
		return true
	})
	slices.Sort(values3)
	if !slices.Equal(values, values3) {
		t.Error("slices are different")
	}
	var value1 T
	set1.Range(func(s T) bool {
		value1 = s
		return s != values[1]
	})

	if value1 != values[1] {
		t.Errorf("expected %v got %v", values[1], value1)
	}
}
