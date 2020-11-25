package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver
)

const defaultMaxDbConnections int = 15

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
	maxConnections := getEnvVarOrDefault("MAX_DB_CONNECTIONS", defaultMaxDbConnections)
	db.SetMaxOpenConns(maxConnections)
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
			_, err = bStmt.Exec(qry.BeforeArgs...)
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

func updateLeaderboard(bracket string, leaderboard []leaderboardEntry) {
	const deleteQuery string = `DELETE FROM leaderboards WHERE region=$1 AND bracket=$2`
	const qry string = `INSERT INTO leaderboards
		(region, bracket, player_id, ranking, rating, season_wins, season_losses)
		SELECT $1, $2, (SELECT id FROM players WHERE realm_id=$3 AND blizzard_id=$4), $5, $6, $7, $8
		WHERE EXISTS (SELECT 1 FROM players WHERE realm_id=$3 AND blizzard_id=$4)`

	deleteArgs := []interface{}{region, bracket}
	args := make([][]interface{}, 0)

	for _, entry := range leaderboard {
		params := []interface{}{
			region,
			bracket,
			entry.RealmID,
			entry.BlizzardID,
			entry.Rank,
			entry.Rating,
			entry.SeasonWins,
			entry.SeasonLosses}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args, Before: deleteQuery, BeforeArgs: deleteArgs})
	logger.Printf("%s %s leaderboard set with %d entries", region, bracket, numInserted)
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
	var deleteTalentQuery string = `DELETE FROM players_talents WHERE player_id IN (`
	const talentQuery string = `INSERT INTO players_talents (player_id, talent_id)
		SELECT $1, $2 WHERE EXISTS (SELECT 1 FROM talents WHERE id=$2)`
	var deletePvPTalentQuery string = `DELETE FROM players_pvp_talents WHERE player_id IN (`
	const pvpTalentQuery string = `INSERT INTO players_pvp_talents (player_id, pvp_talent_id)
		SELECT $1, $2 WHERE EXISTS (SELECT 1 FROM pvp_talents WHERE id=$2)`
	deleteArgs := make([]interface{}, 0)
	talentArgs := make([][]interface{}, 0)
	pvpTalentArgs := make([][]interface{}, 0)

	ctr := 1
	for id, talents := range playersTalents {
		deleteTalentQuery += fmt.Sprintf("$%d,", ctr)
		deletePvPTalentQuery += fmt.Sprintf("$%d,", ctr)
		deleteArgs = append(deleteArgs, id)
		for _, talent := range talents.Talents {
			talentArgs = append(talentArgs, []interface{}{id, talent})
		}
		for _, pvptalent := range talents.PvPTalents {
			pvpTalentArgs = append(pvpTalentArgs, []interface{}{id, pvptalent})
		}
		ctr++
	}
	deleteTalentQuery = strings.TrimRight(deleteTalentQuery, ",")
	deletePvPTalentQuery = strings.TrimRight(deletePvPTalentQuery, ",")
	deleteTalentQuery += ")"
	deletePvPTalentQuery += ")"

	numInserted := insert(query{SQL: talentQuery, Args: talentArgs,
		Before: deleteTalentQuery, BeforeArgs: deleteArgs})
	logger.Printf("Mapped %d players=>talents", numInserted)
	numInserted = insert(query{SQL: pvpTalentQuery, Args: pvpTalentArgs,
		Before: deletePvPTalentQuery, BeforeArgs: deleteArgs})
	logger.Printf("Mapped %d players=>PvP talents", numInserted)
}

