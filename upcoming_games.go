package main

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func (g *UpcomingGame) itemId() int {
	return g.Id
}

func (g *UpcomingGame) itemType() string {
	return "game"
}

type UpcomingGame struct {
	Id          int
	Date        *time.Time
	Description string
	Responses   []GameResponse
}

type GameResponse struct {
	Name    string
	Players []string
}

func teamUpcomingGames(DB *sqlx.DB, t team) []UpcomingGame {
	return upcomingGames(DB, []team{t})
}

// Only show responses on player homepage and on team page when player plays on
// that team
func playerUpcomingGames(DB *sqlx.DB, p *player) []UpcomingGame {
	teams := []team{}
	err := DB.Select(
		&teams,
		"select t.* from teams t join players_teams on team_id = t.id where player_id = ?",
		p.Id)
	checkErr(err, "player upcoming games teams filter")
	return upcomingGames(DB, teams)
}

func upcomingGames(DB *sqlx.DB, teams []team) (games []UpcomingGame) {
	teamIds := []int{}

	if len(teams) == 0 {
		return games
	}

	for _, t := range teams {
		teamIds = append(teamIds, t.Id)
	}

	query, args, err := sqlx.In(
		`select g.id as id,
                 g.time as date,
                 g.description as description
                 from teams t
                 join games g on g.team_id = t.id
                 where g.time >= ? and t.id in (?)
                 order by date;`,
		time.Now().Truncate(24*time.Hour).Unix(),
		teamIds,
	)
	checkErr(err, "binding to teams")
	query = DB.Rebind(query)

	err = DB.Select(&games, query, args...)
	checkErr(err, "upcoming games")

	fmt.Printf("Got %d games\n", len(games))
	responsesForGames(DB, games)

	return games
}

func responsesForGames(DB *sqlx.DB, games []UpcomingGame) {
	for i, game := range games {
		r := responsesForGame(DB, game.Id)
		games[i].Responses = r
	}
}

func responsesForGame(DB *sqlx.DB, id int) []GameResponse {
	var r [4]GameResponse
	respMap := map[string]int{
		"Yes":      0,
		"No":       1,
		"Maybe":    2,
		"No Reply": 3,
	}

	for k, v := range respMap {
		r[v].Name = k
	}

	rows, err := DB.Queryx(`
select
  case
    when pg.status = 'N' then 'No'
    when pg.status = 'Y' then 'Yes'
    when pg.status = 'M' then 'Maybe'
    else 'No Reply'
  end as status,
  name
from games g
join players_teams pt using(team_id)
join players p on pt.player_id = p.id
left join players_games pg on pg.game_id = g.id and pg.player_id = p.id
where g.id = ?
order by status desc, name`, id)
	checkErr(err, "game show query")
	defer rows.Close()
	for rows.Next() {
		var status, name string
		rows.Scan(&status, &name)
		idx := respMap[status]
		r[idx].Players = append(r[idx].Players, name)
	}
	return r[:]
}
