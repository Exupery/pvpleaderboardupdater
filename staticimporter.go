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
	for _, r := range regions {
		importRealms(r)
	}
	importRaces()
	importClasses()
	importSpecsAndTalents()
	importPvPTalents()
	importAchievements()

	logger.Println("Static data import complete")
}

func parseRealms(data *[]byte) []realm {
	type Realms struct {
		Realms []realm
	}
	var realms Realms
	err := json.Unmarshal(*data, &realms)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]realm, 0)
	}
	return realms.Realms
}

func importRealms(region string) {
	var realmJSON *[]byte = getDynamic(region, "realm/index")
	var realms []realm = parseRealms(realmJSON)
	logger.Printf("Found %d %s realms", len(realms), region)
	addRealms(&realms, region)
}

func parseRaces(data *[]byte) []race {
	type Races struct {
		Races []race
	}
	var races Races
	err := json.Unmarshal(*data, &races)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]race, 0)
	}
	return races.Races
}

func importRaces() {
	var racesJSON *[]byte = getStatic(region, "playable-race/index")
	var races []race = parseRaces(racesJSON)
	logger.Printf("Found %d races", len(races))
	addRaces(&races)
}

func parseClasses(data *[]byte) []class {
	type Classes struct {
		Classes []class
	}
	var classes Classes
	err := json.Unmarshal(*data, &classes)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]class, 0)
	}
	return classes.Classes
}

func importClasses() {
	var classesJSON *[]byte = getStatic(region, "playable-class/index")
	var classes []class = parseClasses(classesJSON)
	logger.Printf("Found %d classes", len(classes))
	addClasses(&classes)
}

func importSpecsAndTalents() {
	var specsJSON *[]byte = getStatic(region, "playable-specialization/index")
	var specs []spec = parseSpecs(specsJSON)
	logger.Printf("Found %d specializations", len(specs))
	addSpecs(&specs)
	var talents []talent = make([]talent, 0)
	for _, spec := range specs {
		talents = append(talents, spec.Talents...)
	}
	addTalents(&talents)
}

func parseSpecs(data *[]byte) []spec {
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
		return make([]spec, 0)
	}
	var specIDs []int = make([]int, 0)
	for _, cs := range specsJSON.CharacterSpecializations {
		specIDs = append(specIDs, cs.ID)
	}
	return getFullSpecInfo(specIDs)
}

func getFullSpecInfo(specIDs []int) []spec {
	var specs []spec = make([]spec, 0)
	var ch chan spec = make(chan spec, len(specIDs))
	for _, i := range specIDs {
		go getSpec(ch, i)
	}
	for range specIDs {
		specs = append(specs, <-ch)
	}
	return specs
}

func getSpec(ch chan spec, specID int) {
	type RoleJSON struct {
		Role string `json:"type"`
	}
	type SpecJSON struct {
		ID            int
		PlayableClass class `json:"playable_class"`
		Name          string
		Media         keyedValue
		Role          RoleJSON
		TalentTiers   []talentTierJSON `json:"talent_tiers"`
	}
	var path string = fmt.Sprintf("playable-specialization/%d", specID)
	var icon = getIcon(region, path)
	var specJSON *[]byte = getStatic(region, path)
	var s SpecJSON
	json.Unmarshal(*specJSON, &s)
	ch <- spec{
		s.ID,
		s.PlayableClass.ID,
		s.Name,
		s.Role.Role,
		icon,
		getFullSpecTalents(specID, s.TalentTiers)}
}

func getFullSpecTalents(specID int, talentTiers []talentTierJSON) []talent {
	var talents []talent = make([]talent, 0)
	type TalentJSON struct {
		ID            int
		Spell         keyedValue
		PlayableClass class `json:"playable_class"`
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
			talent := talent{
				id,
				talentDetails.Spell.ID,
				talentDetails.PlayableClass.ID,
				specID,
				talentDetails.Spell.Name,
				icon,
				tier,
				col}
			talents = append(talents, talent)
		}
	}
	return talents
}

func importPvPTalents() {
	var talentsJSON *[]byte = getStatic(region, "pvp-talent/index")
	var pvpTalents []pvpTalent = parsePvPTalents(talentsJSON)
	logger.Printf("Found %d PvP Talents", len(pvpTalents))
	addPvPTalents(&pvpTalents)
}

func parsePvPTalents(data *[]byte) []pvpTalent {
	type PvPTalentsJSON struct {
		PvPTalents []keyedValue `json:"pvp_talents"`
	}
	var pvpTalentsJSON PvPTalentsJSON
	err := json.Unmarshal(*data, &pvpTalentsJSON)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return make([]pvpTalent, 0)
	}
	var pvpTalents []pvpTalent = make([]pvpTalent, 0)
	var ch chan pvpTalent = make(chan pvpTalent, len(pvpTalentsJSON.PvPTalents))
	for _, keyedValue := range pvpTalentsJSON.PvPTalents {
		go getPvPTalent(ch, keyedValue.ID)
	}
	for range pvpTalentsJSON.PvPTalents {
		pvpTalents = append(pvpTalents, <-ch)
	}
	return pvpTalents
}

func getPvPTalent(ch chan pvpTalent, id int) {
	type PvPTalentJSON struct {
		Spell                  keyedValue
		PlayableSpecialization keyedValue `json:"playable_specialization"`
	}
	var pvpTalentJSON *[]byte = getStatic(region, fmt.Sprintf("pvp-talent/%d", id))
	var talentDetails PvPTalentJSON
	json.Unmarshal(*pvpTalentJSON, &talentDetails)
	icon := getIcon(region, fmt.Sprintf("spell/%d", talentDetails.Spell.ID))
	ch <- pvpTalent{
		id,
		talentDetails.Spell.Name,
		talentDetails.Spell.ID,
		talentDetails.PlayableSpecialization.ID,
		icon}
}

func parseAchievements(data *[]byte) []achievement {
	type Achievements struct {
		ID           int
		Name         string
		Achievements []keyedValue
	}

	var achievements Achievements
	err := json.Unmarshal(*data, &achievements)
	var pvpAchievements []achievement = make([]achievement, 0)
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
	var ch chan achievement = make(chan achievement, len(achievementIDs))
	for _, id := range achievementIDs {
		go getPvPAchievement(ch, id)
	}
	for range achievementIDs {
		pvpAchievements = append(pvpAchievements, <-ch)
	}

	return pvpAchievements
}

func getPvPAchievement(ch chan achievement, id int) {
	ch <- getAchievement(id)
}

func getAchievement(id int) achievement {
	type PvPAchievementJSON struct {
		ID          int
		Name        string
		Description string
	}
	var pvpAchievementJSON *[]byte = getStatic(region, fmt.Sprintf("achievement/%d", id))
	var pvpAchievementJSONDetails PvPAchievementJSON
	json.Unmarshal(*pvpAchievementJSON, &pvpAchievementJSONDetails)
	return achievement{
		id,
		pvpAchievementJSONDetails.Name,
		pvpAchievementJSONDetails.Description}
}

func importAchievements() {
	var achievementsJSON *[]byte = getStatic(region, fmt.Sprintf("achievement-category/%d", pvpFeatsOfStrengthCategory))
	var achievements []achievement = parseAchievements(achievementsJSON)
	var seasonalCount int = len(achievements)
	logger.Printf("Found %d seasonal achievements", seasonalCount)
	for _, id := range achievementIDs {
		achievement := getAchievement(id)
		achievements = append(achievements, achievement)
	}
	logger.Printf("Found %d non-seasonal achievements", len(achievements)-seasonalCount)
	addAchievements(&achievements)
}
