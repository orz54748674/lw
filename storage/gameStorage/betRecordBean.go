package gameStorage

import (
	"time"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/game"
)

const maxPageNum = 200 //查询最大页数
type BetRecord struct {
	ID          uint64             `bson:"-" json:"-"`
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Uid         string             `bson:"Uid" json:"Uid" gorm:"type:varchar(32);index"`
	UserType    int8               `bson:"UserType" json:"UserType" gorm:"index"`
	Channel     string             `bson:"Channel" json:"Channel" gorm:"type:varchar(32);index"`
	CreateAt    time.Time          `bson:"CreateAt" json:"CreateAt" gorm:"index"`
	GameType    game.Type          `bson:"GameType" json:"GameType"`
	GameId      string             `bson:"GameId" json:"GameId" gorm:"index"` //游戏ID
	GameNo      string             `bson:"GameNo" json:"GameNo"`              //游戏期号，没有的传空
	GameResult  string             `bson:"GameResult" json:"GameResult"`      //这一期的开奖结果，数据结构每个游戏自定义，可json,可 逗号分割
	Income      int64              `bson:"Income" json:"Income"`              //纯利润,不包含本金
	BetAmount   int64              `bson:"BetAmount" json:"BetAmount"`        //下注额
	BetDetails  string             `bson:"BetDetails" json:"BetDetails"`      //下注详情 每个游戏结构自己定义
	CurBalance  int64              `bson:"CurBalance" json:"CurBalance"`      //当前余额
	SysProfit   int64              `bson:"SysProfit" json:"SysProfit"`        //系统抽水
	BotProfit   int64              `bson:"BotProfit" json:"BotProfit"`        //机器人抽水
	AgentProfit int64              `bson:"AgentProfit" json:"AgentProfit"`    //代理收益
	Ip          string             `bson:"Ip" json:"Ip"`                      //
	UpdateAt    time.Time          `bson:"UpdateAt" json:"UpdateAt"`          //
	Status      int8               `bson:"Status" json:"Status"`              // 记录状态
}
