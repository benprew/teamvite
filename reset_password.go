package main

import (
	"database/sql"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func ResetPassword(DB *QueryLogger, email string, password string) error {
	pHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	p, err := findByEmail(DB, email)
	if p.Id == 0 || err != nil {
		return fmt.Errorf("no user found [email=%s,error=%v]", email, err)
	}
	p.Password = sql.NullString{String: string(pHash[:]), Valid: true}

	log.Printf("Updating password to: '%s' hash: %s\n", password, p.Password.String)
	res, err := DB.NamedExec("update players set password = :password where id = :id", p)
	if err != nil {
		log.Print(err)
	}
	rows, err := res.RowsAffected()
	fmt.Printf("rows updated: %d\n", rows)
	DB.Exec("commit")

	return err
}
