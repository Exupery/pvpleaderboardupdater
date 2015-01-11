package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type Leaderboard struct {
	Rows []LeaderboardEntry
}

type LeaderboardEntry struct {
	Name string
	Ranking int
	Rating int
	RealmId int
	RealmName string
	RealmSlug string
	RaceId int
	ClassId int
	SpecId int
	FactionId int
	GenderId int
	SeasonWins int
	SeasonLosses int
	WeeklyWins int
	WeeklyLosses int
}

var logger *log.Logger = log.New(os.Stdout, "main|", log.Ltime|log.Lmicroseconds)
const errPrefix string = "[ERROR]"

var uriBase string = "https://us.battle.net/api/wow/"

func main() {
	logger.Println("PvPLeaderBoard Starting")
	if len(os.Args) > 1 {
		uriBase = os.Args[1]
	}
	logger.Printf("WoW API URIs using '%s'", uriBase)
	getLeaderboard("2v2")
	logger.Println("PvPLeaderBoard Complete")
}

func get(path string) *[]byte {
	resp, err := http.Get(uriBase + path)

	if err != nil {
		logger.Printf("%s GET failed: %s", errPrefix, err)
		return nil
	}
	if resp.StatusCode != 200 {
		logger.Printf("%s non-200 status code: %s", errPrefix, err)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		logger.Printf("%s reading body failed: %s", errPrefix, err)
		return nil
	}

	logger.Printf("%s returned %v bytes", path, len(body))
	return &body
}

func parseLeaderboard(data *[]byte) []LeaderboardEntry {
	var leaderboard Leaderboard
	err := json.Unmarshal(*data, &leaderboard)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]LeaderboardEntry, 0)
	}
	return leaderboard.Rows
}

func getLeaderboard(bracket string) {
	var leaderboardJson *[]byte = get("leaderboard/" + bracket)
	var entries []LeaderboardEntry = parseLeaderboard(leaderboardJson)
	logger.Printf("Parsed %v %s entries", len(entries), bracket)
}
