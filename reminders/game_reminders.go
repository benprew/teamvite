package reminders

import (
	"bytes"
	"database/sql"
	"errors"
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
	thttp "github.com/benprew/teamvite/http"
	"github.com/benprew/teamvite/sqlite"
)

type mail struct {
	Sender     string
	SenderName string
	To         []string
	Subject    string
	Body       string
}

type ReminderService struct {
	db     *sql.DB
	smtp   teamvite.SMTPConfig
	sms    teamvite.SMSConfig
	domain string
}

// NewGameService returns a new instance of GameService.
func NewReminderService(db *sql.DB, SMTP teamvite.SMTPConfig, SMS teamvite.SMSConfig, domain string) *ReminderService {
	return &ReminderService{db: db, smtp: SMTP, sms: SMS, domain: domain}
}

func checkErr(err error, msg string) {
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Println("[ERROR]: ", msg, err)
	}
}

func (s *ReminderService) SendGameReminders() error {
	query := `
	SELECT
		p.id AS player_id,
		p.name AS player_name,
		p.email as player_email,
		p.phone as player_phone,
		g.id AS game_id,
		g.time AS game_time,
		g.description AS game_description,
		t.name AS team_name,
		t.division_id AS division_id,
		pt.remind_email AS remind_email,
		pt.remind_sms AS remind_sms
	FROM players p
	JOIN players_games pg ON p.id = pg.player_id
	JOIN games g ON pg.game_id = g.id
	JOIN player_teams pt ON p.id = pt.player_id
	JOIN teams t ON pt.team_id = t.id
	WHERE
		g.time BETWEEN datetime('now') AND datetime('now', '+5 days')
		AND (NOT pg.reminder_sent OR pg.status = '');
	`
	rows, err := s.db.Query(query)
	if err != nil {
		log.Println("querying for reminders:", err)
		return err
	}
	defer rows.Close()

	emailSent := 0
	smsSent := 0
	messages := make(map[string]string, 1000)
	reminders := []string{}
	var mKey string

	for rows.Next() {
		var p teamvite.Player
		var g teamvite.Game
		var tName string
		var divID int
		var remindEmail bool
		var remindSMS bool
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Email,
			&p.Phone,
			&g.ID,
			&g.Time,
			&g.Description,
			&tName,
			&divID,
			&remindEmail,
			&remindSMS)
		if err != nil {
			log.Println("reading reminder rows", err)
			return err
		}

		mKey = fmt.Sprintf("%s-%d", tName, divID)
		reminderSent := false
		if remindEmail {
			if err := s.emailReminder(p, g); err != nil {
				checkErr(err, "Sending email")
			} else {
				reminderSent = true
				emailSent++
			}
		}

		if remindSMS {
			if err := s.sendTwilioSMSMessage(p, g); err != nil {
				checkErr(err, "Sending SMS")
			} else {
				reminderSent = true
				smsSent++
			}
		}

		if reminderSent {
			// reminders are in the format expected for players_games,
			// with status and reminder_sent columns populated.
			reminders = append(reminders, fmt.Sprintf("(%d, %d, '?', true)", p.ID, g.ID))
		}
		messages[mKey] = fmt.Sprintf("%s - email: %d, sms: %d", g.Description, emailSent, smsSent)
	}

	if len(reminders) > 0 {
		fmt.Println(reminders)
		query := fmt.Sprintf(`
			INSERT INTO players_games (player_id, game_id, status, reminder_sent)
			VALUES %s
			ON CONFLICT(player_id, game_id)
			DO UPDATE SET reminder_sent = true;
			`,
			strings.Join(reminders, ","))
		log.Println(query)
		_, err = s.db.Exec(query)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	fmt.Println(messages)
	return nil
}

