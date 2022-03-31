DROP TABLE tokens;

DROP TABLE sessions;

DROP TABLE sessions_new;

CREATE TABLE sessions (
    id varchar(128) NOT NULL PRIMARY KEY,
    player_id NOT NULL,
    ip varchar,
    expires_on datetime NOT NULL DEFAULT 2556144000, -- 1/1/2051
    FOREIGN KEY (player_id) REFERENCES players (id)
);
