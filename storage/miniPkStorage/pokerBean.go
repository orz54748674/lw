package miniPkStorage

import (
	"time"
	"vn/framework/mongo-driver/bson/primitive"
)

type PkPlay struct {
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid" `       //
	Name        string             `bson:"Name" json:"Name"`                // 名字
	PrizeType   int8               `bson:"PrizeType" json:"PrizeType"`      //
	Odds        int64              `bson:"Odds" json:"Odds"`                //
	Example     []int8             `bson:"Example" json:"Example" gorm:"-"` // 中奖例子
	Description string             `bson:"Description" json:"Description"`  //
}

type PokerRecord struct {
	Oid              primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`      //
	Uid              string             `bson:"Uid" json:"Uid"`                //  用户id
	NickName         string             `bson:"NickName" json:"NickName"`      //	昵称
	Number           string             `bson:"Number" json:"Number"`          //
	BetAmount        int64              `bson:"BetAmount" json:"BetAmount"`    //
	Pokers           []int8             `bson:"Pokers" json:"Pokers" gorm:"-"` //
	Profit           int64              `bson:"Profit" json:"Profit"`          //
	Bonus            int64              `bson:"Bonus" json:"Bonus"`            //
	Pump             int64              `bson:"Pump" json:"Pump"`              //
	PumpAmount       int64              `bson:"PumpAmount" json:"PumpAmount"`  //  需要/100
	PrizeType        int8               `bson:"PrizeType" json:"PrizeType"`    //
	CreateAt         time.Time          `bson:"CreateAt" json:"CreateAt"`      //
	UpdateAt         time.Time          `bson:"UpdateAt" json:"UpdateAt"`      //
	transactionUnits []string           `bson:"-" json:"-" gorm:"-"`
}

type AutoIncr struct {
	Oid       primitive.ObjectID `bson:"_id,omitempty" json:"Oid"` //
	TableName string             `bson:"TableName" json:"TableName"`
	Field     string             `bson:"Field" json:"Field"`
	Value     int64              `bson:"Value" json:"Value"`
}
