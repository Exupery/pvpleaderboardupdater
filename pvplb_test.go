package pvpleaderboardupdater

import (
	"testing"
)

func TestCreateToken(t *testing.T) {
	var token string = createToken()
	if len(token) == 0 {
		t.Error("Creating token failed")
	}
}

func TestGet(t *testing.T) {
	var resp *[]byte = getDynamic("US", "token")
	if len(*resp) == 0 {
		t.Error("No response from GET")
	}
}

func TestParseRealms(t *testing.T) {
	var realmJSON *[]byte = getStatic("TODO", "realm/status")
	var realms []Realm = parseRealms(realmJSON)

	if realms == nil || len(realms) == 0 {
		t.Error("Parsing realms failed")
	}
}

func TestParseRaces(t *testing.T) {
	var racesJSON *[]byte = getStatic("TODO", "data/character/races")
	var races []Race = parseRaces(racesJSON)

	if races == nil || len(races) == 0 {
		t.Error("Parsing races failed")
	}
}

func TestParseClasses(t *testing.T) {
	var classesJSON *[]byte = getStatic("TODO", "data/character/classes")
	var classes []Class = parseClasses(classesJSON)

	if classes == nil || len(classes) == 0 {
		t.Error("Parsing classes failed")
	}
}

func TestParseSpecsTalents(t *testing.T) {
	var specs *[]Spec
	var talents *[]Talent
	var classes *[]Class = retrieveClasses()
	specs, talents = retrieveSpecsTalents(classes)

	if specs == nil || len(*specs) == 0 {
		t.Error("Parsing specs failed")
	}
	if talents == nil || len(*talents) == 0 {
		t.Error("Parsing talents failed")
	}
}

func TestClassSlugToIdMap(t *testing.T) {
	const msg = "Creating Class Slug=>Id map failed"
	var classes *[]Class = retrieveClasses()

	if classes == nil || len(*classes) == 0 {
		t.Error(msg)
	}

	var slugIDMap map[string]int = classSlugToIDMap(classes)
	if slugIDMap == nil || len(slugIDMap) != len(*classes) {
		t.Error(msg)
	}
}

func TestParseAchievements(t *testing.T) {
	var achievementsJSON *[]byte = getStatic("TODO", "data/character/achievements")
	var achievements []Achievement = parseAchievements(achievementsJSON)

	if achievements == nil || len(achievements) == 0 {
		t.Error("Parsing achievements failed")
	}
}

func TestParsePlayerDetails(t *testing.T) {
	var playerJSON *[]byte = getDynamic("TODO", "character/tichondrius/Exupery?fields=talents,guild,achievements,stats,items")
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