func (s *ReminderService) emailReminder(p teamvite.Player, g teamvite.Game) error {
	log.Printf("Sending reminder to: %s\n", p.Email)
	session := sqlite.NewSessionService(s.db)
	token, err := session.New(p.ID, nil, time.Hour*24*7)
	if err != nil {
		log.Println("creating token: ", err)
		return err
	}
	reminderURL := fmt.Sprintf("https://%s%s?%s=%s", s.domain, thttp.UrlFor(&g, "show"), thttp.SESSION_KEY, token)
	body, err := reminderEmailBody(reminderParams{Player: &p, Game: &g, ReminderURL: reminderURL})
	if err != nil {
		log.Println("building reminder email body: ", err)
		return err
	}
	request := mail{
		Sender:     "team@teamvite.com",
		SenderName: "Teamvite",
		To:         []string{p.Email},
		Subject:    fmt.Sprintf("Next Game: %s %s", g.Time.Format(""), g.Description),
		Body:       body,
	}
	msg := buildMessage(request)
	auth := smtp.PlainAuth("", s.smtp.Username, s.smtp.Password, s.smtp.Hostname)
	addr := fmt.Sprintf("%s:%d", s.smtp.Hostname, s.smtp.Port)
	return smtp.SendMail(addr, auth, request.Sender, request.To, []byte(msg))
}

type reminderParams struct {
	Player      *teamvite.Player
	Game        *teamvite.Game
	ReminderURL string
}

var reminderTemplate = `
Dear {{ .Player.Name }},<br>
This is your game reminder.  The next game is:<br>
<blockquote>
  {{ .Game.Time.Format "Mon Jan 2 3:04PM" }} {{ .Game.Description }}
</blockquote>

Can you make the game?
<ul>
  <li><a href="{{ statusURL .ReminderURL "Y" }}">Yes</a></li>
  <li><a href="{{ statusURL .ReminderURL "N" }}">No</a></li>
  <li><a href="{{ statusURL .ReminderURL "M" }}">Maybe</a></li>
</ul>


Thank you for using Teamvite!
`

func statusURL(URL string, status string) string {
	return fmt.Sprintf("%s&status=%s", URL, status)
}

func reminderEmailBody(params reminderParams) (string, error) {
	// TODO: make reminders in params? As an array?
	// Doing it in the template complicates building the template and needed functions
	fMap := template.FuncMap{
		"statusURL": statusURL,
	}
	templates := template.New("layout.tmpl").Funcs(fMap)
	tmpl, err := template.Must(templates.Clone()).New("content").Parse(reminderTemplate)
	if err != nil {
		checkErr(err, "parsing ReminderEmail template")
		return "", err
	}
	var w bytes.Buffer
	if err = tmpl.ExecuteTemplate(&w, "content", params); err != nil {
		log.Println("[ERROR]", err)
		return "", err
	}

	return w.String(), nil
}

func buildMessage(mail mail) string {
	msg := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	msg += fmt.Sprintf("From: %s <%s>\r\n", mail.SenderName, mail.Sender)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(mail.To, ";"))
	msg += fmt.Sprintf("Subject: %s\r\n", mail.Subject)
	msg += fmt.Sprintf("\r\n%s\r\n", mail.Body)

	return msg
}

func (s *ReminderService) sendTwilioSMSMessage(p teamvite.Player, g teamvite.Game) error {
	accountSid := s.sms.Sid
	u := fmt.Sprintf("%s/Accounts/%s/Messages.json", s.sms.API, accountSid)
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
	postParams.Add("From", s.sms.From)
	postParams.Add("Body", body)
	postParams.Add("StatusCallback", "https://www.teamvite.com/sms")

	log.Println(postParams)
	log.Println(u)
	req, err := http.NewRequest("POST", u, strings.NewReader(postParams.Encode()))
	checkErr(err, "creating twilio post")
	req.SetBasicAuth(s.sms.Sid, s.sms.Token)
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
{{ .Time.Format "Mon Jan 2 3:04PM" }} {{ .Description }}
Reply
YES/NO/MAYBE/STOP`

func smsBody(g teamvite.Game) (string, error) {
	tmpl, err := template.New("content").Parse(smsReminderTemplate)
	if err != nil {
		checkErr(err, "parsing smsReminderTemplate")
		return "", err
	}
	var w bytes.Buffer
	if err = tmpl.ExecuteTemplate(&w, "content", g); err != nil {
		log.Println("[ERROR]", err)
		return "", err
	}

	return w.String(), nil
}
