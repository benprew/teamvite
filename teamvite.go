package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"internal/session"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const SessionKey = "teamvite-session"

func main() {
	db, err := sqlx.Connect("sqlite3", "file:teamvite.db?_foreign_keys=1")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s := server{
		DB:       db,
		MsgStore: NewMessageStore(),
	}

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", s.routes()))
}

func (s *server) GetUser(req *http.Request) (usr player) {
	t, ok := req.URL.Query()[SessionKey]
	var err error
	var sess session.Session
	if ok && t[0] != "" {
		sid := t[0]
		log.Printf("Finding player with token: %s\n", sid)
		sess, err = session.LoadSession(s.DB, sid, session.RequestIP(req))
		checkErr(err, "getting player from token")
	}

	// if we couldn't get player from token, try getting from session
	if sess.ID == "" {
		sid, _ := session.SidFromCookie(req, SessionKey)
		sess, err = session.LoadSession(s.DB, sid, session.RequestIP(req))
		checkErr(err, "loading session from cookie")
	}

	err = s.DB.Get(&usr, "select * from players where id = ?", sess.PlayerID)
	checkErr(err, fmt.Sprintf("get user from session: %d", sess.PlayerID))
	return
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
