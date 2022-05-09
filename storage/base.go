package storage

import (
	"time"
	"vn/framework/mongo-driver/bson/primitive"
)

type Base struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt time.Time          `bson:"UpdateAt" json:"UpdateAt"`
}

func (b *Base) AddBefore() {
	b.Oid = primitive.NewObjectID()
	b.CreateAt = time.Now()
	b.UpdateAt = time.Now()
}

func (b *Base) UpdateBefore() {
	b.UpdateAt = time.Now()
}
