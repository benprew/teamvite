package sqlite

import (
	"net"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestLoadSessionIpMismatch(t *testing.T) {
	db := Open(":memory:")
	srv := NewSessionService(db)

	_, err := db.Exec(`
CREATE TABLE sessions (
    id varchar(128) NOT NULL PRIMARY KEY,
    player_id NOT NULL,
    ip varchar,
    expires_on datetime NOT NULL DEFAULT 2556144000, -- 1/1/2051
    FOREIGN KEY (player_id) REFERENCES players (id)
);`)
	panicIf(err)

	// create dummy session
	_, err = db.Exec("insert into sessions (id, player_id, ip) values (123, 123, '127.0.0.2')")
	panicIf(err)

	s, err := srv.Load("123", net.ParseIP("127.0.0.1"))
	if s.PlayerID != 0 {
		t.Errorf("Session was valid [session=%v]", s)
	}
	if err == nil {
		t.Errorf("No error")
	}
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
