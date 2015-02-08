package main

import (
	_ "github.com/lib/pq"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var db sql.DB = dbConnect()

func dbConnect() sql.DB {
	var dbUrl string = os.Getenv("DB_URL")
	if dbUrl == "" {
		logger.Fatalf("%s Environment variable 'DB_URL' NOT set! Aborting.", fatalPrefix)
	}

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		logger.Printf("%s Unable to connect to database: %s", errPrefix, err)	
	}
	return *db
}

// retrieving results isn't as easily abstracted as inserts/updates
// so (for now) use this template as a base in appropriate methods
func queryTemplate() {
	rows, err := db.Query("")
	if err != nil {
		logger.Printf("%s %s", errPrefix, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		err := rows.Scan(&id)
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
		}

	}
}

func insert(qry Query) int64 {
	var numInserted int64 = 0
	txn, _ := db.Begin()
	stmt, _ := txn.Prepare(qry.Sql)

	if qry.Before != "" {
		bStmt, _ := txn.Prepare(qry.Before)
		var err error = nil
		if len(qry.BeforeArgs) == 0 {
			_, err = bStmt.Exec()
		} else {
			_, err = bStmt.Exec(qry.BeforeArgs...)
		}
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
			return 0
		}
	}

	for _, params := range qry.Args {
		res, err := stmt.Exec(params...)
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
			return 0
		}
		affected, _ := res.RowsAffected()
		numInserted += affected
	}

	txn.Commit()
	return numInserted
}

func setLeaderboard(bracket string, entries *[]LeaderboardEntry, playerSlugIdMap *map[string]int) {
	before := fmt.Sprintf("TRUNCATE TABLE bracket_%s", bracket)
	qry := fmt.Sprintf(`INSERT INTO bracket_%s 
		(ranking, player_id, rating, season_wins, season_losses, last_update)
		VALUES ($1, $2, $3, $4, $5, NOW())`, bracket)
	args := make([][]interface{}, 0)

	for _, entry := range *entries {
		id := (*playerSlugIdMap)[entry.Name + entry.RealmSlug]
		if id > 0 {
			params := []interface{}{
				entry.Ranking,
				id,
				entry.Rating,
				entry.SeasonWins,
				entry.SeasonLosses}
			args = append(args, params)
		}
	}

	numInserted := insert(Query{Sql: qry, Args: args, Before: before})
	logger.Printf("%s leaderboard set with %d entries", bracket, numInserted)
}

func addPlayers(players *[]Player) {
	const qry string =
		`INSERT INTO players (name, realm_slug) SELECT $1, $2
		WHERE NOT EXISTS (SELECT 1 FROM players WHERE name=$3 AND realm_slug=$4)`
	args := make([][]interface{}, 0)

	for _, player := range *players {
		if player.RealmSlug != "" && player.Name != "" {
			params := []interface{}{player.Name, player.RealmSlug, player.Name, player.RealmSlug}
			args = append(args, params)
		}
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Added %v players", numInserted)
}

func updatePlayers(players *map[int]Player) bool {
	updated := updatePlayerDetails(players)
	if updated > 0 {
		updatePlayerTalents(players)
		updatePlayerGlyphs(players)
		updatePlayerStats(players)
		updatePlayerAchievements(players)
		return true
	} else {
		logger.Printf("%s Updated NO player details (%d expected)", errPrefix, len(*players))
		return false
	}
}

func getPlayerIdMap(players *[]Player) *map[int]Player {
	var m map[int]Player = make(map[int]Player)
	rows, err := db.Query("SELECT id, name, realm_slug FROM players")
	if err != nil {
		logger.Printf("%s %s", errPrefix, err)
	}
	defer rows.Close()
	var t map[string]int = make(map[string]int)
	for rows.Next() {
		var id int
		var name string
		var realm_slug string
		err := rows.Scan(&id, &name, &realm_slug)
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
		}
		t[name + realm_slug] = id
	}

	for _, player := range *players {
		var id int = t[player.Name + player.RealmSlug]
		if id > 0 {
			m[id] = player
		}
	}
	return &m
}

