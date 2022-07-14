package rsbStorage

import (
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/storage/gameStorage"
)

var cRsbRecord = "rsbRecord"
var cRsbConf = "rsbConf"

func GetRsbRecord(uid string, offset, limit int) []RsbRecord {
	c := common.GetMongoDB().C(cRsbRecord)
	var record []RsbRecord
	query := bson.M{"uid": uid}
	c.Find(query).Sort("-updateTime").Skip(offset).Limit(limit).All(&record)
	return record
}

func GetOpenRes(uid string, limit int) []OpenRes {
	c := common.GetMongoDB().C(cRsbRecord)
	var openRes []OpenRes
	query := bson.M{"uid": uid}
	fields := bson.M{"winPos": 1}
	c.Find(query).Select(fields).Sort("-updateTime").Limit(limit).All(&openRes)
	return openRes
}

func InsertRecord(record RsbRecord) {
	c := common.GetMongoDB().C(cRsbRecord)
	if error := c.Insert(&record); error != nil {
		log.Info("Insert mail error: %s", error)
	}
}

func GetRsbConf() (int64, int64, int64) {
	c := common.GetMongoDB().C(cRsbConf)
	var rsbConf RsbConf
	if err := c.Find(nil).One(&rsbConf); err != nil {
		rsbConf.MingPercent = 20
		rsbConf.AnPercent = 50
		rsbConf.InitBalance = 100000
		if err = c.Insert(&rsbConf); err != nil {
			log.Info("Insert rsbConf error: %s", err)
		}
		return rsbConf.InitBalance, rsbConf.MingPercent, rsbConf.AnPercent
	}

	gameProfit := gameStorage.QueryProfit(game.Roshambo)
	return gameProfit.BotBalance, rsbConf.MingPercent, rsbConf.AnPercent
}
