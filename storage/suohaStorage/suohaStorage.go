package suohaStorage

import (
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var (
	cBjlRecord      = "bjlRecord"
	cSuoHaGameConf  = "suohaGameConf"
	cSuoHaTableInfo = "suohaTableInfo"
	cSuoHaRanking   = "suohaRanking"
)

func InitBjlStorage() {
	c := common.GetMongoDB().C(cBjlRecord)
	key := bsonx.Doc{{Key: "createTime", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(30*24*3600)); err != nil {
		log.Error("create cBjlRecord Index: %s", err)
	}
}

func GetUserRecord(uid string, offset, limit int) []UserGameRecord {
	c := common.GetMongoDB().C(cBjlRecord)
	var records []UserGameRecord
	query := bson.M{"uid": uid}
	c.Find(query).Sort("-createTime").Skip(offset).Limit(limit).All(&records)
	return records
}

func InsertRecord(records []UserGameRecord) {
	c := common.GetMongoDB().C(cBjlRecord)
	for _, v := range records {
		if error := c.Insert(&v); error != nil {
			log.Info("Insert mail error: %s", error)
		}
	}
}

func GetGameConf() Conf {
	c := common.GetMongoDB().C(cSuoHaGameConf)
	var conf Conf
	err := c.Find(nil).One(&conf)
	if err != nil && err == mongo.ErrNoDocuments {
		conf.BaseConf = []BaseInfo{
			{Base: 100, MinEnter: 1000, MaxEnter: 5000},
			{Base: 500, MinEnter: 5000, MaxEnter: 25000},
			{Base: 1000, MinEnter: 10000, MaxEnter: 50000},
			{Base: 2000, MinEnter: 20000, MaxEnter: 100000},
			{Base: 5000, MinEnter: 50000, MaxEnter: 250000},
			{Base: 10000, MinEnter: 100000, MaxEnter: 500000},
			{Base: 20000, MinEnter: 200000, MaxEnter: 1000000},
			{Base: 50000, MinEnter: 500000, MaxEnter: 25000000},
		}
		if err := c.Insert(&conf); err != nil {
			log.Info("Insert mail error: %s", err)
		}
	}
	return conf
}

func GetSuoHaTableInfo() []TableInfo {
	c := common.GetMongoDB().C(cSuoHaTableInfo)
	var tableInfos []TableInfo
	if err := c.Find(nil).All(&tableInfos); err != nil {
		log.Info("GetSuoHaTableInfo err:%s", err.Error())
	}
	return tableInfos
}

func InsertSuoHaTableInfo(tableInfo TableInfo) error {
	c := common.GetMongoDB().C(cSuoHaTableInfo)
	if err := c.Insert(&tableInfo); err != nil {
		return err
	}
	return nil
}

func UpsertSuoHaTableInfo(info TableInfo) error {
	c := common.GetMongoDB().C(cSuoHaTableInfo)
	query := bson.M{"_id": info.Oid}
	update := bson.M{"$set": bson.M{"CurPlayer": info.CurPlayer}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func GetSuoHaRanking(offset, limit int) []RankingInfo {
	var ranking []RankingInfo
	c := common.GetMongoDB().C(cSuoHaRanking)
	if err := c.Find(nil).Sort("-TotalWinScore").Skip(offset).Limit(limit).All(&ranking); err != nil {
		log.Info("GetSuoHaRanking err:%s", err.Error())
	}
	return ranking
}
