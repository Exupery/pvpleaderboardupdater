package main

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var db *sql.DB = dbConnect()

var realmSlugs = make(map[int]string)

func dbConnect() *sql.DB {
	var dbURL string = getEnvVar("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Fatalf("%s Unable to connect to database: %s", fatalPrefix, err)
	}
	err = db.Ping()
	if err != nil {
		logger.Fatalf("%s Unable to access database: %s", fatalPrefix, err)
	}
	return db
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

func insert(qry query) int64 {
	var numInserted int64 = 0
	txn, err := db.Begin()
	if err != nil {
		logger.Fatalf("%s %s", fatalPrefix, err)
	}
	stmt, err := txn.Prepare(qry.SQL)
	if err != nil {
		logger.Fatalf("%s %s", fatalPrefix, err)
	}

	if qry.Before != "" {
		bStmt, _ := txn.Prepare(qry.Before)
		var err error = nil
		if len(qry.BeforeArgs) == 0 {
			_, err = bStmt.Exec()
		} else {
			for _, param := range qry.BeforeArgs {
				_, err = bStmt.Exec(param)
			}
		}
		if err != nil {
			logger.Printf("%s Before query failed: %s", errPrefix, err)
			return 0
		}
	}

	for _, params := range qry.Args {
		res, err := stmt.Exec(params...)
		if err != nil {
			logger.Printf("%s %s. Parameters: %v", errPrefix, err, params)
			return 0
		}
		affected, _ := res.RowsAffected()
		numInserted += affected
	}

	txn.Commit()
	return numInserted
}

func execute(sql string) {
	_, err := db.Exec(sql)

	if err != nil {
		logger.Printf("%s %s", errPrefix, err)
	}
}

func setLeaderboard(bracket string, entries *map[string]*leaderboardEntry, playerSlugIDMap *map[string]int) {
	qry := fmt.Sprintf(`INSERT INTO bracket_%s
		(ranking, player_id, rating, season_wins, season_losses, last_update)
		VALUES ($1, $2, $3, $4, $5, NOW())`, bracket)
	args := make([][]interface{}, 0)

	for slug, entry := range *entries {
		id := (*playerSlugIDMap)[slug]
		if id > 0 {
			params := []interface{}{
				entry.Rank,
				id,
				entry.Rating,
				entry.SeasonWins,
				entry.SeasonLosses}
			args = append(args, params)
		}
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("%s leaderboard set with %d entries", bracket, numInserted)
}

func addPlayers(players []*player) {
	const qry string = `INSERT INTO players (name, realm_id, blizzard_id, class_id, spec_id,
		faction_id, race_id, gender, guild) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (realm_id, blizzard_id) DO UPDATE SET name=$1, spec_id=$5, faction_id=$6,
		race_id=$7, gender=$8, guild=$9, last_update=NOW()`
	args := make([][]interface{}, 0)

	for _, player := range players {
		params := []interface{}{player.Name, player.RealmID, player.BlizzardID, player.ClassID,
			player.SpecID, player.FactionID, player.RaceID, player.Gender, player.Guild}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Added or updated %d players", numInserted)
}

func getPlayerIDs(players []*player) map[string]int {
	var m map[string]int = make(map[string]int)
	rows, err := db.Query("SELECT id, realm_id, blizzard_id FROM players")
	if err != nil {
		logger.Printf("%s %s", errPrefix, err)
	}
	defer rows.Close()
	var t map[string]int = make(map[string]int)
	for rows.Next() {
		var id int
		var realmID int
		var blizzardID int
		err := rows.Scan(&id, &realmID, &blizzardID)
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
		}
		key := playerKey(realmID, blizzardID)
		t[key] = id
	}

	for _, player := range players {
		tKey := playerKey(player.RealmID, player.BlizzardID)
		var id int = t[tKey]
		if id > 0 {
			m[player.Path] = id
		}
	}
	return m
}

func addPlayerTalents(playersTalents map[int]playerTalents) {
	const deleteTalentQuery string = `DELETE FROM players_talents WHERE player_id=$1`
	const talentQuery string = `INSERT INTO players_talents (player_id, talent_id) VALUES ($1, $2)`
	const deletePvPTalentQuery string = `DELETE FROM players_pvp_talents WHERE player_id=$1`
	const pvpTalentQuery string = `INSERT INTO players_pvp_talents (player_id, pvp_talent_id) VALUES ($1, $2)`
	deleteArgs := make([]interface{}, 0)
	talentArgs := make([][]interface{}, 0)
	pvpTalentArgs := make([][]interface{}, 0)

	for id, talents := range playersTalents {
		deleteArgs = append(deleteArgs, id)
		for _, talent := range talents.Talents {
			talentArgs = append(talentArgs, []interface{}{id, talent})
		}
		for _, pvptalent := range talents.PvPTalents {
			pvpTalentArgs = append(pvpTalentArgs, []interface{}{id, pvptalent})
		}
	}

	numInserted := insert(query{SQL: talentQuery, Args: talentArgs,
		Before: deleteTalentQuery, BeforeArgs: deleteArgs})
	logger.Printf("Mapped %d players=>talents", numInserted)
	numInserted = insert(query{SQL: pvpTalentQuery, Args: pvpTalentArgs,
		Before: deletePvPTalentQuery, BeforeArgs: deleteArgs})
	logger.Printf("Mapped %d players=>PvP talents", numInserted)
}

func updatePlayerAchievements(players *map[int]*player) {
	const qry string = `INSERT INTO players_achievements (player_id, achievement_id, achieved_at) SELECT $1, $2, $3
		WHERE NOT EXISTS (SELECT 1 FROM players_achievements WHERE player_id=$4 AND achievement_id=$5)`
	args := make([][]interface{}, 0)

	// validIds := getAchievementIds()
	// for id, player := range *players {
	// for idx, achievID := range player.AchievementIDs {
	// 	if (*validIds)[achievID] {
	// 		achievedAt := time.Unix(player.AchievementTimestamps[idx]/1000, 0)
	// 		args = append(args, []interface{}{id, achievID, achievedAt, id, achievID})
	// 	}
	// }
	// }

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Mapped %d players=>achievements", numInserted)
}

func addPlayerStats(playersStats map[int]stats) {
	const qry string = `INSERT INTO players_stats
		(player_id, strength, agility, intellect, stamina, critical_strike, haste,
		versatility, mastery, leech, dodge, parry)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) ON CONFLICT(player_id)
		DO UPDATE SET strength=$2, agility=$3, intellect=$4, stamina=$5, critical_strike=$6,
		haste=$7, versatility=$8, mastery=$9, leech=$10, dodge=$11, parry=$12`
	args := make([][]interface{}, 0)

	for id, ps := range playersStats {
		stats := []interface{}{
			id,
			ps.Strength,
			ps.Agility,
			ps.Intellect,
			ps.Stamina,
			ps.CriticalStrike,
			ps.Haste,
			ps.Versatility,
			ps.Mastery,
			ps.Leech,
			ps.Dodge,
			ps.Parry}
		args = append(args, stats)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Mapped %d players=>stats", numInserted)
}

func updatePlayerItems(players *map[int]*player) {
	const qry string = `INSERT INTO players_items
		(player_id, average_item_level, average_item_level_equipped, head, neck, shoulder, back, chest, shirt,
		tabard, wrist, hands, waist, legs, feet, finger1, finger2, trinket1, trinket2, mainhand, offhand)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`
	args := make([][]interface{}, 0)
	items := make(map[int]item)

	// var ctr int = 1
	// for id, player := range *players {
	// pi := player.Items
	// playerItems := []interface{}{
	// 	id,
	// 	pi.AverageItemLevel,
	// 	pi.AverageItemLevelEquipped,
	// 	pi.Head.ID,
	// 	pi.Neck.ID,
	// 	pi.Shoulder.ID,
	// 	pi.Back.ID,
	// 	pi.Chest.ID,
	// 	pi.Shirt.ID,
	// 	pi.Tabard.ID,
	// 	pi.Wrist.ID,
	// 	pi.Hands.ID,
	// 	pi.Waist.ID,
	// 	pi.Legs.ID,
	// 	pi.Feet.ID,
	// 	pi.Finger1.ID,
	// 	pi.Finger2.ID,
	// 	pi.Trinket1.ID,
	// 	pi.Trinket2.ID,
	// 	pi.MainHand.ID,
	// 	pi.OffHand.ID}
	// args = append(args, playerItems)
	// apppendItems(&items, pi)
	// 	ctr++
	// }

	updateItems(&items)
	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Mapped %d players=>items", numInserted)
}

func apppendItems(itemsMap *map[int]item, itemsToAdd items) {
	items := []item{itemsToAdd.Head, itemsToAdd.Neck, itemsToAdd.Shoulder, itemsToAdd.Back, itemsToAdd.Chest,
		itemsToAdd.Shirt, itemsToAdd.Tabard, itemsToAdd.Wrist, itemsToAdd.Hands, itemsToAdd.Waist,
		itemsToAdd.Legs, itemsToAdd.Feet, itemsToAdd.Finger1, itemsToAdd.Finger2, itemsToAdd.Trinket1,
		itemsToAdd.Trinket2, itemsToAdd.MainHand, itemsToAdd.OffHand}

	for _, item := range items {
		if item.ID > 0 {
			(*itemsMap)[item.ID] = item
		}
	}
}

func updateItems(items *map[int]item) {
	const qry string = `INSERT INTO items (id, name, icon, item_level)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (SELECT 1 FROM items WHERE id=$5)`
	args := make([][]interface{}, 0)

	for _, item := range *items {
		params := []interface{}{item.ID, item.Name}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d items", numInserted)
}

func setUpdateTime() {
	execute(`INSERT INTO metadata (key, last_update)
		SELECT 'update_time', NOW()
		WHERE NOT EXISTS (SELECT 1 FROM metadata WHERE key='update_time')`)

	execute("UPDATE metadata SET last_update=NOW() WHERE key='update_time'")
}

func addRealms(realms *[]realm, region string) {
	const qry string = `INSERT INTO realms (id, slug, name, region)
	VALUES($1, $2, $3, $4) ON CONFLICT DO NOTHING`
	args := make([][]interface{}, 0)

	for _, realm := range *realms {
		params := []interface{}{realm.ID, realm.Slug, realm.Name, region}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d realms", numInserted)
}

func addRaces(races *[]race) {
	const qry string = `INSERT INTO races (id, name) VALUES($1, $2) ON CONFLICT DO NOTHING`
	args := make([][]interface{}, 0)

	for _, race := range *races {
		params := []interface{}{race.ID, race.Name}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d races", numInserted)
}

func addClasses(classes *[]class) {
	const qry string = `INSERT INTO classes (id, name) VALUES($1, $2) ON CONFLICT DO NOTHING`
	args := make([][]interface{}, 0)

	for _, class := range *classes {
		params := []interface{}{class.ID, class.Name}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d classes", numInserted)
}

func addSpecs(specs *[]spec) {
	const qry string = `INSERT INTO specs (id, class_id, name, role, icon)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET icon = $5`
	args := make([][]interface{}, 0)

	for _, spec := range *specs {
		params := []interface{}{spec.ID, spec.ClassID, spec.Name, spec.Role, spec.Icon}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted or updated %d specs", numInserted)
}

func addTalents(talents *[]talent) {
	const deleteQuery string = `TRUNCATE TABLE talents CASCADE`
	const qry string = `INSERT INTO talents (id, spell_id, class_id, spec_id, name, icon, tier, col)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (id) DO UPDATE SET spec_id = NULL`
	args := make([][]interface{}, 0)

	for _, talent := range *talents {
		params := []interface{}{talent.ID, talent.SpellID, talent.ClassID, talent.SpecID, talent.Name,
			talent.Icon, talent.Tier, talent.Column}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args, Before: deleteQuery})
	logger.Printf("Inserted %d talents", numInserted)
}

func addPvPTalents(pvpTalents *[]pvpTalent) {
	const deleteQuery string = `TRUNCATE TABLE pvp_talents CASCADE`
	const qry string = `INSERT INTO pvp_talents (id, spell_id, spec_id, name, icon)
		VALUES ($1, $2, $3, $4, $5)`
	args := make([][]interface{}, 0)

	for _, talent := range *pvpTalents {
		params := []interface{}{talent.ID, talent.SpellID, talent.SpecID, talent.Name, talent.Icon}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args, Before: deleteQuery})
	logger.Printf("Inserted %d PvP talents", numInserted)
}

func addAchievements(achievements *[]achievement) {
	const qry string = `INSERT INTO achievements (id, name, description)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`
	args := make([][]interface{}, 0)

	for _, achiev := range *achievements {
		params := []interface{}{achiev.ID, achiev.Title, achiev.Description}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d achievements", numInserted)
}

func classIDSpecNameToSpecIDMap() *map[string]int {
	var m map[string]int = make(map[string]int)
	rows, err := db.Query("SELECT id, class_id, name FROM specs")
	if err != nil {
		logger.Printf("%s %s", errPrefix, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var classID int
		var name string
		err := rows.Scan(&id, &classID, &name)
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
		}
		m[strconv.Itoa(classID)+name] = id
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

func getRealmSlug(id int) string {
	slug, ok := realmSlugs[id]
	if ok {
		return slug
	}
	mapRealmSlugs()
	return realmSlugs[id]
}

func mapRealmSlugs() {
	rows, err := db.Query("SELECT id, slug FROM realms")
	if err != nil {
		logger.Printf("%s %s", errPrefix, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var slug string
		err := rows.Scan(&id, &slug)
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
		}
		realmSlugs[id] = slug
	}
}
