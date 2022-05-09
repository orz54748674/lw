package userStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
)

type DeviceBlack struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"oid"`
	Uuid     string             `bson:"Uuid" json:"uuid"`
	Status   int                `bson:"Status" json:"status"`
	CreateAt time.Time          `bson:"CreateAt" json:"createAt"`
	UpdateAt time.Time          `bson:"UpdateAt" json:"updateAt"`
}
type IpBlack struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"oid"`
	Ip     string             `bson:"Ip" json:"Ip"`
	Status   int                `bson:"Status" json:"status"`
	CreateAt time.Time          `bson:"CreateAt" json:"createAt"`
	UpdateAt time.Time          `bson:"UpdateAt" json:"updateAt"`
}
var cDeviceBlack = "deviceBlack"
var cIpBlack = "ipBlack"

func DeviceIsBlack(uuid string) bool {
	c := common.GetMongoDB().C(cDeviceBlack)
	query := bson.M{"Uuid": uuid, "Status": 1}
	var black DeviceBlack
	c.Find(query).One(&black)
	if black.Oid.IsZero() {
		return false
	} else {
		return true
	}
}
func IpIsBlack(uuid string) bool {
	c := common.GetMongoDB().C(cIpBlack)
	query := bson.M{"Ip": uuid, "Status": 1}
	var black IpBlack
	c.Find(query).One(&black)
	if black.Oid.IsZero() {
		return false
	} else {
		return true
	}
}