package http

import (
	"fmt"
	"log"
	"net/http"
	"time"

	teamvite "github.com/benprew/teamvite"

	"golang.org/x/crypto/bcrypt"
)

func (s *Server) GetUser(req *http.Request) (usr *teamvite.Player) {
	return teamvite.UserFromContext(req.Context())
}

func (s *Server) userLogout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logout(w, r)
	})
}

// Revoke user sessions, clear cookies and redirect to home page
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	sids := SidsFromCookie(r, SESSION_KEY)
	for _, sid := range sids {
		if err := s.SessionService.Revoke(sid); err != nil {
			s.Error(w, r, err)
			return
		}
	}
	s.setSession(w, teamvite.Session{})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) userLogin() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.RenderTemplate(w, r, "views/user/login.tmpl", nil)
	})
}

func (s *Server) userLoginPost() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			s.Error(w, r, err)
			return
		}
		email := r.PostForm.Get("email")
		password := []byte(r.PostForm.Get("password"))
		players, n, err := s.PlayerService.FindPlayers(r.Context(), teamvite.PlayerFilter{Email: email})
		if err != nil || n != 1 {
			msg := fmt.Sprintf("No user found for email: %s", email)
			SetFlash(w, msg)
			log.Println("[ERROR] ", msg, err)
			http.Redirect(w, r, "/user/login", http.StatusFound)
			return
		}
		player := players[0]
		hash := []byte(player.Password)
		if player.Password == "" {
			msg := "user must reset password"
			SetFlash(w, msg)
			log.Println(msg)
			http.Redirect(w, r, "/user/login", http.StatusFound)
			return
		}

		// Comparing the password with the hash
		err = bcrypt.CompareHashAndPassword(hash, password)
		if err != nil {
			msg := "incorrect password"
			SetFlash(w, msg)
			log.Println(msg, err, string(password[:]), string(hash[:]))
			http.Redirect(w, r, "/user/login", http.StatusFound)
			return
		}
		log.Println("DEBUG: logging in as user:", player)
		userID := player.ID
		ip := RequestIP(r)
		log.Println("sessionservice", s.SessionService)
		session, err := s.SessionService.New(userID, ip, time.Hour*24*30)
		if err != nil {
			s.Error(w, r, err)
			return
		}

		s.setSession(w, session)
		log.Printf("created session [player_id=%d]\n", userID)
		http.Redirect(w, r, UrlFor(player, "show"), http.StatusFound)
	})
}
