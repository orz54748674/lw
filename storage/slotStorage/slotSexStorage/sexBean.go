package slotSexStorage

import "vn/framework/mongo-driver/bson/primitive"

type Symbol string

const (
	S1 Symbol = "S1" //
	S2 Symbol = "S2" //
	S3 Symbol = "S3" //
	S4 Symbol = "S4" //
	S5 Symbol = "S5" //
	S6 Symbol = "S6" //
	S7 Symbol = "S7" //
	S8 Symbol = "S8" //

	JACKPOT Symbol = "JACKPOT" //
	WILD    Symbol = "WILD"    //
	SCATTER Symbol = "SCATTER" //
	BONUS   Symbol = "BONUS"   //
)

type RoomData struct {
	ID         int64                `bson:"-" json:"-"`
	Oid        primitive.ObjectID   `bson:"_id,omitempty" json:"Oid"`
	Jackpot    []int64              `bson:"Jackpot" json:"Jackpot"`       //金奖池
	TablesInfo map[string]TableInfo `bson:"TablesInfo" json:"TablesInfo"` //桌子信息
}
type Conf struct {
	InitJackpot          []int64 `bson:"InitJackpot" json:"InitJackpot"`             //初始奖池
	ProfitPerThousand    int     `bson:"ProfitPerThousand" json:"ProfitPerThousand"` //系统抽水 2%
	PoolScaleThousand    int     `bson:"PoolScaleThousand" json:"PoolScaleThousand"` //入奖千分比
	BonusTimeOut         int     `bson:"BonusTimeOut" json:"BonusTimeOut"`
	BotProfitPerThousand int     `bson:"BotProfitPerThousand" json:"BotProfitPerThousand"` //机器人抽水 8%
	FreeGameMinTimes     []int   `bson:"FreeGameMinTimes" json:"FreeGameMinTimes"`         //
	BonusGameMinTimes    []int   `bson:"BonusGameMinTimes" json:"BonusGameMinTimes"`       //
}
type TableInfo struct {
	TableID string `bson:"TableID" json:"TableID"`
}
