package apiStorage

import (
	"context"
	"fmt"
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/log"
)

type SboFunc func(db *mongo.Database, ctx context.Context, m *SboBetRecord) (err error)

var (
	cSboBetRecord        = "SboBetRecord"
	AddSboBetRecord      = "addSboBetRecord"
	AddSboBetAmount      = "addSboBetAmount"
	SettleSboBetRecord   = "settleSboBetRecord"
	CancelSboBetRecord   = "cancelSboBetRecord"
	RollbackSboBetRecord = "rollbackSboBetRecord"
	SboFuncMap           = map[string]SboFunc{
		"addSboBetRecord":      addSboBetRecord,
		"settleSboBetRecord":   settleSboBetRecord,
		"cancelSboBetRecord":   cancelSboBetRecord,
		"rollbackSboBetRecord": rollbackSboBetRecord,
		"addSboBetAmount":      addSboBetAmount,
	}
)

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *SboBetRecord) SetName() string {
	return cSboBetRecord
}

func (m *SboBetRecord) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *SboBetRecord) TransferCodeExists() bool {
	record := &SboBetRecord{}
	if err := m.C().Find(bson.M{"TransferCode": m.TransferCode}).One(record); err == nil {
		return true
	} else if err == mongo.ErrNoDocuments {
		return false
	} else {
		log.Error("SboBetRecord TransferCodeIsExists Err：%s", err.Error())
		return true
	}
}

func (m *SboBetRecord) GetRecords(transferCode string) (res []*SboBetRecord, err error) {
	res = []*SboBetRecord{}
	err = m.C().Find(bson.M{"TransferCode": transferCode}).All(&res)
	return
}

/**
 *  @title rollbackSboBetRecord
 *  @description	新增记录
 */
func (m *SboBetRecord) RollbackSboBetRecord() (err error) {
	c := m.C()
	find := bson.M{"TransferCode": m.TransferCode, "ProductType": m.ProductType, "GameType": m.GameType, "Status": "settled"}
	update := bson.M{
		"$set": bson.M{
			"Status":   "running",
			"UpdateAt": time.Now(),
			"WinLoss":  0,
			"Rollback": true,
		},
	}
	_, err = c.UpdateMany(context.Background(), find, update)
	return
}

func (m *SboBetRecord) GetStatus(pType, gType int, tCode, tId string) (record *SboBetRecord, err error) {
	record = &SboBetRecord{}
	find := bson.M{"TransferCode": tCode, "TransactionId": tId, "ProductType": pType, "GameType": gType}
	log.Debug("%v", find)
	err = m.C().Find(find).One(record)
	return
}

func (m *SboBetRecord) GetTransferCodeStatus(tCode string) (record *SboBetRecord, err error) {
	record = &SboBetRecord{}
	find := bson.M{"TransferCode": tCode}
	log.Debug("%v", find)
	err = m.C().Find(find).One(record)
	return
}

func (m *SboBetRecord) GetRecord(tCode, tId string) (record *SboBetRecord, err error) {
	record = &SboBetRecord{}
	find := bson.M{"TransferCode": tCode, "TransactionId": tId}
	log.Debug("%v", find)
	err = m.C().Find(find).One(record)
	return
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *SboBetRecord) SetTransactionUnits(transactionUnits ...string) {
	m.transactionUnits = transactionUnits
}

/**
 *  @title TransactionUnit
 *  @description 事务操作单元
 */
func (m *SboBetRecord) TransactionUnit(db *mongo.Database, ctx context.Context) (err error) {
	for _, funcKey := range m.transactionUnits {
		Func, ok := SboFuncMap[funcKey]
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
 *  @title addSboBetRecord
 *  @description	新增记录
 */
func addSboBetRecord(db *mongo.Database, ctx context.Context, m *SboBetRecord) (err error) {
	m.CreateAt = time.Now()
	m.UpdateAt = time.Now()
	c := db.Collection(m.SetName())

	_, err = c.InsertOne(ctx, m)
	return
}

/**
 *  @title addSboBetRecord
 *  @description	新增记录
 */
func addSboBetAmount(db *mongo.Database, ctx context.Context, m *SboBetRecord) (err error) {
	c := db.Collection(m.SetName())
	find := bson.M{"TransferCode": m.TransferCode, "ProductType": m.ProductType, "GameType": m.GameType}
	update := bson.M{
		"$set": bson.M{
			"Amount":   m.Amount,
			"UpdateAt": time.Now(),
		},
	}
	_, err = c.UpdateOne(ctx, find, update)
	return
}

/**
 *  @title settleSboBetRecord
 *  @description	新增记录
 */
func settleSboBetRecord(db *mongo.Database, ctx context.Context, m *SboBetRecord) (err error) {
	c := db.Collection(m.SetName())
	find := bson.M{"TransferCode": m.TransferCode, "ProductType": m.ProductType, "GameType": m.GameType, "Status": "running"}
	update := bson.M{
		"$set": bson.M{
			"ResultType":      m.ResultType,
			"ResultTime":      m.ResultTime,
			"Status":          "settled",
			"CommissionStake": m.CommissionStake,
			"GameResult":      m.GameResult,
			"UpdateAt":        time.Now(),
			"WinLoss":         m.WinLoss,
		},
	}
	_, err = c.UpdateMany(ctx, find, update)
	return
}

/**
 *  @title cancelSboBetRecord
 *  @description	新增记录
 */
func cancelSboBetRecord(db *mongo.Database, ctx context.Context, m *SboBetRecord) (err error) {
	c := db.Collection(m.SetName())
	find := bson.M{"TransferCode": m.TransferCode, "ProductType": m.ProductType, "GameType": m.GameType}
	if !m.IsCancelAll {
		find["TransactionId"] = m.TransactionId
	}
	update := bson.M{
		"$set": bson.M{
			"Status":             "void",
			"CancelBeforeStatus": m.Status,
			"IsCancelAll":        m.IsCancelAll,
			"UpdateAt":           time.Now(),
		},
	}
	_, err = c.UpdateMany(ctx, find, update)
	return
}

/**
 *  @title rollbackSboBetRecord
 *  @description	新增记录
 */
func rollbackSboBetRecord(db *mongo.Database, ctx context.Context, m *SboBetRecord) (err error) {
	c := db.Collection(m.SetName())
	find := bson.M{"TransferCode": m.TransferCode, "ProductType": m.ProductType, "GameType": m.GameType, "Rollback": false}
	update := bson.M{
		"$set": bson.M{
			"Status":   "running",
			"UpdateAt": time.Now(),
			"WinLoss":  0,
			"Rollback": true,
		},
	}
	_, err = c.UpdateMany(ctx, find, update)
	return
}
