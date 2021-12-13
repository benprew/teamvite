package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

func (t *team) itemId() int {
	return t.Id
}

func (t *team) itemType() string {
	return "team"
}

func (t *team) Players(DB *sqlx.DB) []player {
	var p []player
	err := DB.Select(&p, "select p.* from players p join players_teams on p.id = player_id where team_id = ? order by p.name", t.Id)
	checkErr(err, "players for teams")
	return p
}

func (t *team) Games(DB *sqlx.DB) (g []game) {
	err := DB.Select(&g, "select * from games where team_id = ? order by time", t.Id)
	checkErr(err, "games for team")
	return
}

func (t *team) nextGame(DB *sqlx.DB) (g game, ok bool) {
	err := DB.Get(&g, "select * from games where team_id = ? and time >= strftime('%s', date('now')) order by time limit 1", t.Id)
	if err == nil {
		ok = true
	}
	checkErr(err, "nextGame")
	return
}

type team struct {
	Id         int    `db:"id,primarykey,autoincrement"`
	Name       string `db:"name,size:128"`
	DivisionId int    `db:"division_id"`
}

type teamShowParams struct {
	Team       *team
	Players    []player
	Games      []UpcomingGame
	ShowStatus bool
	IsManager  bool
}

type teamEditParams struct {
	Team       *team
	Players    []player
	Games      []UpcomingGame
	ShowStatus bool
}

type listTeam struct {
	Id       int
	Name     string
	Division string
}

type teamListParams struct {
	Teams []listTeam
	Q     string // query
}

type teamCtx struct {
	Team     team
	Template string
}

func buildTeamContext(DB *sqlx.DB, r *http.Request) (teamCtx, error) {
	ctx, err := BuildContext(DB, r)
	if err != nil {
		return teamCtx{}, err
	}

	return teamCtx{
		Team:     ctx.Model.(team),
		Template: ctx.Template,
	}, nil
}

func (s *server) teamShow() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := buildTeamContext(s.DB, r)
		if err != nil {
			log.Printf("[ERROR] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}

		var isManager bool
		var status bool
		row := s.DB.QueryRow("select is_manager, true from players_teams where team_id = ? and player_id = ?", ctx.Team.Id, s.getUser(r).Id)
		row.Scan(&isManager, &status)
		templateParams := teamShowParams{
			Players:    ctx.Team.Players(s.DB),
			Team:       &ctx.Team,
			Games:      teamUpcomingGames(s.DB, ctx.Team),
			ShowStatus: status,
			IsManager:  isManager,
		}

		s.RenderTemplate(w, r, ctx.Template, templateParams)
	})
}

func (s *server) teamEdit() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := buildTeamContext(s.DB, r)
		if err != nil {
			log.Printf("[ERROR] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}

		templateParams := teamEditParams{
			Team:       &ctx.Team,
			Players:    ctx.Team.Players(s.DB),
			Games:      teamUpcomingGames(s.DB, ctx.Team),
			ShowStatus: false,
		}
		s.RenderTemplate(w, r, ctx.Template, templateParams)
	})

}

func (s *server) teamList() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// params := httprouter.ParamsFromContext(r.Context())
		ctx, err := buildTeamContext(s.DB, r)
		if err != nil {
			log.Printf("[ERROR] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}

		t := []listTeam{}
		nameWhere := ""
		nameQuery := ""

		q, ok := r.URL.Query()["q"]
		if ok && q[0] != "" {
			nameWhere = "where t.name like ?"
			nameQuery = q[0]
		}

		query := fmt.Sprintf(
			"select t.id, t.name, d.name as division from teams t join divisions d on d.id = division_id %s order by 3, 2 limit 100",
			nameWhere)

		log.Println("query:", query, "val:", nameQuery)

		err = s.DB.Select(&t, query, fmt.Sprintf("%%%s%%", nameQuery))
		checkErr(err, "team list")

		templateParams := teamListParams{
			Q:     nameQuery,
			Teams: t,
		}
		s.RenderTemplate(w, r, ctx.Template, templateParams)
	})
}

func (s *server) teamAddPlayer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := buildTeamContext(s.DB, r)
		if err != nil {
			log.Printf("[ERROR] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}

		if err := r.ParseForm(); err != nil {
			checkErr(err, "parsing add_player form")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		email := r.PostForm.Get("email")
		name := r.PostForm.Get("name")

		if email == "" {
			log.Println("WARN: email is required to add player")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p, err := findByEmail(s.DB, email)
		checkErr(err, "add_player findByEmail")
		if errors.Is(err, sql.ErrNoRows) {
			_, err = s.DB.Exec("insert into players (email, name) values (?, ?)", email, name)
			checkErr(err, "add_player creating new player")
		}
		p, err = findByEmail(s.DB, email)
		checkErr(err, "add_player findByEmail")
		s.DB.Exec("insert into players_teams (player_id, team_id) values (?, ?)", p.Id, ctx.Team.Id)
		http.Redirect(w, r, urlFor(&ctx.Team, "edit"), http.StatusFound)
	})
}

func (s *server) teamRemovePlayer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := buildTeamContext(s.DB, r)
		if err != nil {
			log.Printf("[ERROR] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			checkErr(err, "parsing remove_player form")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		playerId := r.PostForm.Get("player_id")

		_, err = s.DB.Exec("delete from players_teams where player_id = ? and team_id = ?", playerId, ctx.Team.Id)
		checkErr(err, "teamRemovePlayer")
		http.Redirect(w, r, urlFor(&ctx.Team, "edit"), http.StatusFound)
	})
}

type TeamCalendarParams struct {
	Team       team
	Games      []CalendarGame
	CreateTime time.Time
}

type CalendarGame struct {
	Url         string
	Description string
	Start       *time.Time
	End         *time.Time
}

func (s *server) teamCalendar() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := buildTeamContext(s.DB, r)
		if err != nil {
			log.Printf("[ERROR] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}
		games := ctx.Team.Games(s.DB)
		cg := []CalendarGame{}

		for _, g := range games {
			e := g.Time.Add(time.Minute * GameLength)
			cg = append(cg, CalendarGame{
				Url:         fmt.Sprintf("https://www.teamvite.com%s", urlFor(&g, "show")),
				Description: g.Description,
				Start:       g.Time,
				End:         &e,
			})
		}

		params := TeamCalendarParams{
			Team:       ctx.Team,
			Games:      cg,
			CreateTime: time.Now(),
		}

		t := template.Must(template.ParseFiles("views/team/calendar.ics.tmpl"))
		w.Header().Set("Content-Type", "text/calendar;charset=utf-8")
		t.Execute(w, params)
	})
}

func (s *server) teamCreate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			checkErr(err, "parsing team create form")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		name := r.PostForm.Get("name")
		divisionId := r.PostForm.Get("division_id")

		_, err := s.DB.Exec("insert into teams (name, division_id) values (?, ?)", name, divisionId)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			checkErr(err, "creating team: ")
			return
		}
		t := team{}
		err = s.DB.Get(&t, "select * from teams where name = ? and division_id = ?", name, divisionId)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			checkErr(err, "creating team: ")
			return
		}

		http.Redirect(w, r, urlFor(&t, "show"), http.StatusFound)
	})
}

func playerEmails(players []player) string {
	emails := []string{}
	for _, p := range players {
		emails = append(emails, p.Email)
	}

	return strings.Join(emails, ",")
}
