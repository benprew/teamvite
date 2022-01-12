package main

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	"github.com/swithek/sessionup"
)

type server struct {
	DB     *sqlx.DB
	Router *httprouter.Router
	Mgr    *sessionup.Manager
}

func (s *server) routes() http.Handler {
	r := httprouter.New()
	r.GET("/css/*filepath", serveStatic)
	r.GET("/favicon.ico", serveStatic)
	r.GET("/robots.txt", serveStatic)
	r.Handler("GET", "/", s.Mgr.Public(s.root()))

	r.Handler("GET", "/send_game_reminders", s.Mgr.Public(s.SendGameReminders()))

	r.Handler("GET", "/user/login", s.Mgr.Public(s.userLogin()))
	r.Handler("POST", "/user/login", s.Mgr.Public(s.userLoginPost()))
	r.Handler("GET", "/user/logout", s.Mgr.Auth(s.userLogout()))

	r.Handler("GET", "/player/:id/show", s.Mgr.Public(s.playerShow()))
	// r.Handler("GET", "/player/:id/edit", s.Mgr.Auth(s.playerEdit()))
	// r.Handler("POST", "/player/:id/edit", playerUpdate)
	// r.Handler("PATCH", "/player/:id/edit", playerUpdate)
	// r.Handler("GET", "/player", playerList)

	r.Handler("GET", "/team", s.Mgr.Public(s.teamList()))
	r.Handler("GET", "/team/:id/show", s.Mgr.Public(s.teamShow()))
	r.Handler("GET", "/team/:id/edit", s.Mgr.Public(s.teamEdit())) // auth is handled in teamEdit
	// r.Handler("PATCH", "/team/:id/edit", s.Mgr.Auth(s.teamUpdate()))
	r.Handler("POST", "/team/:id/add_player", s.Mgr.Auth(s.teamAddPlayer()))
	r.Handler("POST", "/team/:id/remove_player", s.Mgr.Auth(s.teamRemovePlayer()))
	r.Handler("GET", "/team/:id/calendar.ics", s.Mgr.Public(s.teamCalendar()))
	r.Handler("POST", "/team", s.Mgr.Public(s.teamCreate()))

	// Handles game responses.  Done as a GET so you can follow links in email
	r.Handler("GET", "/game/:id/show", s.Mgr.Public(s.gameShow()))
	r.Handler("POST", "/game", s.Mgr.Public(s.GameCreate()))

	// JSON APIs
	r.Handler("GET", "/season", s.Mgr.Public(s.SeasonList()))
	r.Handler("GET", "/division", s.Mgr.Public(s.DivisionList()))

	s.Router = r

	return r
}

type appHandler func(http.ResponseWriter, *http.Request) error

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (s *server) root() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
	})
}

//go:embed static
var static embed.FS

func serveStatic(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	staticSub, err := fs.Sub(static, "static")
	checkErr(err, "subdir of static")
	handler := http.FileServer(http.FS(staticSub))
	handler.ServeHTTP(w, r)
}
