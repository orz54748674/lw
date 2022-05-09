package rsbStorage

type RsbRecord struct {
	Uid        string    `bson:"uid" json:"uid"`   //玩家uid
	GameNo     string    `bson:"gameNo" json:"gameNo"` //游戏编号
	WinPos     int       `bson:"winPos" json:"winPos"` //
	BetCount   int64     `bson:"betCount" json:"betCount"`
	Score      int64     `bson:"score" json:"score"`
	Res        []int     `bson:"res" json:"res"`
	UpdateTime int64 `bson:"updateTime" json:"updateTime"` //插入时间
}

type OpenRes struct {
	Uid string `bson:"uid" json:"uid"`
	WinPos int `json:"winPos" json:"winPos"`
	UpdateTime int64 `bson:"updateTime" json:"updateTime"` //插入时间
}

type RsbConf struct {
	AnPercent int64 `bson:"anPercent" json:"anPercent"`
	MingPercent int64 `bson:"mingPercent" json:"mingPercent"`
	InitBalance int64 `bson:"initBalance" json:"initBalance"`
}
