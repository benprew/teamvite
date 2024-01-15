package http

import (
	"log"
	"net"
	"net/http"
	"strings"

	teamvite "github.com/benprew/teamvite"
)

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
// func (s *Server) New(playerID int, IP net.IP, sessionLen time.Duration) (Session, error) {
// 	session := teamvite.Session{
// 		ID:        genSessionID(25),
// 		PlayerID:  playerID,
// 		IP:        IP,
// 		ExpiresOn: time.Now().Add(sessionLen),
// 	}
// 	return session, saveSession(DB, session)

// }

// func saveSession(DB *sqlx.DB, s Session) error {
// 	if s.ipStr == "" {
// 		s.ipStr = s.IP.String()
// 	}
// 	_, err := DB.NamedExec("insert into sessions (id, player_id, ip, expires_on) values (:id, :player_id, :ip, :expires_on)", s)
// 	return err
// }

// the Load verb is common in the codebase and means to load a struct from the database
// func LoadSession(DB *sqlx.DB, sid string, ip net.IP) (Session, error) {
// 	s := Session{}
// 	err := DB.Get(&s, "select * from sessions where id = ? and expires_on >= ?", sid, time.Now().Unix())
// 	log.Printf("loadSession [dbSession=%v]", s)
// 	if err != nil {
// 		return Session{}, err
// 	}
// 	s.IP = net.ParseIP(s.ipStr)
// 	if ip != nil && s.IP != nil && !ip.Equal(s.IP) {
// 		msg := fmt.Sprintf("ip mismatch [req=%s db=%s]", ip, s.IP)
// 		log.Printf("[WARN] %s\n", msg)
// 		return Session{}, fmt.Errorf(msg)
// 	}
// 	log.Printf("loaded session [sid=%s, ip=%s, session=%v]", sid, s.IP, s)
// 	return s, err
// }

// func Revoke(DB *sqlx.DB, sid string) error {
// 	_, err := DB.Exec("delete from sessions where id = ?", sid)
// 	log.Printf("Revoked session [sid=%s]", sid)
// 	return err
// }

func SidsFromCookie(r *http.Request, keyName string) (sid []string) {
	for _, c := range r.Cookies() {
		if c.Name != keyName {
			continue
		}
		log.Printf("Loaded cookie [cookie=%s sid=%s]", c, c.Value)
		sid = append(sid, c.Value)
	}
	return sid
}

func RequestIP(r *http.Request) net.IP {
	if r.Header == nil {
		log.Println("[ERROR] RequestIP: r.Header is nil")
		return nil
	}
	forwarded := strings.Split(r.Header.Get("X-FORWARDED-FOR"), ",")
	if forwarded == nil {
		log.Println("[ERROR] RequestIP: X-FORWARDED-FOR is nil")
		return nil
	}
	ip := forwarded[len(forwarded)-1]

	if ip == "" {
		// RemoteAddr is a string IP:Port ex 127.0.0.1:59690
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	return net.ParseIP(ip)
}

func (s *Server) SetCookie(w http.ResponseWriter, session teamvite.Session) {
	cookie := &http.Cookie{
		Name:     SESSION_KEY,
		Value:    session.ID,
		Domain:   s.Domain,
		Expires:  session.ExpiresOn,
		Secure:   false, // teamvite serves http behind nginx proxy
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}
