package sqlp

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"text/template"
)

var data0 = map[string]any{
	"table": "`student`",
	"class": 1002,
	"name":  "cook",
}
var data1 = []map[string]any{
	{
		"name":  "alex",
		"class": 1001,
	},
	{
		"name":  "bob",
		"class": 1001,
	},
}

const name = "namespace.gotmpl"
const tmpl = `
{{define "sample0" -}}
UPDATE ${table}
SET
  {{if (ne nil (index . "class"))}}class = #{class}, {{end}}
  name = #{name}
WHERE name = #{name}
{{end}}

{{define "sample1" -}}
SELECT *
FROM ${table}
WHERE 1=1
  {{if (ne nil (index . "class"))}}AND class = #{class}{{end}}
{{end}}

{{define "sample2" -}}
INSERT INTO student
(name, class)
VALUES
  {{ $n := len . }}
  {{range $i, $e := .}}
    (#{@{{$i}}.name},#{@{{$i}}.class}){{if ne $n (plus 1 $i)}}, {{end}}
  {{end}}
{{end}}
`

func TestMain(m *testing.M) {
	parseFile = func(t *template.Template, _ string) (*template.Template, error) {
		return t.Parse(tmpl)
	}
	os.Exit(m.Run())
}

func TestFuncMap(t *testing.T) {
	if funcPlus(1, 1) != 2 {
		t.Fail()
	}
	if funcTrim("one, two, three, ", ',') != "one, two, three" {
		t.Fail()
	}

	if funcJoin(",", "", "one") != "one" {
		t.Fail()
	}
	if funcJoin(",", "one", "two") != "one,two" {
		t.Fail()
	}
	if funcJoin(",", "one", "") != "one" {
		t.Fail()
	}
}

func TestMapper(t *testing.T) {
	m0, err := ParseFile(name)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	m1, err := ParseFile(name)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	testMapperTemplate(t, m0, m1, "sample0", data0)
	testMapperTemplate(t, m0, m1, "sample1", data0)
	testMapperTemplate(t, m0, m1, "sample2", data1)
}

func testMapperTemplate(t *testing.T, m0, m1 *Mapper, name string, data any) {
	b0 := &strings.Builder{}
	if err := m0.t.ExecuteTemplate(b0, name, data); err != nil {
		t.Errorf("%s", err.Error())
	}

	b1 := &strings.Builder{}
	if err := m1.t.ExecuteTemplate(b1, name, data); err != nil {
		t.Errorf("%s", err.Error())
	}

	if b0.String() != b1.String() {
		t.Errorf("%s", "template result not equal")
	}
}

type fakeSqlpoint string

func (fakeSqlpoint) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}
func (fakeSqlpoint) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func TestErrExecutor(t *testing.T) {
	m, err := ParseFile(name)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	if err = m.Use("").Ctx(context.Background()).check(); err == nil {
		t.Fail()
	}
}

func TestExec(t *testing.T) {
	m, err := ParseFile(name)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	if _, err = m.Use("sample0").At(fakeSqlpoint("")).Exec(data0); err != nil {
		t.Errorf("%s", err.Error())
	}
}

func TestExecBatch(t *testing.T) {
	m, err := ParseFile(name)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	if _, err = m.Use("sample2").At(fakeSqlpoint("")).ExecBatch(data1...); err != nil {
		t.Errorf("%s", err.Error())
	}
}

func TestScan(t *testing.T) {
	m, err := ParseFile(name)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	if err = m.Use("sample1").At(fakeSqlpoint("")).Scan(nil, data0); err != nil {
		t.Errorf("%s", err.Error())
	}
}
