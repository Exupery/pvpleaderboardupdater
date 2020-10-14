package pvpleaderboardupdater

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var logger *log.Logger = log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds)

const errPrefix string = "[ERROR]"
const fatalPrefix string = "[FATAL]"
const warnPrefix string = "[WARN]"

var uriBase string
var apiKey string

func main() {
	logger.Println("Updating PvPLeaderBoard")
	flag.StringVar(&uriBase, "base", "https://us.api.battle.net/wow/", "WoW API base URI")
	var importStatic *bool = flag.Bool("static", false, "Import static data (e.g. races, classes, realms, etc)")
	flag.Parse()
	logger.Printf("WoW API URIs using '%s'", uriBase)

	if *importStatic {
		importStaticData()
	} else {
		brackets := [4]string{"2v2", "3v3", "rbg"}
		for _, bracket := range brackets {
			updatePlayersAndLeaderboard(bracket)
		}

		setUpdateTime()
	}

	logger.Println("Updating PvPLeaderBoard Complete")
}

func getEnvVar(envVar string) string {
	var value string = os.Getenv(envVar)
	if value == "" {
		logger.Fatalf("%s Environment variable '%s' NOT set! Aborting.", fatalPrefix, envVar)
	}

	return value
}

func get(path string) *[]byte {
	if apiKey == "" {
		apiKey = getEnvVar("BATTLE_NET_API_KEY")
	}
	var params string = "locale=en_US&apikey=" + apiKey
	var sep string
	if strings.Count(path, "?") == 0 {
		sep = "?"
	} else {
		sep = "&"
	}

	resp, err := http.Get(uriBase + path + sep + params)

	if err != nil {
		logger.Printf("%s GET '%s' failed: %s", errPrefix, path, err)
		return nil
	}
	if resp.StatusCode != 200 {
		logger.Printf("%s non-200 status code for '%s': %v", warnPrefix, path, resp.StatusCode)
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
	var playerIdMap *map[int]*Player = getPlayerIdMap(players)
	if len(*playerIdMap) > 0 {
		updated := updatePlayers(playerIdMap)
		if updated {
			updateLeaderboard(playerIdMap, leaderboard, bracket)
		} else {
			logger.Printf("%s Updating player details failed, NOT updating %s leaderboard", errPrefix, bracket)
		}
	} else {
		logger.Printf("%s %s player ID map empty (%d expected)", errPrefix, bracket, len(players))
	}
}

func updateLeaderboard(playerIdMap *map[int]*Player, leaderboard *map[string]*LeaderboardEntry, bracket string) {
	var playerSlugIdMap map[string]int = make(map[string]int)
	for id, player := range *playerIdMap {
		playerSlugIdMap[player.Name+player.RealmSlug] = id
	}

	setLeaderboard(bracket, leaderboard, &playerSlugIdMap)
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
	var leaderboardJson *[]byte = get("leaderboard/" + bracket)
	var entries *map[string]*LeaderboardEntry = parseLeaderboard(leaderboardJson)
	logger.Printf("Parsed %v %s entries", len(*entries), bracket)

	return entries
}

func getPlayersFromLeaderboard(entries *map[string]*LeaderboardEntry) []*Player {
	players := make([]*Player, 0)

	for _, entry := range *entries {
		player := Player{
			Name:      entry.Name,
			ClassId:   entry.ClassId,
			FactionId: entry.FactionId,
			RaceId:    entry.RaceId,
			RealmSlug: entry.RealmSlug,
			Gender:    entry.GenderId}
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
		Id int
	}
	type TalentJson struct {
		Spell Spell
	}
	type TalentsJson struct {
		Talents  []TalentJson
		Spec     Spec
		Selected bool
	}
	type PlayerJson struct {
		Name                string
		Class               int
		Race                int
		Gender              int
		AchievementPoints   int
		TotalHonorableKills int
		Guild               Guild
		Achievements        Achievements
		Talents             []TalentsJson
		Stats               Stats
		Items               Items
	}

	var player PlayerJson
	err := json.Unmarshal(*data, &player)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return nil
	}

	var specId int
	var talentIds []int = make([]int, 0)

	for _, t := range player.Talents {
		if t.Selected {
			specId = (*classSpecMap)[strconv.Itoa(player.Class)+t.Spec.Name]
			for _, talent := range t.Talents {
				talentIds = append(talentIds, talent.Spell.Id)
			}
		}
	}

	p := Player{
		Name:                  player.Name,
		ClassId:               player.Class,
		SpecId:                specId,
		RaceId:                player.Race,
		Guild:                 player.Guild.Name,
		Gender:                player.Gender,
		Stats:                 player.Stats,
		TalentIds:             talentIds,
		Items:                 player.Items,
		AchievementIds:        player.Achievements.AchievementsCompleted,
		AchievementTimestamps: player.Achievements.AchievementsCompletedTimestamp,
		AchievementPoints:     player.AchievementPoints,
		HonorableKills:        player.TotalHonorableKills}

	return &p
}

func getPlayerDetails(playerMap *map[string]*Player) []*Player {
	players := make([]*Player, 0)
	classSpecMap := classIdSpecNameToSpecIdMap()
	const path string = "character/%s/%s?fields=talents,guild,achievements,stats,items"
	for _, player := range *playerMap {
		// realm may be empty if character is transferring
		if player.RealmSlug != "" {
			var playerJson *[]byte = get(fmt.Sprintf(path, player.RealmSlug, player.Name))
			if playerJson != nil {
				var p *Player = parsePlayerDetails(playerJson, classSpecMap)
				if p != nil {
					p.RealmSlug = player.RealmSlug
					p.FactionId = player.FactionId
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
	if player.ClassId == 0 {
		logger.Printf(msg, warnPrefix, player.Name, player.RealmSlug, "ClassId")
		return false
	}
	if player.SpecId == 0 {
		logger.Printf(msg, warnPrefix, player.Name, player.RealmSlug, "SpecId")
		return false
	}
	if player.RaceId == 0 {
		logger.Printf(msg, warnPrefix, player.Name, player.RealmSlug, "RaceId")
		return false
	}
	return true
}
