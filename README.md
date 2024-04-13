# Teamvite - recreational sports game management

Teamvite is an web app I wrote to manage my recreational sports teams. It supports sending reminders (email or sms) and tracking responses. Most views are mobile-optimized for ease of use.

Players can also download an ical calendar that has all the games and updates automatically.

I chose to write Teamvite in Go because of its low resource requirements compared to python or ruby and because I wanted more experience with Go. Note that this is actually v2 of this app, the first being picklespears (written in ruby).

This is the first app I've written in Go, so there are things I've likely missed. Feel free to open an issue with feedback on things I could do better. Thanks!

## System design

### Goals:
- simple deployment, development, and dependency processes
- mobile and desktop friendly
- low resource requirements
- reasonable security
- trade features for simpler design
- minimal dependencies

### Deployment Design

Teamvite is statically compiled using musl with sqlite bindings. (see bin/build.sh)

A faux fs is created to store all the text templates within the binary itself (see templates.go)

Deploys are done by copying the binary, moving the symlink and restarting the webserver. (see bin/deploy.sh)

The only library dependencies are ones that I don't feel qualified to write (bcrypt and sqlite3 bindings) or are very similar to any implementation I would write (httprouter).

Teamvite can be run without a proxy webserver (ex. nginx) in front of it, and will deliver static files.

### Package Layout

System module/package design follows ideas proposed in [Standard-Package-Layout](https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1#.ds38va3pp) and [Style-guidelines-for-Go-Packages](https://rakyll.org/style-packages/)

Packages:
- http: anything related to sending or receiving http traffic. html templates
- sqlite: anything that interacts with the sqlite database
- teamvite: anything specific to the teamvite app, domain objects, and interfaces
- reminders: send game reminders via email and SMS

This follows recommendations from Ben Johnson on how to organize Go modules and
Ted Kaminski on using modules to hide implementation details.

Views use Go's default template library and db access is done through sqlx.

### Use of contex.Context

Teamvite is an open system. A logged in user can see other teams and players on those teams, but unless they are a manager or looking at themselves, they shouldn't be able to make changes.

In the http package, the request context will contain the domain type based on the current route. So if the url includes /team/ the context will include a Team struct in it. Same for players and games.


## Interal Documentation

Documentation for common actions. Mostly a reference for me to remind myself how to do them.


### Add new games (for Portland Indoor season)
1. Insert new season in seasons table
   ```
     INSERT INTO seasons (name) VALUES ('2022-winter');
   ```
2. Copy team placement email into file (ex 2022-winter-placements.txt)
   File should be formatted like:
   ```
   <div_name>
   <team>
   .
   .
   <blank_line>
   <div_name>
   ```
3. Save new games once schedule is out (on laptop)
   ```
     SEASON=summer
     bin/pdx_indoor_schedule.py $SEASON 2022
   ```
4. Make sure there any naming differences are captured
   ```
     env SEASON=summer /bin/bash -c 'diff -u <(cut -f3 -d\| pi_games-$SEASON-2022.txt |sort -u) <(sort 2022-$SEASON-placements.txt)' |less
   ```

5. Update teams and divisions from placements file
   Note: has to be run from digitalocean droplet
   ```
     ./pdx_indoor_team_placements.py \
         2022-$SEASON-placements.txt \
         --db /var/www/teamvite/teamvite.db
   ```
6. Upload games to teamvite.com (on laptop)
   ```
     env APP_URL=https://teamvite.com bin/upload_schedule.rb \
       pi_games-$SEASON-2022.txt |tee upload_schedule.log
   ```
7. Email new schedule to team

8. Profit!


### Building and deploying

    bin/build.sh && bin/deploy.sh

### Testing Game Reminders
- Install [MailHog](https://github.com/mailhog/MailHog)
- start mailhog `MailHog`
- check config.json is set to localhost port 1025
- send reminders:  `curl --silent http://teamvitedev.com:8080/send_game_reminders -H 'Content-Type: "application/json"' |jq`
- view them on mailhog `http://0.0.0.0:8025/`

### Testing SMS callbacks
docs on post body
https://www.twilio.com/docs/messaging/guides/webhook-request


    curl -i -X POST --silent \
         http://teamvitedev.com:8080/sms \
         -H "Content-Type: application/x-www-form-urlencoded" \
         --data 'From=%2B19715347840&Body=yessir'

    curl -i -X POST --silent \
         http://teamvitedev.com:8080/sms \
         -H "Content-Type: application/x-www-form-urlencoded" \
         --data 'From=%2B19715347840&Body=nope'

    curl -i -X POST --silent \
         http://teamvitedev.com:8080/sms \
         -H "Content-Type: application/x-www-form-urlencoded" \
         --data 'From=%2B19715347840&Body=maybe?'

    curl -i -X POST --silent \
         http://teamvitedev.com:8080/sms \
         -H "Content-Type: application/x-www-form-urlencoded" \
         --data 'From=%2B19715347840&Body=STOP'

    curl -i -X POST --silent \
         http://teamvitedev.com:8080/sms \
         -H "Content-Type: application/x-www-form-urlencoded" \
         --data 'From=%2B19715347840&Body=stop'

    curl -i -X POST --silent \
         http://teamvitedev.com:8080/sms \
         -H "Content-Type: application/x-www-form-urlencoded" \
         --data 'From=1234&Body=stop'

    curl -i -X POST --silent \
         http://teamvitedev.com:8080/sms \
         -H "Content-Type: application/x-www-form-urlencoded" \
         --data 'Body=stop'


### Reset Password

    ./teamvite resetpassword [email] [password]

