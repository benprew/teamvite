package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/benprew/teamvite"
)

type Mail struct {
	Sender     string
	SenderName string
	To         []string
	Subject    string
	Body       string
}

type SendReminderParams struct {
	Teams    []teamvite.Team
	Messages []string
}

func ReminderUrl(g teamvite.Game, token string, status string) string {
	return fmt.Sprintf("https://%s%s?%s=%s&status=%s", CONFIG.Servername, urlFor(&g, "show"), SESSION_KEY, token, status)

}

// The players_teams db table
type PlayerTeam struct {
	PlayerID    int  `db:"player_id"`
	TeamID      int  `db:"team_id"`
	IsManager   bool `db:"is_manager"`
	RemindEmail bool `db:"remind_email"`
	RemindSMS   bool `db:"remind_sms"`
}

func (s *Server) SendGameReminders() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var playerTeams []PlayerTeam
		s.DB.Select(&playerTeams, "select * from players_teams order by team_id, player_id")
		var t teamvite.Team
		var ng teamvite.Game
		var mKey string
		messages := make(map[string]string, 1000)
		reminders := []string{}
		for _, pt := range playerTeams {
			if pt.TeamID != t.ID {
				t = loadTeam(s.DB, uint64(pt.TeamID))
				ng, _ = t.NextGame(s.DB)
				mKey = fmt.Sprintf("%s-%d", t.Name, t.DivisionID)
			}

			if ng.ID == 0 || ng.Time.After(time.Now().Add(time.Hour*24*5)) {
				messages[mKey] = "No upcoming unreminded games"
				continue
			}

			log.Printf("Sending reminders for game: %s\n", ng.Description)

			emailSent := 0
			smsSent := 0
			p := loadPlayer(s.DB, uint64(pt.PlayerID))
			if !needsGameReminder(s.DB, p.ID, ng.ID) {
				log.Printf("Reminder already sent to: %s\n", p.Name)
				messages[mKey] = fmt.Sprintf("%s - email: %d, sms: %d", ng.Description, emailSent, smsSent)
				continue
			}

			reminderSent := false
			if pt.RemindEmail {
				if err := s.emailReminder(p, ng); err != nil {
					checkErr(err, "Sending email")
				} else {
					reminderSent = true
					emailSent++
				}
			}

			if pt.RemindSMS {
				if err := sendTwilioSMSMessage(p, ng); err != nil {
					checkErr(err, "Sending SMS")
				} else {
					reminderSent = true
					smsSent++
				}
			}

			if reminderSent {
				reminders = append(reminders, fmt.Sprintf("(%d, %d)", p.ID, ng.ID))
			}
			messages[mKey] = fmt.Sprintf("%s - email: %d, sms: %d", ng.Description, emailSent, smsSent)
		}
		if len(reminders) > 0 {
			fmt.Println(reminders)
			query := fmt.Sprintf(`
				drop table if exists tmp_reminders;
				create table tmp_reminders(player_id int, game_id int);
				insert into tmp_reminders (player_id, game_id) values %s;
				insert into players_games
					select player_id, game_id, '?', true from tmp_reminders as tr
						where player_id = tr.player_id and game_id = tr.game_id
					on conflict (player_id, game_id) do update set reminder_sent = true;
				drop table tmp_reminders;`,
				strings.Join(reminders, ","))
			log.Println(query)
			_, err := s.DB.Exec(query)
			checkErr(err, "updating reminders_sent")
		}
		w.Header().Set("Content-Type", JSON)
		json.NewEncoder(w).Encode(messages)
	})
}

func (s *Server) emailReminder(p teamvite.Player, g teamvite.Game) error {
	log.Printf("Sending reminder to: %s\n", p.Email)
	token, err := session.New(s.DB.Queryer, p.ID, nil, time.Hour*24*7)
	if err != nil {
		return err
	}
	body, err := reminderEmailBody(reminderParams{Player: &p, Game: &g, Token: token.ID})
	if err != nil {
		return err
	}
	request := Mail{
		Sender:     "team@teamvite.com",
		SenderName: "Teamvite",
		To:         []string{p.Email},
		Subject:    fmt.Sprintf("Next Game: %s %s", g.Time.Format(""), g.Description),
		Body:       body,
	}
	msg := buildMessage(request)
	auth := smtp.PlainAuth("", CONFIG.SMTP.Username, CONFIG.SMTP.Password, CONFIG.SMTP.Hostname)
	addr := fmt.Sprintf("%s:%d", CONFIG.SMTP.Hostname, CONFIG.SMTP.Port)
	return smtp.SendMail(addr, auth, request.Sender, request.To, []byte(msg))
}

