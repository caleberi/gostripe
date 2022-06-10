package driver

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DBConfiguration struct {
	ConnMaxLifetime, ConnMaxIdleTime, MaxIdleConns time.Duration
	MaxOpenConns                                   int
}

func ConfigureDB(dsn string, config DBConfiguration) (*sql.DB, error) {
	db, err := OpenDB(dsn)
	if err != nil {
		return nil, err
	}

	if config.ConnMaxIdleTime != 0 {
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	if config.MaxIdleConns != 0 {
		db.SetConnMaxIdleTime(config.MaxIdleConns)
	}

	if config.MaxOpenConns != 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}

	if config.ConnMaxLifetime != 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	return db, nil
}

func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()

	if err != nil {
		return nil, err
	}

	return db, err
}
