package templatehelper_test

import (
	"strings"
	"testing"

	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/templatehelper"
	"github.com/andrescosta/goico/pkg/test"
)

type data struct {
	Long  string
	Short string
}

func TestRender(t *testing.T) {
	t.Parallel()
	b := strings.Builder{}
	template1 := `Usage: {{.Long | trim | capitalize}}`
	d := data{Long: "  longdata   "}
	err := Render(&b, template1, d)
	test.Nil(t, err)
	if b.String() != "Usage: Longdata" {
		t.Errorf(`expected "Usage: Longdata" got %s`, b.String())
	}
	b.Reset()
	template2 := `Usage: {{.Long}}`
	d2 := data{Long: " longdata "}
	err = Render(&b, template2, d2)
	test.Nil(t, err)
	if b.String() != "Usage:  longdata " {
		t.Errorf(`expected "Usage:  longdata " got %s`, b.String())
	}
	b.Reset()
	template3 := `Usage: {{.Long | trim}}`
	d3 := data{Long: " longdata "}
	err = Render(&b, template3, d3)
	test.Nil(t, err)
	if b.String() != "Usage: longdata" {
		t.Errorf(`expected "Usage:  longdata " got %s`, b.String())
	}
	b.Reset()
	template4 := `Usage: {{.Long | capitalize}}`
	d4 := data{Long: "longdata "}
	err = Render(&b, template4, d4)
	test.Nil(t, err)
	if b.String() != "Usage: Longdata " {
		t.Errorf(`expected "Usage:  Longdata " got %s`, b.String())
	}
}
