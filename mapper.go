package sqlp

import (
	"context"
	"io/fs"
	"strings"
	"text/template"
)

func funcPlus(x, y int) int {
	return x + y
}
func funcTrim(s string, r rune) string {
	return strings.TrimRightFunc(s, func(c rune) bool {
		return c == ' ' || c == r
	})
}
func funcJoin(sep string, a, b string) string {
	if len(a) < 1 {
		return b
	}
	if len(b) < 1 {
		return a
	}
	return a + sep + b
}

func baseTemplate() (t *template.Template) {
	t = template.New("base")
	t.Funcs(template.FuncMap{
		"plus": funcPlus,
		"trim": funcTrim,
		"join": funcJoin,
	})
	return
}

var (
	parseFile = func(t *template.Template, fname string) (*template.Template, error) {
		return t.ParseFiles(fname)
	}
	parseFS = func(t *template.Template, fsys fs.FS, fname string) (*template.Template, error) {
		return t.ParseFS(fsys, fname)
	}
)

func ParseFile(fname string) (m *Mapper, err error) {
	t := baseTemplate()
	if _, err = parseFile(t, fname); err != nil {
		return
	}
	m = newMapper(t)
	return
}

func ParseFS(fsys fs.FS, fname string) (m *Mapper, err error) {
	t := baseTemplate()
	if _, err = parseFS(t, fsys, fname); err != nil {
		return
	}
	m = newMapper(t)
	return
}

func newMapper(t *template.Template) *Mapper {
	return &Mapper{t: t}
}

type Mapper struct {
	t *template.Template
}

func (m *Mapper) Use(sql string) *executor {
	return &executor{
		t:    m.t,
		sqlm: sql,
		ctx:  context.Background(),
	}
}
