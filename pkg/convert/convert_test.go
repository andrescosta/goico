package convert_test

import (
	"testing"

	"github.com/andrescosta/goico/pkg/collection"
	. "github.com/andrescosta/goico/pkg/convert"
)

type d1 struct {
	one string
}
type d2 struct {
	two string
}

func TestSliceWithFn(t *testing.T) {
	d1s := []d1{{"1"}, {"3"}, {"5"}}
	set := collection.NewSet("1", "3", "5")
	d2s := SliceWithFn(d1s, func(d d1) d2 { return d2{two: d.one} })
	if len(d1s) != len(d2s) {
		t.Errorf("size are different %d - %d", len(d1s), len(d2s))
	}
	for _, d := range d2s {
		if !set.Has(d.two) {
			t.Errorf("expected %s", d.two)
		}
	}
}
