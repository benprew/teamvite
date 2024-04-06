package http

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	teamvite "github.com/benprew/teamvite"
	"github.com/julienschmidt/httprouter"
)

func UrlFor(i teamvite.Item, action string) string {
	id := i.ItemID()
	name := i.ItemType()
	return fmt.Sprintf("/%s/%d/%s", name, id, action)
}

func (s *Server) routes() http.Handler {
	r := httprouter.New()
	r.GET("/css/*filepath", serveStatic)
	r.GET("/favicon.ico", serveStatic)
	r.GET("/robots.txt", serveStatic)
	r.Handler("GET", "/", s.routeWithMiddleware(s.root()))

	r.Handler("GET", "/sms", s.SMS())
	r.Handler("POST", "/sms", s.SMS())
	r.Handler("POST", "/test_sms_receiver/Accounts/:id/Messages.json", s.TestSMSReceiver())

	r.Handler("GET", "/user/login", s.userLogin())
	r.Handler("POST", "/user/login", s.userLoginPost())
	r.Handler("GET", "/user/logout", s.routeWithMiddleware(s.userLogout()))

	r.Handler("GET", "/player/:id/show", s.routeWithMiddleware(s.playerShow()))
	r.Handler("GET", "/player/:id/edit", s.routeWithMiddleware(s.PlayerEdit()))
	r.Handler("POST", "/player/:id/edit", s.routeWithMiddleware(s.PlayerUpdate()))
	r.Handler("PATCH", "/player/:id/edit", s.routeWithMiddleware(s.PlayerUpdate()))
	// r.Handler("GET", "/player", playerList)

	r.Handler("GET", "/team", s.routeWithMiddleware(s.teamList()))
	r.Handler("GET", "/team/:id/show", s.routeWithMiddleware(s.teamShow()))
	r.Handler("GET", "/team/:id/edit", s.routeWithMiddleware(s.teamEdit()))
	// r.Handler("PATCH", "/team/:id/edit", s.teamUpdate()))
	r.Handler("POST", "/team/:id/add_player", s.routeWithMiddleware(s.teamAddPlayer()))
	r.Handler("POST", "/team/:id/remove_player", s.routeWithMiddleware(s.teamRemovePlayer()))
	r.Handler("GET", "/team/:id/calendar.ics", s.routeWithMiddleware(s.teamCalendar()))
	r.Handler("POST", "/team", s.routeWithMiddleware(s.teamCreate()))

	// Handles game responses.  Done as a GET so you can follow links in email
	r.Handler("GET", "/game/:id/show", s.routeWithMiddleware(s.gameShow()))
	r.Handler("POST", "/game", s.routeWithMiddleware(s.GameCreate()))

	// JSON APIs
	r.Handler("GET", "/season", s.routeWithMiddleware(s.SeasonList()))
	r.Handler("GET", "/division", s.routeWithMiddleware(s.DivisionList()))

	return r
}

func (s *Server) routeWithMiddleware(handler http.Handler) http.Handler {
	return s.authMiddleware(s.routeModelMiddleware(handler))
}

func (s *Server) root() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := s.GetUser(r)

		if u == nil {
			http.Redirect(w, r, "/user/login", http.StatusFound)
			return
		}
		http.Redirect(w, r, UrlFor(u, "show"), http.StatusFound)
	})
}

//go:embed static
var static embed.FS

func serveStatic(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	staticSub, err := fs.Sub(static, "static")
	if err != nil {
		log.Printf("Failed to get subdir of static: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	handler := http.FileServer(http.FS(staticSub))
	handler.ServeHTTP(w, r)
}
