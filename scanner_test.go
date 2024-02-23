package sqlp

import (
	"database/sql"
	"testing"
)

type fakerows struct {
	count   uint
	columns []string
}

func (r *fakerows) Columns() ([]string, error) {
	return r.columns, nil
}
func (r *fakerows) Err() error {
	return nil
}
func (r *fakerows) Next() bool {
	next := (r.count > 0)
	r.count -= 1
	return next
}
func (r *fakerows) Scan(...any) error {
	return nil
}

func initRowsZero() *fakerows {
	return &fakerows{
		count: 0,
	}
}
func initRowsOne() *fakerows {
	return &fakerows{
		count:   1,
		columns: []string{"a"},
	}
}
func initRowsTwo() *fakerows {
	return &fakerows{
		count:   2,
		columns: []string{"a", "b"},
	}
}

func TestScanMap(t *testing.T) {
	var m0 map[string]any
	if err := scanAny(&m0, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}

	var m1 map[string]any
	_ = scanAny(&m1, initRowsTwo())
	var m2 map[string]string
	_ = scanAny(&m2, initRowsTwo())
	var m3 map[string]string
	_ = scanAny(&m3, initRowsZero())
}

func TestScanDest(t *testing.T) {
	var v string
	_ = scanAny(v, initRowsTwo())

	var s *string
	_ = scanAny(s, initRowsTwo())
}

func TestScanField(t *testing.T) {
	var s0 string
	if err := scanAny(&s0, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}

	var s1 string
	_ = scanAny(&s1, initRowsTwo())

	var nullstr sql.NullString
	if err := scanAny(&nullstr, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}
}

func TestScanStruct(t *testing.T) {
	var v0 struct {
		B string `sqlp:""`
		A string `sqlp:"a"`
		c string
	}
	if err := scanAny(&v0, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}

	var v1 struct{ S string }
	_ = scanAny(&v1, initRowsOne())
}

func TestScanSliceMap(t *testing.T) {
	var m0 []map[string]any
	if err := scanAny(&m0, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}
	var m1 *[]map[string]any
	if err := scanAny(&m1, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}
	var m2 []*map[string]any
	if err := scanAny(&m2, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}

	var m3 []map[string]string
	_ = scanAny(&m3, initRowsTwo())
}

func TestScanSliceStruct(t *testing.T) {
	var v0 []struct {
		S string `sqlp:"s"`
	}
	if err := scanAny(&v0, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}

	var v1 *[]struct {
		S string `sqlp:"s"`
	}
	if err := scanAny(&v1, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}

	var v2 []*struct {
		S string `sqlp:"s"`
	}
	if err := scanAny(&v2, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}
}

func TestScanSliceField(t *testing.T) {
	var s0 []string
	if err := scanAny(&s0, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}

	var s1 []*string
	if err := scanAny(&s1, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}

	var nullstr []sql.NullString
	if err := scanAny(&nullstr, initRowsOne()); err != nil {
		t.Errorf("%s", err.Error())
	}
}
