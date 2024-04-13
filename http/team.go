package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	teamvite "github.com/benprew/teamvite"
)

type teamShowParams struct {
	Team      *teamvite.Team
	Players   []*teamvite.Player
	Games     []*teamvite.Game
	IsManager bool
}

type teamListParams struct {
	Teams []*teamvite.Team
	Q     string // query
}

func (s *Server) teamShow() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		team := teamvite.TeamFromContext(ctx)
		players, _, err := s.PlayerService.FindPlayers(r.Context(), teamvite.PlayerFilter{TeamID: &team.ID})
		if err != nil {
			s.Error(w, r, err)
			return
		}
		games, _, err := s.GameService.FindGames(
			r.Context(),
			teamvite.GameFilter{TeamID: team.ID, Time: time.Now().Unix()})
		if err != nil {
			s.Error(w, r, err)
			return
		}

		templateParams := teamShowParams{
			Players:   players,
			Team:      team,
			Games:     games,
			IsManager: s.isManager(r.Context(), team),
		}

		s.RenderTemplate(w, r, teamvite.TemplateFromContext(ctx), templateParams)
	})
}

func (s *Server) teamEdit() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		team := teamvite.TeamFromContext(ctx)
		// TODO: should this be NotAuthorized instead?
		if !s.isManager(r.Context(), team) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}
		players, _, err := s.PlayerService.FindPlayers(r.Context(), teamvite.PlayerFilter{TeamID: &team.ID})
		if err != nil {
			s.Error(w, r, err)
			return
		}
		games, _, err := s.GameService.FindGames(
			r.Context(),
			teamvite.GameFilter{TeamID: team.ID, Time: time.Now().Unix()})
		if err != nil {
			s.Error(w, r, err)
			return
		}

		templateParams := teamShowParams{
			Team:    team,
			Players: players,
			Games:   games,
		}
		s.RenderTemplate(w, r, teamvite.TemplateFromContext(ctx), templateParams)
	})

}

func (s *Server) teamList() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// TODO: Add filtering by division name (for creating games)
		nameQuery := ""
		q := r.URL.Query().Get("name")
		nameQuery = "%" + q + "%"

		teams, _, err := s.TeamService.FindTeams(r.Context(), teamvite.TeamFilter{Name: &nameQuery})
		if err != nil {
			s.Error(w, r, err)
			return
		}

		switch r.Header.Get("Content-type") {
		case JSON:
			w.Header().Set("Content-Type", JSON)
			json.NewEncoder(w).Encode(teams)
		default:
			templateParams := teamListParams{
				Q:     q,
				Teams: teams,
			}
			s.RenderTemplate(w, r, teamvite.TemplateFromContext(ctx), templateParams)
		}
	})
}

func (s *Server) teamAddPlayer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		team := teamvite.TeamFromContext(ctx)

		if !s.isManager(r.Context(), team) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}

		if err := r.ParseForm(); err != nil {
			s.Error(w, r, teamvite.Errorf(teamvite.EINVALID, "Invalid request."))
			return
		}

		email := r.PostForm.Get("email")
		name := r.PostForm.Get("name")

		if email == "" {
			s.Error(w, r, teamvite.Errorf(teamvite.EINVALID, "Email is required."))
			return
		}
		p, n, err := s.PlayerService.FindPlayers(r.Context(), teamvite.PlayerFilter{Email: email, Limit: 1})
		if err != nil || n > 1 {
			s.Error(w, r, err)
			return
		}
		var player teamvite.Player

		if n == 0 {
			player = teamvite.Player{Name: name, Email: email}
			err = s.PlayerService.CreatePlayer(r.Context(), &player)
			if err != nil {
				s.Error(w, r, err)
				return
			}
		} else {
			player = *p[0]
		}

		err = s.TeamService.AddPlayer(teamvite.NewContextWithUser(r.Context(), &player), team)
		if err != nil {
			s.Error(w, r, err)
			return
		}
		http.Redirect(w, r, UrlFor(team, "edit"), http.StatusFound)
	})
}

func (s *Server) teamRemovePlayer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		team := teamvite.TeamFromContext(ctx)
		if !s.isManager(r.Context(), team) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}

		if err := r.ParseForm(); err != nil {
			s.Error(w, r, teamvite.Errorf(teamvite.EINVALID, "invalid form %v", err))
			return
		}
		playerID, _ := strconv.Atoi(r.PostForm.Get("player_id"))
		player, err := s.PlayerService.FindPlayerByID(r.Context(), uint64(playerID))
		if err != nil {
			s.Error(w, r, err)
			return
		}

		if err := s.TeamService.RemovePlayer(teamvite.NewContextWithUser(r.Context(), player), team); err != nil {
			s.Error(w, r, err)
			return
		}
		http.Redirect(w, r, UrlFor(team, "edit"), http.StatusFound)
	})
}

type TeamCalendarParams struct {
	Team       teamvite.Team
	Games      []CalendarGame
	CreateTime time.Time
}

type CalendarGame struct {
	Url         string
	Description string
	Start       *time.Time
	End         *time.Time
}

func (s *Server) teamCalendar() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		team := teamvite.TeamFromContext(ctx)
		games, _, err := s.GameService.FindGames(r.Context(), teamvite.GameFilter{TeamID: team.ID})
		if err != nil {
			s.Error(w, r, err)
			return
		}

		cg := []CalendarGame{}

		for _, g := range games {
			e := g.Time.Add(time.Minute * teamvite.GameLength)
			cg = append(cg, CalendarGame{
				Url:         fmt.Sprintf("https://www.teamvite.com%s", UrlFor(g, "show")),
				Description: g.Description,
				Start:       g.Time,
				End:         &e,
			})
		}

		params := TeamCalendarParams{
			Team:       *team,
			Games:      cg,
			CreateTime: time.Now(),
		}

		filename := "views/team/calendar.ics.tmpl"
		// parse as text/template to avoid html escaping
		t := template.Must(template.ParseFS(views, filename))
		w.Header().Set("Content-Type", "text/calendar;charset=utf-8")
		t.ExecuteTemplate(w, "calendar.ics.tmpl", params)
	})
}

// curl -i -X POST --silent \
// http://teamvitedev.com:8080/team \
// -H 'Content-Type: application/json' \
// --data '{"name":"Test Team","division_id":1}'
func (s *Server) teamCreate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-type")
		var t teamvite.Team
		switch contentType {
		case JSON:
			if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
				return
			}
			err := s.TeamService.CreateTeam(r.Context(), &t)
			if err != nil {
				s.Error(w, r, err)
				return
			}
			w.Header().Set("Content-Type", JSON)
			json.NewEncoder(w).Encode(t)
		default:
			if err := r.ParseForm(); err != nil {
				s.Error(w, r, teamvite.Errorf(teamvite.EINVALID, "%s", err))
				return
			}
			d, _ := strconv.Atoi(r.PostForm.Get("division_id"))
			t = teamvite.Team{
				Name:       r.PostForm.Get("name"),
				DivisionID: uint64(d),
			}
			err := s.TeamService.CreateTeam(r.Context(), &t)
			if err != nil {
				s.Error(w, r, err)
				return
			}
			http.Redirect(w, r, UrlFor(&t, "show"), http.StatusFound)
		}
	})
}

func playerEmails(players []*teamvite.Player) string {
	emails := []string{}
	for _, p := range players {
		emails = append(emails, p.Email)
	}

	return strings.Join(emails, ",")
}

func (s *Server) isManager(ctx context.Context, team *teamvite.Team) bool {
	return s.TeamService.IsManagedBy(ctx, team)
}
