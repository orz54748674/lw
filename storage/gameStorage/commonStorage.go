package gameStorage

import (
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var(
	cGameCommon = "gameCommon"
)
func InitGameCommon() {
	c := common.GetMongoDB().C(cGameCommon)
	key := bsonx.Doc{{Key: "Uid",Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key,options.Index().SetUnique(true));err != nil{
		log.Error("create gameCommon Index: %s",err)
	}
	log.Info("init gameCommon of mongo db")
}
func IncGameCommonData(uid string,vnd int64) {
	c := common.GetMongoDB().C(cGameCommon)
	query := bson.M{"Uid": uid}
	update := bson.M{
		"$inc": bson.M{"InRoomNeedVnd": vnd,
		},
	}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
}
func UpsertInRoomNeedVnd(uid string,vnd int64) {
	c := common.GetMongoDB().C(cGameCommon)
	query := bson.M{"Uid": uid}
	update := bson.M{
		"$set": bson.M{"InRoomNeedVnd": vnd,
		},
	}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
}
func QueryGameCommonData(uid string) *GameCommonData {
	c := common.GetMongoDB().C(cGameCommon)
	query := bson.M{"Uid": uid}
	var data GameCommonData
	if err := c.Find(query).One(&data); err != nil {
		return &GameCommonData{}
	}
	return &data
}
func RemoveAllGameCommonData() {
	c := common.GetMongoDB().C(cGameCommon)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}

