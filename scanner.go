package sqlp

import (
	"database/sql"
	"errors"
	"reflect"
	"strconv"
)

const tagName = "sqlp"

// ErrTooManyResults triggered when sql result rows more than one and no more variable to scan
var ErrTooManyResults = errors.New("expected one result (or nil), but found multiple")

var sqlScannerInterface = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

type sqlrows interface {
	Next() bool
	Columns() ([]string, error)
	Scan(...any) error
	Err() error
}

func scanAny(dest any, rows sqlrows) error {
	if reflect.ValueOf(rows).IsNil() {
		return nil
	}
	if !rows.Next() {
		return sql.ErrNoRows
	}

	ptr := reflect.ValueOf(dest)
	if ptr.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to scan destination")
	}
	if ptr.IsNil() {
		return errors.New("nil pointer passed to scan destination")
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	length := len(columns)

	value := ptr.Elem()
	switch value.Kind() {
	case reflect.Map:
		switch value.Interface().(type) {
		case map[string]any:
			addr, _ := ptr.Interface().(*map[string]any)
			return scanMap(addr, rows, columns, length, false)
		default:
			return errors.New("only passed map type: *map[string]any, received: " + ptr.Type().String())
		}

	case reflect.Struct:
		if ptr.Type().Implements(sqlScannerInterface) {
			return scanField(ptr, rows, length, false)
		}
		return scanStruct(value, rows, columns, length, false)

	case reflect.Slice:
		t := value.Type().Elem()

		kind := t.Kind()
		isPtr := (kind == reflect.Ptr)
		if isPtr {
			kind = t.Elem().Kind()
		}

		switch kind {
		case reflect.Map:
			switch value.Interface().(type) {
			case []map[string]any, []*map[string]any:
				return scanSliceMap(value, isPtr, rows, columns, length)
			default:
				return errors.New("only passed type: []map[string]any or []*map[string]any, received: " + ptr.Type().String())
			}

		case reflect.Struct:
			p := t
			if !isPtr {
				p = reflect.PointerTo(t)
			}
			if p.Implements(sqlScannerInterface) {
				return scanSliceField(value, p, isPtr, rows, length)
			}
			return scanSliceStruct(value, p, isPtr, rows, columns, length)

		default:
			p := t
			if !isPtr {
				p = reflect.PointerTo(t)
			}
			return scanSliceField(value, p, isPtr, rows, length)
		}

	default:
		return scanField(ptr, rows, length, false)
	}
}

func scanMap(dest *map[string]any, rows sqlrows, cols []string, length int, fromSlice bool) error {
	if *dest == nil {
		*dest = make(map[string]any, length)
	}

	values := make([]any, length)
	for i := range values {
		values[i] = new(any)
	}

	err := rows.Scan(values...)
	if err != nil {
		return err
	}

	m := *dest
	for i, col := range cols {
		m[col] = *(values[i].(*any))
	}

	if !fromSlice {
		if rows.Next() {
			return ErrTooManyResults
		}
	}
	return rows.Err()
}

func scanStruct(v reflect.Value, rows sqlrows, cols []string, length int, fromSlice bool) error {

	numField := v.NumField()
	mapTagToField := make(map[string]reflect.Value, numField)

	t := v.Type()
	for i := 0; i < numField; i++ {
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}

		ft := t.Field(i)
		tag := ft.Tag.Get(tagName)
		if tag == "" {
			continue
		}
		mapTagToField[tag] = fv
	}

	if len(mapTagToField) == 0 {
		return errors.New("required at least one tag[`" + tagName + "`] in structure")
	}

	values := make([]any, length)
	for i, col := range cols {
		field, ok := mapTagToField[col]
		if ok {
			values[i] = field.Addr().Interface()
		} else {
			values[i] = new(any)
		}
	}

	err := rows.Scan(values...)
	if err != nil {
		return err
	}

	mapColToResult := make(map[string]reflect.Value, length)
	for i, col := range cols {
		mapColToResult[col] = reflect.ValueOf(values[i])
	}

	for i := 0; i < numField; i++ {
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}

		ft := t.Field(i)
		tag := ft.Tag.Get(tagName)
		if tag == "" {
			continue
		}

		r, ok := mapColToResult[tag]
		if ok {
			fv.Set(r.Elem())
		}
	}

	if !fromSlice {
		if rows.Next() {
			return ErrTooManyResults
		}
	}
	return rows.Err()
}

func scanField(ptr reflect.Value, rows sqlrows, length int, fromSlice bool) error {
	if length > 1 {
		return errors.New("expected one column, but found " + strconv.FormatInt(int64(length), 10))
	}

	err := rows.Scan(ptr.Interface())
	if err != nil {
		return err
	}

	if !fromSlice {
		if rows.Next() {
			return ErrTooManyResults
		}
	}
	return rows.Err()
}

func scanSliceMap(value reflect.Value, elemIsPtr bool, rows sqlrows, cols []string, length int) error {
	var err error

	m := make([]map[string]any, 0, 10)

	flag := true
	for flag {
		var mm map[string]any
		err = scanMap(&mm, rows, cols, length, true)
		if err != nil {
			return err
		}
		m = append(m, mm)

		flag = rows.Next()
	}

	if elemIsPtr {
		mp := make([]*map[string]any, len(m))
		for i := range m {
			mp[i] = &m[i]
		}
		value.Set(reflect.ValueOf(mp))
	} else {
		value.Set(reflect.ValueOf(m))
	}

	return nil
}

func scanSliceStruct(value reflect.Value, elemTypePtr reflect.Type, elemIsPtr bool, rows sqlrows, cols []string, length int) error {
	var err error

	flag := true
	for flag {
		mm := reflect.New(elemTypePtr.Elem())
		err = scanStruct(mm.Elem(), rows, cols, length, true)
		if err != nil {
			return err
		}

		if elemIsPtr {
			value.Set(reflect.Append(value, mm))
		} else {
			value.Set(reflect.Append(value, mm.Elem()))
		}

		flag = rows.Next()
	}

	return nil
}

func scanSliceField(value reflect.Value, elemTypePtr reflect.Type, elemIsPtr bool, rows sqlrows, length int) error {
	var err error

	flag := true
	for flag {
		mm := reflect.New(elemTypePtr.Elem())
		err = scanField(mm, rows, length, true)
		if err != nil {
			return err
		}

		if elemIsPtr {
			value.Set(reflect.Append(value, mm))
		} else {
			value.Set(reflect.Append(value, mm.Elem()))
		}

		flag = rows.Next()
	}

	return nil
}
