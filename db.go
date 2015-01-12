package main

import (
	_ "github.com/lib/pq"
	"database/sql"
	"os"
)

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
	var db sql.DB = dbConnect()
	logger.Println(db)	// TODO DELME
	// TODO UPSERT ENTRIES
}