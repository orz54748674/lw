package slotStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/game"
)

var (
	cJackpotRecord = "SlotJackpotRecord"
)

const maxPageNum = 100

type JackpotRecord struct {
	ID          uint64                 `bson:"-" json:"-"`
	Oid         primitive.ObjectID     `bson:"_id,omitempty" json:"Oid"`
	CreateAt    time.Time              `bson:"CreateAt" json:"CreateAt"`
	Uid         string                 `bson:"Uid" json:"Uid"`
	NickName    string                 `bson:"NickName" json:"NickName"`
	BetV        int64                  `bson:"BetV" json:"BetV"`
	GetJackpot  int64                  `bson:"GetJackpot" json:"GetJackpot"`
	GameType    game.Type              `bson:"GameType" json:"GameType"`
	JackpotType string                 `bson:"JackpotType" json:"JackpotType"`
	GameId      string                 `bson:"GameId" json:"GameId"` //游戏期号，没有的传空
	DetailInfo  map[string]interface{} `bson:"DetailInfo" json:"DetailInfo"`
}

func InitJackpotRecord(incDataExpireDay time.Duration) {
	c := common.GetMongoDB().C(cJackpotRecord)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create JackpotRecord Index: %s", err)
	}
	//_ = common.GetMysql().AutoMigrate(&JackpotRecord{})
}
func InsertJackpotRecord(uid string, nickName string, betV int64, getJackpot int64, gameType game.Type, jackpotType string, gameId string, detailInfo map[string]interface{}) {
	jackpotRecord := JackpotRecord{
		Oid:         primitive.NewObjectID(),
		Uid:         uid,
		NickName:    nickName,
		BetV:        betV,
		GetJackpot:  getJackpot,
		GameType:    gameType,
		JackpotType: jackpotType,
		GameId:      gameId,
		DetailInfo:  detailInfo,
		CreateAt:    utils.Now(),
	}
	c := common.GetMongoDB().C(cJackpotRecord)
	if err := c.Insert(&jackpotRecord); err != nil {
		log.Error(err.Error())
	}
}
func QueryJackpotRecord(offset int, pageSize int, gameType game.Type) []JackpotRecord {
	c := common.GetMongoDB().C(cJackpotRecord)
	if pageSize != 0 {
		if offset/pageSize > maxPageNum {
			return []JackpotRecord{}
		}
	}
	var selector map[string]interface{}
	selector = bson.M{"GameType": gameType}
	var betRecord []JackpotRecord
	err := c.Find(selector).Sort("-CreateAt").Skip(offset).Limit(pageSize).All(&betRecord)
	if err != nil {
		return []JackpotRecord{}
	}
	if betRecord == nil {
		return []JackpotRecord{}
	}
	return betRecord
}
func QueryJackpotRecordTotal(gameType game.Type) int64 {
	c := common.GetMongoDB().C(cJackpotRecord)
	var selector map[string]interface{}
	selector = bson.M{"GameType": gameType}
	count, _ := c.Find(selector).Count()
	return count
}
