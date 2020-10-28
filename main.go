package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var logger *log.Logger = log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds)

const errPrefix string = "[ERROR]"
const fatalPrefix string = "[FATAL]"
const warnPrefix string = "[WARN]"

const defaultGroupSize int = 100

var region = "US"

func main() {
	start := time.Now()
	logger.Println("Updating PvPLeaderBoard DB")
	importStaticData()
	groupSize := groupSize()
	season := getCurrentSeason()
	// TODO HANDLE REGIONS
	leaderboards := make(map[string][]LeaderboardEntry)
	// brackets := []string{"2v2", "3v3", "rbg"}
	// for _, bracket := range brackets {
	bracket := "2v2" // TODO DELME
	leaderboard := getLeaderboard(bracket, season)
	logger.Printf("Found %d players on %s %s leaderboard", len(leaderboard), region, bracket)
	leaderboards[bracket] = leaderboard
	players := getPlayersFromLeaderboards(leaderboards)
	logger.Printf("Found %d unique players across %s leaderboards", len(players), region)
	groups := split(players, groupSize)
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(groups))
	for _, group := range groups {
		go importPlayers(group, &waitGroup)
	}
	waitGroup.Wait()
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

func groupSize() int {
	var size = os.Getenv("GROUP_SIZE")
	if size == "" {
		return defaultGroupSize
	}
	i, err := strconv.Atoi(size)
	if err != nil {
		logger.Printf("%s Cannot convert '%s' to int, using default group size (%d).",
			warnPrefix, size, defaultGroupSize)
		return defaultGroupSize
	}
	return i
}

func split(slice []*Player, groupSize int) [][]*Player {
	groups := make([][]*Player, 0)
	if len(slice) <= groupSize {
		return append(groups, slice)
	}

	group := make([]*Player, 0)
	for i, p := range slice {
		group = append(group, p)
		if (i+1)%groupSize == 0 {
			groups = append(groups, group)
			group = make([]*Player, 0)
		}
	}

	return groups
}

func getCurrentSeason() int {
	type Seasons struct {
		Seasons       []KeyedID
		CurrentSeason KeyedID `json:"current_season"`
	}
	var seasonsJSON *[]byte = getDynamic(region, "pvp-season/index")
	var seasons Seasons
	err := json.Unmarshal(*seasonsJSON, &seasons)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return 0
	}
	return seasons.CurrentSeason.ID
}

func getLeaderboard(bracket string, season int) []LeaderboardEntry {
	type RealmJSON struct {
		Slug string
		ID   int
	}
	type CharacterJSON struct {
		Name  string
		ID    int
		Realm RealmJSON
	}
	type WinLossJSON struct {
		Played int
		Won    int
		Lost   int
	}
	type LeaderboardEntryJSON struct {
		Rank       int
		Rating     int
		Character  CharacterJSON
		WinsLosses WinLossJSON `json:"season_match_statistics"`
	}
	type LeaderBoardJSON struct {
		Entries []LeaderboardEntryJSON
	}
	var leaderboardJSON *[]byte = getDynamic(region, fmt.Sprintf("pvp-season/%d/pvp-leaderboard/%s",
		season, bracket))
	var leaderboard LeaderBoardJSON
	err := json.Unmarshal(*leaderboardJSON, &leaderboard)
	var leaderboardEntries []LeaderboardEntry = make([]LeaderboardEntry, 0)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return leaderboardEntries
	}
	for _, entry := range leaderboard.Entries {
		leaderboardEntry := LeaderboardEntry{
			entry.Character.Name,
			entry.Character.Realm.ID,
			entry.Character.ID,
			entry.Rank,
			entry.Rating,
			entry.WinsLosses.Won,
			entry.WinsLosses.Lost}
		leaderboardEntries = append(leaderboardEntries, leaderboardEntry)
	}
	max, err := strconv.Atoi(os.Getenv("MAX_PER_BRACKET"))
	if err == nil && max < len(leaderboardEntries) {
		return leaderboardEntries[0:max]
	}
	return leaderboardEntries
}

func getPlayersFromLeaderboards(leaderboards map[string][]LeaderboardEntry) []*Player {
	players := make(map[string]*Player, 0)
	for _, entries := range leaderboards {
		for _, entry := range entries {
			key := playerKey(entry.RealmID, entry.BlizzardID)
			_, exists := players[key]
			if exists {
				continue
			}
			path := fmt.Sprintf("%s/%s",
				getRealmSlug(entry.RealmID), url.QueryEscape(strings.ToLower(entry.Name)))
			player := Player{
				Name:       entry.Name,
				BlizzardID: entry.BlizzardID,
				RealmID:    entry.RealmID,
				Path:       path}
			players[key] = &player
		}
	}
	var p []*Player = make([]*Player, 0)
	for _, player := range players {
		p = append(p, player)
	}
	return p
}

func playerKey(realmID, blizzardID int) string {
	return fmt.Sprintf("%d-%d", realmID, blizzardID)
}

func importPlayers(players []*Player, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for _, player := range players {
		setPlayerDetails(player)
	}
	foundPlayers := make([]*Player, 0)
	for _, player := range players {
		if player.ClassID == 0 {
			logger.Printf("No details found for %s, skipping", player.Path)
			continue
		}
		foundPlayers = append(foundPlayers, player)
	}
	addPlayers(foundPlayers)
	// TODO IMPORT/INSERT TALENTS
	// TODO IMPORT/INSERT STATS
	// TODO IMPORT/INSERT ITEMS
	// TODO IMPORT/INSERT ACHIEVS
}

func setPlayerDetails(player *Player) {
	type ProfileJSON struct {
		Gender         TypedName
		Faction        TypedName
		Race           KeyedValue
		CharacterClass KeyedValue `json:"character_class"`
		ActiveSpec     KeyedValue `json:"active_spec"`
		Guild          KeyedValue
	}
	var profileJSON *[]byte = getProfile(region, player.Path)
	if profileJSON == nil {
		return
	}
	var profile ProfileJSON
	err := json.Unmarshal(*profileJSON, &profile)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return
	}

	if profile.Gender.Type == "FEMALE" {
		player.Gender = 1
	} else {
		player.Gender = 0
	}

	if profile.Faction.Type == "HORDE" {
		player.FactionID = 67
	} else {
		player.FactionID = 469
	}

	player.RaceID = profile.Race.ID
	player.ClassID = profile.CharacterClass.ID
	player.SpecID = profile.ActiveSpec.ID
	player.Guild = profile.Guild.Name
}
