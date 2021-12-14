package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/swithek/sessionup"
	"github.com/swithek/sessionup/memstore"
)

const HOSTNAME = "teamvitedev.com"

func main() {
	store := memstore.New(time.Minute * 5)

	s := server{
		DB:       initDb(),
		Mgr:      sessionup.NewManager(store, sessionup.Secure(false)),
		HostName: "teamvitedev.com",
	}
	defer s.DB.Close()

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", s.routes()))
}

func (s *server) getUser(req *http.Request) (user player) {
	t, ok := req.URL.Query()["token"]
	if ok && t[0] != "" {
		log.Printf("Finding player with token: %s\n", t)
		if p, err := PlayerForToken(s.DB, t[0]); err == nil {
			log.Println(p)
			// TODO: create a session here so we can set messages
			return p
		} else {
			log.Println(p, err)
		}
	}

	ss, ok := s.getSession(req)
	if ok {
		userId := ss.UserKey
		err := s.DB.Get(&user, "select * from players where id = ?", userId)
		checkErr(err, fmt.Sprintf("Getting user from session: %s", userId))
	}
	return user
}

func (s *server) getSession(req *http.Request) (sessionup.Session, bool) {
	var ss []sessionup.Session

	ss, err := s.Mgr.FetchAll(req.Context())
	checkErr(err, "Getting sessions from manager")
	log.Printf("session_len: %d sessions: %v", len(ss), ss)
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

func initDb() *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", "file:teamvite.db?_foreign_keys=1")
	checkErr(err, "sql.Open failed")
	return db
}

func checkErr(err error, msg string) {
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatalln(msg, err)
	}
}
