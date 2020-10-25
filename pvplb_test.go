package main

import (
	"testing"
)

const testRegion = "US"

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
	var realms []Realm = parseRealms(realmJSON)

	if realms == nil || len(realms) == 0 {
		t.Error("Parsing realms failed")
	}
	t.Logf("Found and parsed %v realms", len(realms))
}

func TestParseRaces(t *testing.T) {
	var racesJSON *[]byte = getStatic(testRegion, "playable-race/index")
	var races []Race = parseRaces(racesJSON)

	if races == nil || len(races) == 0 {
		t.Error("Parsing races failed")
	}
	t.Logf("Found and parsed %v races", len(races))
}

func TestParseClasses(t *testing.T) {
	var classesJSON *[]byte = getStatic(testRegion, "playable-class/index")
	var classes []Class = parseClasses(classesJSON)

	if classes == nil || len(classes) == 0 {
		t.Error("Parsing classes failed")
	}
	t.Logf("Found and parsed %v classes", len(classes))
}

func TestParseSpecsTalents(t *testing.T) {
	var specsJSON *[]byte = getStatic(testRegion, "playable-specialization/index")
	var specs []Spec = parseSpecs(specsJSON)

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
	var pvpTalents []PvPTalent = parsePvPTalents(talentsJSON)

	if pvpTalents == nil || len(pvpTalents) == 0 {
		t.Error("Parsing PvP Talents failed")
	}
	t.Logf("Found and parsed %v PvP Talents", len(pvpTalents))
}

func TestParseAchievements(t *testing.T) {
	var achievementsJSON *[]byte = getStatic(testRegion, "achievement-category/15270")
	var achievements []Achievement = parseAchievements(achievementsJSON)

	if achievements == nil || len(achievements) == 0 {
		t.Error("Parsing achievements failed")
	}
	t.Logf("Found and parsed %v PvP achievements", len(achievements))
}

func TestParsePlayerDetails(t *testing.T) {
	var playerJSON *[]byte = getDynamic(testRegion, "character/tichondrius/Exupery?fields=talents,guild,achievements,stats,items")
	m := map[string]int{"9Affliction": 265}
	var player *Player = parsePlayerDetails(playerJSON, &m)

	if player == nil {
		t.Error("Parsing player details failed")
	}

	if len(player.AchievementIDs) == 0 {
		t.Error("Parsing player AchievementIds failed")
	}

	if len(player.AchievementTimestamps) == 0 {
		t.Error("Parsing player AchievementTimestamps failed")
	}

	if len(player.TalentIDs) == 0 {
		t.Error("Parsing player TalentIds failed")
	}

	if player.Stats.Sta == 0 {
		t.Error("Parsing player Stats failed")
	}

	if player.Items.AverageItemLevel == 0 {
		t.Error("Parsing player AverageItemLevel failed")
	}
}
