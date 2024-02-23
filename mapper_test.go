package sqlp

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"
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

type fakeFsys string

func (fakeFsys) Open(string) (fs.File, error) {
	return &openFile{f: &file{name: name, data: tmpl}}, nil
}

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
	m1, err := ParseFS(fakeFsys(""), name)
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
	m, err := ParseFS(fakeFsys(""), name)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	if err = m.Use("").Ctx(context.Background()).check(); err == nil {
		t.Fail()
	}
}

func TestExec(t *testing.T) {
	m, err := ParseFS(fakeFsys(""), name)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	if _, err = m.Use("sample0").At(fakeSqlpoint("")).Exec(data0); err != nil {
		t.Errorf("%s", err.Error())
	}
}

func TestExecBatch(t *testing.T) {
	m, err := ParseFS(fakeFsys(""), name)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	if _, err = m.Use("sample2").At(fakeSqlpoint("")).ExecBatch(data1...); err != nil {
		t.Errorf("%s", err.Error())
	}
}

func TestScan(t *testing.T) {
	m, err := ParseFS(fakeFsys(""), name)
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	if err = m.Use("sample1").At(fakeSqlpoint("")).Scan(nil, data0); err != nil {
		t.Errorf("%s", err.Error())
	}
}

// split splits the name into dir and elem as described in the
// comment in the FS struct above. isDir reports whether the
// final trailing slash was present, indicating that name is a directory.
func split(name string) (dir, elem string, isDir bool) {
	if name[len(name)-1] == '/' {
		isDir = true
		name = name[:len(name)-1]
	}
	i := len(name) - 1
	for i >= 0 && name[i] != '/' {
		i--
	}
	if i < 0 {
		return ".", name, isDir
	}
	return name[:i], name[i+1:], isDir
}

// A file is a single file in the FS.
// It implements fs.FileInfo and fs.DirEntry.
type file struct {
	// The compiler knows the layout of this struct.
	// See cmd/compile/internal/staticdata's WriteEmbed.
	name string
	data string
}

func (f *file) Name() string               { _, elem, _ := split(f.name); return elem }
func (f *file) Size() int64                { return int64(len(f.data)) }
func (f *file) ModTime() time.Time         { return time.Time{} }
func (f *file) IsDir() bool                { _, _, isDir := split(f.name); return isDir }
func (f *file) Sys() any                   { return nil }
func (f *file) Type() fs.FileMode          { return f.Mode().Type() }
func (f *file) Info() (fs.FileInfo, error) { return f, nil }

func (f *file) Mode() fs.FileMode {
	if f.IsDir() {
		return fs.ModeDir | 0555
	}
	return 0444
}

func (f *file) String() string {
	return fs.FormatFileInfo(f)
}

// An openFile is a regular file open for reading.
type openFile struct {
	f      *file // the file itself
	offset int64 // current read offset
}

func (f *openFile) Close() error               { return nil }
func (f *openFile) Stat() (fs.FileInfo, error) { return f.f, nil }

func (f *openFile) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.f.data)) {
		return 0, io.EOF
	}
	if f.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.f.name, Err: fs.ErrInvalid}
	}
	n := copy(b, f.f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

// An openDir is a directory open for reading.
type openDir struct {
	f      *file  // the directory file itself
	files  []file // the directory contents
	offset int    // the read offset, an index into the files slice
}

func (d *openDir) Close() error               { return nil }
func (d *openDir) Stat() (fs.FileInfo, error) { return d.f, nil }

func (d *openDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.f.name, Err: errors.New("is a directory")}
}

func (d *openDir) ReadDir(count int) ([]fs.DirEntry, error) {
	n := len(d.files) - d.offset
	if n == 0 {
		if count <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	if count > 0 && n > count {
		n = count
	}
	list := make([]fs.DirEntry, n)
	for i := range list {
		list[i] = &d.files[d.offset+i]
	}
	d.offset += n
	return list, nil
}
