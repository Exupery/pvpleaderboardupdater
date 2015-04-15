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

func TestParseSpecsTalentsGlyphs(t *testing.T) {
	var specs *[]Spec
	var talents *[]Talent
	var glyphs *[]Glyph
	var classes *[]Class = retrieveClasses()
	specs, talents, glyphs = retrieveSpecsTalentsGlyphs(classes)

	if specs == nil || len(*specs) == 0 {
		t.Error("Parsing specs failed")
	}
	if talents == nil || len(*talents) == 0 {
		t.Error("Parsing talents failed")
	}
	if glyphs == nil || len(*glyphs) == 0 {
		t.Error("Parsing glyphs failed")
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

func TestGlyphSpellIdsMap(t *testing.T) {
	const msg = "Creating Glyph Spell IDs map failed"
	var glyphs *[]Glyph
	var classes *[]Class = retrieveClasses()
	_, _, glyphs = retrieveSpecsTalentsGlyphs(classes)

	var glyphSpellIds map[string]int = glyphSpellIdsMap()
	if glyphSpellIds == nil || len(glyphSpellIds) < len(*glyphs) {
		t.Error(msg)
	}

	for _, glyph := range *glyphs {
		if glyphSpellIds[glyph.Name] == 0 {
			t.Error(glyph.Name + " missing from glyphSpellIds")
		}
	}
}