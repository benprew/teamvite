package http

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	teamvite "github.com/benprew/teamvite"
	"golang.org/x/crypto/bcrypt"
)

type playerShowParams struct {
	Player *teamvite.Player
	IsUser bool
	Teams  []teamvite.PlayerTeam
	Games  []*teamvite.Game
}

func (s *Server) playerShow() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := teamvite.UserFromContext((r.Context()))
		player := teamvite.PlayerFromContext(r.Context())
		template := teamvite.TemplateFromContext(r.Context())
		teams, err := s.PlayerService.Teams(r.Context())
		if err != nil {
			s.Error(w, r, err)
			return
		}

		games, _, err := s.GameService.FindGames(
			r.Context(),
			teamvite.GameFilter{PlayerID: player.ID, Time: time.Now().Unix()})
		if err != nil {
			s.Error(w, r, err)
			return
		}

		templateParams := playerShowParams{
			Player: player,
			IsUser: *player == *user,
			Teams:  teams,
			Games:  games,
		}
		log.Printf("playerShow: rendering template: %s\n", template)
		s.RenderTemplate(w, r, template, templateParams)
	})
}

func (s *Server) PlayerEdit() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := teamvite.UserFromContext((r.Context()))
		player := teamvite.PlayerFromContext(r.Context())
		template := teamvite.TemplateFromContext(r.Context())

		if user == nil || player == nil || player.ID != user.ID {
			s.Error(w, r, fmt.Errorf("must be logged in as player"))
			return
		}

		teams, err := s.PlayerService.Teams(r.Context())
		if err != nil {
			s.Error(w, r, err)
			return
		}

		games, _, err := s.GameService.FindGames(
			r.Context(),
			teamvite.GameFilter{PlayerID: player.ID, Time: time.Now().Unix()})
		if err != nil {
			s.Error(w, r, err)
			return
		}

		templateParams := playerShowParams{
			Player: player,
			IsUser: *player == *user,
			Teams:  teams,
			Games:  games,
		}
		log.Printf("playerEdit: rendering template: %s, params: %v\n", template, templateParams)
		if err = s.RenderTemplate(w, r, template, templateParams); err != nil {
			s.Error(w, r, err)
			return
		}
	})
}

func ReminderID(teamID uint64) string {
	return fmt.Sprintf("reminders_%d", teamID)
}

func (s *Server) PlayerUpdate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			s.Error(w, r, err)
			return
		}

		log.Println("=======PLAYER UPDATE FORM")
		log.Println(r.PostForm)

		log.Printf("playerUpdate\n")

		user := teamvite.UserFromContext((r.Context()))
		player := teamvite.PlayerFromContext(r.Context())

		if *player != *user {
			http.Error(w, "Must be logged in as player", http.StatusUnauthorized)
			return
		}

		player.Name = r.PostForm.Get("name")
		player.Email = r.PostForm.Get("email")
		tel := teamvite.UnTelify(r.PostForm.Get("phone"))
		if tel != -1 {
			player.Phone = tel
		} else {
			log.Printf("[WARN] invalid phone number: %s\n", r.PostForm.Get("phone"))

			SetFlash(w, "Invalid phone number, must be 10 digits")
			http.Redirect(w, r, UrlFor(user, "edit"), http.StatusFound)
			return
		}
		pass := r.PostForm.Get("password")
		if pass != "" {
			pHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Printf("Updating password to: %s\n", string(pHash[:]))
			player.Password = sql.NullString{String: string(pHash[:]), Valid: true}
		}

		// Note: updates happen whether things have changed or not
		if err := s.PlayerService.UpdatePlayer(r.Context()); err != nil {
			s.Error(w, r, err)
			return
		}

		// Update reminders
		teams, err := s.PlayerService.Teams(r.Context())
		if err != nil {
			s.Error(w, r, err)
			return
		}
		for _, pt := range teams {
			reminders := r.Form[ReminderID(pt.Team.ID)]
			log.Println("reminders:", reminders)
			pt.RemindEmail = false
			pt.RemindSMS = false
			for _, r := range reminders {
				log.Println("reminder:", r)
				log.Println(pt)
				if r == "email" {
					pt.RemindEmail = true
				}
				if r == "sms" {
					pt.RemindSMS = true
				}
			}
			log.Println(pt)
			if err := s.PlayerService.UpdatePlayerTeam(r.Context(), &pt); err != nil {
				s.Error(w, r, err)
				return
			}
		}

		http.Redirect(w, r, UrlFor(user, "show"), http.StatusFound)
	})
}
