package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/benprew/teamvite"
	_ "github.com/mattn/go-sqlite3"
)

func Open(dsn string) *sql.DB {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		panic(err)
	}

	return db
	// return &ql.QueryLogger{Queryer: db, Logger: log.Default()}
}

// FormatLimitOffset returns a SQL string for a given limit & offset.
// Clauses are only added if limit and/or offset are greater than zero.
func FormatLimitOffset(limit, offset int) string {
	if limit > 0 && offset > 0 {
		return fmt.Sprintf(`LIMIT %d OFFSET %d`, limit, offset)
	} else if limit > 0 {
		return fmt.Sprintf(`LIMIT %d`, limit)
	} else if offset > 0 {
		return fmt.Sprintf(`OFFSET %d`, offset)
	}
	return ""
}

// FormatError tries to format a sqlite error as a teamvite error.
// Otherwise returns the original error.
func FormatError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case strings.Contains(err.Error(), "UNIQUE constraint"):
		return teamvite.Errorf(teamvite.ECONFLICT, err.Error())
	default:
		return err
	}
}
