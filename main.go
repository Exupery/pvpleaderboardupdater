package main

import (
	"crypto/sha256"
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

	cmap "github.com/orcaman/concurrent-map/v2"
)

var logger *log.Logger = log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds)

const errPrefix string = "[ERROR]"
const fatalPrefix string = "[FATAL]"
const warnPrefix string = "[WARN]"

const defaultGroupSize int = 100

var loginStaleSeconds int64 = int64(getEnvVarOrDefault("LAST_LOGIN_STALE_HOURS", 999) * 60 * 60)

var region = "US"
var regions = []string{"EU", "US"}

func main() {
	start := time.Now()
	logger.Println("Updating PvPLeaderBoard DB")
	importStaticData()
	groupSize := getEnvVarOrDefault("GROUP_SIZE", defaultGroupSize)
	season := getCurrentSeason()
	foundPlayers := false
	markStalePlayerTalents()
	for _, r := range regions {
		region = r
		leaderboards := make(map[string][]leaderboardEntry)

		// REGULAR BRACKETS
		brackets := []string{"2v2", "3v3", "rbg"}
		for _, bracket := range brackets {
			leaderboard := getLeaderboard(bracket, season)
			logger.Printf("Found %d players on %s %s leaderboard", len(leaderboard), region, bracket)
			if len(leaderboard) == 0 {
				continue
			}
			leaderboards[bracket] = leaderboard
		}
		// SOLO SHUFFLE
		soloLeaderboards := getPrefixedLeaderboards(season, "shuffle")
		for specID, name := range soloLeaderboards {
			leaderboard := getLeaderboard(name, season)
			logger.Printf("Found %d players on %s %s leaderboard", len(leaderboard), region, name)
			if len(leaderboard) == 0 {
				continue
			}
			bracket := fmt.Sprintf("solo_%d", specID)
			leaderboards[bracket] = leaderboard
		}
		// BATTLEGROUND BLITZ
		blitzLeaderboards := getPrefixedLeaderboards(season, "blitz")
		for specID, name := range blitzLeaderboards {
			leaderboard := getLeaderboard(name, season)
			logger.Printf("Found %d players on %s %s leaderboard", len(leaderboard), region, name)
			if len(leaderboard) == 0 {
				continue
			}
			bracket := fmt.Sprintf("blitz_%d", specID)
			leaderboards[bracket] = leaderboard
		}

		players := getPlayersFromLeaderboards(leaderboards)
		logger.Printf("Found %d unique players across %s leaderboards", len(players), region)
		if len(players) == 0 {
			continue
		} else {
			foundPlayers = true
		}
		groups := split(players, groupSize)
		var waitGroup sync.WaitGroup
		waitGroup.Add(len(groups))

		// Player items/gear will have A LOT of overlap so use a
		// singular global collection for that so all the upserts
		// are minimized and happen only once per update.
		playersItems := cmap.New[items]()

		for _, group := range groups {
			go importPlayers(group, &waitGroup, &playersItems)
		}
		waitGroup.Wait()

		addItems(squashItems(&playersItems))
		addPlayerItems(&playersItems)

		for bracket, leaderboard := range leaderboards {
			updateLeaderboard(bracket, leaderboard)
		}
	}
	if foundPlayers {
		logger.Println("Cleaning up...")
		purgeStalePlayers()
		setUpdateTime()
	}
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
		logger.Printf("%s parsing season failed: %s", warnPrefix, err)
		return 0
	}
	season := seasons.CurrentSeason.ID
	logger.Printf("Current season: %d", season)
	return season
}

