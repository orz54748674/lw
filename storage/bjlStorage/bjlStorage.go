package bjlStorage

import (
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var (
	cBjlRecord = "bjlRecord"
	cBjlGameConf = "bjlGameConf"
)

func InitBjlStorage() {
	c := common.GetMongoDB().C(cBjlRecord)
	key := bsonx.Doc{{Key: "createTime",Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key,options.Index().
		SetExpireAfterSeconds(30*24*3600));err != nil{
		log.Error("create cBjlRecord Index: %s",err)
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
		if error := c.Insert(&v); error != nil{
			log.Info("Insert mail error: %s", error)
		}
	}
}

func GetGameConf() Conf {
	c := common.GetMongoDB().C(cBjlGameConf)
	var conf Conf
	err := c.Find(nil).One(&conf)
	if err != nil {
		conf.BetTime = 20000
		conf.BotProfitPerThousand = 80
		conf.CheckoutTime = 10000
		conf.ChipList = []int64{1000, 5000, 10000, 50000, 100000, 500000, 1000000, 5000000, 10000000, 50000000}
		conf.KickRoomCnt = 3
		conf.OddList = []int64{200, 195, 900, 1100, 1100}
		conf.ProfitPerThousand = 0
		conf.SendCardTime = 30000
		conf.PosBetLimit = []int64{200000000, 200000000, 60000000, 60000000, 60000000}
		conf.MinBet = 1000
		conf.MaxBet = 200000000
		if err := c.Insert(&conf); err != nil {
			log.Info("Insert mail error: %s", err)
		}
	}
	return conf
}
