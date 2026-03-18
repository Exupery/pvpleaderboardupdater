package main

import (
	"math"
	"testing"

	cmap "github.com/orcaman/concurrent-map/v2"
)

const testRegion = "US"
const testSeason = 37
const testPlayerPath = "emerald-dream/exuperjun"

func TestCreateToken(t *testing.T) {
	var token string = createToken()
	if len(token) == 0 {
		t.Error("Creating token failed")
	}
	t.Logf("Created token '%s'", token)
}

func TestGet(t *testing.T) {
	var resp *[]byte = getDynamic(testRegion, "token/")
	if len(*resp) == 0 {
		t.Error("No response from GET")
	}
}

func TestSafeUnmarshal(t *testing.T) {
	type EmptyInterface struct {
	}
	var empty EmptyInterface
	err := safeUnmarshal(nil, &empty)
	if err == nil {
		t.Error("Error should be returned when unmarshalling nil")
	}

	var realmJSON *[]byte = getDynamic(testRegion, "realm/index")
	realmSlice := (*realmJSON)[1:4]
	err = safeUnmarshal(&realmSlice, &empty)
	if err == nil {
		t.Error("Error should be returned when unmarshalling partial JSON")
	}
}

func TestParseRealms(t *testing.T) {
	var realmJSON *[]byte = getDynamic(testRegion, "realm/index")
	var realms []realm = parseRealms(realmJSON)

	if len(realms) == 0 {
		t.Error("Parsing realms failed")
	}
	t.Logf("Found and parsed %v realms", len(realms))
}

func TestParseRaces(t *testing.T) {
	var racesJSON *[]byte = getStatic(testRegion, "playable-race/index")
	var races []race = parseRaces(racesJSON)

	if len(races) == 0 {
		t.Error("Parsing races failed")
	}
	t.Logf("Found and parsed %v races", len(races))
}

func TestParseClasses(t *testing.T) {
	var classesJSON *[]byte = getStatic(testRegion, "playable-class/index")
	var classes []class = parseClasses(classesJSON)

	if len(classes) == 0 {
		t.Error("Parsing classes failed")
	}
	t.Logf("Found and parsed %v classes", len(classes))
}

func TestParseSpecs(t *testing.T) {
	var specsJSON *[]byte = getStatic(testRegion, "playable-specialization/index")
	var specs []spec = parseSpecs(specsJSON)

	if len(specs) == 0 {
		t.Error("Parsing specs failed")
	}
	t.Logf("Found and parsed %v specs", len(specs))

	for _, spec := range specs {
		if len(spec.Icon) == 0 {
			t.Error("Parsing specs failed")
		}
	}
}

func TestGetBestPaths(t *testing.T) {
	paths := map[string]string{
		"talent-tree/3333/playable-specialization/11": "whatever",
		"talent-tree/1111/playable-specialization/11": "whatever",
		"talent-tree/2222/playable-specialization/12": "whatever",
		"talent-tree/7777/playable-specialization/12": "whatever",
		"talent-tree/3333/playable-specialization/12": "whatever",
		"talent-tree/6666/playable-specialization/10": "whatever",
	}

	bestPaths := getBestPaths(paths)

	if len(bestPaths) != 4 {
		t.Errorf("Expected 4 best paths but found %v", len(bestPaths))
	}

	if bestPaths[0] != "talent-tree/3333/playable-specialization/11" {
		t.Errorf("Incorrect path for element 0: %v", bestPaths[0])
	}
	if bestPaths[1] != "talent-tree/6666/playable-specialization/10" {
		t.Errorf("Incorrect path for element 1: %v", bestPaths[1])
	}
	if bestPaths[2] != "talent-tree/7777/playable-specialization/12" {
		t.Errorf("Incorrect path for element 2: %v", bestPaths[2])
	}

	// Temporary hardcode of spriest due to Blizzard API bug
	if bestPaths[3] != "talent-tree/795/playable-specialization/258" {
		t.Errorf("Incorrect path for element 2: %v", bestPaths[3])
	}

	t.Logf("Found %v talent tree paths", len(bestPaths))
}

