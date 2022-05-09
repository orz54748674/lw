package lobbyStorage

import (
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/game"
)

type SumGrade struct { //累计战绩
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"` //uid
	GameType    game.Type          `bson:"GameType"`                 //累计赢
	Win         uint64             `bson:"Win"`                      //累计赢
	WinRank     uint64             `bson:"WinRank"`                  //累计赢 排行
	WinAndLost  int64              `bson:"WinAndLost"`               //累计输赢
	WinCount    uint64             `bson:"WinCount"`                 //累计获胜场次 一定大于0
	PlayCount   uint64             `bson:"PlayCount"`                //累次场次
	MaxWinCount uint64             `bson:"MaxWinCount"`              //最高连胜场次
	CurWinCount uint64             `bson:"CurWinCount"`              //当前连续获胜场次 用计算maxWinCount
	UpdateTime  time.Time          `bson:"UpdateAt"`
}

type LobbyBanner struct {
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	Uid        primitive.ObjectID `bson:"Uid"`
	UserName   string             `bson:"UserName"`
	GameType   string             `bson:"GameType"`
	WinType    string             `bson:"WinType"`
	Amount     int64              `bson:"Amount"`
	CreateTime time.Time          `bson:"CreateAt"`
}

var (
	WinTypeNormal  = "win"
	WinTypeJackpot = "winJackpot"
)

func NewSumGrade(id primitive.ObjectID, gameType game.Type) *SumGrade {
	sumGrade := &SumGrade{
		Oid:        id,
		GameType:   gameType,
		UpdateTime: utils.Now(),
	}
	return sumGrade
}
type Notice struct {
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	AdminId    int64             `bson:"AdminId"`
	Title      string             `bson:"Title"`
	Content    string             `bson:"Content"`
	CreateTime time.Time          `bson:"CreateAt"`
}

type GameType string //
const (
	Normal	GameType = "normal" //本地常规
	Slot	GameType = "slot" //老虎机
	Card	GameType = "card" //牌类
	Lottery	GameType = "lottery" //彩票
	Mini	GameType = "mini" //Mini
	Api		GameType = "Api" //Api
)
type LobbyGameLayout struct {
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	GameName   game.Type             `bson:"GameName"`
	GameType   GameType             `bson:"GameType"` //normal常规 slot lottery card mini Api
	SortType   int   			  `bson:"SortType"`
	LobbyPos   int   			  `bson:"LobbyPos"`
	IsHot	   int				  `bson:"IsHot"`   //1热门 0
	Status	   int                `bson:"Status"` //1 开启 0关闭
	IsNotAllowPlay int 			  `bson:"IsNotAllowPlay"` //是否不允许陪玩号玩 1 不允许 0 允许
	UpdateTime time.Time          `bson:"UpdateTime"`
}
type BubbleType string //
const (
	NormalActivity	BubbleType = "NormalActivity" //普通活动
	DayActivity	BubbleType = "DayActivity" //每日任务
	Mail	BubbleType = "Mail" //邮件
	VipActivity		BubbleType = "VipActivity" //
)
type LobbyBubble struct {
	Uid 	   string 			  `bson:"Uid",json:"Uid"`
	BubbleType BubbleType 		  `bson:"BubbleType",json:"BubbleType"`
	Num 	   int 		  		  `bson:"Num",json:"Num"`//数量
	UpdateAt   time.Time          `bson:"UpdateAt",json:"UpdateAt"`
}