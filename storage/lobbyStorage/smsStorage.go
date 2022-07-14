package lobbyStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

type Sms struct {
	Area     int64     `bson:"Area"`
	Phone    int64     `bson:"Phone"`
	Code     string    `bson:"Code"`
	Event    string    `bson:"Event"`
	CreateAt time.Time `bson:"CreateAt"`
}

var (
	cSms      = "smsCode"
	EventBind = "bind"
)

func InsertSms(sms *Sms) {
	c := common.GetMongoDB().C(cSms)
	if err := c.Insert(sms); err != nil {
		log.Error(err.Error())
	}
}

func QuerySms(area int64, phone int64, event string) *Sms {
	c := common.GetMongoDB().C(cSms)
	query := bson.M{"Area": area, "Phone": phone, "Event": event}
	var sms Sms
	if err := c.Find(query).One(&sms); err != nil {
		return nil
	}
	return &sms
}

func InitSms(smsExpire time.Duration) {
	c := common.GetMongoDB().C(cSms)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(int32(smsExpire/time.Second))); err != nil {
		log.Error("create Sms Index: %s", err)
	}
	log.Info("init token of mongo db")
}