type reminderParams struct {
	Player *teamvite.Player
	Game   *teamvite.Game
	Token  string
}

var reminderTemplate = `
Dear {{ .Player.Name }},<br>
This is your game reminder.  The next game is:<br>
<blockquote>
  {{ .Game.Time.Format "Mon Jan 2 3:04PM" }} {{ .Game.Description }}
</blockquote>

Can you make the game?
<ul>
  <li><a href="{{ ReminderUrl .Game .Token "Y"}}">Yes</a></li>
  <li><a href="{{ ReminderUrl .Game .Token "N"}}">No</a></li>
  <li><a href="{{ ReminderUrl .Game .Token "M"}}">Maybe</a></li>
</ul>


Thank you for using Teamvite!
`

func reminderEmailBody(params reminderParams) (string, error) {
	tmpl, err := template.Must(templates.Clone()).New("content").Parse(reminderTemplate)
	if err != nil {
		checkErr(err, "ReminderEmail template")
		return "", err
	}
	var w bytes.Buffer
	if err = tmpl.ExecuteTemplate(&w, "content", params); err != nil {
		log.Println("[ERROR]", err)
		return "", err
	}

	return w.String(), nil
}

func buildMessage(mail Mail) string {
	msg := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	msg += fmt.Sprintf("From: %s <%s>\r\n", mail.SenderName, mail.Sender)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(mail.To, ";"))
	msg += fmt.Sprintf("Subject: %s\r\n", mail.Subject)
	msg += fmt.Sprintf("\r\n%s\r\n", mail.Body)

	return msg
}

func sendTwilioSMSMessage(p teamvite.Player, g teamvite.Game) error {
	accountSid := CONFIG.SMS.Sid
	u := fmt.Sprintf("%s/Accounts/%s/Messages.json", CONFIG.SMS.API, accountSid)
	client := &http.Client{Timeout: time.Second * 10}

	body, err := smsBody(g)
	if err != nil {
		return err
	}

	// HTTP requests to the API are protected with HTTP Basic
	// authentication. To learn more about how Twilio handles authentication,
	// please refer to our security documentation.
	//
	// In short, you will use your Twilio Account SID as the username and your
	// Auth Token as the password for HTTP Basic authentication with Twilio.
	//
	// curl -G https://api.twilio.com/2010-04-01/Accounts \
	//   -u <YOUR_ACCOUNT_SID>:<YOUR_AUTH_TOKEN>
	postParams := url.Values{}
	postParams.Add("To", fmt.Sprintf("+1%d", p.Phone))
	postParams.Add("From", CONFIG.SMS.From)
	postParams.Add("Body", body)
	postParams.Add("StatusCallback", "https://www.teamvite.com/sms")

	log.Println(postParams)
	log.Println(u)
	req, err := http.NewRequest("POST", u, strings.NewReader(postParams.Encode()))
	checkErr(err, "creating twilio post")
	req.SetBasicAuth(CONFIG.SMS.Sid, CONFIG.SMS.Token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	checkErr(err, "posting to SMS gateway")

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	checkErr(err, "SMS gateway response")
	log.Println(string(respBody))

	return err
}

var smsReminderTemplate = `
Teamvite Game Reminder:
{{ .Game.Time.Format "Mon Jan 2 3:04PM" }} {{ .Game.Description }}
Reply
YES/NO/MAYBE/STOP`

func smsBody(g teamvite.Game) (string, error) {
	tmpl, err := template.Must(templates.Clone()).New("content").Parse(smsReminderTemplate)
	if err != nil {
		checkErr(err, "parsing smsReminderTemplate")
		return "", err
	}
	var w bytes.Buffer
	if err = tmpl.ExecuteTemplate(&w, "content", struct{ Game game }{Game: g}); err != nil {
		log.Println("[ERROR]", err)
		return "", err
	}

	return w.String(), nil
}
