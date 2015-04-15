package main

import (
	"encoding/json"
	"strings"
)

func importStaticData() {
	logger.Println("Beginning import of static data")
	importRealms()
	importRaces()
	importFactions()
	importAchievements()

	var classes *[]Class = retrieveClasses()
	logger.Printf("Parsed %v classes", len(*classes))
	importClasses(classes)

	// specs, talents, and glyphs share an endpoint and are grouped by class
	var specs *[]Spec
	var talents *[]Talent
	var glyphs *[]Glyph
	specs, talents, glyphs = retrieveSpecsTalentsGlyphs(classes)
	logger.Printf("Parsed %v specs", len(*specs))
	importSpecs(specs)
	logger.Printf("Parsed %v talents", len(*talents))
	importTalents(talents)
	logger.Printf("Parsed %v glyphs", len(*glyphs))
	importGlyphs(glyphs)

	logger.Println("Static data import complete")
}

func parseRealms(data *[]byte) []Realm {
	type Realms struct  {
		Realms []Realm
	}
	var realms Realms
	err := json.Unmarshal(*data, &realms)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]Realm, 0)
	}
	return realms.Realms
}

func importRealms() {
	var realmJson *[]byte = get("realm/status")
	var realms []Realm = parseRealms(realmJson)
	logger.Printf("Parsed %v realms", len(realms))
	addRealms(&realms)
}

func parseRaces(data *[]byte) []Race {
	type Races struct  {
		Races []Race
	}
	var races Races
	err := json.Unmarshal(*data, &races)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]Race, 0)
	}
	return races.Races
}

func importRaces() {
	var racesJson *[]byte = get("data/character/races")
	var races []Race = parseRaces(racesJson)
	logger.Printf("Parsed %v races", len(races))
	addRaces(&races)
}

func importFactions() {
	// No faction data via API
	factions := []Faction{Faction{0, "Alliance"}, Faction{1, "Horde"}}
	logger.Printf("Parsed %v factions", len(factions))
	addFactions(&factions)
}

func parseClasses(data *[]byte) []Class {
	type Classes struct  {
		Classes []Class
	}
	var classes Classes
	err := json.Unmarshal(*data, &classes)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]Class, 0)
	}
	return classes.Classes
}

func retrieveClasses() *[]Class {
	var classesJson *[]byte = get("data/character/classes")
	var classes []Class = parseClasses(classesJson)
	return &classes
}

func importClasses(classes *[]Class) {
	addClasses(classes)
}

func retrieveSpecsTalentsGlyphs(classes *[]Class) (*[]Spec, *[]Talent, *[]Glyph) {
	var specs []Spec = make([]Spec, 0)
	var talents []Talent = make([]Talent, 0)
	var glyphs []Glyph = make([]Glyph, 0)

	type Spell struct {
		Id int
		Name string
		Icon string
		Description string
	}
	type TalentJson struct {
		Tier int
		Column int
		Spell Spell
	}
	type ClassData struct {
		Class string
		Glyphs []Glyph
		Talents [][][]TalentJson
		Specs []Spec
	}

	var m map[string]ClassData
	var data *[]byte = get("data/talents")
	err := json.Unmarshal(*data, &m)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return &specs, &talents, &glyphs
	}

	var classIds map[string]int = classSlugToIdMap(classes)
	var specIds map[string]int = specSlugToIdMap()
	var glyphSpellIds map[string]int = glyphSpellIdsMap()

	for _, v := range m {
		var classId int = classIds[v.Class]
		for _, spec := range v.Specs {
			var specId int = specIds[v.Class + spec.Name]
			if specId == 0 {
				logger.Printf("%s ID not found for spec %s", errPrefix, v.Class + spec.Name)
			}
			spec.ClassId = classId
			spec.Id = specId
			specs = append(specs, spec)
		}
		for _, glyph := range v.Glyphs {
			glyph.ClassId = classId
			glyph.SpellId = glyphSpellIds[glyph.Name]
			if glyph.SpellId == 0 {
				logger.Printf("%s SpellID not found for '%s'", errPrefix, glyph.Name)
			}
			glyphs = append(glyphs, glyph)
		}
		for _, t := range v.Talents {
			for _, t1 := range t {
				for _, t2 := range t1 {
					var talent Talent = Talent{
						t2.Spell.Id,
						classId,
						t2.Spell.Name,
						t2.Spell.Description,
						t2.Spell.Icon,
						t2.Tier,
						t2.Column}
					talents = append(talents, talent)
				}
			}
		}
	}

	return &specs, &talents, &glyphs
}

func classSlugToIdMap(classes *[]Class) map[string]int {
	var m map[string]int = make(map[string]int)
	for _, c := range *classes {
		var slug string = strings.ToLower(c.Name)
		slug = strings.Replace(slug, " ", "-", -1)
		m[slug] = c.Id
	}

	return m
}

func specSlugToIdMap() map[string]int {
	// Spec name=>ID mapping not available via API
	return map[string]int{
		"mageArcane": 62,
		"mageFire": 63,
		"mageFrost": 64,
		"paladinHoly": 65,
		"paladinProtection": 66,
		"paladinRetribution": 70,
		"warriorArms": 71,
		"warriorFury": 72,
		"warriorProtection": 73,
		"druidBalance": 102,
		"druidFeral": 103,
		"druidGuardian": 104,
		"druidRestoration": 105,
		"death-knightBlood": 250,
		"death-knightFrost": 251,
		"death-knightUnholy": 252,
		"hunterBeast Mastery": 253,
		"hunterMarksmanship": 254,
		"hunterSurvival": 255,
		"priestDiscipline": 256,
		"priestHoly": 257,
		"priestShadow": 258,
		"rogueAssassination": 259,
		"rogueCombat": 260,
		"rogueSubtlety": 261,
		"shamanElemental": 262,
		"shamanEnhancement": 263,
		"shamanRestoration": 264,
		"warlockAffliction": 265,
		"warlockDemonology": 266,
		"warlockDestruction": 267,
		"monkBrewmaster": 268,
		"monkWindwalker": 269,
		"monkMistweaver": 270}
}

func importSpecs(specs *[]Spec) {
	addSpecs(specs)
}

func importTalents(talents *[]Talent) {
	addTalents(talents)
}

func importGlyphs(glyphs *[]Glyph) {
	addGlyphs(glyphs)
}

func parseAchievements(data *[]byte) []Achievement {
	var pvpAchievements []Achievement = make([]Achievement, 0)
	type AchievementCategory struct  {
		Id int
		Name string
		Achievements []Achievement
		Categories []AchievementCategory
	}
	type Achievements struct  {
		Achievements []AchievementCategory
	}

	var achievements Achievements
	err := json.Unmarshal(*data, &achievements)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return pvpAchievements
	}

	for _, ac := range achievements.Achievements {
		if ac.Name == "Player vs. Player" {
			for _, acc := range ac.Categories {
				if acc.Name == "Rated Battleground" || acc.Name == "Arena" {
					pvpAchievements = append(pvpAchievements, acc.Achievements...)
				}
			}
		}
	}

	return pvpAchievements
}

func importAchievements() {
	var achievementsJson *[]byte = get("data/character/achievements")
	var achievements []Achievement = parseAchievements(achievementsJson)
	logger.Printf("Parsed %v achievements", len(achievements))
	addAchievements(&achievements)
}