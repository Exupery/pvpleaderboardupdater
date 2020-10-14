package pvpleaderboardupdater

type Query struct {
	Sql        string
	Args       [][]interface{}
	Before     string
	BeforeArgs []interface{}
}

type Realm struct {
	Slug        string
	Name        string
	Battlegroup string
	Timezone    string
	Type        string
}

type Race struct {
	Id   int
	Name string
	Side string
}

type Faction struct {
	Id   int
	Name string
}

type Class struct {
	Id   int
	Name string
}

type Spec struct {
	Id              int
	ClassId         int
	Name            string
	Role            string
	Description     string
	BackgroundImage string
	Icon            string
}

type Talent struct {
	Id          int
	ClassId     int
	Name        string
	Description string
	Icon        string
	Tier        int
	Column      int
}

type Achievement struct {
	Id          int
	Title       string
	Points      int
	Description string
	Icon        string
}

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

type LeaderboardEntry struct {
	Name         string
	Ranking      int
	Rating       int
	RealmId      int
	RealmName    string
	RealmSlug    string
	RaceId       int
	ClassId      int
	SpecId       int
	FactionId    int
	GenderId     int
	SeasonWins   int
	SeasonLosses int
	WeeklyWins   int
	WeeklyLosses int
}

type Player struct {
	Name                  string
	ClassId               int
	SpecId                int
	FactionId             int
	RaceId                int
	RealmSlug             string
	Guild                 string
	Gender                int
	Stats                 Stats
	AchievementIds        []int
	AchievementTimestamps []int64
	TalentIds             []int
	AchievementPoints     int
	HonorableKills        int
	Items                 Items
}

type TooltipParam struct {
	Enchant int
}

type Item struct {
	Id            int
	Name          string
	Icon          string
	Quality       int
	ItemLevel     int
	TooltipParams TooltipParam
	Armor         int
	Context       string
}

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
