package gameStorage

import (
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

type GameReconnect struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty",json:"Oid"`
	Uid      string             `bson:"Uid"`
	ServerID string             `bson:"ServerID"，json:"ServerID"` //
}

var (
	cGameReconnect = "gameReconnect"
)

func InitGameReconnect() {
	c := common.GetMongoDB().C(cGameReconnect)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetUnique(true)); err != nil {
		log.Error("create gameReconnect Index: %s", err)
	}
	log.Info("init gameReconnect of mongo db")
}
func UpsertGameReconnect(uid string, serverID string) {
	c := common.GetMongoDB().C(cGameReconnect)
	query := bson.M{"Uid": uid}
	update := bson.M{"ServerID": serverID}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
}
func QueryGameReconnect(uid string) string {
	c := common.GetMongoDB().C(cGameReconnect)
	query := bson.M{"Uid": uid}
	var gameReconnect GameReconnect
	if err := c.Find(query).One(&gameReconnect); err != nil {
		return ""
	}
	return gameReconnect.ServerID
}
func RemoveAllReconnect() {
	c := common.GetMongoDB().C(cGameReconnect)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}
func RemoveReconnectByUid(uid string) (err error) {
	c := common.GetMongoDB().C(cGameReconnect)
	selector := bson.M{"Uid": uid}
	err = c.Remove(selector)
	if err != nil {
		//log.Error(err.Error())
	}
	return err
}
