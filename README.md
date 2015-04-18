[![Build Status](https://travis-ci.org/Exupery/pvpleaderboardupdater.svg)](https://travis-ci.org/Exupery/pvpleaderboardupdater)
# PvPLeaderBoard Updater

Updates the World of Warcraft PvP leaderboard data consumed by [pvpleaderboard](https://github.com/Exupery/pvpleaderboard)

Use: run `pvpleaderboard` to update the `$DB_URL` database with the current data from Blizzard's [API](http://blizzard.github.io/api-wow-docs/)

Flags:
* `-static` updates the achievement, class, faction, glyph, race, realm, specialization, and talent data
* `-base ARG` uses `ARG` as the base URI for Blizzard's API instead of the default (https://us.battle.net/api/wow/)

Environment variables:
* `DB_URL` the URL of the [PostgreSQL] database to use (required)
* `TEST_BASE_URI` the base URI of the API to use with tests (required to run tests)
* `MAX_PER_BRACKET` maximum number of players to retrieve per bracket (optional, will retrieve all players for each bracket if not set)

