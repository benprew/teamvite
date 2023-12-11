package teamvite

type Mail struct {
	Sender     string
	SenderName string
	To         []string
	Subject    string
	Body       string
}

type SendReminderParams struct {
	Teams    []Team
	Messages []string
}

// The players_teams db table
type PlayerTeam struct {
	PlayerID    int  `db:"player_id"`
	TeamID      int  `db:"team_id"`
	IsManager   bool `db:"is_manager"`
	RemindEmail bool `db:"remind_email"`
	RemindSMS   bool `db:"remind_sms"`
}
