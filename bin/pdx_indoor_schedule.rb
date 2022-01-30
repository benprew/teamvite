#!/usr/bin/env ruby
# frozen_string_literal: true

require 'open-uri'
require 'uri'
require 'set'
require 'date'

URL = 'https://pdxindoorsoccer.com/wp-content/schedules'

SEASONS = %w[
  spring
  summer
  1fall
  2fall
  winter
].freeze

LEAGUES = %w[men women coed].freeze
DIVISIONS = 1..6
SUBDIVISIONS = ['', 'A', 'B', 'C'].freeze

class BuildDb
  def initialize(season, year = DateTime.now.year)
    raise "invalid season '#{season}'. Seasons: #{SEASONS}" unless SEASONS.include?(season)

    @season = season
    @season_url = "#{URL}/#{season}"
    @games = Queue.new
    @workq = build_work_queue
    @year = year
  end

  def run
    pool = (0..2).map do |_i|
      Thread.new do
        while (info = @workq.deq(false))
          (league, division, sub_div) = info
          file = "/#{league}/DIV%20#{division}#{sub_div}.TXT"
          url = URI.parse(@season_url + file)
          warn "working on: #{url}"
          begin
            url.open(read_timeout: 2) do |f|
              f.each do |line|
                data = _parse_schedule_line(_clean_line(line))
                next unless data

                div = "#{league[0]}#{division}#{sub_div.downcase}"
                desc = "#{data[:home]} vs #{data[:away]}"

                g = {
                  season: "#{@year}-#{@season}",
                  division: div,
                  description: desc,
                  time: data[:time]
                }

                @games << g.merge({ team: data[:home] })
                @games << g.merge({ team: data[:away] })
              end
            end
          rescue OpenURI::HTTPError, Net::ReadTimeout, Net::OpenTimeout
            ''
            # warn "Error opening file #{file} : #{$!}"
          end
        end
      end
    end

    pool.map(&:join)

    games_file = "pi_games-#{@season}-#{@year}.txt"
    File.open(games_file, 'w') do |fh|
      header = %i[season division team time description]
      until @games.empty?
        data = @games.pop
        fh.puts header.map { |i| data[i] }.join('|')
      end
    end

    warn "games written to #{games_file}"
  end

  def build_work_queue
    workq = Queue.new
    LEAGUES.each do |league|
      DIVISIONS.each do |division|
        SUBDIVISIONS.each do |sub_div|
          workq << [league, division, sub_div]
        end
      end
    end
    workq.close
    workq
  end

  def _parse_schedule_line(line)
    return unless line.match(/\w/)

    m = /\w{3}\s+(\w{3})\s+(\d{1,2})\s+([0-9:]+|MIDNITE:?\d*|NOON:?\d*)\s*(AM|PM)?\s+(.*)VS(.*)/.match(line)
    return unless m && m[6]

    hour = m[3]
    am_pm = m[4]
    if hour == 'NOON'
      hour = '12:00'
      am_pm = 'PM'
    end
    if hour == 'MIDNITE'
      hour = '11:59'
      am_pm = 'PM'
    end
    time = DateTime.parse("#{@year} #{m[1]} #{m[2]} #{hour} #{am_pm}")
    # the Dec/Jan boundary without a year means we may try to create a jan game in the wrong year
    # puts "#{m[1]} #{m[2]} #{hour} #{am_pm} - #{time} - #{time < Date.today - 120}"
    time = DateTime.new(time.year + 1, time.month, time.day, time.hour, time.min) if time < Date.today - 120

    {
      home: m[5].strip,
      away: m[6].strip,
      time: time
    }
  end

  def _clean_line(line)
    line.encode('utf-8', 'ISO-8859-1').strip.gsub(/\s+/, ' ').upcase.gsub(%r{[^A-Z0-9:&!./ ]}, '')
  rescue ArgumentError
    open('bad_lines.txt', 'a') do |fh|
      fh.write line
    end
    warn "bad line saved to bad_lines.txt #{line}"
    ''
  end
end

BuildDb.new(*ARGV).run
