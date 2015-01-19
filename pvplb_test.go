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
	var racesJson *[]byte = get("character/races")
	var races []Race = parseRaces(racesJson)

	if races == nil || len(races) == 0 {
		t.Error("Parsing races failed")
	}
}

func TestParseClasses(t *testing.T) {
	var classesJson *[]byte = get("character/classes")
	var classes []Class = parseClasses(classesJson)

	if classes == nil || len(classes) == 0 {
		t.Error("Parsing classes failed")
	}
}

func TestParseSpecsTalentsGlyphs(t *testing.T) {
	var specs *[]Spec
	var talents *[]Talent
	var glyphs *[]Glyph
	specs, talents, glyphs = retrieveSpecsTalentsGlyphs()

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