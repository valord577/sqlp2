package main

import (
	"example/mapper"
	"fmt"

	sqlp "github.com/valord577/sqlp2"
)

const (
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
	var (
		s           string
		e           error
		placeholder []any
	)

	var m *sqlp.Mapper
	if m, e = sqlp.ParseFS(mapper.FS, namespace); e != nil {
		panic(e)
	}

	s, placeholder, e = m.Use("sampleCreate").Parse(sqlp.SqlModeNormal)
	if e != nil {
		panic(e)
	} else {
		print(s, placeholder)
	}

	s, placeholder, e = m.Use("sampleInsertBatch").Parse(sqlp.SqlModeBatch, rows[:2]...)
	if e != nil {
		panic(e)
	} else {
		print(s, placeholder)
	}
}

func print(s string, placeholder []any) {
	fmt.Printf("sql: %s\n", s)
	fmt.Printf("placeholder: %s\n", placeholder)
	fmt.Printf("\n")
}
