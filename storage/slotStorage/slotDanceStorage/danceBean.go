package slotDanceStorage

import "vn/framework/mongo-driver/bson/primitive"

type Symbol string
const (
	D1     Symbol = "D1"        //
	D2     Symbol = "D2"        //
	D3     Symbol = "D3"        //
	D4     Symbol = "D4"        //
	D5     Symbol = "D5"        //
	D6     Symbol = "D6"        //
	D7     Symbol = "D7"        //
	D8     Symbol = "D8"        //
	WILD     Symbol = "WILD"     //
	SCATTER  Symbol = "SCATTER"  //
)

type RoomData struct {
	ID             int64                `bson:"-" json:"-"`
	Oid            primitive.ObjectID   `bson:"_id,omitempty" json:"Oid"`
	TablesInfo     map[string]TableInfo `bson:"TablesInfo" json:"TablesInfo"` //桌子信息
}
type Conf struct {
	BotProfitPerThousand int   `bson:"BotProfitPerThousand" json:"BotProfitPerThousand"` //机器人抽水 8%
	FreeGameMinTimes    int   `bson:"FreeGameMinTimes" json:"FreeGameMinTimes"` //
}
type TableInfo struct {
	TableID  string `bson:"TableID" json:"TableID"`
}