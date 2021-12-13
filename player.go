package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
)

type player struct {
	Id          int            `db:"id,primarykey,autoincrement"`
	Name        string         `db:"name,size:64"`
	Email       string         `db:"email,size:128"`
	Password    sql.NullString `db:"password,size:256,default:''"`
	PhoneNumber sql.NullString `db:"phone_number,size:32"`
}

func (p *player) itemId() int {
	return p.Id
}

func (p *player) itemType() string {
	return "player"
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

type PlayerShowParams struct {
	Player     player
	Teams      []team
	Games      []UpcomingGame
	ShowStatus bool
}

func (s *server) playerShow() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := BuildContext(s.DB, r)
		if err != nil {
			log.Printf("[ERROR] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}

		p := ctx.Model.(player)
		if p.Id == 0 {
			http.NotFound(w, r)
			return
		}
		templateParams := PlayerShowParams{
			Player:     p,
			Teams:      p.Teams(s.DB),
			ShowStatus: p == s.getUser(r),
			Games:      playerUpcomingGames(s.DB, &p),
		}
		log.Printf("playerShow: rending template: %s\n", ctx.Template)
		s.RenderTemplate(w, r, ctx.Template, templateParams)
	})
}

type playerCtx struct {
	Player   player
	Template string
}

// func buildPlayerContext(DB *sqlx.DB, r *http.Request) (playerCtx, error) {
// 	routeInfo, err := buildRouteInfo(r)
// 	if err != nil {
// 		return playerCtx{}, err
// 	}

// 	// This handles team/list which doesn't have a team associated with it
// 	nullT := team{}
// 	t := loadTeam(DB, routeInfo.Id)
// 	if routeInfo.Id > 0 && t == nullT {
// 		return teamCtx{}, fmt.Errorf("no team found for id: %d", routeInfo.Id)
// 	}

// 	return playerCtx{
// 		Player:   loadPlayer(DB, routeInfo.Id),
// 		Template: routeInfo.Template,
// 	}, nil
// }
