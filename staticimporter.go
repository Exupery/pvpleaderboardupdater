package main

import "encoding/json"

func importStaticData() {
	logger.Println("Beginning import of static data")
	importRealms()
	importRaces()
	importFactions()
	importClasses()
	importAchievements()

	// specs, talents, and glyphs share an endpoint and are grouped by class
	var specs *[]Spec
	var talents *[]Talent
	var glyphs *[]Glyph
	specs, talents, glyphs = retrieveSpecsTalentsGlyphs()
	println(*specs, *talents, *glyphs) // TODO DELME
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
	// TODO WRITE REALMS TO DB
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
	var racesJson *[]byte = get("character/races")
	var races []Race = parseRaces(racesJson)
	logger.Printf("Parsed %v races", len(races))
	// TODO WRITE RACES TO DB
}

func importFactions() {
	// No faction data via API
	factions := [2]Faction{Faction{0, "Alliance"}, Faction{1, "Horde"}}
	logger.Printf("Parsed %v factions", len(factions))
	// TODO WRITE FACTIONS TO DB
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

func importClasses() {
	var classesJson *[]byte = get("character/classes")
	var classes []Class = parseClasses(classesJson)
	logger.Printf("Parsed %v classes", len(classes))
	// TODO WRITE CLASSES TO DB
}

func retrieveSpecsTalentsGlyphs() (*[]Spec, *[]Talent, *[]Glyph) {
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

	for _, v := range m {
		logger.Println(v.Class)	// TODO DELME
		specs = append(specs, v.Specs...)
		glyphs = append(glyphs, v.Glyphs...)
		for _, t := range v.Talents {
			for _, t1 := range t {
				for _, t2 := range t1 {
					var talent Talent = Talent{
						t2.Spell.Id,
						v.Class,
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

func importSpecs(specs *[]Spec) {
	// TODO WRITE SPECS TO DB
}

func importTalents(talents *[]Talent) {
	// TODO WRITE TALENTS TO DB
}

func importGlyphs(glyphs *[]Glyph) {
	// TODO WRITE GLYPHS TO DB
}

func importAchievements() {
	// TODO IMPORT ACHIEVEMENTS
	// TODO WRITE ACHIEVEMENTS TO DB
}