func TestTalentTreePaths(t *testing.T) {
	paths := getTalentTreePaths()

	if len(paths) == 0 {
		t.Error("Getting talent tree paths failed")
	}
	t.Logf("Found %v talent tree paths", len(paths))
}

func TestExtractTalentTreePath(t *testing.T) {
	var specCases = map[string]string{
		"https://a.b.c/d/e/talent-tree/781/playable-specialization/270?f": "talent-tree/781/playable-specialization/270",
		"foo/talent-tree/1234/playable-specialization/5678/bar":           "talent-tree/1234/playable-specialization/5678",
		"talent-tree/1/playable-specialization/2":                         "talent-tree/1/playable-specialization/2",
		"talent-tree/11/playable-specialization/22":                       "talent-tree/11/playable-specialization/22",
		"talent-tree/111/playable-specialization/222":                     "talent-tree/111/playable-specialization/222",
		"talent-tree/1/invalid/playable-specialization/2":                 "",
		"invalidtalent-tree/1/playable-specialization/2":                  "",
		"talent-tree/1/playable-specializationinvalid/2":                  "",
		"talent-tree/1/playable-specialization/invalid/2":                 "",
		"talent-tree/1/invalid/2":                                         "",
		"talent-tree/playable-specialization/2":                           "",
		"talent-tree/1/playable-specialization":                           "",
		"talent-tree/1/playable-specialization/invalid/":                  "",
	}

	for href, expected := range specCases {
		actual := parseSpecTalentTreePath(href)
		if actual != expected {
			t.Errorf("Returned '%s' for '%s' but expected '%s'", actual, href, expected)
		}
	}
}

func TestGetTalentsFromTree(t *testing.T) {
	path := "talent-tree/1000/playable-specialization/270"
	talents := getTalentsFromTree(path)

	if len(talents) == 0 {
		t.Error("Getting talents from talent tree failed")
	}
	t.Logf("Found and parsed %v talents", len(talents))
}

func TestParsePvPTalents(t *testing.T) {
	var talentsJSON *[]byte = getStatic(region, "pvp-talent/index")
	var pvpTalents []pvpTalent = parsePvPTalents(talentsJSON)

	if len(pvpTalents) == 0 {
		t.Error("Parsing PvP Talents failed")
	}
	t.Logf("Found and parsed %v PvP Talents", len(pvpTalents))
}

func TestParseAchievements(t *testing.T) {
	var achievementsJSON *[]byte = getStatic(testRegion, "achievement-category/15270")
	var achievements []achievement = parseAchievements(achievementsJSON)

	if len(achievements) == 0 {
		t.Error("Parsing achievements failed")
	}
	t.Logf("Found and parsed %v PvP achievements", len(achievements))
}

func TestGetCurrentSeason(t *testing.T) {
	var currentSeason = getCurrentSeason()

	if currentSeason == 0 {
		t.Error("Determining current season failed")
	}
	t.Logf("Current PvP season is %d", currentSeason)
}

func TestGetPrefixedLeaderboards(t *testing.T) {
	var soloLeaderboards = getPrefixedLeaderboards(testSeason, "shuffle")
	if len(soloLeaderboards) == 0 {
		t.Error("No Solo Shuffle Leaderboards Found")
	}

	t.Logf("Found %d Solo Shuffle Leaderboards", len(soloLeaderboards))

	var blitzLeaderboards = getPrefixedLeaderboards(testSeason, "blitz")
	if len(blitzLeaderboards) == 0 {
		t.Error("No Blitz Leaderboards Found")
	}

	t.Logf("Found %d Blitz Leaderboards", len(blitzLeaderboards))
}

func TestGetSpecIDFromLeaderboardName(t *testing.T) {
	var cases = map[string]int{
		"2v2":                         0,
		"rbg":                         0,
		"some-invalid-string":         0,
		"shuffle-deathknight-frost":   251,
		"shuffle-hunter-beastmastery": 253,
		"shuffle-warlock-affliction":  265,
		"shuffle-monk-mistweaver":     270,
		"blitz-evoker-devastation":    1467,
	}

	for name, expected := range cases {
		actual := getSpecIDFromLeaderboardName(name)
		if actual != expected {
			t.Errorf("Returned '%d' for '%s' but expected '%d'", actual, name, expected)
		}
	}
}

