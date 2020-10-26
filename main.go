package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

var logger *log.Logger = log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds)

const errPrefix string = "[ERROR]"
const fatalPrefix string = "[FATAL]"
const warnPrefix string = "[WARN]"

var uriBase string
var apiKey string

var region = "US"

func main() {
	start := time.Now()
	logger.Println("Updating PvPLeaderBoard DB")
	importStaticData()
	// TODO HANDLE REGIONS
	// brackets := []string{"2v2", "3v3", "rbg"}
	// for _, bracket := range brackets {
	// 	updatePlayersAndLeaderboard(bracket)
	// }
	// TODO PURGE STALE DATA
	// setUpdateTime()
	end := time.Now()
	logger.Printf("Updating PvPLeaderBoard Complete after %v", end.Sub(start))
}

func getEnvVar(envVar string) string {
	var value string = os.Getenv(envVar)
	if value == "" {
		logger.Fatalf("%s Environment variable '%s' NOT set! Aborting.", fatalPrefix, envVar)
	}

	return value
}

func updatePlayersAndLeaderboard(bracket string) {
	playerMap := make(map[string]*Player)

	var leaderboard *map[string]*LeaderboardEntry = getLeaderboard(bracket)
	lbPlayers := getPlayersFromLeaderboard(leaderboard)
	max, err := strconv.Atoi(os.Getenv("MAX_PER_BRACKET"))
	if err != nil || max < 0 || max > len(lbPlayers) {
		max = len(lbPlayers)
	}
	for _, player := range lbPlayers[0:max] {
		playerMap[player.Name+player.RealmSlug] = player
	}

	logger.Printf("Found %v players in the %s bracket", len(playerMap), bracket)

	players := getPlayerDetails(&playerMap)
	addPlayers(players)
	var playerIDMap *map[int]*Player = getPlayerIDMap(players)
	if len(*playerIDMap) > 0 {
		updated := updatePlayers(playerIDMap)
		if updated {
			updateLeaderboard(playerIDMap, leaderboard, bracket)
		} else {
			logger.Printf("%s Updating player details failed, NOT updating %s leaderboard", errPrefix, bracket)
		}
	} else {
		logger.Printf("%s %s player ID map empty (%d expected)", errPrefix, bracket, len(players))
	}
}

func updateLeaderboard(playerIDMap *map[int]*Player, leaderboard *map[string]*LeaderboardEntry, bracket string) {
	var playerSlugIDMap map[string]int = make(map[string]int)
	for id, player := range *playerIDMap {
		playerSlugIDMap[player.Name+player.RealmSlug] = id
	}

	setLeaderboard(bracket, leaderboard, &playerSlugIDMap)
}

func parseLeaderboard(data *[]byte) *map[string]*LeaderboardEntry {
	entries := make(map[string]*LeaderboardEntry)
	type Leaderboard struct {
		Rows []LeaderboardEntry
	}
	var leaderboard Leaderboard
	err := json.Unmarshal(*data, &leaderboard)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return &entries
	}

	for _, entry := range leaderboard.Rows {
		e := LeaderboardEntry(entry)
		entries[entry.Name+entry.RealmSlug] = &e
	}
	return &entries
}

func getLeaderboard(bracket string) *map[string]*LeaderboardEntry {
	var leaderboardJSON *[]byte = getDynamic(region, "leaderboard/"+bracket)
	var entries *map[string]*LeaderboardEntry = parseLeaderboard(leaderboardJSON)
	logger.Printf("Parsed %v %s entries", len(*entries), bracket)

	return entries
}

func getPlayersFromLeaderboard(entries *map[string]*LeaderboardEntry) []*Player {
	players := make([]*Player, 0)

	for _, entry := range *entries {
		player := Player{
			Name:      entry.Name,
			ClassID:   entry.ClassID,
			FactionID: entry.FactionID,
			RaceID:    entry.RaceID,
			RealmSlug: entry.RealmSlug,
			Gender:    entry.GenderID}
		players = append(players, &player)
	}

	return players
}

func parsePlayerDetails(data *[]byte, classSpecMap *map[string]int) *Player {
	type Guild struct {
		Name string
	}
	type Achievements struct {
		AchievementsCompleted          []int
		AchievementsCompletedTimestamp []int64
	}
	type Spell struct {
		ID int
	}
	type TalentJSON struct {
		Spell Spell
	}
	type TalentsJSON struct {
		Talents  []TalentJSON
		Spec     Spec
		Selected bool
	}
	type PlayerJSON struct {
		Name                string
		Class               int
		Race                int
		Gender              int
		AchievementPoints   int
		TotalHonorableKills int
		Guild               Guild
		Achievements        Achievements
		Talents             []TalentsJSON
		Stats               Stats
		Items               Items
	}

	var player PlayerJSON
	err := json.Unmarshal(*data, &player)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return nil
	}

	var specID int
	var talentIds []int = make([]int, 0)

	for _, t := range player.Talents {
		if t.Selected {
			specID = (*classSpecMap)[strconv.Itoa(player.Class)+t.Spec.Name]
			for _, talent := range t.Talents {
				talentIds = append(talentIds, talent.Spell.ID)
			}
		}
	}

	p := Player{
		Name:                  player.Name,
		ClassID:               player.Class,
		SpecID:                specID,
		RaceID:                player.Race,
		Guild:                 player.Guild.Name,
		Gender:                player.Gender,
		Stats:                 player.Stats,
		TalentIDs:             talentIds,
		Items:                 player.Items,
		AchievementIDs:        player.Achievements.AchievementsCompleted,
		AchievementTimestamps: player.Achievements.AchievementsCompletedTimestamp,
		AchievementPoints:     player.AchievementPoints,
		HonorableKills:        player.TotalHonorableKills}

	return &p
}

func getPlayerDetails(playerMap *map[string]*Player) []*Player {
	players := make([]*Player, 0)
	classSpecMap := classIDSpecNameToSpecIDMap()
	const path string = "character/%s/%s?fields=talents,guild,achievements,stats,items"
	for _, player := range *playerMap {
		// realm may be empty if character is transferring
		if player.RealmSlug != "" {
			var playerJSON *[]byte = getDynamic(region, fmt.Sprintf(path, player.RealmSlug, player.Name))
			if playerJSON != nil {
				var p *Player = parsePlayerDetails(playerJSON, classSpecMap)
				if p != nil {
					p.RealmSlug = player.RealmSlug
					p.FactionID = player.FactionID
					if playerIsValid(p) {
						players = append(players, p)
					}
				}
			}
		}
	}

	return players
}

func playerIsValid(player *Player) bool {
	if player.Name == "" || player.RealmSlug == "" {
		return false
	}
	const msg string = "%s %s-%s has no %s"
	if player.ClassID == 0 {
		logger.Printf(msg, warnPrefix, player.Name, player.RealmSlug, "ClassId")
		return false
	}
	if player.SpecID == 0 {
		logger.Printf(msg, warnPrefix, player.Name, player.RealmSlug, "SpecId")
		return false
	}
	if player.RaceID == 0 {
		logger.Printf(msg, warnPrefix, player.Name, player.RealmSlug, "RaceId")
		return false
	}
	return true
}
