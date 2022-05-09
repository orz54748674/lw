package apiStorage

import (
	"context"
	"fmt"
	"time"
	"vn/common"

	// "vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

type AwcFunc func(db *mongo.Database, ctx context.Context, m *AwcBetRecord) (err error)

var (
	AddAwcRecord        = "addAwcRecord"
	SettleAwcRecord     = "settleAwcRecord"
	CancelAwcRecord     = "cancelAwcRecord"
	UnsettleAwcRecord   = "unsettleAwcRecord"
	VoidAwcRecord       = "voidAwcRecord"
	cAwcBetRecord       = "AwcBetRecord"
	VoidSettleAwcRecord = "voidSettleAwcRecord"
	awcFuncMap          = map[string]AwcFunc{
		"addAwcRecord":        addAwcRecord,
		"settleAwcRecord":     settleAwcRecord,
		"cancelAwcRecord":     cancelAwcRecord,
		"voidAwcRecord":       voidAwcRecord,
		"unsettleAwcRecord":   unsettleAwcRecord,
		"voidSettleAwcRecord": voidSettleAwcRecord,
	}
)

func InitAwcBetRecord(day int64) {
	c := common.GetMongoDB().C(cAwcBetRecord)
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
func (m *AwcBetRecord) SetName() string {
	return cAwcBetRecord
}

func (m *AwcBetRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *AwcBetRecord) SetTransactionUnits(transactionUnits ...string) {
	m.transactionUnits = transactionUnits
}

func (m *AwcBetRecord) GetRecords(platformTxIds []string) (records []*AwcBetRecord, err error) {
	find := bson.M{"PlatformTxID": bson.M{"$in": platformTxIds}, "SettleStatus": NotSettle}
	records = []*AwcBetRecord{}
	err = m.C().Find(find).All(&records)
	return
}

func (m *AwcBetRecord) GetSettledRecords(platformTxIds []string) (records []*AwcBetRecord, err error) {
	find := bson.M{"PlatformTxID": bson.M{"$in": platformTxIds}, "SettleStatus": IsSettle}
	records = []*AwcBetRecord{}
	err = m.C().Find(find).All(&records)
	return
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *AwcBetRecord) TransactionUnit(db *mongo.Database, ctx context.Context) (err error) {
	for _, funcKey := range m.transactionUnits {
		Func, ok := awcFuncMap[funcKey]
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
 *  @title addAwcRecord
 *  @description	新增记录
 */
func addAwcRecord(db *mongo.Database, ctx context.Context, m *AwcBetRecord) (err error) {
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())

	_, err = c.InsertOne(ctx, m)
	return
}

/**
 *  @title settleAwcRecord
 *  @description	结算
 */
func settleAwcRecord(db *mongo.Database, ctx context.Context, m *AwcBetRecord) (err error) {
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())
	find := bson.M{"PlatformTxID": m.PlatformTxID, "SettleStatus": NotSettle}
	set := bson.M{"$set": bson.M{"WinAmount": m.WinAmount, "TxTime": m.TxTime, "UpdateTime": m.UpdateTime, "Turnover": m.Turnover, "GameInfo": m.GameInfo, "SettleStatus": IsSettle}}
	_, err = c.UpdateOne(ctx, find, set)
	return
}

/**
 *  @title cancelAwcRecord
 *  @description	取消
 */
func cancelAwcRecord(db *mongo.Database, ctx context.Context, m *AwcBetRecord) (err error) {
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())
	find := bson.M{"PlatformTxID": m.PlatformTxID, "SettleStatus": NotSettle}
	set := bson.M{"$set": bson.M{"UpdateTime": m.UpdateAt, "GameInfo": m.GameInfo, "SettleStatus": Colse}}
	_, err = c.UpdateOne(ctx, find, set)
	return
}

/**
 *  @title voidAwcRecord
 *  @description	取消
 */
func voidAwcRecord(db *mongo.Database, ctx context.Context, m *AwcBetRecord) (err error) {
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())
	find := bson.M{"PlatformTxID": m.PlatformTxID, "SettleStatus": NotSettle}
	set := bson.M{"$set": bson.M{"UpdateTime": m.UpdateTime, "GameInfo": m.GameInfo, "SettleStatus": VoidBet, "VoidType": m.VoidType}}
	_, err = c.UpdateOne(ctx, find, set)
	return
}

/**
 *  @title unsettleAwcRecord
 *  @description	取消
 */
func unsettleAwcRecord(db *mongo.Database, ctx context.Context, m *AwcBetRecord) (err error) {
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())
	find := bson.M{"PlatformTxID": m.PlatformTxID, "SettleStatus": IsSettle}
	set := bson.M{"$set": bson.M{"UpdateTime": m.UpdateTime, "GameInfo": m.GameInfo, "SettleStatus": NotSettle, "VoidType": m.VoidType, "WinAmount": 0}}
	_, err = c.UpdateOne(ctx, find, set)
	return
}

/**
 *  @title voidSettleAwcRecord
 *  @description	取消
 */
func voidSettleAwcRecord(db *mongo.Database, ctx context.Context, m *AwcBetRecord) (err error) {
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())
	find := bson.M{"PlatformTxID": m.PlatformTxID, "SettleStatus": IsSettle}
	set := bson.M{"$set": bson.M{"UpdateTime": m.UpdateTime, "SettleStatus": ViodSettle, "VoidType": m.VoidType}}
	_, err = c.UpdateOne(ctx, find, set)
	return
}
