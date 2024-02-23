package main

import (
	"example/mapper"
	"fmt"

	"github.com/valord577/sqlp2"
)

const (
	dsn       = ""
	namespace = "namespace.gotmpl"
)

var rows = []map[string]any{
	{
		"name":  "alex",
		"class": 1001,
	},
	{
		"name":  "bob",
		"class": 1001,
	},
	{
		"name":  "cook",
		"class": 1001,
	},
}

func main() {
	var e error
	if e = open(); e != nil {
		panic(e)
	}
	defer free()

	var m *sqlp.Mapper
	if m, e = sqlp.ParseFS(mapper.FS, namespace); e != nil {
		panic(e)
	}
	if _, e = m.Use("sampleCreate").At(sqldb).Exec(); e != nil {
		panic(e)
	}

	_, e = m.Use("sampleInsertBatch").At(sqldb).ExecBatch(rows[:2]...)
	if e != nil {
		panic(e)
	}
	_, e = m.Use("sampleInsertOne").At(sqldb).Exec(rows[2])
	if e != nil {
		panic(e)
	}

	var maps []map[string]any
	e = m.Use("sampleSelect").At(sqldb).Scan(&maps, map[string]any{"class": 1001})
	if e != nil {
		panic(e)
	}
	fmt.Printf("%#v\n", maps)

	rows[2]["class"] = 1002
	_, e = m.Use("sampleUpdate").At(sqldb).Exec(rows[2])
	if e != nil {
		panic(e)
	}
}
