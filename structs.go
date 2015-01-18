package main

type Realm struct {
	Slug string
	Name string
	Battlegroup string
	Timezone string
	Type string
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