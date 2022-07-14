package gameStorage

import (
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/game"
)

var (
	cGameWinLoseRecord = "gameWinLoseRecord"
)

type WinLoseRecord struct {
	ID       int64              `bson:"-" json:"-"`
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	GameType game.Type          `bson:"GameType" json:"GameType"`
	NickName string             `bson:"NickName" json:"NickName"`
	Score    int64              `bson:"Score" json:"Score"`
}

func InitGameWinLoseRecord() {
	c := common.GetMongoDB().C(cGameWinLoseRecord)
	key := bsonx.Doc{{Key: "GameType", Value: bsonx.Int32(1)}, {Key: "NickName", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create game WinLoseRecord Index: %s", err)
	}
}
func IncGameWinLoseScore(gameType game.Type, nickName string, amount int64) {
	c := common.GetMongoDB().C(cGameWinLoseRecord)
	query := bson.M{"GameType": gameType, "NickName": nickName}
	update := bson.M{
		"$inc": bson.M{"Score": amount},
	}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
}
func GetGameWinLoseRank(gameType game.Type, limit int) []WinLoseRecord {
	c := common.GetMongoDB().C(cGameWinLoseRecord)
	var winLoseRecord []WinLoseRecord
	query := bson.M{"GameType": gameType}
	err := c.Find(query).Sort("-Score").Limit(limit).All(&winLoseRecord)
	if err != nil {
		return []WinLoseRecord{}
	}
	if winLoseRecord == nil {
		return []WinLoseRecord{}
	}
	return winLoseRecord
}
