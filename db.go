package main

import (
	"database/sql"
	"strconv"
	"strings"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	cmap "github.com/orcaman/concurrent-map/v2"
)

const defaultMaxDbConnections int = 15

var db *sql.DB = dbConnect()

var lock sync.Mutex

var realmSlugs = make(map[int]string)

func dbConnect() *sql.DB {
	var dbURL string = getEnvVar("DB_URL")

	db, err := sql.Open("pgx", dbURL)
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
		var bRes sql.Result
		if len(qry.BeforeArgs) == 0 {
			bRes, err = bStmt.Exec()
		} else {
			bRes, err = bStmt.Exec(qry.BeforeArgs...)
		}
		if err != nil {
			logger.Printf("%s Before query failed: %s", errPrefix, err)
			return 0
		}
		bQuery := qry.Before[0:24]
		bAffected, _ := bRes.RowsAffected()
		logger.Printf("Before query '%s' impacted %d rows", bQuery, bAffected)
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
	if len(leaderboard) == 0 {
		return
	}
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
		faction_id, race_id, gender, guild, last_login, profile_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, to_timestamp($10), $11)
		ON CONFLICT (realm_id, blizzard_id) DO UPDATE SET name=$1, spec_id=$5, faction_id=$6,
		race_id=$7, gender=$8, guild=$9, last_login=to_timestamp($10), last_update=NOW(), profile_id=$11`
	args := make([][]interface{}, 0)

	for _, player := range players {
		if player.SpecID == 0 {
			continue
		}
		params := []interface{}{player.Name, player.RealmID, player.BlizzardID, player.ClassID,
			player.SpecID, player.FactionID, player.RaceID, player.Gender, player.Guild,
			player.LastLogin, player.ProfileID}
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

// Mark all existing player_talent and player_pvp_talent entries
// as stale so we can delete any that aren't set to false after
// all the addPlayerTalents calls have concluded.
func markStalePlayerTalents() {
	const talentQuery string = `UPDATE players_talents SET stale=TRUE`
	const pvpTalentQuery string = `UPDATE players_pvp_talents SET stale=TRUE`
	execute(talentQuery)
	execute(pvpTalentQuery)
}

func addPlayerTalents(playersTalents map[int]playerTalents) {
	if len(playersTalents) == 0 {
		return
	}
	const talentQuery string = `INSERT INTO players_talents (player_id, talent_id, stale)
		SELECT $1, $2, FALSE WHERE EXISTS (SELECT 1 FROM talents WHERE id=$2) ON CONFLICT (player_id, talent_id) DO UPDATE SET stale=FALSE`
	const pvpTalentQuery string = `INSERT INTO players_pvp_talents (player_id, pvp_talent_id, stale)
		SELECT $1, $2, FALSE WHERE EXISTS (SELECT 1 FROM pvp_talents WHERE id=$2) ON CONFLICT (player_id, pvp_talent_id) DO UPDATE SET stale=FALSE`
	talentArgs := make([][]interface{}, 0)
	pvpTalentArgs := make([][]interface{}, 0)

	for id, talents := range playersTalents {
		for _, talent := range talents.Talents {
			talentArgs = append(talentArgs, []interface{}{id, talent})
		}
		for _, pvptalent := range talents.PvPTalents {
			pvpTalentArgs = append(pvpTalentArgs, []interface{}{id, pvptalent})
		}
	}

	logger.Printf("Upserting up to %d players=>talents", len(playersTalents))
	numInserted := insert(query{SQL: talentQuery, Args: talentArgs})
	logger.Printf("Mapped %d players=>talents", numInserted)

	logger.Printf("Upserting up to %d players=>PvP talents", len(playersTalents))
	numInserted = insert(query{SQL: pvpTalentQuery, Args: pvpTalentArgs})
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

	logger.Printf("Upserting up to %d players=>achievements", len(playerAchievements))
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

	logger.Printf("Upserting up to %d players=>stats", len(playersStats))
	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Mapped %d players=>stats", numInserted)
}

func addPlayerItems(playersItems *cmap.ConcurrentMap[string, items]) {
	const qry string = `INSERT INTO players_items
		(player_id, head, neck, shoulder, back, chest, shirt,
		tabard, wrist, hands, waist, legs, feet, finger1, finger2, trinket1, trinket2, mainhand, offhand)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (player_id) DO UPDATE SET head=$2, neck=$3, shoulder=$4, back=$5, chest=$6,
		shirt=$7, tabard=$8, wrist=$9, hands=$10, waist=$11, legs=$12, feet=$13, finger1=$14,
		finger2=$15, trinket1=$16, trinket2=$17, mainhand=$18, offhand=$19`
	args := make([][]interface{}, 0)

	// for id, pi := range playersItems {
	for tpl := range playersItems.Iter() {
		id, _ := strconv.Atoi(tpl.Key)
		pi := tpl.Val
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

	logger.Println("Upserting players=>items")
	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Mapped %d players=>items", numInserted)
}

func addItems(equippedItems map[int]item) {
	// Make this method effectively single-threaded since so many players are
	// wearing many of the same items - this avoids deadlocks at the DB level
	lock.Lock()
	defer lock.Unlock()

	const qry string = `INSERT INTO items (id, name, quality) VALUES ($1, $2, $3) ON CONFLICT (id)
		DO UPDATE SET name=$2, quality=$3, last_update=NOW()`

	args := make([][]interface{}, 0)
	for id, item := range equippedItems {
		if id == 0 {
			continue
		}
		args = append(args, []interface{}{id, item.Name, item.Quality})
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d items", numInserted)
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
		// Do not unset icon if we couldn't retrieve it due to Blizzard API flakiness
		if spec.Icon != "" {
			args = append(args, params)
		}
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted or updated %d specs", numInserted)
}

func addTalents(talents *[]talent) {
	if len(*talents) == 0 {
		return
	}
	const staleQuery string = `UPDATE talents SET stale=TRUE`
	execute(staleQuery)

	const qry string = `INSERT INTO talents (id, spell_id, class_id, spec_id, name, icon,
		node_id, display_row, display_col, stale, cat, hero_specs) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, FALSE, $10, $11) ON
		CONFLICT (id) DO UPDATE SET spell_id = $2, class_id = $3, spec_id = $4, name = $5, icon = $6, node_id = $7, display_row = $8, display_col = $9, stale = FALSE, cat = $10, hero_specs = $11`
	args := make([][]interface{}, 0)

	for _, talent := range *talents {
		params := []interface{}{talent.ID, talent.SpellID, talent.ClassID, talent.SpecID, talent.Name, talent.Icon, talent.NodeID, talent.Row, talent.Col, talent.Cat, talent.HeroSpecs}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted or updated %d talents", numInserted)

	const deleteStaleQuery string = `DELETE FROM talents WHERE stale=TRUE`
	execute(deleteStaleQuery)
}

func addPvPTalents(pvpTalents *[]pvpTalent) {
	if len(*pvpTalents) == 0 {
		return
	}
	const staleQuery string = `UPDATE pvp_talents SET stale=TRUE`
	execute(staleQuery)

	const qry string = `INSERT INTO pvp_talents (id, spell_id, spec_id, name, icon, stale)
		VALUES ($1, $2, $3, $4, $5, FALSE)
		ON CONFLICT (id) DO UPDATE SET name = $4, icon = $5, stale = FALSE`
	args := make([][]interface{}, 0)

	for _, talent := range *pvpTalents {
		params := []interface{}{talent.ID, talent.SpellID, talent.SpecID, talent.Name, talent.Icon}
		args = append(args, params)
	}

	numInserted := insert(query{SQL: qry, Args: args})
	logger.Printf("Inserted %d PvP talents", numInserted)

	const deleteStaleQuery string = `DELETE FROM pvp_talents WHERE stale=TRUE`
	execute(deleteStaleQuery)
}

func addAchievements(achievements *[]achievement) {
	const qry string = `INSERT INTO achievements (id, name, description, icon)
		VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET icon = $4`
	args := make([][]interface{}, 0)

	for _, achiev := range *achievements {
		params := []interface{}{achiev.ID, achiev.Title, achiev.Description, achiev.Icon}
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

func getSpecIDForClassSpec(clazz string, spec string) int {
	// Can't simply lookup IDs via names because in the solo shuffle key
	// they strip out spaces (i.e. without a placeholder)
	rows, err := db.Query("SELECT specs.id AS id, classes.name AS c, specs.name AS s FROM specs JOIN classes ON specs.class_id=classes.id")
	if err != nil {
		logger.Printf("%s %s", errPrefix, err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var c string
		var s string
		err := rows.Scan(&id, &c, &s)
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
			return 0
		}
		clazzSlug := strings.ReplaceAll(strings.ToLower(c), " ", "")
		specSlug := strings.ReplaceAll(strings.ToLower(s), " ", "")
		if clazzSlug == clazz && specSlug == spec {
			return id
		}
	}

	return 0
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
