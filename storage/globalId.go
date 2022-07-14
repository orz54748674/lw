package storage

import (
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var (
	cGlobal   = "global"
	KeyUser   = "userId"
	KeyGameDx = "gameDx"
	KeyBotId  = "botId"
)

func NewGlobalId(t string) int64 {
	c := common.GetMongoDB().C(cGlobal)
	selector := bson.M{"t": t}
	update := bson.M{"$inc": bson.M{"curId": 1}}
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error(err.Error())
	}
	return int64(GetLastId(t))
}
func GetLastId(t string) int64 {
	c := common.GetMongoDB().C(cGlobal)
	selector := bson.M{"t": t}
	query := make(map[string]interface{})
	if err := c.Find(selector).One(query); err != nil { //初始化数据库
		log.Error("unInit mongo")
		return 0
	}
	lastUid, _ := utils.ConvertInt(query["curId"])
	return lastUid
}
func InitGlobal() {
	c := common.GetMongoDB().C(cGlobal)
	key := bsonx.Doc{{Key: "t", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetUnique(true)); err != nil {
		log.Error("create global id Index: %s", err)
	}
}
