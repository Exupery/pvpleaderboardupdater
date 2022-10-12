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
	importSpecsAndTalents()
	importPvPTalents()
	importAchievements()
	importCovenants()
	importSoulbinds()
	importConduits()

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
	safeUnmarshal(specJSON, &s)
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
			safeUnmarshal(talentJSON, &talentDetails)
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

func importCovenants() {
	var covenantsJSON *[]byte = getStatic(region, "covenant/index")
	var covenants []covenant = parseCovenants(covenantsJSON)
	logger.Printf("Found %d covenants", len(covenants))
	addCovenants(&covenants)
}

func parseCovenants(data *[]byte) []covenant {
	type Covenants struct {
		Covenants []keyedValue
	}
	var covenantsJSON Covenants
	var covenants []covenant = make([]covenant, 0)
	err := safeUnmarshal(data, &covenantsJSON)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return covenants
	}
	for _, c := range covenantsJSON.Covenants {
		icon := getIcon(region, fmt.Sprintf("covenant/%d", c.ID))
		covenants = append(covenants, covenant{c.ID, c.Name, icon})
	}
	return covenants
}

func importSoulbinds() {
	var soulbindsJSON *[]byte = getStatic(region, "covenant/soulbind/index")
	var soulbinds []soulbind = parseSoulbinds(soulbindsJSON)
	logger.Printf("Found %d soulbinds", len(soulbinds))
	addSoulbinds(&soulbinds)
}

func parseSoulbinds(data *[]byte) []soulbind {
	type Soulbinds struct {
		Soulbinds []keyedValue
	}
	var soulbindsJSON Soulbinds
	var soulbinds []soulbind = make([]soulbind, 0)
	err := safeUnmarshal(data, &soulbindsJSON)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return soulbinds
	}
	for _, sb := range soulbindsJSON.Soulbinds {
		soulbinds = append(soulbinds, soulbind{sb.ID, sb.Name})
	}
	return soulbinds
}

func importConduits() {
	var conduitsJSON *[]byte = getStatic(region, "covenant/conduit/index")
	var conduits []conduit = parseConduits(conduitsJSON)
	logger.Printf("Found %d conduits", len(conduits))
	addConduits(&conduits)
}

func parseConduits(data *[]byte) []conduit {
	type Conduits struct {
		Conduits []keyedValue
	}
	var conduitsJSON Conduits
	var conduits []conduit = make([]conduit, 0)
	err := safeUnmarshal(data, &conduitsJSON)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return conduits
	}
	for _, c := range conduitsJSON.Conduits {
		spellID := getConduitSpellID(c.ID)
		conduits = append(conduits, conduit{c.ID, spellID, c.Name})
	}
	return conduits
}

func getConduitSpellID(conduitID int) int {
	type ConduitSpellToolTip struct {
		Spell keyedValue
	}
	type ConduitRank struct {
		ID           int
		Tier         int
		SpellTooltip ConduitSpellToolTip `json:"spell_tooltip"`
	}
	type Conduit struct {
		ID    int
		Name  string
		Ranks []ConduitRank
	}
	var conduitJSON Conduit
	var data *[]byte = getStatic(region, fmt.Sprintf("covenant/conduit/%d", conduitID))
	err := safeUnmarshal(data, &conduitJSON)
	if err != nil {
		logger.Printf("%s json parsing failed: %s", errPrefix, err)
		return 0
	}
	if conduitJSON.Ranks == nil || len(conduitJSON.Ranks) == 0 {
		return 0
	}
	return conduitJSON.Ranks[0].SpellTooltip.Spell.ID
}
