package main

/* Structs used across multiple layers */

// Query : SQL query with optional args
type Query struct {
	SQL    string
	Args   [][]interface{}
	Before string
}

// Realm : realm info
type Realm struct {
	ID   int
	Slug string
	Name string
}

// Race : playable race
type Race struct {
	ID   int
	Name string
}

// Faction : player faction
type Faction struct {
	ID   int
	Name string
}

// Class : player class
type Class struct {
	ID   int
	Name string
}

// Spec : class specialization
type Spec struct {
	ID      int
	ClassID int
	Name    string
	Role    string
	Icon    string
	Talents []Talent
}

// Talent : talent info
type Talent struct {
	ID      int
	SpellID int
	ClassID int
	SpecID  int
	Name    string
	Icon    string
	Tier    int
	Column  int
}

// PvPTalent : PvP talent info
type PvPTalent struct {
	ID      int
	Name    string
	SpellID int
	SpecID  int
	Icon    string
}

// Achievement : completed achievement info
type Achievement struct {
	ID          int
	Title       string
	Description string
}

// Stats : player stat info
type Stats struct {
	Str               int
	Agi               int
	Int               int
	Sta               int
	Spr               int
	CritRating        int
	HasteRating       int
	AttackPower       int
	MasteryRating     float64
	MultistrikeRating float64
	Versatility       float64
	LeechRating       float64
	DodgeRating       float64
	ParryRating       float64
}

// LeaderboardEntry : a singular listing on a leaderboard
type LeaderboardEntry struct {
	Name         string
	RealmID      int
	BlizzardID   int
	Rank         int
	Rating       int
	SeasonWins   int
	SeasonLosses int
}

// Player : player info
type Player struct {
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
}

// Item : an equippable item
type Item struct {
	ID        int
	Name      string
	Icon      string
	Quality   int
	ItemLevel int
	Armor     int
	Context   string
}

// Items : a player's equipped items
type Items struct {
	AverageItemLevel         int
	AverageItemLevelEquipped int
	Head                     Item
	Neck                     Item
	Shoulder                 Item
	Back                     Item
	Chest                    Item
	Shirt                    Item
	Tabard                   Item
	Wrist                    Item
	Hands                    Item
	Waist                    Item
	Legs                     Item
	Feet                     Item
	Finger1                  Item
	Finger2                  Item
	Trinket1                 Item
	Trinket2                 Item
	MainHand                 Item
	OffHand                  Item
}
