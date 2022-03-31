#!/usr/bin/env ruby

require 'date'
require 'uri'
require 'net/http'
require 'json'

raise 'APP_URL must be in environment' unless ENV['APP_URL']

APP_URL = ENV['APP_URL']

def add_game_for_team(season_id, team_id, game_time, description)
  uri = URI("#{APP_URL}/game")

  req = Net::HTTP::Post.new(uri)
  req['Content-Type'] = 'application/json'

  g = {
    team_id: team_id,
    season_id: season_id,
    time: game_time.iso8601, # iso8601 formatted time, UTC TZ
    description: description
  }

  req.body = g.to_json

  res = Net::HTTP.start(uri.host, uri.port, use_ssl: APP_URL =~ /https/) do |http|
    http.request(req)
  end

  if res.code != '200' && res.body !~ /UNIQUE constraint failed: games.team_id, games.time/
    raise "#{uri} bad: #{res.code} #{res.body} for #{g}"
  end

  puts "adding game: #{res.body}"
end

@seasons = {}
def get_season(name)
  check_get @seasons, name, "/season?name=#{name}"
end

@teams = {}
def get_team(name)
  check_get @teams, name, "/team?name=#{name}"
end

def check_get(memo, name, path)
  unless memo[name]
    uri = URI("#{APP_URL}#{path}")

    req = Net::HTTP::Get.new(uri)
    req['Content-Type'] = 'application/json'

    res = Net::HTTP.start(uri.hostname, uri.port, use_ssl: APP_URL =~ /https/) do |http|
      http.request(req)
    end

    raise "#{uri} bad: #{res.code} #{res.body}" if res.code != '200'

    objs = JSON.parse(res.body).select { |n| n['name'] == name }
    raise "Too many responses: #{objs}" if objs.length > 1

    memo[name] = objs[0]
  end
  memo[name]
end

ARGV.each do |filename|
  puts "uploading games in #{filename}"
  f = File.new(filename)
  num_games = 0
  f.each do |line|
    line.chop!
    (season, division, team, time, description) = line.split '|'

    game_time = DateTime.parse time

    s = get_season(season)
    t = get_team(team)

    puts("ERROR invalid team: #{team}") if t == nil

    add_game_for_team(s['id'], t['id'], game_time, description)
    num_games += 1
  end
  puts "created #{num_games} games"
end
