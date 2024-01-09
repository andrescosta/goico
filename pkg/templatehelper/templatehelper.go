package templatehelper

import (
	"io"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

func Render(w io.Writer, templ string, data any) error {
	t := template.New("top")
	t.Funcs(template.FuncMap{"trim": strings.TrimSpace, "capitalize": capitalize})
	template.Must(t.Parse(templ))
	if err := t.Execute(w, data); err != nil {
		return err
	}
	return nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToTitle(r)) + s[n:]
}
