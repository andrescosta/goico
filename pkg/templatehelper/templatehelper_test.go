package templatehelper_test

import (
	"strings"
	"testing"

	. "github.com/andrescosta/goico/pkg/templatehelper"
)

type data struct {
	Long  string
	Short string
}

func TestRender(t *testing.T) {
	b := strings.Builder{}
	template1 := `Usage: {{.Long | trim | capitalize}}`
	d := data{Long: "  longdata   "}
	if err := Render(&b, template1, d); err != nil {
		t.Errorf("Render %s", err)
	}
	if b.String() != "Usage: Longdata" {
		t.Errorf(`expected "Usage: Longdata" got %s`, b.String())
	}
	b.Reset()
	template2 := `Usage: {{.Long}}`
	d2 := data{Long: " longdata "}
	if err := Render(&b, template2, d2); err != nil {
		t.Errorf("Render %s", err)
	}
	if b.String() != "Usage:  longdata " {
		t.Errorf(`expected "Usage:  longdata " got %s`, b.String())
	}
	b.Reset()
	template3 := `Usage: {{.Long | trim}}`
	d3 := data{Long: " longdata "}
	if err := Render(&b, template3, d3); err != nil {
		t.Errorf("Render %s", err)
	}
	if b.String() != "Usage: longdata" {
		t.Errorf(`expected "Usage:  longdata " got %s`, b.String())
	}
	b.Reset()
	template4 := `Usage: {{.Long | capitalize}}`
	d4 := data{Long: "longdata "}
	if err := Render(&b, template4, d4); err != nil {
		t.Errorf("Render %s", err)
	}
	if b.String() != "Usage: Longdata " {
		t.Errorf(`expected "Usage:  Longdata " got %s`, b.String())
	}
}
