#!/usr/bin/env python3

from datetime import datetime, timedelta
from multiprocessing import Pool
from threading import Thread
import re
import sys
import json
import queue
import urllib.request, urllib.parse
import time

URL = "https://pdxindoorsoccer.com/wp-content/schedules"
SEASONS = [
    "spring",
    "summer",
    "1fall",
    "2fall",
    "winter",
]
LEAGUES = ["men", "women", "multi-gender"]
DIVISIONS = ["1", "2", "3", "4", "5", "6"]
SUBDIVISIONS = ["", "A", "B", "C"]

season = sys.argv[1]
year = sys.argv[2]

q = queue.Queue()


def main():
    for n in build_queue():
        q.put(n)

    print(q.qsize())
    workers = []
    for i in range(3):
        workers.append(Thread(target=build_schedule))
        workers[-1].start()
    q.join()
    for n in workers:
        n.join()


def build_schedule():
    while True:
        try:
            info = q.get_nowait()
            if not info:
                return
            (schedule, div_id) = get_schedule(info)
            if schedule:
                games = parse_schedule(schedule, div_id)
                write_schedule(games, div_id)
            q.task_done()
        except queue.Empty:
            return


# mp is faster, but I couldn't get the queue to work, so it has to write game to
# individual files
# def main_mp():
#     with Pool(5) as p:
#         p.map(build_schedule_mp, build_queue())


# def build_schedule_mp(info):
#     (schedule, div_id) = get_schedule(info)
#     if not schedule:
#         return
#     games = parse_schedule(schedule, div_id)
#     write_schedule(games, div_id)


def get_schedule(info):
    try:
        (league, sub_div, div) = info
        file = urllib.parse.quote(f"{league}/DIV {div}{sub_div}.TXT")
        req = urllib.request.Request(f"{URL}/{season}/{file}")
        # pdxindoor rejects urllib user agent
        req.add_header("User-Agent", "python-requests/2.27.1")
        content = ""
        with urllib.request.urlopen(req) as resp:
            # if resp.status != 200:
            #     print(f"ERROR: Invalid HTTP status [status={resp.status}]")
            #     return None, None
            content = resp.read()
        print(f"Found schedule for: {file}")
        league_id = "c" if league == "multi-gender" else league[0]
        div_id = f"{league_id}{div}{sub_div.lower()}"
        return content, div_id
    except urllib.error.HTTPError as e:
        if e.reason != "Not Found":
            print(f"HTTP Error: {e.reason}")
        return None, None


def parse_schedule(schedule, div_id):
    games = []
    for l in _translit(schedule).split("\n"):
        data = _parse_schedule_line(_clean_line(l))
        if not data:
            continue

        desc = f"{data['home']} vs {data['away']}"

        g = {
            "season": f"{year}-{season}",
            "division": div_id,
            "description": desc,
            # +00:00 is added to match previous version
            "time": data["time"].isoformat() + "+00:00",
        }
        games.append(g | {"team": data["home"]})
        games.append(g | {"team": data["away"]})
    return games


def write_schedule(games, div_id):
    game_keys = ["season", "division", "team", "time", "description"]
    with open(f"pi/{div_id}.TXT", "w") as fh:
        for g in games:
            fh.write("|".join([g[x] for x in game_keys]) + "\n")


def _translit(schedule):
    return schedule.replace(b"\x92", b"'").decode("ascii")


def _clean_line(line):
    line = line.strip().upper()
    line = re.sub(r"\s+", " ", line)
    line = re.sub(r"[^A-Z0-9:&!./' ]", "", line)
    # print(f"after iconv: {line}")
    return line


game_re = re.compile(
    r"\w{3}\s+(\w{3})\s+(\d{1,2})\s+([0-9:]+|MIDNITE:?\d*|NOON:?\d*)\s*(AM|PM)?\s+(.*)VS(.*)"
)


def _parse_schedule_line(line):
    if not re.match(r"\w", line):
        return

    m = re.findall(game_re, line)
    if not m or len(m[0]) < 6:
        return

    (mon, day, hour, am_pm, home, away) = m[0]

    if hour == "NOON":
        hour = "12:00"
        am_pm = "PM"

    if hour == "MIDNITE":
        hour = "11:59"
        am_pm = "PM"

    date_str = f"{year} {mon} {day} {hour}{am_pm}"
    time = datetime.strptime(date_str, "%Y %b %d %I:%M%p")
    # the Dec/Jan boundary without a year means we may try to create a jan game in
    # the wrong year
    # puts "#{m[1]} #{m[2]} #{hour} #{am_pm} - #{time} - #{time <
    # Date.today - 120}"
    if time < datetime.now() + timedelta(days=-120):
        time = time + timedelta(years=1)

    return {
        "home": home.strip(),
        "away": away.strip(),
        "time": time,
    }


def build_queue():
    return [[l, s, d] for l in LEAGUES for d in DIVISIONS for s in SUBDIVISIONS]


if __name__ == "__main__":
    main()