func addPlayerAchievements(playerAchievements map[int][]int) {
	const qry string = `INSERT INTO players_achievements (player_id, achievement_id) VALUES ($1, $2)
		ON CONFLICT (player_id, achievement_id) DO NOTHING`
	args := make([][]interface{}, 0)

	for id, achievements := range playerAchievements {
		for _, achievID := range achievements {
			args = append(args, []interface{}{id, achievID})
		}
	}

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

func addPlayerItems(playersItems map[int]items) {
	const qry string = `INSERT INTO players_items
		(player_id, head, neck, shoulder, back, chest, shirt,
		tabard, wrist, hands, waist, legs, feet, finger1, finger2, trinket1, trinket2, mainhand, offhand)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (player_id) DO UPDATE SET head=$2, neck=$3, shoulder=$4, back=$5, chest=$6,
		shirt=$7, tabard=$8, wrist=$9, hands=$10, waist=$11, legs=$12, feet=$13, finger1=$14,
		finger2=$15, trinket1=$16, trinket2=$17, mainhand=$18, offhand=$19`
	args := make([][]interface{}, 0)

	for id, pi := range playersItems {
		playerItems := []interface{}{
			id,
			pi.Head.ID,
			pi.Neck.ID,
			pi.Shoulder.ID,
			pi.Back.ID,
			pi.Chest.ID,
			pi.Shirt.ID,
			pi.Tabard.ID,
			pi.Wrist.ID,
			pi.Hands.ID,
			pi.Waist.ID,
			pi.Legs.ID,
			pi.Feet.ID,
			pi.Finger1.ID,
			pi.Finger2.ID,
			pi.Trinket1.ID,
			pi.Trinket2.ID,
			pi.MainHand.ID,
			pi.OffHand.ID}
		args = append(args, playerItems)
	}

	addItems(playersItems)
	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Mapped %d players=>items", numInserted)
}

func addItems(playersItems map[int]items) {
	const qry string = `INSERT INTO items (id, name) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`

	items := make(map[int]string, 0)
	for _, pi := range playersItems {
		items[pi.Head.ID] = pi.Head.Name
		items[pi.Neck.ID] = pi.Neck.Name
		items[pi.Shoulder.ID] = pi.Shoulder.Name
		items[pi.Back.ID] = pi.Back.Name
		items[pi.Chest.ID] = pi.Chest.Name
		items[pi.Shirt.ID] = pi.Shirt.Name
		items[pi.Tabard.ID] = pi.Tabard.Name
		items[pi.Wrist.ID] = pi.Wrist.Name
		items[pi.Hands.ID] = pi.Hands.Name
		items[pi.Waist.ID] = pi.Waist.Name
		items[pi.Legs.ID] = pi.Legs.Name
		items[pi.Feet.ID] = pi.Feet.Name
		items[pi.Finger1.ID] = pi.Finger1.Name
		items[pi.Finger2.ID] = pi.Finger2.Name
		items[pi.Trinket1.ID] = pi.Trinket1.Name
		items[pi.MainHand.ID] = pi.MainHand.Name
		items[pi.OffHand.ID] = pi.OffHand.Name
	}

	args := make([][]interface{}, 0)
	for id, name := range items {
		if id == 0 {
			continue
		}
		args = append(args, []interface{}{id, name})
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d items", numInserted)
}

func addPlayerSoulbinds(playerSoulbinds map[int]playerSoulbind) {
	const covQry string = `INSERT INTO players_covenants (player_id, covenant_id) VALUES ($1, $2)
		ON CONFLICT (player_id) DO UPDATE SET covenant_id=$2`
	covArgs := make([][]interface{}, 0)
	const soulbindQry string = `INSERT INTO players_soulbinds (player_id, soulbind_id) VALUES ($1, $2)
		ON CONFLICT (player_id) DO UPDATE SET soulbind_id=$2`
	soulbindArgs := make([][]interface{}, 0)
	var deleteConduitQuery string = `DELETE FROM players_conduits WHERE player_id IN (`
	deleteArgs := make([]interface{}, 0)
	const conduitQry string = `INSERT INTO players_conduits (player_id, conduit_id)
		SELECT $1, $2 WHERE EXISTS (SELECT 1 FROM conduits WHERE id=$2)
		ON CONFLICT (player_id, conduit_id) DO NOTHING`
	conduitArgs := make([][]interface{}, 0)

	ctr := 1
	for id, soulbind := range playerSoulbinds {
		if soulbind.Covenant == 0 {
			continue
		}
		covArgs = append(covArgs, []interface{}{id, soulbind.Covenant})

		if soulbind.Soulbind == 0 {
			continue
		}
		soulbindArgs = append(soulbindArgs, []interface{}{id, soulbind.Soulbind})

		deleteConduitQuery += fmt.Sprintf("$%d,", ctr)
		deleteArgs = append(deleteArgs, id)
		for _, conduitID := range soulbind.Conduits {
			conduitArgs = append(conduitArgs, []interface{}{id, conduitID})
		}
		ctr++
	}
	deleteConduitQuery = strings.TrimRight(deleteConduitQuery, ",")
	deleteConduitQuery += ")"

	if len(covArgs) > 0 {
		numInserted := insert(query{SQL: covQry, Args: covArgs})
		logger.Printf("Mapped %d players=>covenants", numInserted)
	}
	if len(soulbindArgs) > 0 {
		numInserted := insert(query{SQL: soulbindQry, Args: soulbindArgs})
		logger.Printf("Mapped %d players=>soulbinds", numInserted)
	}
	if len(conduitArgs) > 0 {
		numInserted := insert(query{SQL: conduitQry, Args: conduitArgs,
			Before: deleteConduitQuery, BeforeArgs: deleteArgs})
		logger.Printf("Mapped %d players=>conduits", numInserted)
	}
}

func setUpdateTime() {
	execute(`INSERT INTO metadata (key, last_update) VALUES ('update_time', NOW())
		ON CONFLICT (key) DO UPDATE SET last_update=NOW()`)
}

func purgeStalePlayers() {
	execute("SELECT purge_old_players()")
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

func addCovenants(covenants *[]covenant) {
	const qry string = `INSERT INTO covenants (id, name, icon) VALUES($1, $2, $3) ON CONFLICT
		(id) DO UPDATE SET icon=$3`
	args := make([][]interface{}, 0)

	for _, covenant := range *covenants {
		params := []interface{}{covenant.ID, covenant.Name, covenant.Icon}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d covenants", numInserted)
}

func addSoulbinds(soulbinds *[]soulbind) {
	const qry string = `INSERT INTO soulbinds (id, name) VALUES($1, $2) ON CONFLICT DO NOTHING`
	args := make([][]interface{}, 0)

	for _, soulbind := range *soulbinds {
		params := []interface{}{soulbind.ID, soulbind.Name}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d soulbinds", numInserted)
}

func addConduits(conduits *[]conduit) {
	const qry string = `INSERT INTO conduits (id, spell_id, name) VALUES($1, $2, $3) ON CONFLICT DO NOTHING`
	args := make([][]interface{}, 0)

	for _, conduit := range *conduits {
		params := []interface{}{conduit.ID, conduit.SpellID, conduit.Name}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d conduits", numInserted)
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

func getAchievementIds() map[int]bool {
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
	return m
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
