package http

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	teamvite "github.com/benprew/teamvite"
)

func (s *Server) SMS() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		player, status, err := s.playerFromMessage(r)
		if err != nil {
			response := "ERROR: Unknown number"
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(response))
			return
		}

		ctx := teamvite.NewContextWithPlayer(r.Context(), "", player)

		nextGame, err := s.PlayerService.NextRemindedGame(ctx, player.ID)
		if err != nil {
			response := "Internal Error"
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(response))
			return
		}

		response := s.setStatusForGame(ctx, nextGame, status)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(response))
	})
}

func (s *Server) playerFromMessage(r *http.Request) (*teamvite.Player, string, error) {
	// TODO: check basic auth
	// TODO: validate request signature
	// https://www.twilio.com/docs/usage/security#http-authentication
	// message := make(map[string]string)

	// TODO: use SmsMessageSid to get unique message id
	// https://support.twilio.com/hc/en-us/articles/223134387-What-is-a-Message-SID-
	// https://www.twilio.com/docs/glossary/what-is-a-sid

	err := r.ParseForm()
	if err != nil || r.PostForm == nil {
		return nil, "", err
	}

	rawTel := r.PostForm.Get("From")
	if len(rawTel) < 10 {
		return nil, "", fmt.Errorf("ERROR: Unknown number")
	}
	tel := teamvite.UnTelify(rawTel[2:])
	reg, _ := regexp.Compile("[^a-zA-Z]+")
	// get clean message (Yes should fuzzy-match y/Yes/YES/yes)
	msg := strings.ToUpper(reg.ReplaceAllString(r.PostForm.Get("Body"), ""))

	log.Printf("Raw tel: %s Parsed tel: %d", r.PostForm.Get("From"), tel)
	log.Println("Message: ", msg)

	// get user from phone number
	players, n, err := s.PlayerService.FindPlayers(r.Context(), teamvite.PlayerFilter{Phone: tel, Limit: 1})
	if err != nil {
		return nil, "", err
	}

	if n == 0 {
		return nil, "", fmt.Errorf("ERROR: Unknown number")
	}

	player := players[0]

	return player, msg[0:1], nil
}

func (s *Server) setStatusForGame(ctx context.Context, g teamvite.Game, status string) string {
	// Find most recently reminded game
	// Note: this has limitations if a player is playing on multiple
	// teams and we send multiple alerts to them.
	// If needed, could get multiple numbers from Twilio to handle
	// multiple teams for a single player
	//
	// Other SMS systems I've seen solve this by having a unique number
	// assigned to the response

	switch status {
	case "Y":
		err := s.GameService.UpdateStatus(ctx, &g, status)
		if err != nil {
			log.Println("Error setting status", err)
			return "Internal Error"
		}
		return "See you at the game"
	case "N":
		err := s.GameService.UpdateStatus(ctx, &g, status)
		if err != nil {
			log.Println("Error setting status", err)
			return "Internal Error"
		}
		return "Sorry you can't make it"
	case "S":
		teams, err := s.PlayerService.Teams(ctx, g.TeamID)
		if err != nil {
			log.Println("Internal Error: Error stopping reminders", err)
			return "Internal Error"
		}
		playerTeam := teams[0]
		playerTeam.RemindSMS = false

		err = s.PlayerService.UpdatePlayerTeam(
			ctx,
			&playerTeam)

		if err != nil {
			log.Println("Internal Error: Error stopping reminders", err)
			return "Internal Error"
		}
		return "Stopping future reminders"
	default:
		return "Unknown reply, valid replies are YES, NO or STOP"
	}
	// docs on post body
	// https://www.twilio.com/docs/messaging/guides/webhook-request
}

// this is the test receiver that captures sms POSTS from send_game_reminders.
func (s *Server) TestSMSReceiver() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.Println("Got SMS message:", string(body))
	})
}
