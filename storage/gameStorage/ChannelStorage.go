package gameStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
)

type Channel struct {
	ID       uint64             `bson:"-" json:"-"`
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Name     string             `bson:"Name"`
	Channel  string             `bson:"Channel"`
	UpdateAt time.Time          `bson:"UpdateAt"`
}

var (
	cChannelManager = "ChannelManager"
)

func InitChannel() {
	//c := common.GetMgo().C(cChannelManager)
	//index := mgo.Index{
	//	Key:        []string{"Oid"},
	//	Unique:     true,
	//	DropDups:   true,
	//	Background: true, // See notes.
	//	Sparse:     true,
	//}
	//if err := c.EnsureIndex(index); err != nil {
	//	log.Error("create Channel err: %s", err)
	//}
	log.Info("init Channel of mongo db")
}
func QueryChannels() []Channel {
	c := common.GetMongoDB().C(cChannelManager)
	var channel []Channel
	err := c.Find(nil).All(&channel)
	if err != nil {
		return []Channel{}
	}
	if channel == nil {
		return []Channel{}
	}
	return channel
}
