package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

const pvpFeatsOfStrengthCategory int = 15270

var achievementIDs = []int{
	// Arena achievements
	401, 405, 404, 1159, 1160, 1161, 5266, 5267, 876, 2090, 2093, 2092, 2091,
	// RBG achievements
	5329, 5326, 5339, 5353, 5341, 5355, 5343, 5356, 6942, 6941}

func importStaticData() {
	logger.Println("Beginning import of static data")
	importRealms() // TODO IMPORT FOR EACH REGION
	importRaces()
	importClasses()
	importSpecsAndTalents()
	importPvPTalents()
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

func importClasses() {
	var classesJSON *[]byte = getStatic(region, "playable-class/index")
	var classes []Class = parseClasses(classesJSON)
	logger.Printf("Parsed %v classes", len(classes))
	addClasses(&classes)
}

func importSpecsAndTalents() *[]Spec {
	var specsJSON *[]byte = getStatic(region, "playable-specialization/index")
	var specs []Spec = parseSpecs(specsJSON)
	return &specs // TODO CALL ADDERS INSTEAD OF RETURNING
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

func importPvPTalents() *[]PvPTalent {
	var talentsJSON *[]byte = getStatic(region, "pvp-talent/index")
	var pvpTalents []PvPTalent = parsePvPTalents(talentsJSON)
	return &pvpTalents // TODO CALL ADDER INSTEAD OF RETURNING
}

func parsePvPTalents(data *[]byte) []PvPTalent {
	type PvPTalentsJSON struct {
		PvPTalents []KeyedValue `json:"pvp_talents"`
	}
	var pvpTalentsJSON PvPTalentsJSON
	err := json.Unmarshal(*data, &pvpTalentsJSON)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]PvPTalent, 0)
	}
	var pvpTalents []PvPTalent = make([]PvPTalent, 0)
	var ch chan PvPTalent = make(chan PvPTalent, len(pvpTalentsJSON.PvPTalents))
	for _, keyedValue := range pvpTalentsJSON.PvPTalents {
		go getPvPTalent(ch, keyedValue.ID)
	}
	for range pvpTalentsJSON.PvPTalents {
		pvpTalents = append(pvpTalents, <-ch)
	}
	return pvpTalents
}

func getPvPTalent(ch chan PvPTalent, id int) {
	type PvPTalentJSON struct {
		Spell                  KeyedValue
		PlayableSpecialization KeyedValue `json:"playable_specialization"`
	}
	var pvpTalentJSON *[]byte = getStatic(region, fmt.Sprintf("pvp-talent/%d", id))
	var talentDetails PvPTalentJSON
	json.Unmarshal(*pvpTalentJSON, &talentDetails)
	icon := getIcon(region, fmt.Sprintf("spell/%d", talentDetails.Spell.ID))
	ch <- PvPTalent{
		id,
		talentDetails.Spell.Name,
		talentDetails.Spell.ID,
		talentDetails.PlayableSpecialization.ID,
		icon}
}

func parseAchievements(data *[]byte) []Achievement {
	type Achievements struct {
		ID           int
		Name         string
		Achievements []KeyedValue
	}

	var achievements Achievements
	err := json.Unmarshal(*data, &achievements)
	var pvpAchievements []Achievement = make([]Achievement, 0)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return pvpAchievements
	}

	var achievementIDs []int = make([]int, 0)
	for _, ac := range achievements.Achievements {
		if strings.Contains(ac.Name, "Season") {
			achievementIDs = append(achievementIDs, ac.ID)
		}
	}
	var ch chan Achievement = make(chan Achievement, len(achievementIDs))
	for _, id := range achievementIDs {
		go getPvPAchievement(ch, id)
	}
	for range achievementIDs {
		pvpAchievements = append(pvpAchievements, <-ch)
	}

	return pvpAchievements
}

func getPvPAchievement(ch chan Achievement, id int) {
	type PvPAchievementJSON struct {
		ID          int
		Name        string
		Description string
	}
	var pvpAchievementJSON *[]byte = getStatic(region, fmt.Sprintf("achievement/%d", id))
	var pvpAchievementJSONDetails PvPAchievementJSON
	json.Unmarshal(*pvpAchievementJSON, &pvpAchievementJSONDetails)
	ch <- Achievement{
		id,
		pvpAchievementJSONDetails.Name,
		pvpAchievementJSONDetails.Description}
}

func importAchievements() {
	// TODO GET ALL MATCHING achievementIDs
	var achievementsJSON *[]byte = getStatic(region, fmt.Sprintf("achievement-category/%d", pvpFeatsOfStrengthCategory))
	var achievements []Achievement = parseAchievements(achievementsJSON)
	logger.Printf("Parsed %v achievements", len(achievements))
	addAchievements(&achievements)
}
