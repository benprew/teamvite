package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type user player

func (s *server) userLogout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := s.Mgr.Revoke(r.Context(), w); err != nil {
			log.Printf("ERR: %s\n", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		http.Redirect(w, r, "/", http.StatusFound)
	})
}

func (s *server) userLogin() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.RenderTemplate(w, r, "views/user/login.tmpl", nil)
	})
}

func (s *server) userLoginPost() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		checkErr(err, "parsing form")
		email := r.PostForm.Get("email")
		password := []byte(r.PostForm.Get("password"))
		p, err := findByEmail(s.DB, email)
		if err != nil {
			msg := fmt.Sprintf("No user found for email: %s", email)
			s.SetMessage(s.GetUser(r), msg)
			log.Println(msg)
			// render the template here, after setting the error message
			s.RenderTemplate(w, r, "views/user/login.tmpl", nil)
			return
		}
		hash := []byte(p.Password.String)
		if p.Password.String == "" || !p.Password.Valid {
			msg := "user must reset password"
			s.SetMessage(s.GetUser(r), msg)
			log.Println(msg)
			s.RenderTemplate(w, r, "views/user/login.tmpl", nil)
			return
		}

		// Comparing the password with the hash
		err = bcrypt.CompareHashAndPassword(hash, password)
		if err != nil {
			msg := "incorrect password"
			s.SetMessage(s.GetUser(r), msg)
			log.Println(msg, err)
			s.RenderTemplate(w, r, "views/user/login.tmpl", nil)
			return
		}
		log.Println("DEBUG: logging in as user:", p)
		userID := p.Id
		err = s.Mgr.Init(
			w,
			r,
			fmt.Sprint(userID),
			func(m map[string]string) { m["message"] = "" })
		checkErr(err, "session manager init")
		http.Redirect(w, r, urlFor(&p, "show"), http.StatusFound)
	})
}

func (u player) IsManager(DB *sqlx.DB, t team) (isMgr bool) {
	if u.Id == 0 {
		return false
	}
	err := DB.Get(&isMgr, "select is_manager from players_teams where player_id = ? and team_id = ?", u.Id, t.Id)
	checkErr(err, "IsManager")
	return
}
