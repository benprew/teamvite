CREATE TABLE players (
    id integer PRIMARY KEY autoincrement,
    name varchar(64) NOT NULL DEFAULT '',
    email varchar(128) NOT NULL UNIQUE,
    password varchar(1024) NOT NULL DEFAULT '',
    phone int8 NOT NULL DEFAULT 0
);

CREATE TABLE teams (
    id integer PRIMARY KEY autoincrement,
    name varchar(64) NOT NULL DEFAULT '',
    division_id integer NOT NULL,
    UNIQUE (name, division_id),
    FOREIGN KEY (division_id) REFERENCES divisions (id)
);

CREATE TABLE divisions (
    id integer NOT NULL PRIMARY KEY autoincrement,
    name varchar(64) NOT NULL UNIQUE
);

CREATE TABLE games (
    id integer PRIMARY KEY autoincrement,
    team_id integer NOT NULL,
    season_id integer NOT NULL,
    time datetime NOT NULL,
    description string NOT NULL DEFAULT '',
    UNIQUE (team_id, time),
    FOREIGN KEY (team_id) REFERENCES teams (id),
    FOREIGN KEY (season_id) REFERENCES seasons (id)
);

CREATE TABLE players_teams (
    player_id integer NOT NULL,
    team_id integer NOT NULL,
    is_manager boolean NOT NULL DEFAULT FALSE,
    remind_email boolean NOT NULL DEFAULT TRUE,
    remind_sms boolean NOT NULL DEFAULT FALSE,
    PRIMARY KEY (team_id, player_id),
    FOREIGN KEY (team_id) REFERENCES teams (id),
    FOREIGN KEY (player_id) REFERENCES players (id)
);

CREATE TABLE players_games (
    player_id integer NOT NULL,
    game_id integer NOT NULL,
    status varchar(32) NOT NULL DEFAULT '?',
    reminder_sent boolean NOT NULL DEFAULT FALSE,
    PRIMARY KEY (player_id, game_id),
    FOREIGN KEY (player_id) REFERENCES players (id),
    FOREIGN KEY (game_id) REFERENCES games (id)
);

CREATE TABLE seasons (
    id integer PRIMARY KEY autoincrement,
    name string NOT NULL UNIQUE
);

CREATE TABLE sessions (
    id varchar(128) NOT NULL PRIMARY KEY,
    player_id NOT NULL,
    ip varchar,
    expires_on datetime NOT NULL DEFAULT 2556144000, -- 1/1/2051
    FOREIGN KEY (player_id) REFERENCES players (id)
);
