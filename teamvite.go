package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const SessionKey = "teamvite-session"

func main() {
	db := getDB()
	defer db.Close()

	if os.Args[1] == "serv" || len(os.Args) == 1 {
		fmt.Printf("Starting teamvite server on port 8080\n")
		serv(db)
	} else if os.Args[1] == "resetpassword" {
		if err := ResetPassword(db, os.Args[2], os.Args[3]); err != nil {
			fmt.Printf("Error: %v", err)
		}
	} else {
		fmt.Printf("ERROR: unknown command %s\n", os.Args[1])
	}
}

// Run as server
func serv(db *QueryLogger) {
	s := server{
		DB:       db,
		MsgStore: NewMessageStore(),
	}

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", s.routes()))
}

func getDB() *QueryLogger {
	db, err := sqlx.Connect("sqlite3", "file:teamvite.db?_foreign_keys=1")
	if err != nil {
		panic(err)
	}

	return &QueryLogger{db, log.Default()}
}

type Item interface {
	itemId() int
	itemType() string
}

func urlFor(i Item, action string) string {
	id := i.itemId()
	name := i.itemType()
	return fmt.Sprintf("/%s/%d/%s", name, id, action)
}

func gravatarKey(email string) string {
	sum := md5.Sum([]byte(email))
	return hex.EncodeToString(sum[:])
}

func checkErr(err error, msg string) {
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Println("[ERROR]: ", msg, err)
	}
}

// format an int as a telephone number
func Telify(num int) string {
	str := fmt.Sprintf("%d", num)
	if len(str) != 10 {
		return ""
	}
	return fmt.Sprintf("%s-%s-%s", str[0:3], str[3:6], str[6:])
}

// convert a string to an integer telephone number.  Valid phone numbers are 10
// characters, no country codes.
//
// returns -1 if the string is not a valid phone number
func UnTelify(str string) int {
	reg, _ := regexp.Compile("[^0-9]+")
	strTel := reg.ReplaceAllString(str, "")
	if len(strTel) == 10 {
		tel, _ := strconv.Atoi(strTel)
		return tel
	}
	return -1
}

type QueryLogger struct {
	queryer *sqlx.DB
	logger  *log.Logger
}

func (p *QueryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	p.logger.Print("SQL===> ", query, args)
	return p.queryer.Query(query, args...)
}

func (p *QueryLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	p.logger.Print("SQL===> ", query, args)
	return p.queryer.Queryx(query, args...)
}

func (p *QueryLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	p.logger.Print("SQL===> ", query, args)
	return p.queryer.QueryRowx(query, args...)
}

func (p *QueryLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	p.logger.Print("SQL===> ", query, args)
	return p.queryer.Exec(query, args...)
}

func (p *QueryLogger) NamedExec(query string, arg interface{}) (sql.Result, error) {
	p.logger.Print("SQL===> ", query, arg)
	return sqlx.NamedExec(p, query, arg)
}

func (p *QueryLogger) Select(dest interface{}, query string, args ...interface{}) error {
	p.logger.Print("SQL===> ", query, args)
	return sqlx.Select(p, dest, query, args...)
}

func (p *QueryLogger) Get(dest interface{}, query string, args ...interface{}) error {
	p.logger.Print("SQL===> ", query, args)
	return sqlx.Get(p, dest, query, args...)
}

func (p *QueryLogger) Close() error {
	return p.queryer.Close()
}

func (p *QueryLogger) DriverName() string {
	return p.queryer.DriverName()
}

func (p *QueryLogger) Rebind(s string) string {
	return p.queryer.Rebind(s)
}

func (p *QueryLogger) BindNamed(s string, i interface{}) (string, []interface{}, error) {
	return p.queryer.BindNamed(s, i)
}
