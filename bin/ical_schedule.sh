#!/bin/bash

# Download and parse an iCal (.ics) file to display schedule
# Usage: ./ical_schedule.sh [-v|--verbose] [team_name]

VERBOSE=0
TEAM_NAME=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=1
            shift
            ;;
        *)
            TEAM_NAME="$1"
            shift
            ;;
    esac
done

URL="https://sportsix.sports-it.com/ical?cid=portlandindoor&id=27062&k=861fc05a34bf88b3c689a89ef8f34384"

curl -sL "$URL" | awk -v team="$TEAM_NAME" -v verbose="$VERBOSE" '
BEGIN { RS="BEGIN:VEVENT"; FS="\n" }
NR > 1 {
    dtstart = ""
    summary = ""
    for (i = 1; i <= NF; i++) {
        if ($i ~ /^DTSTART:/) {
            gsub(/^DTSTART:/, "", $i)
            gsub(/\r/, "", $i)
            dtstart = $i
        }
        if ($i ~ /^SUMMARY:/) {
            gsub(/^SUMMARY:/, "", $i)
            gsub(/\r/, "", $i)
            summary = $i
        }
    }

    # Filter by team name if provided
    if (team != "" && index(summary, team) == 0) next

    if (dtstart != "") {
        # Parse YYYYMMDDTHHMMSS format
        year = substr(dtstart, 3, 2)
        month = substr(dtstart, 5, 2) + 0  # +0 removes leading zero
        day = substr(dtstart, 7, 2) + 0
        hour = substr(dtstart, 10, 2) + 0
        min = substr(dtstart, 12, 2)

        # Convert 24h to 12h format
        if (hour >= 12) {
            ampm = "pm"
            if (hour > 12) hour = hour - 12
        } else {
            ampm = "am"
            if (hour == 0) hour = 12
        }

        # Clean up summary: remove score (e.g., "W 7-1" or "L 2-6")
        game = summary
        gsub(/ [WLT] [0-9]+-[0-9]+ /, " ", game)

        # Print with sortable prefix (YYYYMMDD)
        if (verbose == 1) {
            printf "%s%02d%02d %d/%d/%s %d:%s%s - %s\n", substr(dtstart,1,4), month, day, month, day, year, hour, min, ampm, game
        } else {
            printf "%s%02d%02d %d/%d/%s %d:%s%s\n", substr(dtstart,1,4), month, day, month, day, year, hour, min, ampm
        }
    }
}
' | sort | cut -d' ' -f2-
