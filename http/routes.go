package http

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func urlFor(i Item, action string) string {
	id := i.itemID()
	name := i.itemType()
	return fmt.Sprintf("/%s/%d/%s", name, id, action)
}

func (s *Server) routes() http.Handler {
	r := httprouter.New()
	r.GET("/css/*filepath", serveStatic)
	r.GET("/favicon.ico", serveStatic)
	r.GET("/robots.txt", serveStatic)
	r.Handler("GET", "/", s.root())

	r.Handler("GET", "/sms", s.SMS())
	r.Handler("POST", "/sms", s.SMS())
	r.Handler("POST", "/test_sms_receiver/Accounts/:id/Messages.json", s.TestSMSReceiver())

	r.Handler("GET", "/send_game_reminders", s.SendGameReminders())

	r.Handler("GET", "/user/login", s.userLogin())
	r.Handler("POST", "/user/login", s.userLoginPost())
	r.Handler("GET", "/user/logout", s.userLogout())

	r.Handler("GET", "/player/:id/show", s.playerShow())
	r.Handler("GET", "/player/:id/edit", s.PlayerEdit())
	r.Handler("POST", "/player/:id/edit", s.PlayerUpdate())
	r.Handler("PATCH", "/player/:id/edit", s.PlayerUpdate())
	// r.Handler("GET", "/player", playerList)

	r.Handler("GET", "/team", s.teamList())
	r.Handler("GET", "/team/:id/show", s.teamShow())
	r.Handler("GET", "/team/:id/edit", s.teamEdit())
	// r.Handler("PATCH", "/team/:id/edit", s.teamUpdate()))
	r.Handler("POST", "/team/:id/add_player", s.teamAddPlayer())
	r.Handler("POST", "/team/:id/remove_player", s.teamRemovePlayer())
	r.Handler("GET", "/team/:id/calendar.ics", s.teamCalendar())
	r.Handler("POST", "/team", s.teamCreate())

	// Handles game responses.  Done as a GET so you can follow links in email
	r.Handler("GET", "/game/:id/show", s.gameShow())
	r.Handler("POST", "/game", s.GameCreate())

	// JSON APIs
	r.Handler("GET", "/season", s.SeasonList())
	r.Handler("GET", "/division", s.DivisionList())

	s.Router = r

	return r
}

type appHandler func(http.ResponseWriter, *http.Request) error

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) root() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := s.GetUser(r)
		if u.ID == 0 {
			http.Redirect(w, r, "/user/login", http.StatusFound)
		} else {
			http.Redirect(w, r, urlFor(&u, "show"), http.StatusFound)
		}
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
