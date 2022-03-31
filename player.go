package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type player struct {
	Id       int            `db:"id,primarykey,autoincrement"`
	Name     string         `db:"name,size:64"`
	Email    string         `db:"email,size:128"`
	Password sql.NullString `db:"password,size:256,default:''"`
	Phone    int            `db:"phone"`
}

func (p *player) itemId() int {
	return p.Id
}

func (p *player) itemType() string {
	return "player"
}

func (p *playerTeam) itemId() int {
	return p.Id
}

func (p *playerTeam) itemType() string {
	return "team"
}

func (p *player) Teams(DB *sqlx.DB) (teams []team) {
	err := DB.Select(&teams, "select t.* from teams t join players_teams on t.id = team_id where player_id = ?", p.Id)
	checkErr(err, "player.Teams")
	return teams
}

func findByEmail(DB *sqlx.DB, email string) (p player, err error) {
	err = DB.Get(&p, "select * from players where email = ?", email)
	return p, err
}

type playerTeam struct {
	Id          int    `db:"id,primarykey,autoincrement" json:"id"`
	Name        string `db:"name,size:128" json:"name"`
	DivisionId  int    `db:"division_id" json:"division_id"`
	IsManager   bool   `db:"is_manager"`
	RemindSMS   bool   `db:"remind_sms"`
	RemindEmail bool   `db:"remind_email"`
}

type PlayerShowParams struct {
	Player *player
	IsUser bool
	Teams  []playerTeam
	Games  []UpcomingGame
}

func (s *server) playerShow() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := BuildContext(s.DB, r)
		if err != nil {
			log.Printf("[WARN] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}

		p := ctx.Model.(player)
		u := s.GetUser(r)
		templateParams := PlayerShowParams{
			Player: &p,
			IsUser: p == u,
			Teams:  playerTeams(s.DB, p),
			Games:  playerUpcomingGames(s.DB, &p),
		}
		log.Printf("playerShow: rending template: %s\n", ctx.Template)
		s.RenderTemplate(w, r, ctx.Template, templateParams)
	})
}

func (s *server) PlayerEdit() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := BuildContext(s.DB, r)
		if err != nil {
			log.Printf("[WARN] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}

		u := s.GetUser(r)
		p := ctx.Model.(player)

		if p != u {
			log.Printf("[ERROR] buildRoute: %s\n", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		templateParams := PlayerShowParams{
			Player: &p,
			IsUser: p == u,
			Teams:  playerTeams(s.DB, p),
			Games:  playerUpcomingGames(s.DB, &p),
		}
		log.Printf("playerEdit: rending template: %s\n", ctx.Template)
		s.RenderTemplate(w, r, ctx.Template, templateParams)
	})
}

func playerTeams(DB *sqlx.DB, p player) (pt []playerTeam) {
	err := DB.Select(&pt, "select t.id, t.name, t.division_id, pt.is_manager, pt.remind_email, pt.remind_sms from teams t join players_teams pt on t.id = pt.team_id where player_id = ?", p.Id)
	checkErr(err, "playerTeams")
	return
}

func ReminderId(teamId int) string {
	return fmt.Sprintf("reminders_%d", teamId)
}

func (s *server) PlayerUpdate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := BuildContext(s.DB, r)
		if err != nil {
			log.Printf("[WARN] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}

		if err := r.ParseForm(); err != nil {
			checkErr(err, "parsing add_player form")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Println("=======PLAYER UPDATE FORM")
		log.Println(r.PostForm)

		log.Printf("playerUpdate\n")

		u := s.GetUser(r)
		p := ctx.Model.(player)

		if p != u {
			log.Printf("[ERROR] buildRoute: %s\n", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		p.Name = r.PostForm.Get("name")
		p.Email = r.PostForm.Get("email")
		tel := UnTelify(r.PostForm.Get("phone"))
		if tel != -1 {
			p.Phone = tel
		} else {
			log.Printf("[WARN] invalid phone number: %s\n", r.PostForm.Get("phone"))
			s.SetMessage(u, "Invalid phone number, must be 10 digits")
			http.Redirect(w, r, urlFor(&u, "edit"), http.StatusFound)
			return
		}
		pass := r.PostForm.Get("password")
		if pass != "" {
			pHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Printf("Updating password to: %s\n", pHash)
			p.Password = sql.NullString{String: string(pHash[:]), Valid: true}
		}

		_, err = s.DB.NamedExec("update players set name=:name, email=:email, phone=:phone, password=:password  where id = :id", p)
		checkErr(err, "update player")

		for _, pt := range playerTeams(s.DB, p) {
			reminders := r.Form[ReminderId(pt.Id)]
			log.Println("reminders:", reminders)
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
			_, err := s.DB.NamedExec(
				fmt.Sprintf("update players_teams set remind_email=:remind_email, remind_sms=:remind_sms where player_id = %d and team_id = :id", p.Id),
				pt)
			checkErr(err, "update pt")
		}

		http.Redirect(w, r, urlFor(&u, "show"), http.StatusFound)
	})
}

type playerCtx struct {
	Player   player
	Template string
}
