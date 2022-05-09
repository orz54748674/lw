package apiStorage

import (
	"context"
	"fmt"
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

type WmFunc func(db *mongo.Database, ctx context.Context, m *WmBillRecord) (err error)

var (
	AddWmBillRecord   = "addWmBillRecord"
	cWmBillRecord     = "WmBillRecord"
	SetRollbackStatus = "setRollbackStatus"
	wmFuncMap         = map[string]WmFunc{
		"addWmBillRecord":   addWmBillRecord,
		"setRollbackStatus": setRollbackStatus,
	}
)

func InitWmBillRecord(day int64) {
	c := common.GetMongoDB().C(cWmBillRecord)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().SetExpireAfterSeconds(int32(time.Duration(day)*24*time.Hour/time.Second))); err != nil {
		log.Error("create cAwcBetRecord Index: %s", err)
	}
}

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *WmBillRecord) SetName() string {
	return cWmBillRecord
}

func (m *WmBillRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *WmBillRecord) GetWmBillRecord(dealid string) (res *WmBillRecord, err error) {
	find := bson.M{"Dealid": dealid}
	res = &WmBillRecord{}
	err = m.C().Find(find).One(res)
	return
}

func (m *WmBillRecord) IsExists(code, betId string) bool {
	find := bson.M{"Code": code, "BetId": betId}
	log.Debug("WmBillRecord IsExists:%v", find)
	if err := m.C().Find(find).One(m); err != nil {
		log.Debug("WmBillRecord IsExists err:%s", err.Error())
		return false
	}
	return true
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *WmBillRecord) SetTransactionUnits(transactionUnits ...string) {
	m.transactionUnits = transactionUnits
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *WmBillRecord) TransactionUnit(db *mongo.Database, ctx context.Context) (err error) {
	for _, funcKey := range m.transactionUnits {
		Func, ok := wmFuncMap[funcKey]
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
 *  @title addWmBillRecord
 *  @description	新增记录
 */
func addWmBillRecord(db *mongo.Database, ctx context.Context, m *WmBillRecord) (err error) {
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())

	_, err = c.InsertOne(ctx, m)
	return
}

/**
 *  @title setRollbackStatus
 *  @description	设置滚状态
 */
func setRollbackStatus(db *mongo.Database, ctx context.Context, m *WmBillRecord) (err error) {
	m.UpdateAt = time.Now()
	find := bson.M{"Dealid": m.Dealid, "RollbackStatus": 0}
	update := bson.M{"$set": bson.M{"RollbackStatus": 1, "RollbackTime": m.RollbackTime, "UpdateAt": time.Now()}}
	c := db.Collection(m.SetName())
	_, err = c.UpdateOne(ctx, find, update)
	return
}
