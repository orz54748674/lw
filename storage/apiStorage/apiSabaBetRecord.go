package apiStorage

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

type SabaFunc func(db *mongo.Database, ctx context.Context, m *SabaBetRecord) (err error)

var (
	cSabaBetRecord               = "SabaBetRecord"
	AddSabaBetRecord             = "addSabaBetRecord"
	SettleSabaBetRecord          = "settleSabaBetRecord"
	UnsettleSabaBetRecord        = "unsettleSabaBetRecord"
	CancelSabaBetRecord          = "cancelSabaBetRecord"
	CashOutSabaBetRecord         = "cashOutSabaBetRecord"
	CashOutResettleSabaBetRecord = "cashOutResettleSabaBetRecord"
	SabaFuncMap                  = map[string]SabaFunc{
		"addSabaBetRecord":             addSabaBetRecord,
		"settleSabaBetRecord":          settleSabaBetRecord,
		"unsettleSabaBetRecord":        unsettleSabaBetRecord,
		"cancelSabaBetRecord":          cancelSabaBetRecord,
		"cashOutSabaBetRecord":         cashOutSabaBetRecord,
		"cashOutResettleSabaBetRecord": cashOutResettleSabaBetRecord,
	}
)

/**
 *  @title	InitSabaBetRecord
 *	@description	初始化记录集合并建(number和lotteryCode)复合唯一索引
 */

func InitSabaBetRecord(day int64) {
	record := &SabaBetRecord{}
	// key := bsonx.Doc{{Key: "BetId", Value: bsonx.Int32(1)}}
	// if err := record.C().CreateIndex(key, options.Index().SetUnique(true)); err != nil {
	// 	log.Error("create InitSabaBetRecord Index: %s", err)
	// }
	c := record.C()
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().SetExpireAfterSeconds(int32(time.Duration(day)*24*time.Hour/time.Second))); err != nil {
		log.Error("create cSabaBetRecord Index: %s", err)
	}
}

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *SabaBetRecord) SetName() string {
	return cSabaBetRecord
}

func (m *SabaBetRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *SabaBetRecord) GetRecordByOids(oids []primitive.ObjectID) (res []*SabaBetRecord, err error) {
	find := bson.M{"_id": bson.M{"$in": oids}}
	res = []*SabaBetRecord{}
	err = m.C().Find(find).All(&res)
	return
}

func (m *SabaBetRecord) GetRecordByTxIds(txIds []int64, status string) (res []*SabaBetRecord, err error) {
	find := bson.M{"TxId": bson.M{"$in": txIds}, "SettleStatus": status}
	res = []*SabaBetRecord{}
	err = m.C().Find(find).All(&res)
	return
}

func (m *SabaBetRecord) GetRecordByRefIds(refIds []string, status string) (res []*SabaBetRecord, err error) {
	find := bson.M{"RefId": bson.M{"$in": refIds}}
	if len(status) > 0 {
		find["SettleStatus"] = status
	}
	res = []*SabaBetRecord{}
	err = m.C().Find(find).All(&res)
	return
}
func (m *SabaBetRecord) GetNoCompletedRecord() (res []*SabaBetRecord, err error) {
	find := bson.M{"GameStatus": bson.M{"$nin": []string{"completed", "refund"}}, "MatchId": bson.M{"$gt": 0}}
	res = []*SabaBetRecord{}
	err = m.C().Find(find).All(&res)
	return
}

func (m *SabaBetRecord) GetRecordByMatchIds(matchIds []int, status string) (res []*SabaBetRecord, err error) {
	find := bson.M{"MatchId": bson.M{"$in": matchIds}}
	if len(status) > 0 {
		find["SettleStatus"] = status
	}
	res = []*SabaBetRecord{}
	err = m.C().Find(find).All(&res)
	return
}

func (m *SabaBetRecord) Update(data map[string]interface{}) (err error) {
	find := bson.M{"_id": m.Oid}
	data["UpdateAt"] = time.Now()
	update := bson.M{"$set": data}
	err = m.C().Update(find, update)
	return
}

func (m *SabaBetRecord) SetScore(matchId int) (err error) {
	find := bson.M{"MatchId": matchId}
	update := bson.M{"$set": bson.M{"GameStatus": m.GameStatus, "HomeScore": m.HomeScore, "AwayScore": m.AwayScore, "HtHomeScore": m.HtHomeScore, "HtAwayScore": m.HtAwayScore}}
	_, err = m.C().UpdateMany(context.Background(), find, update)
	return
}

