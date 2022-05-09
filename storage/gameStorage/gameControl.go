package gameStorage

import (
	"context"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/game"
)

type GameReboot struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty",json:"Oid"`
	GameType game.Type          `bson:"GameType"`
	Reboot   string             `bson:"Reboot"，json:"Reboot"` //“true” 停服
}

var (
	cGameReboot = "gameReboot"
)

func InitGameReboot() {
	c := common.GetMongoDB().C(cGameReboot)
	key := bsonx.Doc{{Key: "GameType", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetUnique(true)); err != nil {
		log.Error("create gameReconnect Index: %s", err)
	}
	update := bson.M{"$set": bson.M{"Reboot": "false"}}
	c.UpdateMany(context.Background(), bson.M{}, update)
	log.Info("init gameReconnect of mongo db")
}
func UpsertGameReboot(gameType game.Type, reboot string) {
	c := common.GetMongoDB().C(cGameReboot)
	query := bson.M{"GameType": gameType}
	update := bson.M{"GameType": gameType, "Reboot": reboot}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
}

//当服务器要重启时，开奖需要等待时间的游戏，应在当局牌结束后停止发牌
func QueryGameReboot(gameType game.Type) string {
	c := common.GetMongoDB().C(cGameReboot)
	queryAll := bson.M{"GameType": "all"}
	var gameRebootAll GameReboot
	if err := c.Find(queryAll).One(&gameRebootAll); err != nil {
		//log.Error(err.Error())
	}
	if gameRebootAll.Reboot == "true" {
		return "true"
	}
	query := bson.M{"GameType": gameType}
	var gameReboot GameReboot
	if err := c.Find(query).One(&gameReboot); err != nil {
		log.Error(err.Error())
	}
	return gameReboot.Reboot
}
