package http

import (
	"encoding/json"
	"log"
	"net/http"

	teamvite "github.com/benprew/teamvite"
	"github.com/julienschmidt/httprouter"
)

const JSON = "application/json"
const SESSION_KEY = "teamvite-session"

type Server struct {
	server *http.Server
	router *httprouter.Router

	GameService     teamvite.GameService
	TeamService     teamvite.TeamService
	PlayerService   teamvite.PlayerService
	DivisionService teamvite.DivisionService
	SeasonService   teamvite.SeasonService

	SessionService teamvite.SessionService

	// bind address and domainname for the listener
	Addr   string
	Domain string
}

func NewServer() *Server {
	s := &Server{
		router: httprouter.New(),
		server: &http.Server{},
	}

	return s
}

func (s *Server) Open() (err error) {
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", s.routes()))
	return nil
}

// Run as server
// func serv(db *ql.QueryLogger) {
// 	s := Server{
// 		DB:       db,
// 		MsgStore: NewMessageStore(),
// 	}

// 	s.Open()
// }

// ErrorResponse represents a JSON structure for error output.
type ErrorResponse struct {
	Error string `json:"error"`
}

type ErrorParams struct {
	StatusCode int
	Header     string
	Message    string
}

// Error prints & optionally logs an error message.
func (s *Server) Error(w http.ResponseWriter, r *http.Request, err error) {
	// Extract error code & message.
	code, message := teamvite.ErrorCode(err), teamvite.ErrorMessage(err)

	// Log & report internal errors.
	if code == teamvite.EINTERNAL {
		teamvite.ReportError(r.Context(), err, r)
		LogError(r, err)
	}

	// Print user message to response based on reqeust accept header.
	switch r.Header.Get("Accept") {
	case "application/json":
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(ErrorStatusCode(code))
		json.NewEncoder(w).Encode(&ErrorResponse{Error: message})

	default:
		w.WriteHeader(ErrorStatusCode(code))
		s.RenderTemplate(w, r, "error.tmpl", &ErrorParams{
			StatusCode: ErrorStatusCode(code),
			Header:     "An error has occurred.",
			Message:    message,
		})
	}
}

func (s *Server) routeModelMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeInfo, err := buildRouteInfo(r.URL.EscapedPath())
		if err != nil {
			s.Error(w, r, err)
			return
		}
		if routeInfo.ModelType == "player" {
			player, err := s.PlayerService.FindPlayerByID(r.Context(), routeInfo.ID)
			if err != nil {
				s.Error(w, r, err)
				return
			}
			r = r.WithContext(teamvite.NewContextWithPlayer(r.Context(), routeInfo.Template, player))
		} else if routeInfo.ModelType == "team" {
			team, err := s.TeamService.FindTeamByID(r.Context(), routeInfo.ID)
			if err != nil {
				s.Error(w, r, err)
				return
			}
			r = r.WithContext(teamvite.NewContextWithTeam(r.Context(), routeInfo.Template, team))
		} else if routeInfo.ModelType == "game" {
			game, err := s.GameService.FindGameByID(r.Context(), routeInfo.ID)
			if err != nil {
				s.Error(w, r, err)
				return
			}
			r = r.WithContext(teamvite.NewContextWithGame(r.Context(), routeInfo.Template, game))
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := &teamvite.Player{}
		var sids []string

		// try loading user from request token
		sids = append(sids, sidFromRequest(r))
		// try loading user from cookie
		sids = append(sids, SidsFromCookie(r, SESSION_KEY)[:]...)

		for _, sid := range sids {
			// exit once we have a user, which may be from the token above
			if user.ID != 0 {
				break
			}
			sess, err := s.SessionService.Load(sid, RequestIP(r))
			if err != nil {
				log.Printf("Failed to load session: %s\n", err)
				continue
			}
			user, err = s.PlayerService.FindPlayerByID(r.Context(), sess.PlayerID)
			if err != nil {
				log.Printf("Failed to load player: %s\n", err)
			}
		}
		r = r.WithContext(teamvite.NewContextWithUser(r.Context(), user))
		next.ServeHTTP(w, r)
	})
}

func sidFromRequest(r *http.Request) string {
	sid := r.URL.Query().Get(SESSION_KEY)
	log.Printf("Finding player with token: %s\n", sid)
	return sid
	// sess, err = LoadSession(s.DB.Queryer, sid, RequestIP(req))
	// checkErr(err, "getting player from token")

	// sids, err := r.Cookie(SESSION_KEY)
	// if err != nil {
	// 	return nil, err
	// }
	// return []string{sids.Value}, nil
}

// lookup of application error codes to HTTP status codes.
var codes = map[string]int{
	teamvite.ECONFLICT:       http.StatusConflict,
	teamvite.EINVALID:        http.StatusBadRequest,
	teamvite.ENOTFOUND:       http.StatusNotFound,
	teamvite.ENOTIMPLEMENTED: http.StatusNotImplemented,
	teamvite.EUNAUTHORIZED:   http.StatusUnauthorized,
	teamvite.EINTERNAL:       http.StatusInternalServerError,
}

// ErrorStatusCode returns the associated HTTP status code for a WTF error code.
func ErrorStatusCode(code string) int {
	if v, ok := codes[code]; ok {
		return v
	}
	return http.StatusInternalServerError
}

// LogError logs an error with the HTTP route information.
func LogError(r *http.Request, err error) {
	log.Printf("[http] error: %s %s: %s", r.Method, r.URL.Path, err)
}