func TestGetLeaderboard(t *testing.T) {
	var leaderboard = getLeaderboard("2v2", testSeason)

	if len(leaderboard) == 0 {
		t.Error("Parsing current season failed")
	}
	t.Logf("Found %d players on leaderboard", len(leaderboard))
}

func TestGetPlayersFromLeaderboards(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	var a = getLeaderboard("2v2", testSeason)
	var b = getLeaderboard("3v3", testSeason)
	var players = getPlayersFromLeaderboards(map[string][]leaderboardEntry{"2v2": a, "3v3": b})

	if len(players) == 0 {
		t.Error("Getting leaderboard players failed")
	}
	t.Logf("Found %d players from leaderboards", len(players))
}

func TestSliceSplitting(t *testing.T) {
	max := 100
	slice := make([]*player, 0)
	for i := 0; i < max; i++ {
		slice = append(slice, &player{BlizzardID: i})
	}

	groups := split(slice, len(slice))
	validateSplitting(t, groups, 1, max)

	groups = split(slice, 10)
	validateSplitting(t, groups, len(slice)/10, 10)
}

func validateSplitting(t *testing.T, groups [][]*player, expectedNumGroups, maxGroupSize int) {
	if len(groups) != expectedNumGroups {
		t.Errorf("Returned %d groups, but expected %d", len(groups), expectedNumGroups)
	}
	for _, group := range groups {
		if len(group) > maxGroupSize {
			t.Errorf("Group has %d elements - should not exceed %d", len(group), maxGroupSize)
		}
	}
}

func TestGetPlayerProfileDetails(t *testing.T) {
	player := player{Path: testPlayerPath}
	setPlayerDetails(&player)

	if player.ClassID == 0 {
		t.Error("Player class NOT set")
	}
	if player.SpecID == 0 {
		t.Error("Player spec NOT set")
	}
	if player.FactionID == 0 {
		t.Error("Player faction NOT set")
	}
	if player.RaceID == 0 {
		t.Error("Player race NOT set")
	}
	if player.LastLogin == 0 {
		t.Error("Last login time NOT set")
	}
	t.Logf("Last login %d", player.LastLogin)
}

func TestGetPlayerTalents(t *testing.T) {
	talents := getPlayerTalents(testPlayerPath)
	if len(talents.Talents) == 0 || len(talents.PvPTalents) == 0 {
		t.Error("Getting player talents failed")
	}
	t.Logf("Found %d talents and %d PvP talents", len(talents.Talents), len(talents.PvPTalents))
}

func TestGetPlayerStats(t *testing.T) {
	stats := getPlayerStats(testPlayerPath)
	if stats.Intellect == 0 || stats.Stamina == 0 {
		t.Error("Getting player stats failed")
	}
	t.Logf("Found stats: %v", stats)
}

func TestHighestStat(t *testing.T) {
	a := highestStat(3, 2, 1)
	if a != 3 {
		t.Errorf("Expected 3 as highest stat, not %d", a)
	}
	b := highestStat(2, 3, 1)
	if b != 3 {
		t.Errorf("Expected 3 as highest stat, not %d", b)
	}
	c := highestStat(1, 2, 3)
	if c != 3 {
		t.Errorf("Expected 3 as highest stat, not %d", c)
	}
	ab := highestStat(3, 3, 1)
	if ab != 3 {
		t.Errorf("Expected 3 as highest stat, not %d", ab)
	}
	max := math.MaxInt32
	z := highestStat(float64(max), float64(max), float64(max))
	if z != max {
		t.Errorf("Expected %d as highest stat, not %d", max, z)
	}
}

