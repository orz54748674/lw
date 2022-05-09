package apiCqStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
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

func (m *ApiUser) addBefore() {
	m.Oid = primitive.NewObjectID()
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
}

func (m *ApiUser) updateBefore() {
	m.UpdateAt = time.Now()
}

func GetRecordByMTCode(mtcode string) (record BetRecords, err error) {
	query := bson.M{"mtcode": mtcode}
	var mtMsg MTCode
	cMTCode := common.GetMongoDB().C(cCqMtcode)
	if err = cMTCode.Find(query).One(&mtMsg); err != nil {
		return record, err
	}

	cRecord := common.GetMongoDB().C(cCqRecord)
	if err = cRecord.Find(bson.M{"_id": utils.ConvertOID(mtMsg.RecordOid)}).One(&record); err != nil {
		return record, err
	}
	return record, nil
}

func FindUserByAccount(account string, apiType int8) (ApiUser, bool) {
	var apiUser ApiUser
	c := common.GetMongoDB().C(cApiUser)
	find := bson.M{"Account": account, "Type": apiType}
	if err := c.Find(find).One(&apiUser); err != nil {
		return apiUser, false
	}
	return apiUser, true
}
