package gameStorage

import (
	"vn/framework/mongo-driver/bson/primitive"
)

type GameCommonData struct {
	ID           int64         `bson:"-" json:"-"`
	Oid          primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Uid  	 string 	 `bson:"Uid"`
	InRoomNeedVnd int64  `bson:"InRoomNeedVnd"`
}
