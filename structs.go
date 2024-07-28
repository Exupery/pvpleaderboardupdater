package main

/* Structs used across multiple layers */

// query : SQL query with optional args
type query struct {
	SQL        string
	Args       [][]interface{}
	Before     string
	BeforeArgs []interface{}
}

// realm : realm info
type realm struct {
	ID   int
	Slug string
	Name string
}

// race : playable race
type race struct {
	ID   int
	Name string
}

// faction : player faction
type faction struct {
	ID   int
	Name string
}

// class : player class
type class struct {
	ID   int
	Name string
}

// spec : class specialization
type spec struct {
	ID      int
	ClassID int
	Name    string
	Role    string
	Icon    string
}

// talent : talent info
type talent struct {
	ID        int
	SpellID   int
	ClassID   int
	SpecID    int
	Name      string
	Icon      string
	NodeID    int
	Row       int
	Col       int
	Cat       string
	HeroSpecs []int
}

// pvpTalent : PvP talent info
type pvpTalent struct {
	ID      int
	Name    string
	SpellID int
	SpecID  int
	Icon    string
}

// achievement : completed achievement info
type achievement struct {
	ID          int
	Title       string
	Description string
	Icon        string
}

// stats : player stat info
type stats struct {
	Strength       int32
	Agility        int32
	Intellect      int32
	Stamina        int32
	CriticalStrike int32
	Haste          int32
	Versatility    int32
	Mastery        int32
	Leech          int32
	Dodge          int32
	Parry          int32
}

// leaderboardEntry : a singular listing on a leaderboard
type leaderboardEntry struct {
	Name         string
	RealmID      int
	BlizzardID   int
	Rank         int
	Rating       int
	SeasonWins   int
	SeasonLosses int
}

// player : player info
type player struct {
	Name       string
	BlizzardID int
	RealmID    int
	ClassID    int
	SpecID     int
	FactionID  int
	RaceID     int
	Gender     int
	Guild      string
	Path       string
	LastLogin  int64
	ProfileID  string
}

// item : an equippable item
type item struct {
	ID      int
	Name    string
	Quality string
}

// items : a player's equipped items
type items struct {
	Head      item
	Neck      item
	Shoulder  item
	Back      item
	Chest     item
	Shirt     item
	Tabard    item
	Wrist     item
	Hands     item
	Waist     item
	Legs      item
	Feet      item
	Finger1   item
	Finger2   item
	Trinket1  item
	Trinket2  item
	MainHand  item
	OffHand   item
	Legendary item
}
