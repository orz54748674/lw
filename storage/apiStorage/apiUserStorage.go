package apiStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var (
	cApiUser = "apiUser"
)

/**
 *  @title	InitLotteryRecord
 *	@description	初始化记录集合并建(number和lotteryCode)复合唯一索引
 */

func InitApiUser() {
	record := &ApiUser{}
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}, {Key: "Type", Value: bsonx.Int32(1)}}
	if err := record.C().CreateIndex(key, options.Index().SetUnique(true)); err != nil {
		log.Error("create LotteryRecordProfit Index: %s", err)
	}
}

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *ApiUser) SetName() string {
	return cApiUser
}

func (m *ApiUser) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *ApiUser) Save() error {
	m.addBefore()
	return m.C().Insert(m)
}

func (m *ApiUser) GetApiUser(uid string, apiType int8) (err error) {
	find := bson.M{"Uid": uid, "Type": apiType}
	err = m.C().Find(find).One(m)
	return
}

func (m *ApiUser) GetApiUserByAccount(account string, apiType int8) (err error) {
	find := bson.M{"Account": account, "Type": apiType}
	err = m.C().Find(find).One(m)
	return
}

func (m *ApiUser) GetApiUsersByAccount(accounts []string, apiType int8) (res []*ApiUser, err error) {
	find := bson.M{"Account": bson.M{"$in": accounts}, "Type": apiType}
	res = []*ApiUser{}
	err = m.C().Find(find).All(&res)
	return
}

func (m *ApiUser) addBefore() {
	m.Oid = primitive.NewObjectID()
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
}
