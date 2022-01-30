package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	// TODO: Figure out how to set metadata in sqlite3 (there's no public interface for it currently)
	sqlitestore "github.com/hyzual/sessionup-sqlitestore"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/swithek/sessionup"
)

func main() {
	db, err := sql.Open("sqlite3", "file:teamvite.db?_foreign_keys=1")
	checkErr(err, "unable to open sqlite3 db teamvite.db")
	defer db.Close()
	store, err := sqlitestore.New(db, "sessions", time.Hour*24)
	if err != nil {
		log.Fatal("Unable to init sqlite session store", err)
	}

	sDb := sqlx.NewDb(db, "sqlite3")

	s := server{
		DB:       sDb,
		Mgr:      sessionup.NewManager(store, sessionup.Secure(false)),
		MsgStore: NewMessageStore(),
	}

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", s.routes()))
}

func (s *server) GetUser(req *http.Request) (usr player) {
	t, ok := req.URL.Query()["token"]
	if ok && t[0] != "" {
		log.Printf("Finding player with token: %s\n", t)
		if usr, err := PlayerForToken(s.DB, t[0]); err == nil {
			log.Println(usr)
			return usr
		} else {
			checkErr(err, "getting player from token")
		}
	}

	// if we couldn't get player from token, try getting from session

	ss, ok := s.getSession(req)
	if ok {
		userId := ss.UserKey
		err := s.DB.Get(&usr, "select * from players where id = ?", userId)
		checkErr(err, fmt.Sprintf("get user from session: %s", userId))
	}
	return
}

func (s *server) getSession(req *http.Request) (sessionup.Session, bool) {
	var ss []sessionup.Session

	ss, err := s.Mgr.FetchAll(req.Context())
	checkErr(err, "Getting sessions from manager")
	for _, s := range ss {
		if s.Current {
			return s, true
		}
	}
	return sessionup.Session{}, false
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
