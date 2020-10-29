package main

import (
	"testing"
)

const testRegion = "US"
const testSeason = 29
const testPlayerPath = "emerald-dream/exupery"

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
}

func TestGetPlayerTalents(t *testing.T) {
	talents := getPlayerTalents(testPlayerPath)
	if len(talents.Talents) == 0 || len(talents.PvPTalents) == 0 {
		t.Error("Getting player talents failed")
	}
	t.Logf("Found %d talents and %d PvP talents", len(talents.Talents), len(talents.PvPTalents))
}