func TestGetPlayerItems(t *testing.T) {
	items := getPlayerItems(testPlayerPath)
	if items.Back.ID == 0 || items.Shoulder.ID == 0 {
		t.Error("Getting player items failed")
	}
	t.Logf("Found items: %v", items)
}

func TestSquashPlayerItems(t *testing.T) {
	itemsA := items{Neck: item{178927, "Clouded Focus", "RARE"}, Shoulder: item{}}
	itemsB := items{Neck: item{1, "Foo", "EPIC"}, Shoulder: item{2, "Bar", "EPIC"}}
	itemsC := items{Neck: item{1, "Foo", "EPIC"}, Shoulder: item{3, "Baz", "EPIC"}}

	playersItems := cmap.New[items]()
	playersItems.Set("47", itemsA)
	playersItems.Set("1138", itemsB)
	playersItems.Set("1701", itemsC)
	squashedItems := squashItems(&playersItems)
	seen := make(map[string]bool, 0)

	for id, item := range squashedItems {
		if id == 0 {
			t.Error("Invalid item present after squashing")
		}
		if seen[item.Name] {
			t.Error("Duplicate items not squished")
		}
		seen[item.Name] = true
	}

	t.Logf("Squashed items: %v", squashedItems)
}

func TestGetPlayerAchievements(t *testing.T) {
	achieved := getPlayerAchievements(testPlayerPath, map[int]bool{2092: true, 13989: true})
	if len(achieved) == 0 {
		t.Error("Getting player achievements failed")
	}
	t.Logf("Found achievements: %v", achieved)
}

func TestDetermineAlt(t *testing.T) {
	var altPlayerPath = "emerald-dream/exupery"
	altID := getProfileIdentifier(altPlayerPath)
	mainID := getProfileIdentifier(testPlayerPath)
	if mainID == "" {
		t.Error("Unable to generate valid ID")
	}
	if mainID != altID {
		t.Error("IDs do NOT match for main and alt characters")
	}
	t.Logf("Main ID: %s", mainID)
	t.Logf("Alt ID: %s", altID)
}

func makeTalent(id int) loadoutTalent {
	return loadoutTalent{Tooltip: talentTooltip{Talent: keyedValue{ID: id}}}
}

func talentIDsFromSlice(talents []loadoutTalent) []int {
	ids := make([]int, len(talents))
	for i, t := range talents {
		ids[i] = t.Tooltip.Talent.ID
	}
	return ids
}

func containsID(talents []loadoutTalent, id int) bool {
	for _, t := range talents {
		if t.Tooltip.Talent.ID == id {
			return true
		}
	}
	return false
}

func TestStripHero(t *testing.T) {
	origHeroTalentIds := heroTalentIds
	defer func() { heroTalentIds = origHeroTalentIds }()

	heroTalentIds = map[int]string{
		100: "{256,258}", // Voidweaver keystone
		101: "{256,258}", // Voidweaver talent
		200: "{257,258}", // Archon keystone
		201: "{257,258}", // Archon talent
	}

	talents := []loadoutTalent{
		makeTalent(1),   // class talent
		makeTalent(2),   // class talent
		makeTalent(100), // hero talent (should be stripped)
		makeTalent(200), // hero talent (should be stripped)
	}

	result := stripHero(talents)
	if len(result) != 2 {
		t.Fatalf("Expected 2 talents after stripHero, got %d", len(result))
	}
	if containsID(result, 100) || containsID(result, 200) {
		t.Error("Hero talents should have been stripped")
	}
	if !containsID(result, 1) || !containsID(result, 2) {
		t.Error("Non-hero talents should be preserved")
	}
}

