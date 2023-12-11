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
