package dxStorage

import (
	"encoding/json"
	"time"
	"vn/framework/mongo-driver/bson/primitive"
)

type Notify struct {
	BetSmall      int64     `bson:"BetSmall" json:"BetSmall"`
	BetBig        int64     `bson:"BetBig" json:"BetBig"`
	BetBigCount   int64     `bson:"BetBigCount" json:"BetBigCount"`
	BetSmallCount int64     `bson:"BetSmallCount" json:"BetSmallCount"`
	RefundBig     int64     `bson:"RefundBig"`
	RefundSmall   int64     `bson:"RefundSmall"`
	Jackpot       int64     `bson:"Jackpot" json:"Jackpot"`
	Result        uint8     `bson:"Result" json:"Result"`
	ResultJackpot uint8     `bson:"ResultJackpot" json:"ResultJackpot"` //0 没中，1 中了
	Dice1         uint8     `bson:"Dice1" json:"Dice1"`
	Dice2         uint8     `bson:"Dice2" json:"Dice2"`
	Dice3         uint8     `bson:"Dice3" json:"Dice3"`
	ShowId        int64     `bson:"ShowId"`
	CreateAt      time.Time `bson:"CreateAt"`
}

type Dx struct {
	Notify            `bson:"notify"`
	ID                int64              `bson:"-" json:"-"`
	Oid               primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	RealBetSmall      int64              `bson:"RealBetSmall"`
	RealBetBig        int64              `bson:"RealBetBig"`
	RealBetBigCount   int64              `bson:"RealBetBigCount"`
	RealBetSmallCount int64              `bson:"RealBetSmallCount"`
	RealRefundBig     int64              `bson:"RealRefundBig"`
	RealRefundSmall   int64              `bson:"RealRefundSmall"`
	SystemWin         int64              `bson:"SystemWin"`    //系统输赢
	BotProfit         int64              `bson:"BotProfit"`    //机器人抽水
	SystemProfit      int64              `bson:"SystemProfit"` //系统抽水
	AgentProfit       int64              `bson:"AgentProfit"`  //代理抽水
}

func (Dx) TableName() string {
	return "dx"
}

//func (s *Dx) String() string {
//	b, _ := json.Marshal(s)
//	return string(b)
//}

var (
	ResultBig      uint8 = 2
	ResultSmall    uint8 = 1
	cGameDx              = "dx"
	cGameDxBetLog        = "dxBetLog"
	cGameDxConf          = "dxConf"
	UserTypeNormal       = "user"
	UserTypeBot          = "bot"
	//UserTypeAgent          = "agent"
)

type BetPositionType string
type DxBetLog struct {
	ID          uint64             `bson:"-" json:"-"`
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Uid         string             `bson:"Uid"`
	NickName    string             `bson:"NickName"`
	GameId      int64              `bson:"GameId"`
	Big         int64              `bson:"Big"`
	Small       int64              `bson:"Small"`
	CurBig      int64              `bson:"CurBig"`
	CurSmall    int64              `bson:"CurSmall"`
	UserType    string             `bson:"UserType"`
	MoneyType   string             `bson:"MoneyType"`
	Result      int64              `bson:"Result"`
	Refund      int64              `bson:"Refund"`
	HasCheckout int                `bson:"HasCheckout"` //0 未结算，1 已结算
	CreateAt    time.Time          `bson:"CreateAt"`
}

func (DxBetLog) TableName() string {
	return "dx_bet_log"
}

func (s *DxBetLog) String() string {
	b, _ := json.Marshal(s)
	return string(b)
}

//func BetLog(uid string, money int64, gameShowId int64, big int64,small int64 ) *DxBetLog {
//
//	IncDxBetLog(dxBetLog)
//	return dxBetLog
//}

type Conf struct {
	ProfitPerThousand       int   `bson:"ProfitPerThousand"`    //系统抽水 2%
	BotProfitPerThousand    int   `bson:"BotProfitPerThousand"` //机器人抽水 8%
	JackpotPerThousand      int   `bson:"JackpotPerThousand"`   //奖池抽下注金额的0.3%
	ResultMax               int   `bson:"ResultMax"`
	ResultMini              int   `bson:"ResultMini"`
	MaxMiniDifference       int   `bson:"MaxMiniDifference"`
	ChipList                []int `bson:"ChipList"`
	BotBetChip              []int `bson:"BotBetChip"`
	BotFreedomPersonPercent int   `bson:"BotFreedomPersonPercent"`
}

func newDxConf() *Conf {
	conf := &Conf{
		ProfitPerThousand:    20,
		BotProfitPerThousand: 80,
		JackpotPerThousand:   3,
		ResultMini:           50000000,
		ResultMax:            500000000,
		MaxMiniDifference:    500000000,
		//ResultMini:              10000,
		//ResultMax:               90000,
		//MaxMiniDifference:       100000,
		ChipList:                []int{1000, 10000, 50000, 100000, 500000, 1000000, 10000000, 50000000},
		BotBetChip:              []int{1000, 10000, 50000, 100000, 500000, 1000000},
		BotFreedomPersonPercent: 15,
	}
	return conf
}
