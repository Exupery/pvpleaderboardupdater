package main

import (
	_ "github.com/lib/pq"
	"database/sql"
	"os"
	"strconv"
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
		_, err := txn.Exec(qry.Before)
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

func setLeaderboard(bracket string, entries *[]LeaderboardEntry) {
	println(bracket, len(*entries))	// TODO DELME
	const qry string =
		`INSERT INTO bracket_2v2 (ranking, player_id, rating, season_wins, season_losses, last_update)
		SELECT $1, $2, $3, $4, $5, $6
		WHERE NOT EXISTS (SELECT 1 FROM bracket_2v2 WHERE player_id=$7)`
	args := make([][]interface{}, 0)

	/*for _, entry := range *entries {
		params := []interface{}{}
		args = append(args, params)
	}*/

	numInserted := insert(Query{Sql: qry, Args: args, Before: "TRUNCATE TABLE bracket_2v2"})
	logger.Printf("%s leaderboard set with %v entries", bracket, numInserted)
}

func upsertPlayers(players *[]Player) {
	// postgres doesn't have an upsert mechanism so add new players then update all
	addPlayers(players)
	updatePlayerDetails(players)
	// TODO UPDATE PLAYER SPECS TALENTS GLYPHS
	// TODO UPDATE PLAYER ACHIEVEMENTS
}

func addPlayers(players *[]Player) {
	const qry string =
		`INSERT INTO players (name, realm_slug) SELECT $1, $2
		WHERE NOT EXISTS (SELECT 1 FROM players WHERE name=$3 AND realm_slug=$4)`
	args := make([][]interface{}, 0)

	for _, player := range *players {
		// realm may be empty if character is transferring
		if player.RealmSlug != "" {
			params := []interface{}{player.Name, player.RealmSlug, player.Name, player.RealmSlug}
			args = append(args, params)
		}
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Added %v players", numInserted)
}

func updatePlayerDetails(players *[]Player) {
	const qry string =
		`UPDATE players SET class_id=$1, spec_id=$2, faction_id=$3, race_id=$4, guild=$5,
		gender=$6, achievement_points=$7, honorable_kills=$8, last_update=NOW()
		WHERE name=$9 AND realm_slug=$10`
	args := make([][]interface{}, 0)

	for _, player := range *players {
		params := []interface{}{player.ClassId, player.SpecId, player.FactionId, player.RaceId, player.Guild,
			player.Gender, player.AchievementPoints, player.HonorableKills, player.Name, player.RealmSlug}
		args = append(args, params)
	}

	numInserted := insert(Query{Sql: qry, Args: args})
	logger.Printf("Updated %v players", numInserted)
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