package main

import (
	"math"
	"testing"
)

const testRegion = "US"
const testSeason = 29
const testPlayerPath = "emerald-dream/exuperjun"

func TestCreateToken(t *testing.T) {
	var token string = createToken()
	if len(token) == 0 {
		t.Error("Creating token failed")
	}
	t.Logf("Created token '%s'", token)
}

func TestGet(t *testing.T) {
	var resp *[]byte = getDynamic(testRegion, "token/")
	if len(*resp) == 0 {
		t.Error("No response from GET")
	}
}

func TestSafeUnmarshal(t *testing.T) {
	type EmptyInterface struct {
	}
	var empty EmptyInterface
	err := safeUnmarshal(nil, &empty)
	if err == nil {
		t.Error("Error should be returned when unmarshalling nil")
	}

	var realmJSON *[]byte = getDynamic(testRegion, "realm/index")
	realmSlice := (*realmJSON)[1:4]
	err = safeUnmarshal(&realmSlice, &empty)
	if err == nil {
		t.Error("Error should be returned when unmarshalling partial JSON")
	}
}

func TestParseRealms(t *testing.T) {
	var realmJSON *[]byte = getDynamic(testRegion, "realm/index")
	var realms []realm = parseRealms(realmJSON)

	if realms == nil || len(realms) == 0 {
		t.Error("Parsing realms failed")
	}
	t.Logf("Found and parsed %v realms", len(realms))
}

func TestParseRaces(t *testing.T) {
	var racesJSON *[]byte = getStatic(testRegion, "playable-race/index")
	var races []race = parseRaces(racesJSON)

	if races == nil || len(races) == 0 {
		t.Error("Parsing races failed")
	}
	t.Logf("Found and parsed %v races", len(races))
}

func TestParseClasses(t *testing.T) {
	var classesJSON *[]byte = getStatic(testRegion, "playable-class/index")
	var classes []class = parseClasses(classesJSON)

	if classes == nil || len(classes) == 0 {
		t.Error("Parsing classes failed")
	}
	t.Logf("Found and parsed %v classes", len(classes))
}

func TestParseSpecs(t *testing.T) {
	var specsJSON *[]byte = getStatic(testRegion, "playable-specialization/index")
	var specs []spec = parseSpecs(specsJSON)

	if specs == nil || len(specs) == 0 {
		t.Error("Parsing specs failed")
	}
	t.Logf("Found and parsed %v specs", len(specs))

	for _, spec := range specs {
		if len(spec.Icon) == 0 {
			t.Error("Parsing specs failed")
		}
	}
}

func TestTalentTreePaths(t *testing.T) {
	paths := getTalentTreePaths()

	if paths == nil || len(paths) == 0 {
		t.Error("Getting talent tree paths failed")
	}
	t.Logf("Found %v talent tree paths", len(paths))
}

func TestExtractTalentTreePath(t *testing.T) {
	var cases = map[string]string{
		"https://a.b.c/d/e/talent-tree/781/playable-specialization/270?f": "talent-tree/781/playable-specialization/270",
		"foo/talent-tree/1234/playable-specialization/5678/bar":           "talent-tree/1234/playable-specialization/5678",
		"talent-tree/1/playable-specialization/2":                         "talent-tree/1/playable-specialization/2",
		"talent-tree/11/playable-specialization/22":                       "talent-tree/11/playable-specialization/22",
		"talent-tree/111/playable-specialization/222":                     "talent-tree/111/playable-specialization/222",
		"talent-tree/1/invalid/playable-specialization/2":                 "",
		"invalidtalent-tree/1/playable-specialization/2":                  "",
		"talent-tree/1/playable-specializationinvalid/2":                  "",
		"talent-tree/1/playable-specialization/invalid/2":                 "",
		"talent-tree/1/invalid/2":                                         "",
		"talent-tree/playable-specialization/2":                           "",
		"talent-tree/1/playable-specialization":                           "",
		"talent-tree/1/playable-specialization/invalid/":                  "",
	}

	for href, expected := range cases {
		actual := parseTalentTreePath(href)
		if actual != expected {
			t.Errorf("Returned '%s' for '%s' but expected '%s'", actual, href, expected)
		}
	}
}

func TestGetTalentsFromTree(t *testing.T) {
	path := "talent-tree/781/playable-specialization/270"
	talents := getTalentsFromTree(path, map[int]bool{})

	if talents == nil || len(talents) == 0 {
		t.Error("Getting talents from talent tree failed")
	}
	t.Logf("Found and parsed %v talents", len(talents))
}

func TestParsePvPTalents(t *testing.T) {
	var talentsJSON *[]byte = getStatic(region, "pvp-talent/index")
	var pvpTalents []pvpTalent = parsePvPTalents(talentsJSON)

	if pvpTalents == nil || len(pvpTalents) == 0 {
		t.Error("Parsing PvP Talents failed")
	}
	t.Logf("Found and parsed %v PvP Talents", len(pvpTalents))
}

func TestParseAchievements(t *testing.T) {
	var achievementsJSON *[]byte = getStatic(testRegion, "achievement-category/15270")
	var achievements []achievement = parseAchievements(achievementsJSON)

	if achievements == nil || len(achievements) == 0 {
		t.Error("Parsing achievements failed")
	}
	t.Logf("Found and parsed %v PvP achievements", len(achievements))
}

func TestGetCurrentSeason(t *testing.T) {
	var currentSeason = getCurrentSeason()

	if currentSeason == 0 {
		t.Error("Determining current season failed")
	}
	t.Logf("Current PvP season is %d", currentSeason)
}

