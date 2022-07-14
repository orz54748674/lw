package activityStorage

import (
	"time"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/game"
)

type ActivityStatus string

var (
	Undo     ActivityStatus = "Undo"     //未完成
	Done     ActivityStatus = "Done"     //已完成
	Received ActivityStatus = "Received" //已领取
	Close    ActivityStatus = "Close"    //活动关闭
)

type ActivityType string

var (
	FirstCharge   ActivityType = "FirstCharge"   //首充
	BindPhone     ActivityType = "BindPhone"     //绑手机
	GiftCode      ActivityType = "GiftCode"      //gift code
	TotalCharge   ActivityType = "TotalCharge"   //累计充值
	SignIn        ActivityType = "SignIn"        //签到
	Encouragement ActivityType = "Encouragement" //鼓励金
	DayCharge     ActivityType = "DayCharge"     //每日任务充值类
	DayGame       ActivityType = "DayGame"       //每日任务游戏类
	DayInvite     ActivityType = "DayInvite"     //每日任务邀请类
	Vip           ActivityType = "Vip"           //VIP活动
	VipWeek       ActivityType = "VipWeek"       //VIP每周彩金
	VipChargeGet  ActivityType = "VipChargeGet"  //VIP充值优惠
	TurnTable     ActivityType = "TurnTable"     //转盘
)

var ActivityNormalList = []ActivityType{
	BindPhone,
	TotalCharge,
	SignIn,
	Encouragement,
}
var ActivityDayList = []ActivityType{
	DayCharge,
	DayGame,
	DayInvite,
}
var ActivityVipList = []ActivityType{
	Vip,
}

type ActivityGetType string

var (
	Gold   ActivityGetType = "Gold"   //赠送金币
	Points ActivityGetType = "Points" //赠送积分
)

//type GameStatus string
//var(
//	InBet   		GameStatus = "InBet"//下注中
//	InCheckout		GameStatus = "InCheckout"//结算
//)
type GameDataInBet struct {
	Uid      string    `bson:"Uid",json:"Uid"`
	GameType game.Type `bson:"GameType",json:"GameType"`
	BetCnt   int       `bson:"BetCnt"` //
	CreateAt time.Time `bson:"CreateAt"`
}
type ActivityConf struct {
	Oid          primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	ActivityType ActivityType       `bson:"ActivityType"`
	Status       int                `bson:"Status"` //1 开启 0关闭
	CreateAt     time.Time          `bson:"CreateAt"`
	UpdateAt     time.Time          `bson:"UpdateAt"`
}
type ActivityRecord struct {
	ID         int64              `bson:"-" json:"-"`
	Oid        primitive.ObjectID `bson:"_id,omitempty",json:"Oid"`
	Type       ActivityType       `bson:"Type"，json:"Type"`
	ActivityID string             `bson:"ActivityID",json:"ActivityID"`
	Uid        string             `bson:"Uid",json:"Uid"`
	Charge     int64              `bson:"Charge",json:"Charge"`
	Get        int64              `bson:"Get",json:"Get"`
	GetPoints  float64            `bson:"GetPoints",json:"GetPoints"`
	BetTimes   int64              `bson:"BetTimes",json:"BetTimes"`
	UpdateAt   time.Time          `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time          `bson:"CreateAt",json:"CreateAt" gorm:"index"`
}
