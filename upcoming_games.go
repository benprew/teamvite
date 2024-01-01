package teamvite

// func (g *UpcomingGame) itemID() int {
// 	return g.ID
// }

// func (g *UpcomingGame) itemType() string {
// 	return "game"
// }

// type UpcomingGame struct {
// 	ID          int
// 	Date        *time.Time
// 	Description string
// 	Responses   []GameResponse
// }

// Players that have responded Yes/No/Maybe/No Reply
// TODO: Name should probably be an enum

// func teamUpcomingGames(DB *QueryLogger, t Team) []UpcomingGame {
// 	return upcomingGames(DB, []Team{t})
// }

// // Only show responses on player homepage and on team page when player plays on
// // that team
// func playerUpcomingGames(DB *QueryLogger, p *Player) []UpcomingGame {
// 	teams := []Team{}
// 	err := DB.Select(
// 		&teams,
// 		"select t.* from teams t join players_teams on team_id = t.id where player_id = ?",
// 		p.ID)
// 	checkErr(err, "player upcoming games teams filter")
// 	return upcomingGames(DB, teams)
// }

// func upcomingGames(DB *QueryLogger, teams []Team) (games []UpcomingGame) {
// 	teamIDs := []int{}

// 	if len(teams) == 0 {
// 		return games
// 	}

// 	for _, t := range teams {
// 		teamIDs = append(teamIDs, t.ID)
// 	}

// 	query, args, err := sqlx.In(
// 		`select g.id as id,
//                  g.time as date,
//                  g.description as description
//                  from teams t
//                  join games g on g.team_id = t.id
//                  where g.time >= ? and t.id in (?)
//                  order by date;`,
// 		// TODO: Fix upcoming games time
// 		// hack to get upcoming games for today
// 		// games are stored in local time, but sever time is unix time...
// 		time.Now().Add(10*time.Hour*-1).Unix(),
// 		teamIDs,
// 	)
// 	checkErr(err, "binding to teams")
// 	query = DB.Rebind(query)

// 	err = DB.Select(&games, query, args...)
// 	checkErr(err, "upcoming games")

// 	log.Printf("Got %d games\n", len(games))
// 	ResponsesForGames(DB, games)

// 	return games
// }

// func ResponsesForGames(DB *QueryLogger, games []UpcomingGame) {
// 	for i, game := range games {
// 		r := ResponsesForGame(DB, game.ID)
// 		games[i].Responses = r
// 	}
// }
