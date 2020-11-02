[![Build Status](https://travis-ci.org/Exupery/pvpleaderboardupdater.svg)](https://travis-ci.org/Exupery/pvpleaderboardupdater)
# PvPLeaderBoard Updater

Updates the World of Warcraft PvP leaderboard data consumed by [pvpleaderboard](https://github.com/Exupery/pvpleaderboard)

Use: run `pvpleaderboard` to update the `$DB_URL` database with the current data from Blizzard's [API](https://develop.battle.net/documentation/world-of-warcraft)

Environment variables:
* `DB_URL` the URL of the [PostgreSQL] database to use (required)
* `BATTLE_NET_CLIENT_ID` [battle.net](https://develop.battle.net/) Client ID (required)
* `BATTLE_NET_SECRET` [battle.net](https://develop.battle.net/) Client ID (required)
* `MAX_PER_BRACKET` maximum number of players to retrieve per bracket (optional, will retrieve all players for each bracket if not set)
* `GROUP_SIZE` number of players each goroutine should handle when importing player details (optional)
* `MAX_DB_CONNECTIONS` maximum size of the DB connection pool  (optional)
