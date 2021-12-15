package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

type Mail struct {
	Sender     string
	SenderName string
	To         []string
	Subject    string
	Body       string
}

type SendReminderParams struct {
	Teams    []team
	Messages []string
}

func ReminderUrl(g game, token string, status string) string {
	return fmt.Sprintf("https://%s%s?token=%s&status=%s", CONFIG.Servername, urlFor(&g, "show"), token, status)

}

func (s *server) SendGameReminders() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		teams := []team{}
		s.DB.Select(&teams, "select * from teams")

		messages := make([]string, len(teams))
		reminders := []string{}
		for i, t := range teams {
			g, ok := t.nextGame(s.DB)
			if !ok || g.Time.After(time.Now().Add(time.Hour*24*5)) {
				messages[i] = "No upcoming unreminded games"
				continue
			}
			remindersSent := 0
			log.Printf("Sending reminders for game: %s\n", g.Description)
			for _, p := range t.Players(s.DB) {
				pg := getOrCreateStatus(s.DB, p.Id, g.Id)
				if pg.ReminderSent {
					log.Printf("Reminder already sent to: %s\n", p.Email)
					continue
				}

				// if this fails we'll just try sending again tomorrow
				if err := s.emailReminder(p, g); err != nil {
					checkErr(err, "Sending email")
				} else {
					remindersSent++
					reminders = append(reminders, fmt.Sprintf("(%d, %d)", p.Id, g.Id))
				}
			}
			messages[i] = fmt.Sprintf("Emailed %s to %d players", g.Description, remindersSent)
		}
		if len(reminders) > 0 {
			fmt.Println(reminders)
			query := fmt.Sprintf(`
                        drop table if exists tmp_reminders;
                        create table tmp_reminders(player_id int, game_id int);
                        insert into tmp_reminders (player_id, game_id) values %s;
                        update players_games set reminder_sent = true
                          where (player_id, game_id) in
                            (select player_id, game_id from tmp_reminders);
                        drop table tmp_reminders;`,
				strings.Join(reminders, ","))
			log.Println(query)
			_, err := s.DB.Exec(query)
			checkErr(err, "updating reminders_sent")
		}
		params := SendReminderParams{
			Teams:    teams,
			Messages: messages,
		}

		s.RenderTemplate(w, r, "views/send_game_reminders.tmpl", params)
	})
}

func (s *server) emailReminder(p player, g game) error {
	log.Printf("Sending reminder to: %s\n", p.Email)
	token, err := CreateToken(s.DB, p, time.Now().Add(time.Hour*24*7))
	if err != nil {
		return err
	}
	log.Println("token:", token)
	body, err := reminderEmailBody(ReminderEmailParams{Player: &p, Game: &g, Token: token})
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

type ReminderEmailParams struct {
	Player *player
	Game   *game
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

func reminderEmailBody(params ReminderEmailParams) (string, error) {
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
