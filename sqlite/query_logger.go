package sqlite

import (
	"database/sql"
	"log"

	"github.com/jmoiron/sqlx"
)

type QueryLogger struct {
	Queryer *sqlx.DB
	Logger  *log.Logger
}

func (p *QueryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	p.Logger.Print("SQL===> ", query, args)
	return p.Queryer.Query(query, args...)
}

func (p *QueryLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	p.Logger.Print("SQL===> ", query, args)
	return p.Queryer.Queryx(query, args...)
}

func (p *QueryLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	p.Logger.Print("SQL===> ", query, args)
	return p.Queryer.QueryRowx(query, args...)
}

func (p *QueryLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	p.Logger.Print("SQL===> ", query, args)
	return p.Queryer.Exec(query, args...)
}

func (p *QueryLogger) NamedExec(query string, arg interface{}) (sql.Result, error) {
	p.Logger.Print("SQL===> ", query, arg)
	return sqlx.NamedExec(p, query, arg)
}

func (p *QueryLogger) Select(dest interface{}, query string, args ...interface{}) error {
	p.Logger.Print("SQL===> ", query, args)
	return sqlx.Select(p, dest, query, args...)
}

func (p *QueryLogger) Get(dest interface{}, query string, args ...interface{}) error {
	p.Logger.Print("SQL===> ", query, args)
	return sqlx.Get(p, dest, query, args...)
}

func (p *QueryLogger) Close() error {
	return p.Queryer.Close()
}

func (p *QueryLogger) DriverName() string {
	return p.Queryer.DriverName()
}

func (p *QueryLogger) Rebind(s string) string {
	return p.Queryer.Rebind(s)
}

func (p *QueryLogger) BindNamed(s string, i interface{}) (string, []interface{}, error) {
	return p.Queryer.BindNamed(s, i)
}
