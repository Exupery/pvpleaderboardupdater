package main

import (
	_ "github.com/lib/pq"
	"database/sql"
	"os"
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

func insert(qry string, args [][]interface{}) int64 {
	var numInserted int64 = 0
	txn, _ := db.Begin()
	stmt, _ := txn.Prepare(qry)

	for _, params := range args {
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

func upsertEntries(bracket string, entries *[]LeaderboardEntry) {
	// TODO UPSERT ENTRIES
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

	numInserted := insert(qry, args)
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

	numInserted := insert(qry, args)
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

	numInserted := insert(qry, args)
	logger.Printf("Inserted %v faction", numInserted)
}