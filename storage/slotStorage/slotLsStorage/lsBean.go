package slotLsStorage

import "vn/framework/mongo-driver/bson/primitive"

type Symbol string

const (
	NINE     Symbol = "9"        //9
	TEN      Symbol = "10"       //
	J        Symbol = "J"        //
	Q        Symbol = "Q"        //
	K        Symbol = "K"        //
	A        Symbol = "A"        //
	PACKET   Symbol = "PACKET"   //
	TORTOISE Symbol = "TORTOISE" //
	FISH     Symbol = "FISH"     //
	LION     Symbol = "LION"     //
	PHOENIX  Symbol = "PHOENIX"  //
	JACKPOT  Symbol = "JACKPOT"  //
	WILD     Symbol = "WILD"     //
	SCATTER  Symbol = "SCATTER"  //
)

type RoomData struct {
	ID            int64                `bson:"-" json:"-"`
	Oid           primitive.ObjectID   `bson:"_id,omitempty" json:"Oid"`
	GoldJackpot   []int64              `bson:"GoldJackpot" json:"GoldJackpot"`     //金奖池
	SilverJackpot []int64              `bson:"SilverJackpot" json:"SilverJackpot"` //银奖池
	TablesInfo    map[string]TableInfo `bson:"TablesInfo" json:"TablesInfo"`       //桌子信息
}
type Conf struct {
	InitGoldJackpot      []int64 `bson:"InitGoldJackpot" json:"InitGoldJackpot"`           //初始奖池
	InitSilverJackpot    []int64 `bson:"InitSilverJackpot" json:"InitSilverJackpot"`       //
	PoolScaleThousand    int     `bson:"PoolScaleThousand" json:"PoolScaleThousand"`       //入奖千分比
	BotProfitPerThousand int     `bson:"BotProfitPerThousand" json:"BotProfitPerThousand"` //机器人抽水 8%
	FreeGameMinTimes     int     `bson:"FreeGameMinTimes" json:"FreeGameMinTimes"`         //
}
type TableInfo struct {
	TableID string `bson:"TableID" json:"TableID"`
}
