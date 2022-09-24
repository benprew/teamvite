package main

import (
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

func (s *server) SMS() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: check basic auth
		// TODO: validate request signature
		// https://www.twilio.com/docs/usage/security#http-authentication
		// message := make(map[string]string)

		// TODO: use SmsMessageSid to get unique message id
		// https://support.twilio.com/hc/en-us/articles/223134387-What-is-a-Message-SID-
		// https://www.twilio.com/docs/glossary/what-is-a-sid

		err := r.ParseForm()
		checkErr(err, "Parsing form data")

		rawTel := r.PostForm.Get("From")
		if len(rawTel) < 10 {
			response := "ERROR: Unknown number"
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(response))
			return
		}
		tel := UnTelify(rawTel[2:])
		reg, _ := regexp.Compile("[^a-zA-Z]+")
		// get clean message (Yes should fuzzy-match y/Yes/YES/yes)
		msg := strings.ToUpper(reg.ReplaceAllString(r.PostForm.Get("Body"), ""))

		log.Printf("Raw tel: %s Parsed tel: %d", r.PostForm.Get("From"), tel)
		log.Println("Message: ", msg)

		// get user from phone number
		p := playerByPhone(s.DB, tel)

		if p.Id == 0 {
			log.Printf("[ERROR] Unable to find player with tel: %d", tel)
			response := "ERROR: Unknown number"
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(response))
			return
		} else {
			log.Println("Player: ", p)
		}

		// Note: this has limitations if a player is playing on multiple
		// teams and we send multiple alerts to them.
		// If needed, could get multiple numbers from Twilio to handle
		// multiple teams for a single player
		teams := p.Teams(s.DB)

		// find most recently reminded game
		var nextGame game

		for _, t := range teams {
			g, ok := t.NextGame(s.DB)
			if ok {
				nextGame = g
			}
		}

		var response string

		switch msg[0:1] {
		case "Y":
			setStatus(s.DB, msg[0:1], p.Id, nextGame.Id)
			response = "See you at the game"
		case "N":
			setStatus(s.DB, msg[0:1], p.Id, nextGame.Id)
			response = "Sorry you can't make it"
		case "S":
			_, err := s.DB.Exec(
				"update players_teams set remind_sms = false where player_id = ? and team_id = ?", p.Id, nextGame.TeamId)
			checkErr(err, "stop reminders")
			response = "Stopping future reminders"
		default:
			response = "Unknown reply, valid replies are YES, NO or STOP"
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(response))

		// docs on post body
		// https://www.twilio.com/docs/messaging/guides/webhook-request
	})
}

// this is the test receiver that captures sms POSTS from send_game_reminders.
func (s *server) TestSMSReceiver() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.Println("Got SMS message:", string(body))
	})
}

func playerByPhone(DB *QueryLogger, phone int) (p player) {
	if phone == -1 {
		return
	}
	err := DB.Get(&p, "select * from players where phone = ?", phone)
	checkErr(err, "error loading player: ")
	return
}