func (m *SabaBetRecord) GetNoSetGameData(settleStatus string) (res []*SabaBetRecord, err error) {
	find := bson.M{"IsGameData": bson.M{"$ne": true}, "SettleStatus": settleStatus}
	res = []*SabaBetRecord{}
	err = m.C().Find(find).All(&res)
	return
}

func (m *SabaBetRecord) SetGameData(matchId int, isGameData bool) (err error) {
	find := bson.M{"MatchId": matchId}
	update := bson.M{"$set": bson.M{"IsGameData": isGameData}}
	_, err = m.C().UpdateMany(context.Background(), find, update)
	return
}

/**
 *  @title addSabaBetRecord
 *  @description	新增记录
 */
func (m *SabaBetRecord) AddSabaBetRecord() (err error) {
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	err = m.C().Insert(m)
	return
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *SabaBetRecord) SetTransactionUnits(transactionUnits ...string) {
	m.transactionUnits = transactionUnits
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *SabaBetRecord) TransactionUnit(db *mongo.Database, ctx context.Context) (err error) {
	for _, funcKey := range m.transactionUnits {
		Func, ok := SabaFuncMap[funcKey]
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
 *  @title addSabaBetRecord
 *  @description	新增记录
 */
func addSabaBetRecord(db *mongo.Database, ctx context.Context, m *SabaBetRecord) (err error) {
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())

	_, err = c.InsertOne(ctx, m)
	return
}

/**
 *  @title settleSabaBetRecord
 *  @description	新增记录
 */
func settleSabaBetRecord(db *mongo.Database, ctx context.Context, m *SabaBetRecord) (err error) {
	c := db.Collection(m.SetName())

	find := bson.M{"_id": m.Oid, "SettleStatus": "runing"}
	update := bson.M{
		"$set": bson.M{
			"Payout":       m.Payout,
			"WinlostDate":  m.WinlostDate,
			"UpdateTime":   m.UpdateTime,
			"Status":       m.Status,
			"SettleStatus": m.SettleStatus,
			"UpdateAt":     time.Now(),
		},
	}
	_, err = c.UpdateMany(ctx, find, update)
	return
}

/**
 *  @title unsettleSabaBetRecord
 *  @description	新增记录
 */
func unsettleSabaBetRecord(db *mongo.Database, ctx context.Context, m *SabaBetRecord) (err error) {
	c := db.Collection(m.SetName())

	find := bson.M{"_id": m.Oid, "SettleStatus": "settle"}
	update := bson.M{
		"$set": bson.M{
			"Payout":       0,
			"WinlostDate":  m.WinlostDate,
			"UpdateTime":   m.UpdateTime,
			"Status":       "unsettle",
			"SettleStatus": m.SettleStatus,
			"UpdateAt":     time.Now(),
		},
	}
	_, err = c.UpdateMany(ctx, find, update)
	return
}

/**
 *  @title cashOutSabaBetRecord
 *  @description
 */
func cashOutSabaBetRecord(db *mongo.Database, ctx context.Context, m *SabaBetRecord) (err error) {
	c := db.Collection(m.SetName())

	find := bson.M{"_id": m.Oid, "SettleStatus": "runing"}
	update := bson.M{
		"$set": bson.M{
			"Payout":       m.Payout,
			"UpdateTime":   m.UpdateTime,
			"Status":       m.Status,
			"SettleStatus": m.SettleStatus,
			"CashStatus":   m.CashStatus,
			"UpdateAt":     time.Now(),
		},
	}
	_, err = c.UpdateMany(ctx, find, update)
	return
}

/**
 *  @title cashOutResettleSabaBetRecord
 *  @description
 */
func cashOutResettleSabaBetRecord(db *mongo.Database, ctx context.Context, m *SabaBetRecord) (err error) {
	c := db.Collection(m.SetName())

	find := bson.M{"_id": m.Oid, "SettleStatus": "cashout", "CashStatus": 3}
	update := bson.M{
		"$set": bson.M{
			"Payout":       m.Payout,
			"UpdateTime":   m.UpdateTime,
			"Status":       m.Status,
			"SettleStatus": m.SettleStatus,
			"UpdateAt":     time.Now(),
		},
	}
	_, err = c.UpdateMany(ctx, find, update)
	return
}

/**
 *  @title cancelSabaBetRecord
 *  @description	新增记录
 */
func cancelSabaBetRecord(db *mongo.Database, ctx context.Context, m *SabaBetRecord) (err error) {
	c := db.Collection(m.SetName())

	find := bson.M{"_id": m.Oid, "SettleStatus": "waiting"}
	update := bson.M{
		"$set": bson.M{
			"SettleStatus": m.SettleStatus,
			"UpdateAt":     time.Now(),
		},
	}
	_, err = c.UpdateMany(ctx, find, update)
	return
}
