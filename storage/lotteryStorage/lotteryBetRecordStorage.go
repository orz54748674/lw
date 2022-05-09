package lotteryStorage

import (
	"context"
	"fmt"
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

type TransactionFunc func(db *mongo.Database, ctx context.Context, m *LotteryBetRecord) (err error)

var (
	cLotteryBetRecord       = "lotteryBetRecord"
	ChangePayStatus         = "changePayStatus"
	ChangeSettleStatus      = "changeSettleStatus"
	CancelSettleStatus      = "cancelSettleStatus"
	NotPay             int8 = 0
	PayEnd             int8 = 1
	transactionFuncMap      = map[string]TransactionFunc{
		"changePayStatus":    changePayStatus,
		"changeSettleStatus": changeSettleStatus,
		"cancelSettleStatus": cancelSettleStatus,
	}
)

func InitLotteryBetRecord(day int64) {
	c := common.GetMongoDB().C(cLotteryBetRecord)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().SetExpireAfterSeconds(int32(time.Duration(day)*24*time.Hour/time.Second))); err != nil {
		log.Error("create cLotteryBetRecord Index: %s", err)
	}
}

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *LotteryBetRecord) SetName() string {
	return cLotteryBetRecord
}

func (m *LotteryBetRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

/**
 *  @title	GetUserRecords
 *	@description	获取用户单期所有下注code
 *	@params number string 期号
 *	@params lotteryCode string
 *	@params subPlayCode string
 *	@return	records	[]*LotteryBetRecord 记录数组对象
 *	@return	err	error	错误
 */
func (m *LotteryBetRecord) GetUserRecords(uid, number, lotteryCode, subPlayCode string) (records []*LotteryBetRecord, err error) {
	records = []*LotteryBetRecord{}
	fields := bson.M{"Code": 1, "_id": 0, "PlayCode": 1}
	find := bson.M{"Number": number, "LotteryCode": lotteryCode, "SubPlayCode": subPlayCode, "Uid": uid}
	err = m.C().Find(find).Select(fields).All(&records)
	return
}

/**
 *  @title	Add
 *	@description	增加一条记录
 *	@return	err	error	错误
 */
func (m *LotteryBetRecord) Add() (err error) {
	m.Oid = primitive.NewObjectID()
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	return m.C().Insert(m)
}

/**
 *  @title	GetRecordList
 *	@description	获取用户下注记录
 *	@return	records	[]*LotteryBetRecord	记录对象数组
 *	@return	err	error	错误
 */
func (m *LotteryBetRecord) GetRecordListByUid(uid string, offset, limit int) (records []map[string]interface{}, err error) {
	records = []map[string]interface{}{}
	find := bson.M{"Uid": uid}
	fields := bson.M{
		"OpenTime":      1,
		"LotteryCode":   1,
		"PlayCode":      1,
		"SubPlayCode":   1,
		"Code":          1,
		"UnitBetAmount": 1,
		"TotalAmount":   1,
		"SProfit":       1,
		"SettleStatus":  1,
		"Number":        1,
		"_id":           0}
	err = m.C().Find(find).Select(fields).Sort("-_id").Skip(offset).Limit(limit).All(&records)
	return
}

/**
 *  @title	GetNumberBets
 *	@description	获取指定期数和lotteryCode 的下注列表
 *	@return	records	[]*LotteryBetRecord	记录对象数组
 *	@return	err	error	错误
 */
func (m *LotteryBetRecord) GetNumberBets(number, lotteryCode string, offset, limit int) (records []*LotteryBetRecord, err error) {
	records = []*LotteryBetRecord{}
	find := bson.M{"Number": number, "LotteryCode": lotteryCode, "SettleStatus": 1}
	err = m.C().Find(find).Sort("-_id").Skip(offset).Limit(limit).All(&records)
	return
}

/**
 *  @title	GetBets
 *	@description	获取下注列表
 *	@return	records	[]*LotteryBetRecord	记录对象数组
 *	@return	err	error	错误
 */
func (m *LotteryBetRecord) GetBets(offset, limit int) (records []*LotteryBetRecord, err error) {
	find := bson.M{}
	records = []*LotteryBetRecord{}
	err = m.C().Find(find).Sort("-CreateAt").Skip(offset).Limit(limit).All(&records)
	return
}
func (m *LotteryBetRecord) ModifyBetInfo() (err error) {
	find := bson.M{"_id": m.Oid}
	change := bson.M{"$set": bson.M{"Number": m.Number, "CnNumber": m.CnNumber, "UpdateAt": time.Now(), "OpenTime": m.OpenTime, "AreaCode": m.AreaCode, "CityCode": m.CityCode, "LotteryCode": m.LotteryCode}}
	err = m.C().Update(find, change)
	return
}

/**
 *  @title	GetBetByOids
 *	@description	获取下注列表
 *	@return	records	[]*LotteryBetRecord	记录对象数组
 *	@return	err	error	错误
 */
func (m *LotteryBetRecord) GetBetByOids(oids []primitive.ObjectID) (records []*LotteryBetRecord, err error) {
	find := bson.M{"_id": bson.M{"$in": oids}}
	records = []*LotteryBetRecord{}
	err = m.C().Find(find).All(&records)
	return
}

/**
 *  @title	GetNumberRecords
 *	@description	获取下注列表
 *	@return	records	[]*LotteryBetRecord	记录对象数组
 *	@return	err	error	错误
 */
func (m *LotteryBetRecord) GetNumberRecords(code, number string, offset, limit int) (records []*LotteryBetRecord, err error) {
	find := bson.M{"LotteryCode": code, "Number": number, "SettleStatus": 0}
	records = []*LotteryBetRecord{}
	err = m.C().Find(find).Sort("-CreateAt").Skip(offset).Limit(limit).All(&records)
	return
}

func (m *LotteryBetRecord) SetOpenCode(number, lotteryCode string, openCode map[PrizeLevel][]string) (result *mongo.UpdateResult, err error) {
	find := bson.M{"Number": number, "LotteryCode": lotteryCode, "SettleStatus": 0}
	data := bson.M{"$set": bson.M{"OpenCode": openCode, "SettleStatus": 1, "UpdateAt": time.Now()}}
	result, err = m.C().UpdateMany(context.Background(), find, data)
	return
}

/**
 *  @title ChangeSettleStatus
 *  @description	修改注单结算状态
 */
func (m *LotteryBetRecord) ChangeSettleStatus(settleStatus int) (err error) {
	find := bson.M{"_id": m.Oid, "SettleStatus": 1}
	change := bson.M{"$set": bson.M{"SettleStatus": settleStatus, "SProfit": m.SProfit, "UpdateAt": time.Now()}}
	err = m.C().Update(find, change)
	return
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *LotteryBetRecord) SetTransactionUnits(transactionUnits ...string) {
	m.transactionUnits = transactionUnits
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *LotteryBetRecord) TransactionUnit(db *mongo.Database, ctx context.Context) (err error) {
	for _, funcKey := range m.transactionUnits {
		Func, ok := transactionFuncMap[funcKey]
		if !ok {
			err = fmt.Errorf("not find TransactionUnit func:%s", funcKey)
			return
		}
		if err = Func(db, ctx, m); err != nil {
			return
		}
	}
	return
}

/**
 *  @title changePayStatus
 *  @description	修改注单支付状态
 */
func changePayStatus(db *mongo.Database, ctx context.Context, m *LotteryBetRecord) (err error) {
	find := bson.M{"_id": m.Oid, "PayStatus": 0}
	change := bson.M{"$set": bson.M{"PayStatus": m.PayStatus, "UpdateAt": time.Now()}}
	c := db.Collection(m.SetName())
	_, err = c.UpdateOne(ctx, find, change)
	return
}

/**
 *  @title changeSettleStatusAffair
 *  @description	修改注单结算状态事务操作
 */
func changeSettleStatus(db *mongo.Database, ctx context.Context, m *LotteryBetRecord) (err error) {
	find := bson.M{"_id": m.Oid, "SettleStatus": 1}
	change := bson.M{"$set": bson.M{"SettleStatus": m.SettleStatus, "SProfit": m.SProfit, "UpdateAt": time.Now()}}
	c := db.Collection(m.SetName())
	_, err = c.UpdateOne(ctx, find, change)
	return
}

/**
 *  @title cancelSettleStatus
 *  @description	修改注单结算状态事务操作
 */
func cancelSettleStatus(db *mongo.Database, ctx context.Context, m *LotteryBetRecord) (err error) {
	find := bson.M{"_id": m.Oid, "SettleStatus": 0}
	change := bson.M{"$set": bson.M{"SettleStatus": m.SettleStatus, "SProfit": m.SProfit, "UpdateAt": time.Now()}}
	c := db.Collection(m.SetName())
	_, err = c.UpdateOne(ctx, find, change)
	return
}