func TestGetLeaderboard(t *testing.T) {
	var leaderboard = getLeaderboard("2v2", testSeason)

	if leaderboard == nil || len(leaderboard) == 0 {
		t.Error("Parsing current season failed")
	}
	t.Logf("Found %d players on leaderboard", len(leaderboard))
}

func TestGetPlayersFromLeaderboards(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	var a = getLeaderboard("2v2", testSeason)
	var b = getLeaderboard("3v3", testSeason)
	var players = getPlayersFromLeaderboards(map[string][]leaderboardEntry{"2v2": a, "3v3": b})

	if players == nil || len(players) == 0 {
		t.Error("Getting leaderboard players failed")
	}
	t.Logf("Found %d players from leaderboards", len(players))
}

func TestSliceSplitting(t *testing.T) {
	max := 100
	slice := make([]*player, 0)
	for i := 0; i < max; i++ {
		slice = append(slice, &player{BlizzardID: i})
	}

	groups := split(slice, len(slice))
	validateSplitting(t, groups, 1, max)

	groups = split(slice, 10)
	validateSplitting(t, groups, len(slice)/10, 10)
}

func validateSplitting(t *testing.T, groups [][]*player, expectedNumGroups, maxGroupSize int) {
	if len(groups) != expectedNumGroups {
		t.Errorf("Returned %d groups, but expected %d", len(groups), expectedNumGroups)
	}
	for _, group := range groups {
		if len(group) > maxGroupSize {
			t.Errorf("Group has %d elements - should not exceed %d", len(group), maxGroupSize)
		}
	}
}

func TestGetPlayerProfileDetails(t *testing.T) {
	player := player{Path: testPlayerPath}
	setPlayerDetails(&player)

	if player.ClassID == 0 {
		t.Error("Player class NOT set")
	}
	if player.SpecID == 0 {
		t.Error("Player spec NOT set")
	}
	if player.FactionID == 0 {
		t.Error("Player faction NOT set")
	}
	if player.RaceID == 0 {
		t.Error("Player race NOT set")
	}
	if player.LastLogin == 0 {
		t.Error("Last login time NOT set")
	}
	t.Logf("Last login %d", player.LastLogin)
}

func TestGetPlayerTalents(t *testing.T) {
	talents := getPlayerTalents(testPlayerPath)
	if len(talents.Talents) == 0 || len(talents.PvPTalents) == 0 {
		t.Error("Getting player talents failed")
	}
	t.Logf("Found %d talents and %d PvP talents", len(talents.Talents), len(talents.PvPTalents))
}

func TestGetPlayerStats(t *testing.T) {
	stats := getPlayerStats(testPlayerPath)
	if stats.Intellect == 0 || stats.Stamina == 0 {
		t.Error("Getting player stats failed")
	}
	t.Logf("Found stats: %v", stats)
}

func TestHighestStat(t *testing.T) {
	a := highestStat(3, 2, 1)
	if a != 3 {
		t.Errorf("Expected 3 as highest stat, not %d", a)
	}
	b := highestStat(2, 3, 1)
	if b != 3 {
		t.Errorf("Expected 3 as highest stat, not %d", b)
	}
	c := highestStat(1, 2, 3)
	if c != 3 {
		t.Errorf("Expected 3 as highest stat, not %d", c)
	}
	ab := highestStat(3, 3, 1)
	if ab != 3 {
		t.Errorf("Expected 3 as highest stat, not %d", ab)
	}
	max := math.MaxInt32
	z := highestStat(max, max, max)
	if z != max {
		t.Errorf("Expected %d as highest stat, not %d", max, z)
	}
}

func TestGetPlayerItems(t *testing.T) {
	items := getPlayerItems(testPlayerPath)
	if items.Back.ID == 0 || items.Shoulder.ID == 0 {
		t.Error("Getting player items failed")
	}
	t.Logf("Found items: %v", items)
}

func TestSquashPlayerItems(t *testing.T) {
	itemsA := items{Neck: item{178927, "Clouded Focus", "LEGENDARY"}, Shoulder: item{}}
	itemsB := items{Neck: item{1, "Foo", "EPIC"}, Shoulder: item{2, "Bar", "EPIC"}}
	itemsC := items{Neck: item{1, "Foo", "EPIC"}, Shoulder: item{3, "Baz", "EPIC"}}

	playersItems := map[int]items{47: itemsA, 1138: itemsB, 1701: itemsC}
	squashedItems := squashItems(playersItems)
	seen := make(map[string]bool, 0)

	for id, item := range squashedItems {
		if id == 0 {
			t.Error("Invalid item present after squashing")
		}
		if seen[item.Name] {
			t.Error("Duplicate items not squished")
		}
		seen[item.Name] = true
		if item.Quality != "LEGENDARY" {
			continue
		}
		if item.Name == "Clouded Focus" {
			t.Errorf("Non legendary base-item name found: %s", item.Name)
		}
	}

	t.Logf("Squashed items: %v", squashedItems)
}

func TestGetPlayerAchievements(t *testing.T) {
	achieved := getPlayerAchievements(testPlayerPath, map[int]bool{2092: true, 13989: true})
	if achieved == nil || len(achieved) == 0 {
		t.Error("Getting player achievements failed")
	}
	t.Logf("Found achievements: %v", achieved)
}

func TestDetermineAlt(t *testing.T) {
	var altPlayerPath = "tichondrius/padatika"
	altID := getProfileIdentifier(altPlayerPath)
	mainID := getProfileIdentifier(testPlayerPath)
	if mainID == "" {
		t.Error("Unable to generate valid ID")
	}
	if mainID != altID {
		t.Error("IDs do NOT match for main and alt characters")
	}
	t.Logf("Main ID: %s", mainID)
	t.Logf("Alt ID: %s", altID)
}
