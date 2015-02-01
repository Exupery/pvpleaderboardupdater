package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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
	} else {
		updateLeaderboards()
	}
	
	logger.Println("PvPLeaderBoard Updated")
}

func get(path string) *[]byte {
	resp, err := http.Get(uriBase + path)

	if err != nil {
		logger.Printf("%s GET '%s' failed: %s", errPrefix, path, err)
		return nil
	}
	if resp.StatusCode != 200 {
		logger.Printf("%s non-200 status code for '%s': %v", errPrefix, path, resp.StatusCode)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		logger.Printf("%s reading body failed: %s", errPrefix, err)
		return nil
	}

	return &body
}

func updateLeaderboards() {
	//brackets := []string{"2v2", "3v3", "5v5", "rbg"}
	brackets := []string{"2v2"}	// TODO DELME
	leaderboards := make(map[string][]LeaderboardEntry)
	playerMap := make(map[string]Player)

	for _, bracket := range brackets {
		leaderboards[bracket] = getLeaderboard(bracket)
		lbPlayers := getPlayersFromLeaderboard(leaderboards[bracket])
		max, err := strconv.Atoi(os.Getenv("MAX_PER_BRACKET"))
		if err != nil || max < 0 || max > len(lbPlayers) {
			max = len(lbPlayers)
		}
		for _, player := range lbPlayers[0:max] {
			// name + realm as key to create unique set of players
			playerMap[player.Name + player.RealmSlug] = player
		}
	}

	logger.Printf("Found %v unique players across %v brackets", len(playerMap), len(leaderboards))

	players := getPlayerDetails(playerMap)
	upsertPlayers(&players)
	// TODO SET LEADERBOARDS
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

func getLeaderboard(bracket string) []LeaderboardEntry {
	var leaderboardJson *[]byte = get("leaderboard/" + bracket)
	var entries []LeaderboardEntry = parseLeaderboard(leaderboardJson)
	logger.Printf("Parsed %v %s entries", len(entries), bracket)
	
	return entries
}

func getPlayersFromLeaderboard(entries []LeaderboardEntry) []Player {
	players := make([]Player, 0)

	for _, entry := range entries {
		player := Player{
			Name: entry.Name,
			ClassId: entry.ClassId,
			FactionId: entry.FactionId,
			RaceId: entry.RaceId,
			RealmSlug: entry.RealmSlug,
			Gender: entry.GenderId}
		players = append(players, player)
	}

	return players
}

func parsePlayerDetails(data *[]byte) Player {
	type Guild struct {
		Name string
	}
	type Achievements struct {
		AchievementsCompleted []int
	}
	type Spell struct {
		Id int
	}
	type TalentJson struct {
		Spell Spell
	}
	type GlyphSpell struct {
		Glyph int
	}
	type GlyphJson struct {
		Major []GlyphSpell
		Minor []GlyphSpell
	}
	type Talents struct {
		Talents []TalentJson
		Glyphs GlyphJson
		Spec Spec
	}
	type PlayerJson struct {
		Name string
		Class int
		Race int
		Gender int
		AchievementPoints int
		TotalHonorableKills int
		Guild Guild
		Achievements Achievements
		Talents []Talents
	}

	var player PlayerJson
	err := json.Unmarshal(*data, &player)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return Player{}
	}

	return Player{
		Name: player.Name,
		ClassId: player.Class,
		SpecId: 1, //player.,	// TODO MAP SPEC IDS
		RaceId: player.Race,
		Guild: player.Guild.Name,
		Gender: player.Gender,
		//AchievementIds: player.Achievements.AchievementsCompleted,	// TODO PLAYER=>ACHIEV RELATION
		AchievementPoints: player.AchievementPoints,
		HonorableKills: player.TotalHonorableKills}
}

func getPlayerDetails(playerMap map[string]Player) []Player {
	players := make([]Player, 0)
	const path string = "character/%s/%s?fields=talents,guild,achievements"
	for _, player := range playerMap {
		var playerJson *[]byte = get(fmt.Sprintf(path, player.RealmSlug, player.Name))
		if playerJson != nil {
			var p Player = parsePlayerDetails(playerJson)
			p.RealmSlug = player.RealmSlug
			p.FactionId = player.FactionId
			players = append(players, p)
		}
	}

	return players
}
