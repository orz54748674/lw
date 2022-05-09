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
	cAwcCancelBetRecord = "AwcCancelBetRecord"
)

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *AwcCancelBetRecord) SetName() string {
	return cAwcCancelBetRecord
}

func (m *AwcCancelBetRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}
func (m *AwcCancelBetRecord) AddMany(datas []*AwcCancelBetRecord) error {
	var data = make([]interface{}, len(datas))
	for i, v := range datas {
		v.CreateAt = time.Now()
		v.UpdateAt = time.Now()
		data[i] = v
	}
	return m.C().InsertMany(data)
}

func (m *AwcCancelBetRecord) GetCancelRecord(platformTxID string) (int64, error) {
	find := bson.M{"PlatformTxID": platformTxID}
	return m.C().Find(find).Count()
}

func InitAwcCancelBetRecord() {
	c := common.GetMongoDB().C(cAwcCancelBetRecord)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().SetExpireAfterSeconds(int32(3*24*time.Hour/time.Second))); err != nil {
		log.Error("create cAwcCancelBetRecord Index: %s", err)
	}
}
