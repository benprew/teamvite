package http

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	teamvite "github.com/benprew/teamvite"
)

func UrlFor(i teamvite.Item, action string) string {
	id := i.ItemID()
	name := i.ItemType()
	return fmt.Sprintf("/%s/%d/%s", name, id, action)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /favicon.ico", serveStatic)
	mux.HandleFunc("GET /robots.txt", serveStatic)
	mux.HandleFunc("GET /css/all.css", serveStatic)
	mux.HandleFunc("GET /css/marx.min.css", serveStatic)
	mux.Handle("GET /", s.routeWithMiddleware(s.root()))

	mux.Handle("GET /sms", s.SMS())
	mux.Handle("POST /sms", s.SMS())
	mux.Handle("POST /test_sms_receiver/Accounts/{id}/Messages.json", s.TestSMSReceiver())

	mux.Handle("GET /user/login", s.userLogin())
	mux.Handle("POST /user/login", s.userLoginPost())
	mux.Handle("GET /user/logout", s.routeWithMiddleware(s.userLogout()))

	mux.Handle("GET /player/{id}/show", s.routeWithMiddleware(s.playerShow()))
	mux.Handle("GET /player/{id}/edit", s.routeWithMiddleware(s.PlayerEdit()))
	mux.Handle("POST /player/{id}/edit", s.routeWithMiddleware(s.PlayerUpdate()))
	mux.Handle("PATCH /player/{id}/edit", s.routeWithMiddleware(s.PlayerUpdate()))

	mux.Handle("GET /team", s.routeWithMiddleware(s.teamList()))
	mux.Handle("GET /team/{id}/show", s.routeWithMiddleware(s.teamShow()))
	mux.Handle("GET /team/{id}/edit", s.routeWithMiddleware(s.teamEdit()))
	mux.Handle("POST /team/{id}/add_player", s.routeWithMiddleware(s.teamAddPlayer()))
	mux.Handle("POST /team/{id}/remove_player", s.routeWithMiddleware(s.teamRemovePlayer()))
	mux.Handle("GET /team/{id}/calendar.ics", s.routeWithMiddleware(s.teamCalendar()))
	mux.Handle("POST /team", s.routeWithMiddleware(s.teamCreate()))

	// Handles game responses.  Done as a GET so you can follow links in email
	mux.Handle("GET /game/{id}/show", s.routeWithMiddleware(s.gameShow()))
	mux.Handle("POST /game", s.routeWithMiddleware(s.GameCreate()))

	// JSON APIs
	mux.Handle("GET /season", s.routeWithMiddleware(s.SeasonList()))
	mux.Handle("GET /division", s.routeWithMiddleware(s.DivisionList()))

	return mux
}

func (s *Server) routeWithMiddleware(handler http.Handler) http.Handler {
	return s.sessionMiddleware(s.routeModelMiddleware(handler))
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

func serveStatic(w http.ResponseWriter, r *http.Request) {
	staticSub, err := fs.Sub(static, "static")
	if err != nil {
		log.Printf("Failed to get subdir of static: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	handler := http.FileServer(http.FS(staticSub))
	handler.ServeHTTP(w, r)
}
