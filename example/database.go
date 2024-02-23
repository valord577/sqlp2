package main

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	sqldb  *sql.DB
	dbErr  error
	dbOnce sync.Once
)

func free() {
	if sqldb != nil {
		if err := sqldb.Close(); err != nil {
			fmt.Printf("closing sql.DB, err: %s\n", err.Error())
		}
	}
}

func open() error {
	dbOnce.Do(func() {
		sqldb, dbErr = sql.Open("mysql", dsn)
		if dbErr == nil {
			dbErr = sqldb.Ping()
		}
		if dbErr != nil {
			return
		}

		sqldb.SetConnMaxIdleTime(180 * time.Second)
		sqldb.SetConnMaxLifetime(600 * time.Second)
		sqldb.SetMaxIdleConns(2)
		sqldb.SetMaxOpenConns(4)
	})
	return dbErr
}
