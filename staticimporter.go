package main

import (
	"fmt"
	"strings"
)

const pvpFeatsOfStrengthCategory int = 15270

var achievementIDs = []int{
	// 2v2
	399, 400, 401, 1159,
	// 3v3
	402, 403, 405, 1160, 5266, 5267, 2091,
	// RBG achievements
	5341, 5355, 5343, 5356, 6942, 6941,
}

var realmRegions = []string{"EU", "US", "KR", "TW"}

func importStaticData() {
	logger.Println("Beginning import of static data")
	for _, r := range realmRegions {
		importRealms(r)
	}
	importRaces()
	importClasses()
	importSpecs()
	importTalents()
	importPvPTalents()
	importAchievements()

	logger.Println("Static data import complete")
}

func parseRealms(data *[]byte) []realm {
	type Realms struct {
		Realms []realm
	}
	var realms Realms
	err := safeUnmarshal(data, &realms)
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
	err := safeUnmarshal(data, &races)
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
	err := safeUnmarshal(data, &classes)
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

func importSpecs() {
	var specsJSON *[]byte = getStatic(region, "playable-specialization/index")
	var specs []spec = parseSpecs(specsJSON)
	logger.Printf("Found %d specializations", len(specs))
	addSpecs(&specs)
}

func parseSpecs(data *[]byte) []spec {
	type CharacterSpecializationJSON struct {
		ID int
	}
	type SpecsJSON struct {
		CharacterSpecializations []CharacterSpecializationJSON `json:"character_specializations"`
	}
	var specsJSON SpecsJSON
	err := safeUnmarshal(data, &specsJSON)
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
		PlayableClass keyedValue `json:"playable_class"`
		Name          string
		Media         keyedValue
		Role          RoleJSON
		PvpTalents    []interface{} `json:"pvp_talents"`
		TalentTree    keyedValue    `json:"spec_talent_tree"`
	}
	var path string = fmt.Sprintf("playable-specialization/%d", specID)
	var icon = getIcon(region, path)
	var specJSON *[]byte = getStatic(region, path)
	var s SpecJSON
	safeUnmarshal(specJSON, &s)
	ch <- spec{
		s.ID,
		s.PlayableClass.ID,
		s.Name,
		s.Role.Role,
		icon}
}

func importTalents() {
	var talentsJSON *[]byte = getStatic(region, "talent/index")
	var talentIds []int = parseTalents(talentsJSON)
	logger.Printf("Found %d talent IDs", len(talentIds))
	var talents []talent = make([]talent, 0)
	var ch chan talent = make(chan talent, len(talentIds))
	for _, id := range talentIds {
		go getTalent(ch, id)
	}
	for range talentIds {
		talent := <-ch
		if talent.ID == 0 {
			continue
		}
		talents = append(talents, talent)
	}
	addTalents(&talents)
}

func parseTalents(data *[]byte) []int {
	var ids []int = make([]int, 0)
	type TalentsJSON struct {
		Talents []keyedValue
	}
	var talentsIDs TalentsJSON
	safeUnmarshal(data, &talentsIDs)
	for _, talent := range talentsIDs.Talents {
		ids = append(ids, talent.ID)
	}
	return ids
}

func getTalent(ch chan talent, id int) {
	ch <- parseTalentDetails(getStatic(region, fmt.Sprintf("talent/%d", id)))
}

func parseTalentDetails(data *[]byte) talent {
	type TalentJSON struct {
		ID                     int
		Spell                  keyedValue
		PlayableClass          keyedValue `json:"playable_class"`
		PlayableSpecialization keyedValue `json:"playable_specialization"`
	}
	var talentDetails TalentJSON
	safeUnmarshal(data, &talentDetails)
	var icon string
	if talentDetails.PlayableClass.ID == 0 || talentDetails.Spell.ID == 0 {
		return talent{0, 0, 0, 0, "", ""}
	} else {
		icon = getIcon(region, fmt.Sprintf("spell/%d", talentDetails.Spell.ID))
	}
	return talent{
		talentDetails.ID,
		talentDetails.Spell.ID,
		talentDetails.PlayableClass.ID,
		talentDetails.PlayableSpecialization.ID,
		talentDetails.Spell.Name,
		icon}
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
	err := safeUnmarshal(data, &pvpTalentsJSON)
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
	safeUnmarshal(pvpTalentJSON, &talentDetails)
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
	err := safeUnmarshal(data, &achievements)
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
	safeUnmarshal(pvpAchievementJSON, &pvpAchievementJSONDetails)
	icon := getIcon(region, fmt.Sprintf("achievement/%d", id))
	return achievement{
		id,
		pvpAchievementJSONDetails.Name,
		pvpAchievementJSONDetails.Description,
		icon}
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
