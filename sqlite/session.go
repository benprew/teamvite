package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/benprew/teamvite"
	"github.com/dchest/uniuri"
)

type SessionService struct {
	db *sql.DB
}

// Ensure service implements interface.
var _ teamvite.SessionService = (*SessionService)(nil)

func NewSessionService(db *sql.DB) *SessionService {
	return &SessionService{db: db}
}

// ## Security
// log when a session is created and from what ip address (but don't log
// session-id)
//
// can read session_id from cookie or from params (replace tokens)
//
// look up owasp recommendations for sessions
// https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
//
// The session ID must be long enough to prevent brute force attacks, where an
// attacker can go through the whole range of ID values and verify the existence
// of valid sessions.
// The session ID length must be at least 128 bits (16 bytes).

// sessions:
// id - varchar(128)
// player_id - reference to players table
// ip - varchar()- ip address - Note: reminder tokens won't set this
// expires_on - date - fixed time for tokens (1 wk), can be longer with ip address
// version - int not null default 1 - version of the session - is this needed?
//
// key name: - either param or cookie name
// teamvite-session
// type Session struct {
// 	ID        string    `db:"id"`
// 	PlayerID  int       `db:"player_id"`
// 	ipStr     string    `db:"ip"`
// 	ExpiresOn time.Time `db:"expires_on"`
// 	IP        net.IP
// }

// Creating a Session
//
//  1. when logging in
//     need to set cookie value
//     ip := session.RequestIP(httpRequest)
//     s := session.New(DB, 123, ip, time.Hour * 24 * 30)
//     s.SetCookie(httpRequest, "teamvite-session")
//
//  2. when sending game reminders
//     don't need to set cookie value
//     s := session.New(DB, 123, nil, time.Hour * 24 * 7)
//     ReminderUrl(game, s.ID)
//
// Loading a Session from an ID
//
//  1. From Cookie
//     sid, err := session.SidFromCookie(request, "teamvite-session")
//     checkErr(err)
//     s, err := session.LoadSession(DB, sid)
//
//  2. From Request Param
//     s, err := session.LoadSession(DB, params["teamvite-session"])
//
// Logging a User Out
//
//	session.Revoke(DB, sid)
//
// Session IDs
//
//	Session IDs are strings of length 25 that can include the characters
//	[0-9a-zA-Z] which provides ~148 bits of entropy. OWASP recommendation is 128
//	bits.
func (ss *SessionService) New(playerID uint64, IP net.IP, sessionLen time.Duration) (teamvite.Session, error) {
	s := teamvite.Session{
		ID:        genSessionID(25),
		PlayerID:  playerID,
		IP:        IP,
		ExpiresOn: time.Now().Add(sessionLen),
	}
	return s, ss.SaveSession(s)
}

func (ss *SessionService) SaveSession(s teamvite.Session) error {
	ipStr := s.IP.String()

	_, err := ss.db.Exec(
		`INSERT INTO SESSIONS (id, player_id, ip, expires_on)
		VALUES (?, ?, ?, ?)`,
		s.ID, s.PlayerID, ipStr, s.ExpiresOn)
	return err
}

// the Load verb is common in the codebase and means to load a struct from the database
func (ss *SessionService) Load(sid string, ip net.IP) (teamvite.Session, error) {
	var ipStr string
	s := teamvite.Session{}
	row := ss.db.QueryRow(
		`SELECT id, player_id, ip, expires_on
		FROM sessions
		WHERE id = ? AND expires_on >= ?`,
		sid,
		time.Now().Unix())
	err := row.Scan(&s.ID, &s.PlayerID, &ipStr, &s.ExpiresOn)
	if err != nil {
		return teamvite.Session{}, err
	}
	s.IP = net.ParseIP(ipStr)
	if ip != nil && s.IP != nil && !ip.Equal(s.IP) {
		msg := fmt.Sprintf("ip mismatch [req=%s db=%s]", ip, s.IP)
		log.Printf("[WARN] %s\n", msg)
		return teamvite.Session{}, fmt.Errorf(msg)
	}
	log.Printf("loaded session [sid=%s, ip=%s, session=%v]", sid, s.IP, s)
	return s, err
}

func (ss *SessionService) Revoke(sid string) error {
	_, err := ss.db.Exec("delete from sessions where id = ?", sid)
	log.Printf("Revoked session [sid=%s]", sid)
	return err
}

func genSessionID(length uint) string {
	return uniuri.NewLen(int(length))
}
