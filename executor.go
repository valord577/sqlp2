package sqlp

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"text/template"
)

type sqlmode uint8

// SqlMode is an identifier that
// determines in what way the template is replaced
// before the SQL is executed.
const (
	SqlModeNormal sqlmode = iota
	SqlModeBatch
)

type sqlpoint interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type executor struct {
	t    *template.Template
	sqlm string
	p    sqlpoint
	ctx  context.Context
}

// At sets the implementation of sqlpoint,
// which may be *sql.DB, *sql.Conn or *sql.TX.
func (e *executor) At(p sqlpoint) *executor {
	e.p = p
	return e
}

// Ctx sets context.Context for sqlpoint.
func (e *executor) Ctx(ctx context.Context) *executor {
	e.ctx = ctx
	return e
}

// Parse parses into SQL statements (strings), and provides placeholder arguments.
func (e *executor) Parse(mode sqlmode, args ...map[string]any) (sqls string, placeholder []any, err error) {
	var data any
	data = args
	if mode == SqlModeNormal {
		if len(args) > 0 {
			data = args[0]
		}
	}

	b := &strings.Builder{}
	if err = e.t.ExecuteTemplate(b, e.sqlm, data); err != nil {
		return
	}
	sqls = b.String()

	mapping := func(key string) (value any, exist bool, err error) {
		if len(args) < 1 {
			return
		}
		i := 0
		if mode == SqlModeBatch {
			sp := strings.Split(key, ".")
			if len(sp) != 2 {
				err = errors.New("invalid '@index.key': " + key)
				return
			}
			isp := strings.TrimLeft(sp[0], "@")
			key = sp[1]

			var i64 int64
			if i64, err = strconv.ParseInt(isp, 10, 32); err != nil {
				return
			}
			i = int(i64)
		}
		value, exist = args[i][key]
		return
	}

	sqls, _, err = expand("${", "}", sqls, func(key string) (string, any, bool, error) {
		replaced := ""
		value, exist, e := mapping(key)
		if e == nil {
			switch a := value.(type) {
			case string:
				replaced = a
			case []byte:
				replaced = string(a)
			default:
				e = errors.New("the '" + key + "' must be string or []byte")
			}
		}
		return replaced, value, exist, e
	})
	if err != nil {
		return
	}

	sqls, placeholder, err = expand("#{", "}", sqls, func(key string) (string, any, bool, error) {
		value, exist, e := mapping(key)
		return "?", value, exist, e
	})
	if err != nil {
		return
	}

	sqls = strings.ReplaceAll(sqls, "\r", " ")
	sqls = strings.ReplaceAll(sqls, "\n", " ")
	return
}

func (e *executor) check() (err error) {
	if e.p == nil {
		err = errors.New("undefined sql point")
	}
	return
}

// Exec executes a query with args that doesn't return rows.
func (e *executor) Exec(args ...map[string]any) (sql.Result, error) {
	return e.exec(SqlModeNormal, args...)
}

// ExecBatch executes a query with the array of args that doesn't return rows.
func (e *executor) ExecBatch(args ...map[string]any) (sql.Result, error) {
	return e.exec(SqlModeBatch, args...)
}

func (e *executor) exec(mode sqlmode, args ...map[string]any) (sql.Result, error) {
	if err := e.check(); err != nil {
		return nil, err
	}

	sqls, placeholder, err := e.Parse(mode, args...)
	if err != nil {
		return nil, err
	}
	return e.p.ExecContext(e.ctx, sqls, placeholder...)
}

// Query executes a query with args that returns rows, typically a SELECT.
func (e *executor) Query(args ...map[string]any) (*sql.Rows, error) {
	return e.query(SqlModeNormal, args...)
}

// QueryBatch executes a query with the array of args that returns rows, typically a SELECT.
func (e *executor) QueryBatch(args ...map[string]any) (*sql.Rows, error) {
	return e.query(SqlModeBatch, args...)
}

func (e *executor) query(mode sqlmode, args ...map[string]any) (*sql.Rows, error) {
	if err := e.check(); err != nil {
		return nil, err
	}

	sqls, placeholder, err := e.Parse(mode, args...)
	if err != nil {
		return nil, err
	}
	return e.p.QueryContext(e.ctx, sqls, placeholder...)
}

// Scan executes a query that returns rows, and maps to dest.
func (e *executor) Scan(dest any, args ...map[string]any) error {
	var query func(...map[string]any) (*sql.Rows, error)
	if len(args) > 1 {
		query = e.QueryBatch
	} else {
		query = e.Query
	}

	rs, err := query(args...)
	if err != nil {
		return err
	}
	// if something happens here, we want to make sure the rows are Closed
	defer func() {
		if rs != nil {
			_ = rs.Close()
		}
	}()

	return scanAny(dest, rs)
}

type mapfunc func(key string) (replaced string, value any, exist bool, err error)

func expand(open, close string, s string, f mapfunc) (string, []any, error) {
	var (
		args []any
		err  error
	)
	for {
		pos0 := strings.Index(s, open)
		if pos0 < 0 {
			break
		}
		pos1 := pos0 + len(open)
		pos2 := strings.Index(s[pos1:], close)
		if pos2 < 0 {
			break
		}
		key := s[pos1 : pos1+pos2]
		pos3 := pos1 + pos2 + len(close)
		replaced, value, exist, e := f(key)
		if e != nil {
			err = e
			break
		}

		if exist {
			args = append(args, value)
		}
		s = s[:pos0] + replaced + s[pos3:]
	}
	return s, args, err
}
