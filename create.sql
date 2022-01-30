create table players (
       id integer primary key autoincrement,
       name varchar(64) not null default '',
       email varchar(128) not null unique,
       password varchar(1024) not null default '',
       phone int8 not null default -1
);

create table teams (
       id integer primary key autoincrement,
       name varchar(64) not null default '',
       division_id integer not null,
       UNIQUE (name, division_id),
       FOREIGN KEY (division_id) REFERENCES divisions(id)
);

create table divisions (
       id integer not null primary key autoincrement,
       name varchar(64) not null unique
);

create table games (
       id integer primary key autoincrement,
       team_id integer not null,
       season_id integer not null,
       time datetime not null,
       description string not null default '',
       UNIQUE (team_id, time)
       FOREIGN KEY (team_id) REFERENCES teams(id)
       FOREIGN KEY (season_id) REFERENCES seasons(id)
);

create table players_teams (
       player_id integer not null,
       team_id integer not null,
       is_manager boolean not null default false,
       remind_email boolean not null default true,
       remind_sms boolean not null default false,
       PRIMARY KEY (team_id, player_id),
       FOREIGN KEY (team_id) REFERENCES teams(id),
       FOREIGN KEY (player_id) REFERENCES players(id)
);

create table players_games (
       player_id integer not null,
       game_id integer not null,
       status varchar(32) not null default '?',
       reminder_sent boolean not null default false,
       PRIMARY KEY (player_id, game_id),
       FOREIGN KEY (player_id) REFERENCES players(id),
       FOREIGN KEY (game_id) REFERENCES games(id)
);

create table seasons (
       id integer primary key autoincrement,
       name string not null unique
);

-- tokens is used for generating temporary tokens when sending email reminders
create table tokens (
       id varchar(128) not null primary key,
       player_id not null,
       expires_on datetime not null default 2556144000, -- 1/1/2051
       FOREIGN KEY (player_id) REFERENCES players(id)
);
