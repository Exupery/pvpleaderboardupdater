package pvpleaderboardupdater

import (
	"encoding/json"
	"fmt"
)

func importStaticData() {
	logger.Println("Beginning import of static data")
	importRealms() // TODO IMPORT FOR EACH REGION
	importRaces()

	var classes *[]Class = retrieveClasses()
	logger.Printf("Parsed %v classes", len(*classes))
	addClasses(classes)

	// TODO SPECS AND TALENTS
	importAchievements()

	logger.Println("Static data import complete")
}

func parseRealms(data *[]byte) []Realm {
	type Realms struct {
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
	var realmJSON *[]byte = getDynamic(region, "realm/index")
	var realms []Realm = parseRealms(realmJSON)
	logger.Printf("Parsed %v %s realms", len(realms), region)
	addRealms(&realms)
}

func parseRaces(data *[]byte) []Race {
	type Races struct {
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
	var racesJSON *[]byte = getStatic(region, "data/character/races")
	var races []Race = parseRaces(racesJSON)
	logger.Printf("Parsed %v races", len(races))
	addRaces(&races)
}

func parseClasses(data *[]byte) []Class {
	type Classes struct {
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
	var classesJSON *[]byte = getStatic(region, "data/character/classes")
	var classes []Class = parseClasses(classesJSON)
	return &classes
}

func parseSpecs(data *[]byte) []Spec {
	type CharacterSpecializationJSON struct {
		ID int
	}
	type SpecsJSON struct {
		CharacterSpecializations []CharacterSpecializationJSON `json:"character_specializations"`
	}
	var specsJSON SpecsJSON
	err := json.Unmarshal(*data, &specsJSON)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]Spec, 0)
	}
	var specIDs []int = make([]int, 0)
	for _, cs := range specsJSON.CharacterSpecializations {
		specIDs = append(specIDs, cs.ID)
	}
	return getFullSpecInfo(specIDs)
}

func getFullSpecInfo(specIDs []int) []Spec {
	var specs []Spec = make([]Spec, 0)
	var ch chan Spec = make(chan Spec, len(specIDs))
	for _, i := range specIDs {
		go getSpec(ch, i)
	}
	for range specIDs {
		specs = append(specs, <-ch)
	}
	return specs
}

func getSpec(ch chan Spec, specID int) {
	type RoleJSON struct {
		Role string `json:"type"`
	}
	type SpecJSON struct {
		ID            int
		PlayableClass Class `json:"playable_class"`
		Name          string
		Media         Media
		Role          RoleJSON
		TalentTiers   []TalentTierJSON `json:"talent_tiers"`
	}
	var path string = fmt.Sprintf("playable-specialization/%d", specID)
	var icon = getIcon(region, path)
	var specJSON *[]byte = getStatic(region, path)
	var s SpecJSON
	json.Unmarshal(*specJSON, &s)
	// TODO PVP TALENTS
	ch <- Spec{
		s.ID,
		s.PlayableClass.ID,
		s.Name,
		s.Role.Role,
		icon,
		getFullSpecTalents(s.TalentTiers)}
}

func getFullSpecTalents(talentTiers []TalentTierJSON) []Talent {
	var talents []Talent = make([]Talent, 0)
	type TalentJSON struct {
		ID            int
		Spell         KeyedValue
		PlayableClass Class `json:"playable_class"`
	}
	for _, t := range talentTiers {
		tier := t.TierIndex
		for _, talentEntry := range t.Talents {
			col := talentEntry.ColumnIndex
			id := talentEntry.Talent.ID
			var talentJSON *[]byte = getStatic(region, fmt.Sprintf("talent/%d", id))
			var talentDetails TalentJSON
			json.Unmarshal(*talentJSON, &talentDetails)
			icon := getIcon(region, fmt.Sprintf("spell/%d", talentDetails.Spell.ID))
			talent := Talent{
				id,
				talentDetails.PlayableClass.ID,
				talentDetails.Spell.ID,
				talentDetails.Spell.Name,
				icon,
				tier,
				col}
			talents = append(talents, talent)
		}
	}
	return talents
}

func parseAchievements(data *[]byte) []Achievement {
	var pvpAchievements []Achievement = make([]Achievement, 0)
	type AchievementCategory struct {
		ID           int
		Name         string
		Achievements []Achievement
		Categories   []AchievementCategory
	}
	type Achievements struct {
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
	var achievementsJSON *[]byte = getStatic(region, "data/character/achievements")
	var achievements []Achievement = parseAchievements(achievementsJSON)
	logger.Printf("Parsed %v achievements", len(achievements))
	addAchievements(&achievements)
}
