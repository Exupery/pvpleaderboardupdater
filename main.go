package main

import (
	"encoding/json"
	"errors"
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
var regions = []string{"EU", "US"}

func main() {
	start := time.Now()
	logger.Println("Updating PvPLeaderBoard DB")
	importStaticData()
	groupSize := getEnvVarOrDefault("GROUP_SIZE", defaultGroupSize)
	season := getCurrentSeason()
	for _, r := range regions {
		region = r
		leaderboards := make(map[string][]leaderboardEntry)
		brackets := []string{"2v2", "3v3", "rbg"}
		for _, bracket := range brackets {
			leaderboard := getLeaderboard(bracket, season)
			logger.Printf("Found %d players on %s %s leaderboard", len(leaderboard), region, bracket)
			if len(leaderboard) == 0 {
				continue
			}
			leaderboards[bracket] = leaderboard
		}
		players := getPlayersFromLeaderboards(leaderboards)
		logger.Printf("Found %d unique players across %s leaderboards", len(players), region)
		if len(players) == 0 {
			continue
		}
		groups := split(players, groupSize)
		var waitGroup sync.WaitGroup
		waitGroup.Add(len(groups))
		for _, group := range groups {
			go importPlayers(group, &waitGroup)
		}
		waitGroup.Wait()
		for bracket, leaderboard := range leaderboards {
			updateLeaderboard(bracket, leaderboard)
		}
	}
	purgeStalePlayers()
	setUpdateTime()
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

func getEnvVarOrDefault(envVar string, defaultValue int) int {
	var size = os.Getenv(envVar)
	if size == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(size)
	if err != nil {
		logger.Printf("%s Cannot convert '%s' to int, using %s default (%d).",
			warnPrefix, size, envVar, defaultValue)
		return defaultValue
	}
	return i
}

func split(slice []*player, groupSize int) [][]*player {
	groups := make([][]*player, 0)
	if len(slice) <= groupSize {
		return append(groups, slice)
	}

	group := make([]*player, 0)
	for i, p := range slice {
		group = append(group, p)
		if (i+1)%groupSize == 0 {
			groups = append(groups, group)
			group = make([]*player, 0)
		}
	}

	return groups
}

func getCurrentSeason() int {
	type Seasons struct {
		Seasons       []keyedValue
		CurrentSeason keyedValue `json:"current_season"`
	}
	var seasonsJSON *[]byte = getDynamic(region, "pvp-season/index")
	if seasonsJSON == nil {
		return 0
	}
	var seasons Seasons
	err := safeUnmarshal(seasonsJSON, &seasons)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return 0
	}
	return seasons.CurrentSeason.ID
}

func getLeaderboard(bracket string, season int) []leaderboardEntry {
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
	var leaderboardEntries []leaderboardEntry = make([]leaderboardEntry, 0)
	var leaderboardJSON *[]byte = getDynamic(region, fmt.Sprintf("pvp-season/%d/pvp-leaderboard/%s",
		season, bracket))
	if leaderboardJSON == nil {
		return leaderboardEntries
	}
	var leaderboard LeaderBoardJSON
	err := safeUnmarshal(leaderboardJSON, &leaderboard)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return leaderboardEntries
	}
	for _, entry := range leaderboard.Entries {
		leaderboardEntry := leaderboardEntry{
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

func getPlayersFromLeaderboards(leaderboards map[string][]leaderboardEntry) []*player {
	players := make(map[string]*player, 0)
	for _, entries := range leaderboards {
		for _, entry := range entries {
			key := playerKey(entry.RealmID, entry.BlizzardID)
			_, exists := players[key]
			if exists {
				continue
			}
			path := fmt.Sprintf("%s/%s",
				getRealmSlug(entry.RealmID), url.QueryEscape(strings.ToLower(entry.Name)))
			player := player{
				Name:       entry.Name,
				BlizzardID: entry.BlizzardID,
				RealmID:    entry.RealmID,
				Path:       path}
			players[key] = &player
		}
	}
	var p []*player = make([]*player, 0)
	for _, player := range players {
		p = append(p, player)
	}
	return p
}

func playerKey(realmID, blizzardID int) string {
	return fmt.Sprintf("%d-%d", realmID, blizzardID)
}

func importPlayers(players []*player, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for _, player := range players {
		setPlayerDetails(player)
	}
	foundPlayers := make([]*player, 0)
	for _, player := range players {
		if player.ClassID == 0 {
			logger.Printf("No details found for %s, skipping", player.Path)
			continue
		}
		foundPlayers = append(foundPlayers, player)
	}

	addPlayers(foundPlayers)
	var playerIDs map[string]int = getPlayerIDs(foundPlayers)
	var pvpAchievements map[int]bool = getAchievementIds()

	var playersTalents map[int]playerTalents = make(map[int]playerTalents, 0)
	var playersStats map[int]stats = make(map[int]stats, 0)
	var playersItems map[int]items = make(map[int]items, 0)
	var playersAchievements map[int][]int = make(map[int][]int, 0)
	var playersSoulbinds map[int]playerSoulbind = make(map[int]playerSoulbind, 0)
	for profilePath, dbID := range playerIDs {
		playersTalents[dbID] = getPlayerTalents(profilePath)
		playersStats[dbID] = getPlayerStats(profilePath)
		playersItems[dbID] = getPlayerItems(profilePath)
		playersAchievements[dbID] = getPlayerAchievements(profilePath, pvpAchievements)
		playersSoulbinds[dbID] = getPlayerSoulbind(profilePath)
	}
	addItems(squashItems(playersItems))
	addPlayerTalents(playersTalents)
	addPlayerStats(playersStats)
	addPlayerItems(playersItems)
	addPlayerLegendaries(playersItems)
	addPlayerAchievements(playersAchievements)
	addPlayerSoulbinds(playersSoulbinds)
}

func setPlayerDetails(player *player) {
	type ProfileJSON struct {
		Gender         typedName
		Faction        typedName
		Race           keyedValue
		CharacterClass keyedValue `json:"character_class"`
		ActiveSpec     keyedValue `json:"active_spec"`
		Guild          keyedValue
	}
	var profileJSON *[]byte = getProfile(region, player.Path)
	if profileJSON == nil {
		return
	}
	var profile ProfileJSON
	err := safeUnmarshal(profileJSON, &profile)
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

func getPlayerTalents(path string) playerTalents {
	type Talent struct {
		Talent keyedValue
	}
	type PvPTalent struct {
		Selected Talent
	}
	type Specialization struct {
		Specialization keyedValue
		Talents        []Talent
		PvPTalentSlots []PvPTalent `json:"pvp_talent_slots"`
	}
	type Specializations struct {
		Specializations      []Specialization
		ActiveSpecialization keyedValue `json:"active_specialization"`
	}
	talentPath := path + "/specializations"
	var talentJSON *[]byte = getProfile(region, talentPath)
	if talentJSON == nil {
		return playerTalents{}
	}
	var specializations Specializations
	err := safeUnmarshal(talentJSON, &specializations)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return playerTalents{}
	}

	activeSpecID := specializations.ActiveSpecialization.ID
	talents := make([]int, 0)
	pvpTalents := make([]int, 0)
	for _, spec := range specializations.Specializations {
		if spec.Specialization.ID != activeSpecID {
			continue
		}
		for _, talent := range spec.Talents {
			id := talent.Talent.ID
			if id > 0 {
				talents = append(talents, id)
			}
		}
		for _, pvpTalent := range spec.PvPTalentSlots {
			id := pvpTalent.Selected.Talent.ID
			if id > 0 {
				pvpTalents = append(pvpTalents, id)
			}
		}
		break
	}

	return playerTalents{talents, pvpTalents}
}

func getPlayerStats(path string) stats {
	type RatedStat struct {
		Rating      float64
		RatingBonus float64 `json:"rating_bonus"`
		Value       float64
	}
	type Stat struct {
		Base      int
		Effective int
	}
	type StatJSON struct {
		Strength    Stat
		Agility     Stat
		Intellect   Stat
		Stamina     Stat
		Versatility float64
		Mastery     RatedStat
		Lifesteal   RatedStat
		Dodge       RatedStat
		Parry       RatedStat
		MeleeCrit   RatedStat `json:"melee_crit"`
		MeleeHaste  RatedStat `json:"melee_haste"`
		RangedCrit  RatedStat `json:"ranged_crit"`
		RangedHaste RatedStat `json:"ranged_haste"`
		SpellCrit   RatedStat `json:"spell_crit"`
		SpellHaste  RatedStat `json:"spell_haste"`
	}
	var statsJSON *[]byte = getProfile(region, path+"/statistics")
	if statsJSON == nil {
		return stats{}
	}
	var s StatJSON
	err := safeUnmarshal(statsJSON, &s)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return stats{}
	}
	crit := highestStat(int(s.MeleeCrit.Rating), int(s.RangedCrit.Rating), int(s.SpellCrit.Rating))
	haste := highestStat(int(s.MeleeHaste.Rating), int(s.RangedHaste.Rating), int(s.SpellHaste.Rating))
	return stats{
		Strength:       s.Strength.Effective,
		Agility:        s.Agility.Effective,
		Intellect:      s.Intellect.Effective,
		Stamina:        s.Stamina.Effective,
		CriticalStrike: crit,
		Haste:          haste,
		Versatility:    int(s.Versatility),
		Mastery:        int(s.Mastery.Rating),
		Leech:          int(s.Lifesteal.Rating),
		Dodge:          int(s.Dodge.Rating),
		Parry:          int(s.Parry.Rating)}
}

func getPlayerItems(path string) items {
	type SpellJSON struct {
		Spell       keyedValue
		Description string
	}
	type ItemJSON struct {
		Item    keyedValue
		Slot    typedName
		Name    string
		Quality typedName
		Spells  []SpellJSON
	}
	type ItemsJSON struct {
		EquippedItems []ItemJSON `json:"equipped_items"`
	}
	var itemsJSON *[]byte = getProfile(region, path+"/equipment")
	if itemsJSON == nil {
		return items{}
	}
	var equipped ItemsJSON
	err := safeUnmarshal(itemsJSON, &equipped)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return items{}
	}
	equippedItems := make(map[string]item)
	for _, i := range equipped.EquippedItems {
		if i.Name == "" {
			continue
		}
		if i.Quality.Type == "LEGENDARY" && len(i.Spells) > 0 {
			spell := i.Spells[0].Spell
			spellID := spell.ID
			name := spell.Name
			equippedItems["LEGENDARY_SPELL"] = item{spellID, name, i.Quality.Type}
		}
		equippedItems[i.Slot.Type] = item{i.Item.ID, i.Name, i.Quality.Type}
	}
	return items{
		Head:      equippedItems["HEAD"],
		Neck:      equippedItems["NECK"],
		Shoulder:  equippedItems["SHOULDER"],
		Back:      equippedItems["BACK"],
		Chest:     equippedItems["CHEST"],
		Shirt:     equippedItems["SHIRT"],
		Tabard:    equippedItems["TABARD"],
		Wrist:     equippedItems["WRIST"],
		Hands:     equippedItems["HANDS"],
		Waist:     equippedItems["WAIST"],
		Legs:      equippedItems["LEGS"],
		Feet:      equippedItems["FEET"],
		Finger1:   equippedItems["FINGER_1"],
		Finger2:   equippedItems["FINGER_2"],
		Trinket1:  equippedItems["TRINKET_1"],
		Trinket2:  equippedItems["TRINKET_2"],
		MainHand:  equippedItems["MAIN_HAND"],
		OffHand:   equippedItems["OFF_HAND"],
		Legendary: equippedItems["LEGENDARY_SPELL"]}
}

func squashItems(playersItems map[int]items) map[int]item {
	items := make(map[int]item, 0)

	for _, pi := range playersItems {
		addItem(items, pi.Head)
		addItem(items, pi.Neck)
		addItem(items, pi.Shoulder)
		addItem(items, pi.Back)
		addItem(items, pi.Chest)
		addItem(items, pi.Shirt)
		addItem(items, pi.Tabard)
		addItem(items, pi.Wrist)
		addItem(items, pi.Hands)
		addItem(items, pi.Waist)
		addItem(items, pi.Legs)
		addItem(items, pi.Feet)
		addItem(items, pi.Finger1)
		addItem(items, pi.Finger2)
		addItem(items, pi.Trinket1)
		addItem(items, pi.Trinket2)
		addItem(items, pi.MainHand)
		addItem(items, pi.OffHand)
	}

	return items
}

func addItem(items map[int]item, itemToAdd item) {
	if itemToAdd.ID == 0 {
		return
	}

	if _, exists := items[itemToAdd.ID]; exists {
		// Checking if exists instead of adding anyway to avoid unnecessary API calls for legendaries
		return
	}

	if itemToAdd.Quality == "LEGENDARY" {
		var json *[]byte = getStatic(region, fmt.Sprintf("item/%d", itemToAdd.ID))
		itemToAdd.Name = parseItemName(json)
	}

	items[itemToAdd.ID] = itemToAdd
}

func parseItemName(data *[]byte) string {
	type Item struct {
		Name string
	}
	var item Item
	if data == nil {
		return ""
	}
	err := safeUnmarshal(data, &item)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return ""
	}
	return item.Name
}

