package main

type Query struct {
	Sql string
	Args [][]interface{}
	Before string
}

type Realm struct {
	Slug string
	Name string
	Battlegroup string
	Timezone string
	Type string
}

type Race struct {
	Id int
	Name string
	Side string
}

type Faction struct {
	Id int
	Name string
}

type Class struct {
	Id int
	Name string
}

type Spec struct {
	ClassId int
	Name string
	Role string
	Description string
	BackgroundImage string
	Icon string
}

type Talent struct {
	Id int
	ClassId int
	Name string
	Description string
	Icon string
	Tier int
	Column int
}

type Glyph struct {
	Glyph int
	ClassId int
	Name string
	Icon string
	Item int
	TypeId int
}

type Achievement struct {
	Id int
	Title string
	Points int
	Description string
	Icon string
}

type LeaderboardEntry struct {
	Name string
	Ranking int
	Rating int
	RealmId int
	RealmName string
	RealmSlug string
	RaceId int
	ClassId int
	SpecId int
	FactionId int
	GenderId int
	SeasonWins int
	SeasonLosses int
	WeeklyWins int
	WeeklyLosses int
}

type Player struct {
	Name string
	ClassId int
	SpecId int
	FactionId int
	RaceId int
	RealmSlug string
	Guild string
	Gender int
	AchievementIds []int
	AchievementPoints int
	HonorableKills int
}