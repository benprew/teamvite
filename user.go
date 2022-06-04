package main

import (
	"fmt"
	"internal/session"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type user player

func (u player) IsManager(DB *sqlx.DB, t team) (isMgr bool) {
	err := DB.Get(&isMgr, "select is_manager from players_teams where player_id = ? and team_id = ?", u.Id, t.Id)
	checkErr(err, fmt.Sprintf("unable to check manager [player_id=%d, team_id=%d]", u.Id, t.Id))
	return
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
		sids, err := session.SidsFromCookie(req, SessionKey)
		checkErr(err, "Failed to get sids from cookie")
		for _, sid := range sids {
			sess, err = session.LoadSession(s.DB, sid, session.RequestIP(req))
			if err == nil {
				break
			}
		}
		checkErr(err, "loading session from cookie")
	}

	err = s.DB.Get(&usr, "select * from players where id = ?", sess.PlayerID)
	checkErr(err, fmt.Sprintf("get user from session: %d", sess.PlayerID))
	return
}

func (s *server) userLogout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sids, err := session.SidsFromCookie(r, SessionKey)
		checkErr(err, "Failed to get sids from cookie")
		for _, sid := range sids {
			if err := session.Revoke(s.DB, sid); err != nil {
				checkErr(err, "Failed to revoke session")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
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
		ip := session.RequestIP(r)
		s, err := session.New(s.DB, userID, ip, time.Hour*24*30)
		s.SetCookie(w, SessionKey, CONFIG.Servername)
		checkErr(err, fmt.Sprintf("created session [player_id=%d]", userID))

		// err = s.Mgr.Init(w, r, fmt.Sprint(userID))
		http.Redirect(w, r, urlFor(&p, "show"), http.StatusFound)
	})
}
