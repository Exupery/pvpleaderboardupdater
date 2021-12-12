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

func TestParseSpecsTalents(t *testing.T) {
	var specsJSON *[]byte = getStatic(testRegion, "playable-specialization/index")
	var specs []spec = parseSpecs(specsJSON)

	if specs == nil || len(specs) == 0 {
		t.Error("Parsing specs failed")
	}
	t.Logf("Found and parsed %v specs", len(specs))

	for _, spec := range specs {
		talents := spec.Talents
		if talents == nil || len(talents) == 0 {
			t.Error("Parsing talents failed")
		}
		t.Logf("Found and parsed %v talents for %s", len(talents), spec.Name)
	}
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

func TestParseCovenants(t *testing.T) {
	var covenantJSON *[]byte = getStatic(testRegion, "covenant/index")
	var covenants []covenant = parseCovenants(covenantJSON)

	if covenants == nil || len(covenants) == 0 {
		t.Error("Parsing covenants failed")
	}
	t.Logf("Found and parsed %v covenants", len(covenants))
}

func TestParseSoulbinds(t *testing.T) {
	var soulbindJSON *[]byte = getStatic(testRegion, "covenant/soulbind/index")
	var soulbinds []soulbind = parseSoulbinds(soulbindJSON)

	if soulbinds == nil || len(soulbinds) == 0 {
		t.Error("Parsing soulbinds failed")
	}
	t.Logf("Found and parsed %v soulbinds", len(soulbinds))
}

func TestParseConduits(t *testing.T) {
	var conduitJSON *[]byte = getStatic(testRegion, "covenant/conduit/index")
	var conduits []conduit = parseConduits(conduitJSON)

	if conduits == nil || len(conduits) == 0 {
		t.Error("Parsing conduits failed")
	}
	for _, conduit := range conduits {
		if conduit.SpellID == 0 {
			t.Errorf("Parsing conduit spell ID failed for %d: %s", conduit.ID, conduit.Name)
		}
	}
	t.Logf("Found and parsed %v conduits", len(conduits))
}
