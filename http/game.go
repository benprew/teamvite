package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	teamvite "github.com/benprew/teamvite"
)

type GameCtx struct {
	Game     *teamvite.Game
	Template string
}

type GameShowParams struct {
	User       teamvite.Player
	Game       teamvite.Game
	Responses  []*teamvite.GameResponse
	ShowStatus bool
}

func (s *Server) buildGameContext(r *http.Request) (GameCtx, error) {
	routeInfo, err := buildRouteInfo(r.URL.EscapedPath())
	if err != nil {
		return GameCtx{}, err
	}

	g, err := s.GameService.FindGameByID(r.Context(), routeInfo.ID)
	return GameCtx{Game: g, Template: routeInfo.Template}, err
}

// create assumes a JSON request
// curl -i -X POST --silent \
//   http://teamvitedev.com:8080/game \
//   -H 'Content-Type: application/json' \
//   --data '{"team_id":4369,"time": "2021-12-01T13:00:00Z", "season_id": 1}'

// -- data '{"team": "foobar", "division": "m2a", "season": "2022-Winter", "time":"2021-12-01T13:00:00Z", "description": "foobar vs fubar"}'
func (s *Server) GameCreate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-type")
		log.Println(contentType)
		if contentType != JSON {
			s.Error(
				w,
				r,
				&teamvite.Error{
					Code:    teamvite.EINVALID,
					Message: fmt.Sprintf("Content type: %s not supported", contentType),
				},
			)
			return
		}

		w.Header().Set("Content-Type", JSON)
		var g teamvite.Game
		if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
			s.Error(w, r, err)
			return
		}
		log.Printf("loaded game: %v\n", g)
		err := s.GameService.CreateGame(r.Context(), &g)
		if err != nil {
			s.Error(w, r, err)
			return
		}
		w.Header().Set("Content-Type", JSON)
		json.NewEncoder(w).Encode(g)
	})
}

func (s *Server) gameShow() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.buildGameContext(r)
		if err != nil {
			s.Error(w, r, err)
			return
		}

		g := ctx.Game
		userID := teamvite.PlayerIDFromContext(r.Context())

		status, ok := r.URL.Query()["status"]
		if ok {
			status := strings.ToUpper(status[0])
			msg := ""
			switch status[0:1] {
			case "Y":
				msg = "See you at the game!"
			case "N":
				msg = "Sorry you can't make it"
			case "M":
				msg = "Sh*t or get off the pot!"

			}
			s.GameService.SetStatus(r.Context(), g, status[0:1])
			SetFlash(w, msg)
		}
		_, n, err := s.GameService.FindGames(r.Context(), teamvite.GameFilter{PlayerID: &userID, ID: &g.ID})
		userGameStatus := n == 1
		log.Printf("PLAYER ON TEAM: %t, %d, %d\n", userGameStatus, userID, g.ID)
		if err != nil {
			s.Error(w, r, err)
			return
		}

		responses, err := s.GameService.ResponsesForGame(r.Context(), g)
		if err != nil {
			s.Error(w, r, err)
			return
		}

		templateParams := GameShowParams{
			Game:       *g,
			Responses:  responses,
			ShowStatus: userGameStatus,
		}
		s.RenderTemplate(w, r, ctx.Template, templateParams)
	})
}
