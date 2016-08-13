package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	uriBase = os.Getenv("TEST_BASE_URI")
	os.Exit(m.Run())
}

func TestParseRealms(t *testing.T) {
	var realmJson *[]byte = get("realm/status")
	var realms []Realm = parseRealms(realmJson)

	if realms == nil || len(realms) == 0 {
		t.Error("Parsing realms failed")
	}
}

func TestParseRaces(t *testing.T) {
	var racesJson *[]byte = get("data/character/races")
	var races []Race = parseRaces(racesJson)

	if races == nil || len(races) == 0 {
		t.Error("Parsing races failed")
	}
}

func TestParseClasses(t *testing.T) {
	var classesJson *[]byte = get("data/character/classes")
	var classes []Class = parseClasses(classesJson)

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

	var slugIdMap map[string]int = classSlugToIdMap(classes)
	if slugIdMap == nil || len(slugIdMap) != len(*classes) {
		t.Error(msg)
	}
}

func TestParseAchievements(t *testing.T) {
	var achievementsJson *[]byte = get("data/character/achievements")
	var achievements []Achievement = parseAchievements(achievementsJson)

	if achievements == nil || len(achievements) == 0 {
		t.Error("Parsing achievements failed")
	}
}

func TestParsePlayerDetails(t *testing.T) {
	var playerJson *[]byte = get("character/tichondrius/Exupery?fields=talents,guild,achievements,stats,items")
	m := map[string]int{"9Affliction": 265}
	var player *Player = parsePlayerDetails(playerJson, &m)

	if player == nil {
		t.Error("Parsing player details failed")
	}

	if len(player.AchievementIds) == 0 {
		t.Error("Parsing player AchievementIds failed")
	}

	if len(player.AchievementTimestamps) == 0 {
		t.Error("Parsing player AchievementTimestamps failed")
	}

	if len(player.TalentIds) == 0 {
		t.Error("Parsing player TalentIds failed")
	}

	if player.Stats.Sta == 0 {
		t.Error("Parsing player Stats failed")
	}

	if player.Items.AverageItemLevel == 0 {
		t.Error("Parsing player AverageItemLevel failed")
	}
}
