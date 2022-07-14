package gameStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/game"
)

var (
	cGameInviteRecord = "gameInviteRecord"
)

type GameInviteRecord struct {
	ID              int64              `bson:"-" json:"-"`
	Oid             primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	GameType        game.Type          `bson:"GameType" json:"GameType"`
	GameName        string             `bson:"GameName" json:"GameName"`
	InvitorNickName string             `bson:"InvitorNickName" json:"InvitorNickName"` //邀请人昵称
	BeInvitedUid    string             `bson:"BeInvitedUid" json:"BeInvitedUid"`       //谁被邀请
	RoomId          string             `bson:"RoomId" json:"RoomId"`                   //房间ID
	BaseScore       int64              `bson:"BaseScore" json:"BaseScore"`             //底分
	ServerId        string             `bson:"ServerId" json:"ServerId"`               //底分
	UpdateAt        time.Time          `bson:"UpdateAt" json:"UpdateAt"`               //
}

func InitGameInviteRecord() {
	c := common.GetMongoDB().C(cGameInviteRecord)
	key := bsonx.Doc{{Key: "UpdateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(300)); err != nil {
		log.Error("create gameInviteRecord Index: %s", err)
	}

	key = bsonx.Doc{{Key: "BeInvitedUid", Value: bsonx.Int32(1)}, {Key: "GameType", Value: bsonx.Int32(1)}, {Key: "RoomId", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create gameInviteRecord Index: %s", err)
	}
	key = bsonx.Doc{{Key: "GameType", Value: bsonx.Int32(1)}, {Key: "RoomId", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create gameInviteRecord Index: %s", err)
	}
}
func UpsertGameInviteRecord(record GameInviteRecord) {
	c := common.GetMongoDB().C(cGameInviteRecord)
	query := bson.M{"BeInvitedUid": record.BeInvitedUid, "GameType": record.GameType, "RoomId": record.RoomId}
	if _, err := c.Upsert(query, record); err != nil {
		log.Error(err.Error())
	}
}
func QueryGameInviteRecord(uid string) []GameInviteRecord {
	c := common.GetMongoDB().C(cGameInviteRecord)
	query := bson.M{"BeInvitedUid": uid}
	res := make([]GameInviteRecord, 0)
	if err := c.Find(query).Sort("-UpdateAt").All(&res); err != nil {

	}
	return res
}
func RemoveGameInviteRecord(uid string, roomId string) {
	c := common.GetMongoDB().C(cGameInviteRecord)
	query := bson.M{"BeInvitedUid": uid, "RoomId": roomId}
	if err := c.Remove(query); err != nil {
		log.Info(err.Error())
	}
}
func RemoveAllGameInviteRecord() {
	c := common.GetMongoDB().C(cGameInviteRecord)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}