func getPrefixedLeaderboards(season int, prefix string) map[int]string {
	type Leaderboards struct {
		Leaderboards []keyedValue
	}
	leaderboardsWithPrefix := make(map[int]string, 0)

	path := fmt.Sprintf("pvp-season/%d/pvp-leaderboard/index", season)
	var leaderboardsJSON *[]byte = getDynamic(region, path)
	if leaderboardsJSON == nil {
		return leaderboardsWithPrefix
	}

	var leaderboards Leaderboards
	err := safeUnmarshal(leaderboardsJSON, &leaderboards)
	if err != nil {
		logger.Printf("%s parsing leaderboards failed: %s", warnPrefix, err)
		return leaderboardsWithPrefix
	}

	for _, leaderboard := range leaderboards.Leaderboards {
		name := leaderboard.Name
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		specID := getSpecIDFromLeaderboardName(name)
		if specID == 0 {
			continue
		}
		leaderboardsWithPrefix[specID] = name
	}

	return leaderboardsWithPrefix
}

func getSpecIDFromLeaderboardName(name string) int {
	if !strings.HasPrefix(name, "shuffle") && !strings.HasPrefix(name, "blitz") {
		return 0
	}

	parts := strings.Split(name, "-")
	if len(parts) != 3 {
		logger.Printf("%s Unexpected leaderboard name: %s", warnPrefix, name)
		return 0
	}

	return getSpecIDForClassSpec(parts[1], parts[2])
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
		logger.Printf("%s parsing leaderboard failed: %s", warnPrefix, err)
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

func importPlayers(players []*player, waitGroup *sync.WaitGroup, playersItems *cmap.ConcurrentMap[string, items]) {
	defer waitGroup.Done()
	for _, player := range players {
		setPlayerDetails(player)
	}
	foundPlayers := make([]*player, 0)
	nowish := time.Now().Unix()
	for _, player := range players {
		if player.ClassID == 0 {
			logger.Printf("No details found for %s, skipping", player.Path)
			continue
		}
		if (nowish - player.LastLogin) > loginStaleSeconds {
			continue
		}
		foundPlayers = append(foundPlayers, player)
	}

	addPlayers(foundPlayers)
	var playerIDs map[string]int = getPlayerIDs(foundPlayers)
	var pvpAchievements map[int]bool = getAchievementIds()

	var playersTalents map[int]playerTalents = make(map[int]playerTalents, 0)
	var playersStats map[int]stats = make(map[int]stats, 0)
	var playersAchievements map[int][]int = make(map[int][]int, 0)
	for profilePath, dbID := range playerIDs {
		playerTalents := getPlayerTalents(profilePath)
		// If we couldn't get the player's talents don't bother attempting other data
		if len(playerTalents.Talents) == 0 {
			continue
		}
		playersTalents[dbID] = playerTalents
		playersStats[dbID] = getPlayerStats(profilePath)
		playersAchievements[dbID] = getPlayerAchievements(profilePath, pvpAchievements)

		(*playersItems).SetIfAbsent(strconv.Itoa(dbID), getPlayerItems(profilePath))
	}
	addPlayerTalents(playersTalents)
	addPlayerStats(playersStats)
	addPlayerAchievements(playersAchievements)
}

func setPlayerDetails(player *player) {
	type ProfileJSON struct {
		Gender         typedName
		Faction        typedName
		Race           keyedValue
		CharacterClass keyedValue `json:"character_class"`
		ActiveSpec     keyedValue `json:"active_spec"`
		Guild          keyedValue
		LastLogin      int64 `json:"last_login_timestamp"`
	}
	var profileJSON *[]byte = getProfile(region, player.Path)
	if profileJSON == nil {
		return
	}
	var profile ProfileJSON
	err := safeUnmarshal(profileJSON, &profile)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", warnPrefix, err)
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
	player.LastLogin = profile.LastLogin / 1000
	profileID := getProfileIdentifier(player.Path)
	if profileID != "" {
		player.ProfileID = profileID
	} else {
		// Fallback to avoid false-positives (will only guarantee uniquness
		// within a region but that's fine as leaderboards are per region)
		player.ProfileID = player.Path
	}
}

func getProfileIdentifier(path string) string {
	// All pets are present account-wide (even on characters that cannot use certain pets)
	// so the hash of the pets JSON can serve as a profile thumbprint
	petPath := path + "/collections/pets"
	var petJSON *[]byte = getProfile(region, petPath)
	if petJSON == nil {
		return ""
	}
	jsonStr := string(*petJSON)
	start := strings.Index(jsonStr, "\"pets\":")
	if start < 0 || !strings.Contains(jsonStr, "species") {
		// Player has no pets
		return ""
	}
	pets := jsonStr[start:]

	hash := sha256.Sum256([]byte(pets))
	return fmt.Sprintf("%x", hash)
}

func getPlayerTalents(path string) playerTalents {
	type Tooltip struct {
		Talent keyedValue
	}
	type Talent struct {
		ID      int
		Rank    int
		Tooltip Tooltip
	}
	type Selected struct {
		Talent keyedValue
	}
	type PvPTalent struct {
		Selected Selected
	}
	type Loadout struct {
		Active       bool     `json:"is_active"`
		ClassTalents []Talent `json:"selected_class_talents"`
		SpecTalents  []Talent `json:"selected_spec_talents"`
	}
	type Specialization struct {
		Specialization keyedValue
		Loadouts       []Loadout
		PvPTalentSlots []PvPTalent `json:"pvp_talent_slots"`
		ClassTalents   []Talent    `json:"selected_class_talents"`
		SpecTalents    []Talent    `json:"selected_spec_talents"`
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
		logger.Printf("%s json parsing failed: %s", warnPrefix, err)
		return playerTalents{}
	}

	activeSpecID := specializations.ActiveSpecialization.ID
	talents := make([]int, 0)
	pvpTalents := make([]int, 0)
	var classTalents []Talent
	var specTalents []Talent
	for _, spec := range specializations.Specializations {
		if spec.Specialization.ID != activeSpecID {
			continue
		}

		// Players with a loadout will have class and spec talents there
		for _, loadout := range spec.Loadouts {
			if !loadout.Active {
				continue
			}
			classTalents = loadout.ClassTalents
			specTalents = loadout.SpecTalents
			break
		}
		// Players not using loadouts have talents directly on the spec object
		if len(spec.ClassTalents) > 0 {
			classTalents = spec.ClassTalents
		}
		if len(spec.SpecTalents) > 0 {
			specTalents = spec.SpecTalents
		}

		for _, talent := range classTalents {
			id := talent.Tooltip.Talent.ID
			if id > 0 {
				talents = append(talents, id)
			}
		}
		for _, talent := range specTalents {
			id := talent.Tooltip.Talent.ID
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
		logger.Printf("%s json parsing failed: %s", warnPrefix, err)
		return stats{}
	}
	crit := highestStat(int(s.MeleeCrit.Rating), int(s.RangedCrit.Rating), int(s.SpellCrit.Rating))
	haste := highestStat(int(s.MeleeHaste.Rating), int(s.RangedHaste.Rating), int(s.SpellHaste.Rating))
	return stats{
		Strength:       int32(s.Strength.Effective),
		Agility:        int32(s.Agility.Effective),
		Intellect:      int32(s.Intellect.Effective),
		Stamina:        int32(s.Stamina.Effective),
		CriticalStrike: int32(crit),
		Haste:          int32(haste),
		Versatility:    int32(s.Versatility),
		Mastery:        int32(s.Mastery.Rating),
		Leech:          int32(s.Lifesteal.Rating),
		Dodge:          int32(s.Dodge.Rating),
		Parry:          int32(s.Parry.Rating)}
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
		logger.Printf("%s json parsing failed: %s", warnPrefix, err)
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

func squashItems(playersItems *cmap.ConcurrentMap[string, items]) map[int]item {
	items := make(map[int]item, 0)

	for tpl := range playersItems.Iter() {
		pi := tpl.Val

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

	items[itemToAdd.ID] = itemToAdd
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
		logger.Printf("%s json parsing failed: %s", warnPrefix, err)
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
		return errors.New("data is nil, nothing to unmarshal")
	}

	err := json.Unmarshal(*data, &v)
	if err != nil {
		logger.Printf("%s JSON parsing failed: %s", warnPrefix, err)
		return err
	}

	return nil
}

type playerTalents struct {
	Talents    []int
	PvPTalents []int
}
