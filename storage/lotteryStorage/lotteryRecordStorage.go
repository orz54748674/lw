package lotteryStorage

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
	cLotteryRecord = "lotteryRecord"
)

/**
 *  @title	InitLotteryRecord
 *	@description	初始化记录集合并建(number和lotteryCode)复合唯一索引
 */

func InitLotteryRecord() {
	record := &LotteryRecord{}
	key := bsonx.Doc{{Key: "Number", Value: bsonx.Int32(1)}, {Key: "LotteryCode", Value: bsonx.Int32(1)}}
	if err := record.C().CreateIndex(key, options.Index().SetUnique(true)); err != nil {
		log.Error("create LotteryRecordProfit Index: %s", err)
	}
}

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *LotteryRecord) SetName() string {
	return cLotteryRecord
}

func (m *LotteryRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *LotteryRecord) GetNumberOpenCode(number, lotteryCode string) (record *LotteryRecord, err error) {
	record = &LotteryRecord{}
	fields := bson.M{"OpenCode": 1, "_id": 0}
	find := bson.M{"LotteryCode": lotteryCode, "Number": number}
	err = m.C().Find(find).Select(fields).One(record)
	return
}

/**
 *  @title	GetRecordList
 *	@description	获取日期(期数)指定类型的lottery记录,并把数据放在对象本身上
 *	@param	number	string	日期(期数)
 *	@param	lotteryCode	string	lottery唯一Code
 *	@return	err	error	错误
 */
func (m *LotteryRecord) GetRecordList(lotteryCode string) (records []map[string]interface{}, err error) {
	records = []map[string]interface{}{}
	fields := bson.M{"OpenCode": 1, "_id": 0, "Number": 1}
	find := bson.M{"LotteryCode": lotteryCode}
	err = m.C().Find(find).Select(fields).Sort("-CnNumber").Skip(0).Limit(10).All(&records)
	return
}

func (m *LotteryRecord) GetRecord(number, lotteryCode string) (record map[string]interface{}, err error) {
	fields := bson.M{"OpenCode": 1, "_id": 0, "Number": 1}
	find := bson.M{"LotteryCode": lotteryCode, "Number": number}
	err = m.C().Find(find).Select(fields).Skip(0).Limit(10).One(&record)
	return
}

/**
 *  @title	GetRecords
 *	@description	获取日期(期数)范围内指定类型的lottery记录
 *	@param	startDate	string	开始日期(期数)
 *	@param	endDate	string	结束日期(期数)
 *	@param	lotteryCode	string	lottery唯一Code
 *	@return results []*LotteryRecord	记录对象数组
 *	@return	err	error	错误
 */
func (m *LotteryRecord) GetRecords(startDate, endDate, lotteryCode string) (results []*LotteryRecord, err error) {
	results = []*LotteryRecord{}
	find := bson.M{"LotteryCode": lotteryCode, "CnNumber": bson.M{"$gte": startDate, "$lte": endDate}}
	err = m.C().Find(find).All(&results)
	return
}

/**
 *  @title	IsExist
 *	@description	判断指定期数和类型 lottery 记录是否存在
 *	@param	number	string	日期(期号)
 *	@param	lotteryCode	string	lottery唯一Code
 *	@return count int64	返回记录条数
 *	@return	err	error	错误
 */
func (m *LotteryRecord) IsExist(number, lotteryCode string) (count int64, err error) {
	find := bson.M{"LotteryCode": lotteryCode, "Number": number}
	count, err = m.C().Find(find).Count()
	return
}

/**
 *  @title	Add
 *	@description	增加一条记录
 *	@return	err	error	错误
 */
func (m *LotteryRecord) Add() (err error) {
	m.Oid = primitive.NewObjectID()
	if err = CheckOpenCode(m.AreaCode, m.OpenCode); err != nil {
		return
	}
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	return m.C().Insert(m)
}
