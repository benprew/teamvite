#!/bin/bash

SQLITE_DB=import.db

rm $SQLITE_DB
sqlite3 $SQLITE_DB < ../create.sql

# Note: picklespears docker must be running
declare -A tables=(
    ["divisions"]="select id, name from divisions"
    ["teams"]="select id, name, division_id from teams"
    ["players"]="select id, name, email_address, password_hash, regexp_replace(phone_number, '[^0-9]', '', 'g') as phone_number from players"
    ["seasons"]="select 1 as id, '2021-2fall' as name"
    ["games"]="select id, team_id, 1 as season_id, extract(epoch from date) as time, description from games join teams_games on game_id = games.id where team_id in (52, 185)"
    ["players_teams"]="select player_id, team_id, is_manager from players_teams"
    ["players_games"]="select player_id, game_id, status, reminder_sent from players_games"
)

for name in "${!tables[@]}"; do
    echo exporting $name
    query=${tables[$name]}
    psql --host localhost --user postgres teamvite \
         --no-align \
         --csv \
         --output ${name}.csv \
         -c "$query"

    echo importing $name into $SQLITE_DB
    sqlite3 $SQLITE_DB ".import --csv --skip 1 ${name}.csv ${name}"
done
