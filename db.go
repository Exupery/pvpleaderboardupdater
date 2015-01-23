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

func upsertEntries(bracket string, entries *[]LeaderboardEntry) {
	logger.Println(db)	// TODO DELME
	// TODO UPSERT ENTRIES
}

func addRealms(realms *[]Realm) {
	var numInserted int64 = 0
	const qry string =
		`INSERT INTO realms (slug, name, battlegroup, timezone, type)
		SELECT $1, $2, $3, $4, $5
		WHERE NOT EXISTS (SELECT 1 FROM realms WHERE slug=$6)`
	txn, _ := db.Begin()
	stmt, _ := txn.Prepare(qry)

	for _, realm := range *realms {
		res, err :=
			stmt.Exec(realm.Slug, realm.Name, realm.Battlegroup, realm.Timezone, realm.Type, realm.Slug)
		if err != nil {
			logger.Printf("%s %s", errPrefix, err)
			return
		}
		affected, _ := res.RowsAffected()
		numInserted += affected
	}

	txn.Commit()
	logger.Printf("Inserted %v realms", numInserted)
}