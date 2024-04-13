package teamvite

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

// Build version & commit SHA.
var (
	Version string
	Commit  string
)

// ReportError notifies an external service of errors. No-op by default.
var ReportError = func(ctx context.Context, err error, args ...interface{}) {}

// ReportPanic notifies an external service of panics. No-op by default.
var ReportPanic = func(err interface{}) {}

type Item interface {
	ItemID() uint64
	ItemType() string
}

func GravatarKey(email string) string {
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
