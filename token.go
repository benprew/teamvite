package main

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/jmoiron/sqlx"
)

func PlayerForToken(DB *sqlx.DB, t string) (p player, err error) {
	err = DB.Get(&p, "select p.* from players p where id in (select player_id from tokens where id = ? and expires_on > strftime('%s', datetime('now'))) ", t)
	checkErr(err, "player for token")
	return
}

// saves a new token to db and returns token id
func CreateToken(DB *sqlx.DB, p player, exp time.Time) (string, error) {
	id, err := genToken()
	if err != nil {
		return "", err
	}
	checkErr(err, "generating token")
	_, err = DB.Exec("insert into tokens (id, player_id, expires_on) values (?, ?, ?)", id, p.Id, exp.Unix())
	checkErr(err, "inserting new token")
	return id, err
}

func genToken() (string, error) {
	length := 16
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
