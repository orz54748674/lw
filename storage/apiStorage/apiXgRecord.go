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

type TransactionFunc func(db *mongo.Database, ctx context.Context, m *XgBetRecord) (err error)

var (
	AddXgRecord          = "addXgRecord"
	SettleXgRecord       = "settleXgRecord"
	RollbackXgRecord     = "rollbackXgRecord"
	cXgBetRecord         = "xgBetRecord"
	SetModifiedStatus    = "setModifiedStatus"
	ChangeSettleXgRecord = "changeSettleXgRecord"
	transactionFuncMap   = map[string]TransactionFunc{
		"addXgRecord":          addXgRecord,
		"settleXgRecord":       settleXgRecord,
		"rollbackXgRecord":     rollbackXgRecord,
		"setModifiedStatus":    setModifiedStatus,
		"changeSettleXgRecord": changeSettleXgRecord,
	}
)

func InitXgBetRecord(day int64) {
	c := common.GetMongoDB().C(cXgBetRecord)
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
func (m *XgBetRecord) SetName() string {
	return cXgBetRecord
}

func (m *XgBetRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *XgBetRecord) GetWagerIds() (records []*XgBetRecord, err error) {
	find := bson.M{"ReadResult": 0, "WagerId": bson.M{"$gt": 0}}
	records = []*XgBetRecord{}
	err = m.C().Find(find).Sort("-BetTime").Select(bson.M{"WagerId": 1, "Uid": 1, "_id": 1, "GameType": 1}).All(&records)
	return
}

func (m *XgBetRecord) GetNoReadRecords() (records []*XgBetRecord, err error) {
	find := bson.M{"ReadResult": 0, "WagerId": bson.M{"$gt": 0}}
	records = []*XgBetRecord{}
	err = m.C().Find(find).Sort("-BetTime").All(&records)
	return
}

func (m *XgBetRecord) SetWagerId(wagerId int64, readResult int) {
	find := bson.M{"WagerId": wagerId}
	//m.C().Update(find, bson.M{"$set": bson.M{"ReadResult": readResult}})
	m.C().UpdateMany(context.Background(), find, bson.M{"$set": bson.M{"ReadResult": readResult}})
	return
}

func (m *XgBetRecord) GetRecords(transactionIds []string) (records []*XgBetRecord, err error) {
	find := bson.M{"TransactionId": bson.M{"$in": transactionIds}, "SettleStatus": 0, "SettleRequestId": "", "WagerId": 0}
	records = []*XgBetRecord{}
	err = m.C().Find(find).All(&records)
	return
}

func (m *XgBetRecord) GetRecord(transactionId, user string) (record *XgBetRecord, err error) {
	find := bson.M{"TransactionId": transactionId, "SettleStatus": 0, "SettleRequestId": "", "WagerId": 0}
	err = m.C().Find(find).One(&record)
	return
}

func (m *XgBetRecord) GetRecordByTransactionId(transactionId string) (record *XgBetRecord, err error) {
	find := bson.M{"TransactionId": transactionId}
	err = m.C().Find(find).One(&record)
	return
}

func (m *XgBetRecord) GetRecordByModifiedStatus(gameType string) (records []*XgBetRecord, err error) {
	find := bson.M{"ModifiedStatus": "", "GameType": gameType}
	err = m.C().Find(find).Sort("-BetTime").All(&records)
	return
}

/**
 *  @title settleXgRecord
 *  @description	结算
 */
func (m *XgBetRecord) SettleXgRecord() (err error) {
	find := bson.M{"TransactionId": m.TransactionId, "SettleStatus": 0, "SettleRequestId": "", "WagerId": 0}
	set := bson.M{"$set": bson.M{"SettleStatus": 8, "SettleRequestId": m.SettleRequestId, "WagerId": m.WagerId, "SettleAmount": m.SettleAmount, "UpdateAt": time.Now()}}
	err = m.C().Update(find, set)
	return
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *XgBetRecord) SetTransactionUnits(transactionUnits ...string) {
	m.transactionUnits = transactionUnits
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *XgBetRecord) TransactionUnit(db *mongo.Database, ctx context.Context) (err error) {
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
 *  @title addXgRecord
 *  @description	新增记录
 */
func addXgRecord(db *mongo.Database, ctx context.Context, m *XgBetRecord) (err error) {
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())

	_, err = c.InsertOne(ctx, m)
	return
}

/**
 *  @title settleXgRecord
 *  @description	结算
 */
func settleXgRecord(db *mongo.Database, ctx context.Context, m *XgBetRecord) (err error) {
	c := db.Collection(m.SetName())
	find := bson.M{"TransactionId": m.TransactionId, "SettleStatus": 0, "SettleRequestId": "", "WagerId": 0}
	set := bson.M{"$set": bson.M{"SettleStatus": 8, "SettleRequestId": m.SettleRequestId, "WagerId": m.WagerId, "SettleAmount": m.SettleAmount, "UpdateAt": time.Now()}}
	_, err = c.UpdateOne(ctx, find, set)
	return
}

/**
 *  @title changeSettleXgRecord
 *  @description	修改结算
 */
func changeSettleXgRecord(db *mongo.Database, ctx context.Context, m *XgBetRecord) (err error) {
	c := db.Collection(m.SetName())
	find := bson.M{"TransactionId": m.TransactionId, "SettleStatus": 8, "SettleRequestId": m.SettleRequestId, "WagerId": m.WagerId}
	set := bson.M{"$set": bson.M{"SettleAmount": m.SettleAmount, "UpdateAt": time.Now()}}
	_, err = c.UpdateOne(ctx, find, set)
	return
}

/**
 *  @title rollbackXgRecord
 *  @description	取消
 */
func rollbackXgRecord(db *mongo.Database, ctx context.Context, m *XgBetRecord) (err error) {
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())
	find := bson.M{"TransactionId": m.TransactionId, "SettleStatus": 0, "SettleRequestId": "", "WagerId": 0}
	set := bson.M{"$set": bson.M{"SettleStatus": -1, "SettleRequestId": m.SettleRequestId, "UpdateAt": time.Now()}}
	_, err = c.UpdateOne(ctx, find, set)
	return
}

func setModifiedStatus(db *mongo.Database, ctx context.Context, m *XgBetRecord) (err error) {
	c := db.Collection(m.SetName())
	find := bson.M{"TransactionId": m.TransactionId}
	set := bson.M{"$set": bson.M{"ModifiedStatus": m.ModifiedStatus, "UpdateAt": time.Now()}}
	_, err = c.UpdateOne(ctx, find, set)
	return
}
