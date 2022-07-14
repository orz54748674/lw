package gbsStorage

import (
	"fmt"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var (
	cGuessBigSmallConf   = "guessBigSmallConf"
	cGbsRecord           = "guessBigSmallRecord"
	cGbsPoolRewardRecord = "guessPoolRewardRecord"
)

func InitGbsStorage() {
	c := common.GetMongoDB().C(cGbsRecord)
	key := bsonx.Doc{{Key: "createTime", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(30*24*3600)); err != nil {
		log.Error("create cGbsRecord Index: %s", err)
	}
}

func GetGameConf() []GameConf {
	c := common.GetMongoDB().C(cGuessBigSmallConf)
	var openRes []GameConf
	if err := c.Find(nil).All(&openRes); err != nil || len(openRes) == 0 {
		//log.Info("get guessBigSmall conf err:", err.Error())
		baseArr := []int64{1000, 10000, 50000, 100000, 500000}
		for _, v := range baseArr {
			var gameConf GameConf
			gameConf.Chip = v
			gameConf.PoolVal = 20 * gameConf.Chip
			gameConf.MingPercent = 20
			gameConf.AnPercent = 50
			if error := c.Insert(&gameConf); error != nil {
				log.Info("Insert mail error: %s", error)
			}
			openRes = append(openRes, gameConf)
		}
	}

	return openRes
}

func UpsertPoolVal(chip, val int64) {
	c := common.GetMongoDB().C(cGuessBigSmallConf)
	query := bson.M{"chip": chip}
	update := bson.M{
		"$inc": bson.M{"poolVal": val},
	}

	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
}

func GetGbsRecord(uid string, offset, limit int) []GbsRecord {
	c := common.GetMongoDB().C(cGbsRecord)
	var record []GbsRecord
	query := bson.M{"uid": uid}
	c.Find(query).Sort("-createTime").Skip(offset).Limit(limit).All(&record)
	fmt.Println(".........record", record)
	return record
}

func InsertRecord(record GbsRecord) {
	c := common.GetMongoDB().C(cGbsRecord)
	if error := c.Insert(&record); error != nil {
		log.Info("Insert gbs record error: %s", error)
	}
}

func InsertPoolRewardRecord(record GbsPoolRewardRecord) {
	c := common.GetMongoDB().C(cGbsPoolRewardRecord)
	if err := c.Insert(&record); err != nil {
		log.Info("Insert gbs pool reward record error: %s", err)
	}
}

func GetGbsPoolRewardRecord(offset, limit int) []GbsPoolRewardRecord {
	c := common.GetMongoDB().C(cGbsPoolRewardRecord)
	var record []GbsPoolRewardRecord
	c.Find(nil).Sort("-createTime").Skip(offset).Limit(limit).All(&record)
	return record
}