func getPlayerAchievements(path string, pvpAchievements map[int]bool) []int {
	type AchievementJSON struct {
		ID                 int
		CompletedTimestamp int64 `json:"completed_timestamp"`
	}
	type AchievedJSON struct {
		Achievements []AchievementJSON
	}
	var achievedJSON *[]byte = getProfile(region, path+"/achievements")
	if achievedJSON == nil {
		return make([]int, 0)
	}
	var achieved AchievedJSON
	err := safeUnmarshal(achievedJSON, &achieved)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]int, 0)
	}
	achievedIDs := make([]int, 0)
	for _, achievement := range achieved.Achievements {
		id := achievement.ID
		if pvpAchievements[id] && achievement.CompletedTimestamp > 0 {
			achievedIDs = append(achievedIDs, id)
		}
	}
	return achievedIDs
}

func getPlayerSoulbind(path string) playerSoulbind {
	type SocketJSON struct {
		Conduit keyedValue
	}
	type ConduitSocketJSON struct {
		Socket SocketJSON
	}
	type TraitJSON struct {
		ConduitSocket ConduitSocketJSON `json:"conduit_socket"`
	}
	type SoulbindJSON struct {
		Soulbind keyedValue
		Traits   []TraitJSON
		IsActive bool `json:"is_active"`
	}
	type SoulbindsJSON struct {
		Covenant  keyedValue `json:"chosen_covenant"`
		Soulbinds []SoulbindJSON
	}
	var soulbindJSON *[]byte = getProfile(region, path+"/soulbinds")
	if soulbindJSON == nil {
		return playerSoulbind{}
	}
	var soulbinds SoulbindsJSON
	err := safeUnmarshal(soulbindJSON, &soulbinds)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return playerSoulbind{}
	}

	for _, soulbind := range soulbinds.Soulbinds {
		if !soulbind.IsActive {
			continue
		}
		var soulbindID int = soulbind.Soulbind.ID
		conduitIDs := make([]int, 0)
		for _, conduit := range soulbind.Traits {
			id := conduit.ConduitSocket.Socket.Conduit.ID
			if id == 0 {
				continue
			}
			conduitIDs = append(conduitIDs, id)
		}
		return playerSoulbind{soulbinds.Covenant.ID, soulbindID, conduitIDs}
	}
	return playerSoulbind{Covenant: soulbinds.Covenant.ID}
}

func highestStat(a, b, c int) int {
	if a >= b && a >= c {
		return a
	}
	if b >= a && b >= c {
		return b
	}
	return c
}

func safeUnmarshal(data *[]byte, v interface{}) error {
	if data == nil {
		return errors.New("Data is nil, nothing to unmarshal")
	}

	err := json.Unmarshal(*data, &v)
	if err != nil {
		logger.Printf("%s JSON parsing failed: %s", errPrefix, err)
		return err
	}

	return nil
}

type playerTalents struct {
	Talents    []int
	PvPTalents []int
}
