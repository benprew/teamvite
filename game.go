package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

type game struct {
	Id          int        `db:"id,primarykey,autoincrement"`
	TeamId      int        `db:"team_id"`
	SeasonId    int        `db:"season_id"`
	Time        *time.Time `db:"time"`
	Description string     `db:"description"`
}

type playerGame struct {
	PlayerId     int    `db:"player_id"`
	GameId       int    `db:"game_id"`
	Status       string `db:"status"`
	ReminderSent bool   `db:"reminder_sent"`
}

func (g *game) itemId() int {
	return g.Id
}

func (g *game) itemType() string {
	return "game"
}

type gameShowParams struct {
	User      player
	Game      game
	Responses []GameResponse
}

const GameLength = 60

type gameCtx struct {
	Game     game
	Template string
}

func buildGameContext(DB *sqlx.DB, r *http.Request) (gameCtx, error) {
	ctx, err := BuildContext(DB, r)
	if err != nil {
		return gameCtx{}, err
	}

	return gameCtx{
		Game:     ctx.Model.(game),
		Template: ctx.Template,
	}, nil
}

func (s *server) gameShow() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := buildGameContext(s.DB, r)
		if err != nil {
			log.Printf("[ERROR] buildRoute: %s\n", err)
			http.NotFound(w, r)
			return
		}

		g := ctx.Game
		user := s.getUser(r)

		status, ok := r.URL.Query()["status"]
		if ok {
			msg := ""
			switch status[0] {
			case "Y":
				msg = "See you at the game!"
			case "N":
				msg = "Sorry you can't make it"
			case "M":
				msg = "Sh*t or get off the pot!"

			}
			setStatus(s.DB, status[0], user.Id, g.Id)
			s.SetMessage(w, r, msg)
		}

		templateParams := gameShowParams{
			User:      user,
			Game:      g,
			Responses: responsesForGame(s.DB, g.Id),
		}
		s.RenderTemplate(w, r, ctx.Template, templateParams)

	})
}

// currently takes you to
// http://www.teamvite.com/team/52/index
// with one of the following messages:
// * see you at the game (Yes)
// * sorry you can't make it (No)
// * sh*t or get off the pot (Maybe)
// Should this take you to a "game index" page?  Do I even need one?
// Clicking on attending status could take you to this game index page.
// Shows details about the game and a list of players with their responses
// Would simplify upcoming games template

// game status can be set on the game page, and it will upsert/find_or_create status
// for emails, pass a session_id param and use that to get the player_id
// otherwise just use the current session player_id
func setStatus(DB *sqlx.DB, status string, userId int, gameId int) {
	if status == "" || userId == 0 || gameId == 0 {
		if status != "" && userId == 0 {
			fmt.Println("WARN: Can't set status without a user")
		}
		return
	}

	fmt.Printf("setting game: %d and user: %d status to: %s\n", gameId, userId, status)
	r, err := DB.Exec("update players_games set status = ? where game_id = ? and player_id = ?", status, gameId, userId)
	checkErr(err, "Setting game status")
	numRows, err := r.RowsAffected()
	checkErr(err, "Game status rows affected")

	if numRows == 0 {
		// do an insert
		_, err := DB.Exec("insert into players_games (status, game_id, player_id) values (?, ?, ?)", status, gameId, userId)
		checkErr(err, "Setting game status")

	}
}

func getOrCreateStatus(DB *sqlx.DB, playerId int, gameId int) (pg playerGame) {
	err := DB.Get(&pg, "select * from players_games where player_id = ? and game_id = ?", playerId, gameId)
	checkErr(err, "game getOrCreateStatus")
	if errors.Is(err, sql.ErrNoRows) {
		DB.Exec("insert into players_games (player_id, game_id) values (?, ?)", playerId, gameId)
	}

	err = DB.Get(&pg, "select * from players_games where player_id = ? and game_id = ?", playerId, gameId)
	checkErr(err, "game getOrCreateStatus")

	return
}