package collection_test

import (
	"errors"
	"testing"

	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/collection"
)

func TestFirstOrDefault(t *testing.T) {
	t.Parallel()
	valuess := []string{"A", "A", "B", "B", "C", "C", "D"}
	v := FirstOrDefault(valuess, "X")
	if v != "A" {
		t.Errorf("expected A got %s", v)
	}

	v = FirstOrDefault([]string{}, "Z")
	if v != "Z" {
		t.Errorf("expected Z got %s", v)
	}
}

func TestUnwrapError(t *testing.T) {
	t.Parallel()
	err1 := errors.New("error 1")
	e := UnwrapError(err1)
	if len(e) != 1 {
		t.Errorf("expected 1 got %d", len(e))
	}
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")
	errs := errors.Join(err2, err3, err1)
	set1 := NewSet(err1, err2, err3)
	e = UnwrapError(errs)
	if len(e) != 3 {
		t.Errorf("expected 3 got %d", len(e))
	}
	for _, er := range e {
		if !set1.Has(er) {
			t.Errorf("%v not found", er)
		}
	}
}
