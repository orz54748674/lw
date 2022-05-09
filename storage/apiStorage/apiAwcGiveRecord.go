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

type AwcGiveFunc func(db *mongo.Database, ctx context.Context, m *AwcGiveRecord) (err error)

var (
	cAwcGiveRecord   = "AwcGiveRecord"
	AddAwcGiveRecord = "addAwcGiveRecord"
	awcGiveFuncMap   = map[string]AwcGiveFunc{
		"addAwcGiveRecord": addAwcGiveRecord,
	}
)

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *AwcGiveRecord) SetName() string {
	return cAwcGiveRecord
}

func (m *AwcGiveRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *AwcGiveRecord) SetTransactionUnits(transactionUnits ...string) {
	m.transactionUnits = transactionUnits
}

func (m *AwcGiveRecord) IsExists() bool {
	c := m.C()
	query := bson.M{"PromotionId": m.PromotionId, "PromotionTxId": m.PromotionTxId}
	var record AwcGiveRecord
	if err := c.Find(query).One(&record); err != nil {
		return false
	}
	return true
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *AwcGiveRecord) TransactionUnit(db *mongo.Database, ctx context.Context) (err error) {
	for _, funcKey := range m.transactionUnits {
		Func, ok := awcGiveFuncMap[funcKey]
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
 *  @title addAwcGiveRecord
 *  @description	新增记录
 */
func addAwcGiveRecord(db *mongo.Database, ctx context.Context, m *AwcGiveRecord) (err error) {
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())

	_, err = c.InsertOne(ctx, m)
	return
}

/**
 *  @title	InitAwcGiveRecord
 *	@description	初始化记录集合并建(number和lotteryCode)复合唯一索引
 */

func InitAwcGiveRecord(day int64) {
	record := &AwcGiveRecord{}
	key := bsonx.Doc{{Key: "PromotionTxId", Value: bsonx.Int32(1)}, {Key: "PromotionId", Value: bsonx.Int32(1)}}
	if err := record.C().CreateIndex(key, options.Index().SetUnique(true)); err != nil {
		log.Error("create AwcGiveRecord Index: %s", err)
	}

	c := record.C()
	key = bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().SetExpireAfterSeconds(int32(time.Duration(day)*24*time.Hour/time.Second))); err != nil {
		log.Error("create cAwcGiveRecord Index: %s", err)
	}
}
