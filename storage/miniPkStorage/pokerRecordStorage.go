package miniPkStorage

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

type TransactionFunc func(db *mongo.Database, ctx context.Context, m *PokerRecord) (err error)

var (
	cPrizeRecord       = "PokerRecord"
	AddPokerRecord     = "addPokerRecord"
	transactionFuncMap = map[string]TransactionFunc{
		"addPokerRecord": addPokerRecord,
	}
)

func InitPokerRecord(day int64) {
	c := common.GetMongoDB().C(cPrizeRecord)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().SetExpireAfterSeconds(int32(time.Duration(day)*24*time.Hour/time.Second))); err != nil {
		log.Error("create cPrizeRecord Index: %s", err)
	}
}

func (m *PokerRecord) SetName() string {
	return cPrizeRecord
}
func (m *PokerRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *PokerRecord) GetBigPrizeList(offset, limit int) (records []map[string]interface{}, err error) {
	records = []map[string]interface{}{}
	find := bson.M{"PrizeType": 9}
	fields := bson.M{"CreateAt": 1, "NickName": 1, "BetAmount": 1, "Bonus": 1, "_id": 0}
	err = m.C().Find(find).Select(fields).Sort("-CreateAt").Skip(offset).Limit(limit).All(&records)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return
}

func (m *PokerRecord) GetPrizeList(offset, limit int, uid string) (records []map[string]interface{}, err error) {
	records = []map[string]interface{}{}
	find := bson.M{"Uid": uid}
	fields := bson.M{"Number": 1, "CreateAt": 1, "NickName": 1, "BetAmount": 1, "Bonus": 1, "Pokers": 1, "_id": 0}
	err = m.C().Find(find).Select(fields).Sort("-CreateAt").Skip(offset).Limit(limit).All(&records)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return
}

/**
 *  @title AddPokerRecord
 *  @description	添加
 */
func (m *PokerRecord) AddPokerRecord() (err error) {
	m.UpdateAt = time.Now()
	m.CreateAt = time.Now()
	err = m.C().Insert(m)
	return
}

func (m *PokerRecord) StatsLeaderBoard(offset, limit int) (records []map[string]interface{}, err error) {
	records = []map[string]interface{}{}
	pipe := mongo.Pipeline{
		{{"$group",
			bson.M{
				"_id":         "$NickName",
				"TotalProfit": bson.M{"$sum": bson.M{"$add": "$Profit"}},
			},
		}},
		{{"$sort", bson.M{"TotalProfit": -1}}},
	}
	err = m.C().Pipe(pipe).Skip(offset).Limit(limit).All(&records)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *PokerRecord) SetTransactionUnits(transactionUnits ...string) {
	m.transactionUnits = transactionUnits
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *PokerRecord) TransactionUnit(db *mongo.Database, ctx context.Context) (err error) {
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
 *  @title addPokerRecord
 *  @description	添加
 */
func addPokerRecord(db *mongo.Database, ctx context.Context, m *PokerRecord) (err error) {
	c := db.Collection(m.SetName())
	m.UpdateAt = time.Now()
	m.CreateAt = time.Now()
	_, err = c.InsertOne(ctx, m)
	return
}
