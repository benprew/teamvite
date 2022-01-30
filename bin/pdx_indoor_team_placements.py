#!/usr/bin/env python3

import argparse
import re
import sqlite3


def args():
    parser = argparse.ArgumentParser(description="Parse and update teams for season")
    parser.add_argument("file", type=str, help="filename to parse")
    parser.add_argument("--db", type=str, help="path to db", default="teamvite.db")
    parser.add_argument("--dry-run", action="store_true", help="don't commit changes")

    return parser.parse_args()


def main(args):
    con = sqlite3.connect(args.db)
    cur = con.cursor()
    teams = parse_teams(args.file)
    for d, teams in teams.items():
        print(f"working on {len(teams)} in division: {d}")
        for t in teams:
            update_or_create_team(cur, d, t)
    if args.dry_run:
        print("not commiting changes")
    else:
        con.commit()


def update_or_create_team(cur, division, team):
    row = cur.execute(
        "select count(*) from teams t join divisions d on division_id = d.id where d.name = ? and t.name = ?",
        (division, team),
    ).fetchone()
    if row[0] == 1:
        print(f"skipping team: {team}")
        return

    div_id = 0
    rows = cur.execute(
        "select id from divisions where name = ?", (division,)
    ).fetchall()
    if len(rows) == 0:
        print(f"creating division {division}")
        cur.execute("insert into divisions (name) values (?)", (division,))
        row = cur.execute(
            "select id from divisions where name = ?", (division,)
        ).fetchone()
        div_id = row[0]
    else:
        div_id = rows[0][0]

    if div_id == 0:
        raise Exception(f"unable to find division {division}")

    rows = cur.execute(
        "select d.name, t.id from teams t join divisions d on division_id = d.id where t.name = ?",
        (team,),
    ).fetchall()
    if len(rows) == 0:
        print(f"creating team: {team}")
        cur.execute(
            "insert into teams (division_id, name) values (?, ?)", (div_id, team)
        )
    else:
        # divisions are start with m/w/c for mens/womens/coed if the team is in
        # the same m/w/c, we move them, otherwise we create a new team in the new
        # division
        if len(rows) == 1 and rows[0][0][0] == division[0]:
            (div_name, team_id) = rows[0]
            print(f"updating team {team} from {div_name} to {division}")
            cur.execute(
                "update teams set division_id = ? where id = ?", (div_id, team_id)
            )
        else:
            print(f"team ({team}) {division} already exists in {rows}, ignoring")


def parse_teams(filename):
    teams = {}

    with open(filename, "r") as fh:
        new_div = True
        for line in fh:
            line = line.strip()
            if line == "":
                new_div = True
                continue

            if new_div:
                div_name = re.sub(
                    "^([A-Z])[A-Z']+ ([A-Z0-9]+) ?.*", "\\1\\2", line
                ).lower()
                new_div = False
            else:
                if div_name not in teams:
                    teams[div_name] = []

                teams[div_name] += [line]

    return teams


if __name__ == "__main__":
    main(args())