func TestStripOtherHeroTree(t *testing.T) {
	origHeroTalentIds := heroTalentIds
	defer func() { heroTalentIds = origHeroTalentIds }()

	heroTalentIds = map[int]string{
		100: "{256,258}", // Voidweaver keystone
		101: "{256,258}",
		102: "{256,258}",
		200: "{257,258}", // Archon keystone
		201: "{257,258}",
	}

	// Player chose Voidweaver (3 talents) but API also returned Archon keystone
	heroTalents := []loadoutTalent{
		makeTalent(100), // VW keystone
		makeTalent(101), // VW talent
		makeTalent(102), // VW talent
		makeTalent(200), // AR keystone (should be stripped)
	}

	result := stripOtherHeroTree(heroTalents)
	if len(result) != 3 {
		t.Fatalf("Expected 3 talents after stripOtherHeroTree, got %d", len(result))
	}
	if containsID(result, 200) {
		t.Error("Archon keystone should have been stripped")
	}
	if !containsID(result, 100) || !containsID(result, 101) || !containsID(result, 102) {
		t.Error("Voidweaver talents should be preserved")
	}
}

func TestStripOtherHeroTreeSingleTree(t *testing.T) {
	origHeroTalentIds := heroTalentIds
	defer func() { heroTalentIds = origHeroTalentIds }()

	heroTalentIds = map[int]string{
		100: "{256,258}",
		101: "{256,258}",
	}

	// Normal case: all talents from one tree, should be returned unchanged
	heroTalents := []loadoutTalent{
		makeTalent(100),
		makeTalent(101),
	}

	result := stripOtherHeroTree(heroTalents)
	if len(result) != 2 {
		t.Fatalf("Expected 2 talents for single-tree case, got %d", len(result))
	}
}

func TestClassTalentsHeroKeystonesStripped(t *testing.T) {
	origHeroTalentIds := heroTalentIds
	defer func() { heroTalentIds = origHeroTalentIds }()

	// Simulate real Priest data:
	// Voidweaver talents (hero_specs = Disc + Shadow)
	// Archon talents (hero_specs = Holy + Shadow)
	heroTalentIds = map[int]string{
		122299: "{256,258}", // Void Torrent (Voidweaver keystone)
		122283: "{256,258}", // Devour Matter (Voidweaver)
		122284: "{256,258}", // Void Infusion (Voidweaver)
		122312: "{257,258}", // Halo (Archon keystone)
		122291: "{257,258}", // Resonant Energy (Archon)
	}

	// Simulate a Voidweaver player's API response:
	// class_talents includes the unchosen Archon keystone (Halo)
	classTalents := []loadoutTalent{
		makeTalent(1),      // regular class talent
		makeTalent(2),      // regular class talent
		makeTalent(122312), // Halo - leaked into class talents by Blizzard API
	}

	// spec_talents may also have hero talents
	specTalents := []loadoutTalent{
		makeTalent(10), // spec talent
		makeTalent(11), // spec talent
	}

	// hero_talents only from chosen tree (Voidweaver)
	heroTalents := []loadoutTalent{
		makeTalent(122299), // Void Torrent (VW keystone)
		makeTalent(122283), // Devour Matter
		makeTalent(122284), // Void Infusion
	}

	// Apply the same processing as getPlayerTalents
	classTalents = stripHero(classTalents)
	specTalents = stripHero(specTalents)
	heroTalents = stripOtherHeroTree(heroTalents)

	allTalents := make([]int, 0)
	allTalents = append(allTalents, talentIDsFromSlice(classTalents)...)
	allTalents = append(allTalents, talentIDsFromSlice(specTalents)...)
	allTalents = append(allTalents, talentIDsFromSlice(heroTalents)...)

	// Halo (122312) should NOT be in the final talent list
	for _, id := range allTalents {
		if id == 122312 {
			t.Error("Halo (unchosen Archon keystone) should not be in final talents - it leaked through class talents")
		}
	}

	// Void Torrent (122299) SHOULD be in the final talent list (from hero talents)
	found := false
	for _, id := range allTalents {
		if id == 122299 {
			found = true
		}
	}
	if !found {
		t.Error("Void Torrent (chosen Voidweaver keystone) should be in final talents")
	}

	// Regular class/spec talents should remain
	if len(classTalents) != 2 {
		t.Errorf("Expected 2 class talents after stripping, got %d", len(classTalents))
	}
	if len(specTalents) != 2 {
		t.Errorf("Expected 2 spec talents after stripping, got %d", len(specTalents))
	}
}
