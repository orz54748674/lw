package gbsStorage

type GameState struct {
	Round int `json:"round"`
	CurGolds int64 `json:"curGolds"`
	CurCard int `json:"curCard"`
	ChipList []int64 `json:"chipList"`
	RemainTime int `json:"remainTime"`
	BiggerReward int64 `json:"biggerReward"`
	SmallerReward int64 `json:"smallerReward"`
	AList []int `json:"aList"`
	BWin bool `json:"bWin"`
	SelectChip int64 `json:"selectChip"` //选择的筹码类型
	ShowCards []int `json:"showCards"`   //出现过的牌
	StartTime int64 `json:"startTime"`
	EndTime int64 `json:"endTime"`
	CurTime int64 `json:"curTime"`
}

type GameConf struct {
	Chip int64 `bson:"chip" json:"chip"`
	PoolVal int64 `bson:"poolVal" json:"poolVal"`
	MingPercent int64 `bson:"mingPercent" json:"mingPercent"`
	AnPercent int64 `bson:"anPercent" json:"anPercent"`
}

type GbsRecord struct {
	Uid string `bson:"uid" json:"uid"`
	GameNo string `bson:"gameNo" json:"gameNo"`
	CreateTime int64 `bson:"createTime" json:"createTime"`
	SelectChip int64 `bson:"selectChip" json:"selectChip"`
	Score int64	`bson:"score" json:"score"`
	BWin bool `bson:"bWin" json:"bWin"`
}

type GbsPoolRewardRecord struct {
	CreateTime int64 `bson:"createTime" json:"createTime"`
	Nickname string `bson:"nickname" json:"nickname"`
	SelectChip int64	`bson:"selectChip" json:"selectChip"`
	Reward int64	`bson:"reward" json:"reward"`
}
