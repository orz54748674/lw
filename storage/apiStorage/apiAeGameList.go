package apiStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson/primitive"
)

type AeGame struct {
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	ApiType    int8               `bson:"ApiType" json:"ApiType"`
	CreateAt   time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt   time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	GameCode   string             `bson:"GameCode" json:"GameCode"`
	GameName   string             `bson:"GameName" json:"GameName"`
	Jackpot    bool               `bson:"Jackpot" json:"Jackpot"`
	Thumbnail  string             `bson:"Thumbnail" json:"Thumbnail"`
	Screenshot string             `bson:"Screenshot" json:"Screenshot"`
	Mthumbnail string             `bson:"Mthumbnail" json:"Mthumbnail"`
	Extends    string             `bson:"Extends" json:"extends"`
}

var (
	cAeGame = "AeGame"
)

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *AeGame) SetName() string {
	return cAeGame
}

func (m *AeGame) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *AeGame) Add() error {
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	return m.C().Insert(m)
}