func updatePlayerDetails(players *map[int]Player) int {
	const qry string =
		`UPDATE players SET class_id=$1, spec_id=$2, faction_id=$3, race_id=$4, guild=$5,
		gender=$6, achievement_points=$7, honorable_kills=$8, last_update=NOW() WHERE id=$9`
	args := make([][]interface{}, 0)

	for id, player := range *players {
		params := []interface{}{player.ClassId, player.SpecId, player.FactionId, player.RaceId, player.Guild,
			player.Gender, player.AchievementPoints, player.HonorableKills, id}
		args = append(args, params)
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Updated %v player details", numInserted)
	return int(numInserted)
}

func updatePlayerTalents(players *map[int]Player) {
	var before string = "DELETE FROM players_talents WHERE player_id IN ("
	const qry string = "INSERT INTO players_talents (player_id, talent_id) VALUES ($1, $2)"
	args := make([][]interface{}, 0)
	beforeArgs := make([]interface{}, 0)

	var ctr int = 1
	for id, player := range *players {
		before += fmt.Sprintf("$%d,", ctr)
		beforeArgs = append(beforeArgs, id)
		for _, talent := range player.TalentIds {
			args = append(args, []interface{}{id, talent})
		}
		ctr++
	}

	before = strings.TrimRight(before, ",")
	before += ")"
	numInserted := insert(Query{Sql: qry, Args: args, Before: before, BeforeArgs: beforeArgs})
	logger.Printf("Mapped %v players=>talents", numInserted)
}

func updatePlayerGlyphs(players *map[int]Player) {
	var before string = "DELETE FROM players_glyphs WHERE player_id IN ("
	const qry string = "INSERT INTO players_glyphs (player_id, glyph_id) VALUES ($1, $2)"
	args := make([][]interface{}, 0)
	beforeArgs := make([]interface{}, 0)

	var ctr int = 1
	for id, player := range *players {
		before += fmt.Sprintf("$%d,", ctr)
		beforeArgs = append(beforeArgs, id)
		for _, glyph := range player.GlyphIds {
			args = append(args, []interface{}{id, glyph})
		}
		ctr++
	}

	before = strings.TrimRight(before, ",")
	before += ")"
	numInserted := insert(Query{Sql: qry, Args: args, Before: before, BeforeArgs: beforeArgs})
	logger.Printf("Mapped %v players=>glyphs", numInserted)
}

func updatePlayerAchievements(players *map[int]Player) {
	const qry string =
		`INSERT INTO players_achievements (player_id, achievement_id, achieved_at) SELECT $1, $2, $3
		WHERE NOT EXISTS (SELECT 1 FROM players_achievements WHERE player_id=$4 AND achievement_id=$5)`
	args := make([][]interface{}, 0)

	validIds := getAchievementIds()
	for id, player := range *players {
		for idx, achievId := range player.AchievementIds {
			if (*validIds)[achievId] {
				achievedAt := time.Unix(player.AchievementTimestamps[idx] / 1000, 0)
				args = append(args, []interface{}{id, achievId, achievedAt, id, achievId})
			}
		}
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Mapped %v players=>achievements", numInserted)
}

func updatePlayerStats(players *map[int]Player) {
	var before string = "DELETE FROM players_stats WHERE player_id IN ("
	const qry string = `INSERT INTO players_stats 
		(player_id, strength, agility, intellect, stamina, spirit, critical_strike, haste,
		attack_power, mastery, multistrike, versatility, leech, dodge, parry)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	args := make([][]interface{}, 0)
	beforeArgs := make([]interface{}, 0)

	var ctr int = 1
	for id, player := range *players {
		before += fmt.Sprintf("$%d,", ctr)
		beforeArgs = append(beforeArgs, id)
		ps := player.Stats
		stats := []interface{}{
			id,
			ps.Str,
			ps.Agi,
			ps.Int,
			ps.Sta,
			ps.Spr,
			ps.CritRating,
			ps.HasteRating,
			ps.AttackPower,
			int(ps.MasteryRating),
			int(ps.MultistrikeRating),
			int(ps.Versatility),
			int(ps.LeechRating),
			int(ps.DodgeRating),
			int(ps.ParryRating)}
		args = append(args, stats)
		ctr++
	}

	before = strings.TrimRight(before, ",")
	before += ")"
	numInserted := insert(Query{Sql: qry, Args: args, Before: before, BeforeArgs: beforeArgs})
	logger.Printf("Mapped %v players=>stats", numInserted)
}

func addRealms(realms *[]Realm) {
	const qry string =
		`INSERT INTO realms (slug, name, battlegroup, timezone, type)
		SELECT $1, $2, $3, $4, $5
		WHERE NOT EXISTS (SELECT 1 FROM realms WHERE slug=$6)`
	args := make([][]interface{}, 0)

	for _, realm := range *realms {
		params := []interface{}{realm.Slug, realm.Name, realm.Battlegroup, realm.Timezone, realm.Type, realm.Slug}
		args = append(args, params)
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Inserted %v realms", numInserted)
}

func addRaces(races *[]Race) {
	const qry string =
		`INSERT INTO races (id, name, side) SELECT $1, $2, $3
		WHERE NOT EXISTS (SELECT 1 FROM races WHERE id=$4)`
	args := make([][]interface{}, 0)

	for _, race := range *races {
		params := []interface{}{race.Id, race.Name, race.Side, race.Id}
		args = append(args, params)
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Inserted %v races", numInserted)
}

func addFactions(factions *[]Faction) {
	const qry string =
		`INSERT INTO factions (id, name) SELECT $1, $2
		WHERE NOT EXISTS (SELECT 1 FROM factions WHERE id=$3)`
	args := make([][]interface{}, 0)

	for _, faction := range *factions {
		params := []interface{}{faction.Id, faction.Name, faction.Id}
		args = append(args, params)
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Inserted %v faction", numInserted)
}

func addClasses(classes *[]Class) {
	const qry string =
		`INSERT INTO classes (id, name) SELECT $1, $2
		WHERE NOT EXISTS (SELECT 1 FROM classes WHERE id=$3)`
	args := make([][]interface{}, 0)

	for _, class := range *classes {
		params := []interface{}{class.Id, class.Name, class.Id}
		args = append(args, params)
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Inserted %v classes", numInserted)
}

func addSpecs(specs *[]Spec) {
	const qry string =
		`INSERT INTO specs (id, class_id, name, role, description, background_image, icon)
		SELECT $1, $2, $3, $4, $5, $6, $7
		WHERE NOT EXISTS (SELECT 1 FROM specs WHERE class_id=$8 AND name=$9)`
	args := make([][]interface{}, 0)

	for _, spec := range *specs {
		params := []interface{}{spec.Id, spec.ClassId, spec.Name, spec.Role, spec.Description, 
			spec.BackgroundImage, spec.Icon, spec.ClassId, spec.Name}
		args = append(args, params)
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Inserted %v specs", numInserted)
}

func addTalents(talents *[]Talent) {
	const qry string =
		`INSERT INTO talents (id, class_id, name, description, icon, tier, col)
		SELECT $1, $2, $3, $4, $5, $6, $7
		WHERE NOT EXISTS (SELECT 1 FROM talents WHERE id=$8)`
	args := make([][]interface{}, 0)

	for _, talent := range *talents {
		params := []interface{}{talent.Id, talent.ClassId, talent.Name, talent.Description,
			talent.Icon, talent.Tier, talent.Column, talent.Id}
		args = append(args, params)
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Inserted %v talents", numInserted)
}

func addGlyphs(glyphs *[]Glyph) {
	const qry string =
		`INSERT INTO glyphs (id, class_id, name, icon, item_id, type_id)
		SELECT $1, $2, $3, $4, $5, $6
		WHERE NOT EXISTS (SELECT 1 FROM glyphs WHERE id=$7)`
	args := make([][]interface{}, 0)

	for _, glyph := range *glyphs {
		params := []interface{}{glyph.Glyph, glyph.ClassId, glyph.Name, glyph.Icon,
			glyph.Item, glyph.TypeId, glyph.Glyph}
		args = append(args, params)
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Inserted %v glyphs", numInserted)
}

func addAchievements(achievements *[]Achievement) {
	const qry string =
		`INSERT INTO achievements (id, name, description, icon, points)
		SELECT $1, $2, $3, $4, $5
		WHERE NOT EXISTS (SELECT 1 FROM achievements WHERE id=$6)`
	args := make([][]interface{}, 0)

	for _, achiev := range *achievements {
		params := []interface{}{achiev.Id, achiev.Title, achiev.Description, achiev.Icon,
			achiev.Points, achiev.Id}
		args = append(args, params)
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Inserted %v achievements", numInserted)
}

func classIdSpecNameToSpecIdMap() *map[string]int {
	var m map[string]int = make(map[string]int)
	rows, err := db.Query("SELECT id, class_id, name FROM specs")
	if err != nil {
		logger.Printf("%s %s", errPrefix, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var classId int
		var name string
		err := rows.Scan(&id, &classId, &name)
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
		}
		m[strconv.Itoa(classId) + name] = id
	}
	return &m
}

func getAchievementIds() *map[int]bool {
	var m map[int]bool = make(map[int]bool)
	rows, err := db.Query("SELECT id FROM achievements")
	if err != nil {
		logger.Printf("%s %s", errPrefix, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		err := rows.Scan(&id)
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
		}
		m[id] = true
	}
	return &m
}