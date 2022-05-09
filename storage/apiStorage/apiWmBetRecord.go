package apiStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var (
	cWmBetRecord = "WmBetRecord"
)

/**
 *  @title	InitWmBetRecord
 *	@description	初始化记录集合并建(number和lotteryCode)复合唯一索引
 */

func InitWmBetRecord(day int64) {
	record := &WmBetRecord{}
	key := bsonx.Doc{{Key: "BetId", Value: bsonx.Int32(1)}}
	if err := record.C().CreateIndex(key, options.Index().SetUnique(true)); err != nil {
		log.Error("create InitWmBetRecord Index: %s", err)
	}
	c := record.C()
	key = bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().SetExpireAfterSeconds(int32(time.Duration(day)*24*time.Hour/time.Second))); err != nil {
		log.Error("create cWmBetRecord Index: %s", err)
	}
}

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *WmBetRecord) SetName() string {
	return cWmBetRecord
}

func (m *WmBetRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *WmBetRecord) AddWmBetRecord() (err error) {
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	err = m.C().Insert(m)
	return
}

func (m *WmBetRecord) IsExists() bool {
	c := m.C()
	query := bson.M{"BetId": m.BetId}
	var record WmBetRecord
	if err := c.Find(query).One(&record); err != nil {
		return false
	}
	return true
}
