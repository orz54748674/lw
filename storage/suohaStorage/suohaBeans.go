package suohaStorage

import "vn/framework/mongo-driver/bson/primitive"

type Record struct {
	Result     int  `json:"result"` //开牌结果，闲赢：0，庄赢：1，和：2
	BankerPair bool `json:"bankerPair"`
	PlayerPair bool `json:"playerPair"`
	BankerDian int  `json:"bankerDian"`
	PlayerDian int  `json:"playerDian"`
}

type Process struct {
	ID              int
	ProcessName     string
	ProcessLastTime int
}

type PlayerInfo struct {
	Uid      string `json:"uid"`
	Head     string `json:"head"`
	NickName string `json:"nickName"`
	Seat     int    `json:"seat"`
	Golds    int64  `json:"golds"`
	Score    int64  `json:"score"`
	IsUp     bool   `json:"isUp"`   //是否弃牌
	IsReady  bool   `json:"isReady"` //是否准备
	IsAllIn  bool   `json:"isAllIn"`  //是否allin
}

type BetInfo struct {
	Pos      int   `json:"pos"`
	BetCount int64 `json:"betCount"`
}

type CardInfo struct {
	PlayerCards []int `json:"playerCards"`
	BankerCards []int `json:"bankerCards"`
	PlayerDian  int   `json:"playerDian"`
	BankerDian  int   `json:"bankerDian"`
}

type GameState struct {
	GameNo        string           `json:"gameNo"`
	State         string           `json:"state"`
	St            int64            `json:"st"`
	Et            int64            `json:"et"`
	Base          int64            `json:"base"` //底分
	PlayerInfos   []PlayerInfo     `json:"playerInfos"`
	ServerTime    int64            `json:"serverTime"`
	RemainTime    int64            `json:"remainTime"`
	Round         string           `json:"round"`  //当前轮次
	CurUid        string           `json:"curUid"` //当前操作玩家
	CurBet        int64            `json:"curBet"`  //当前下注金额
	Uid2HandCards map[string][]int `json:"uid2HandCards"` //玩家手牌
}

type BetDetail struct {
	UserID      string `json:"uid"`
	Pos         int    `json:"pos"`
	Coin        int64  `json:"coin"`
	PlayerGolds int64  `json:"playerGolds"`
}

type UserGameRecord struct {
	Uid        string `bson:"uid" json:"uid"`
	GameNo     string `bson:"gameNo" json:"gameNo"`
	Pos        int    `bson:"pos" json:"pos"`
	BetCount   int64  `bson:"betCount" json:"betCount"`
	Score      int64  `bson:"score" json:"score"`
	PlayerDian int    `bson:"playerDian" json:"playerDian"`
	BankerDian int    `bson:"bankerDian" json:"bankerDian"`
	CreateTime int64  `bson:"createTime" json:"createTime"`
}

type BaseInfo struct {
	Base     int64
	MinEnter int64
	MaxEnter int64
}

type Conf struct {
	BaseConf []BaseInfo `bson:"BaseConf" json:"BaseConf"`
}

type TableInfo struct {
	Oid           primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	BaseScore     int64              `bson:"BaseScore" json:"BaseScore"`
	CurPlayer     int                `bson:"CurPlayer" json:"CurPlayer"`
	IsCreateTable bool               `bson:"IsCreateTable" json:"IsCreateTable"`
	Password      int                `bson:"Password" json:"Password"`
	TableNo       int                `bson:"TableNo" json:"TableNo"`
}

type RankingInfo struct {
	Oid           primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Nickname      string             `bson:"Nickname" json:"Nickname"`
	TotalWinScore int64              `bson:"TotalWinScore" json:"TotalWinScore"`
}

type EnterConf struct {
	Base    int64
	MinTake int64
	MaxTake int64
}
