package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var logger *log.Logger = log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds)
const errPrefix string = "[ERROR]"
const fatalPrefix string = "[FATAL]"

var uriBase string

func main() {
	logger.Println("Updating PvPLeaderBoard")
	flag.StringVar(&uriBase, "base", "https://us.battle.net/api/wow/", "WoW API base URI")
	var importStatic *bool = flag.Bool("static", false, "Import static data (e.g. races, classes, realms, etc)")
	flag.Parse()
	logger.Printf("WoW API URIs using '%s'", uriBase)

	if *importStatic {
		importStaticData()
	}

	// brackets := []string{"2v2", "3v3", "5v5", "rbg"}
	brackets := []string{"2v2"}
	for _, bracket := range brackets {
		updateLeaderboard(bracket)
	}
	
	logger.Println("PvPLeaderBoard Updated")
}

func get(path string) *[]byte {
	resp, err := http.Get(uriBase + path)

	if err != nil {
		logger.Printf("%s GET failed: %s", errPrefix, err)
		return nil
	}
	if resp.StatusCode != 200 {
		logger.Printf("%s non-200 status code: %v", errPrefix, resp.StatusCode)
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
	type Leaderboard struct {
		Rows []LeaderboardEntry
	}
	var leaderboard Leaderboard
	err := json.Unmarshal(*data, &leaderboard)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]LeaderboardEntry, 0)
	}
	return leaderboard.Rows
}

func updateLeaderboard(bracket string) {
	var leaderboardJson *[]byte = get("leaderboard/" + bracket)
	var entries []LeaderboardEntry = parseLeaderboard(leaderboardJson)
	logger.Printf("Parsed %v %s entries", len(entries), bracket)
	setLeaderboard(bracket, &entries)
}
