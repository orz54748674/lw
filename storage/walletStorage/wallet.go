package walletStorage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/mongo/readconcern"
	"vn/framework/mongo-driver/mongo/writeconcern"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/gate"
)

func OperateVndBalance(bill *Bill) error {
	b := *bill
	dbTransaction := common.NewDBTransaction()
	operateVndBalance := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		// err = operateWallet(sctx, b)
		err = newOperateWallet(sctx, b, true)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.")
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operateVndBalance); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}
func OperateAgentBalance(bill Bill) {
	dbTransaction := common.NewDBTransaction()
	operateVndBalance := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateAgentV1(sctx, bill)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.")
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operateVndBalance); err != nil {
		log.Error(err.Error())
	}
}

func operateAgentV1(m mongo.SessionContext, bill Bill) error {
	db := m.Client().Database(common.MongoConfig.DbName)
	c := db.Collection(CWallet)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	oid, _ := primitive.ObjectIDFromHex(bill.Uid)
	findM := bson.M{"_id": oid}
	updateM := bson.M{
		"$inc": bson.M{"AgentBalance": bill.Amount},
		"$set": bson.M{"UpdateAt": utils.Now()},
	}
	if _, err := c.UpdateOne(ctx, findM, updateM); err != nil {
		log.Error(err.Error())
		return err
	}
	wallet := QueryWallet(oid)
	bill.Balance = wallet.AgentBalance
	if err := parseBillV1(ctx, db, bill); err != nil {
		return err
	}
	b := bill
	common.ExecQueueFunc(func() {
		var q Wallet
		common.GetMysql().First(&q, "oid=?", b.Uid)
		q.Oid = utils.ConvertOID(b.Uid)
		q.UpdateAt = utils.Now()
		wallet := QueryWallet(q.Oid)
		q.VndBalance = wallet.VndBalance
		q.AgentBalance = wallet.AgentBalance
		common.GetMysql().Save(&q)
		common.GetMysql().Save(&b)
	})
	return nil
}

func operateAgent(m mongo.SessionContext, bill Bill) error {
	db := m.Client().Database(common.MongoConfig.DbName)
	c := db.Collection(CWallet)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	oid, _ := primitive.ObjectIDFromHex(bill.Uid)
	findM := bson.M{"_id": oid}
	updateM := bson.M{
		"$inc": bson.M{"AgentBalance": bill.Amount},
		"$set": bson.M{"UpdateAt": utils.Now()},
	}
	if _, err := c.UpdateOne(ctx, findM, updateM); err != nil {
		log.Error(err.Error())
		return err
	}
	wallet := QueryWallet(oid)
	bill.Balance = wallet.VndBalance
	if err := parseBill(ctx, db, bill); err != nil {
		return err
	}
	b := bill
	common.ExecQueueFunc(func() {
		var q Wallet
		common.GetMysql().First(&q, "oid=?", b.Uid)
		q.Oid = utils.ConvertOID(b.Uid)
		q.UpdateAt = utils.Now()
		wallet := QueryWallet(q.Oid)
		q.AgentBalance = wallet.AgentBalance
		common.GetMysql().Save(&q)
		var queryB Bill
		common.GetMysql().First(&queryB,
			"uid=? and type=? and event=? and event_id=?",
			b.Uid, b.Type, b.Event, b.EventId)
		b.ID = queryB.ID
		common.GetMysql().Save(&b)
	})
	return nil
}

func parseBill(ctx context.Context, db *mongo.Database, bill Bill) error {
	c := db.Collection(CBill)
	//opts := options.Update().SetUpsert(true)
	find := bson.M{
		"Uid":     bill.Uid,
		"Type":    bill.Type,
		"Event":   bill.Event,
		"EventId": bill.EventId,
	}
	cursor, err := c.Find(context.Background(), find)
	if err != nil {
		log.Error(err.Error())
	}
	var billOid primitive.ObjectID
	var queryBill []Bill
	err = cursor.All(context.Background(), &queryBill)
	if err != nil {
		return err
	}
	if len(queryBill) == 0 {
		updateResult, err := c.InsertOne(ctx, bill)
		if err != nil {
			log.Error(err.Error())
			return err
		} else {
			billOid = utils.ConvertOID(updateResult.InsertedID.(primitive.ObjectID).Hex())
		}
	} else {
		update := bson.M{"$inc": bson.M{"Amount": bill.Amount},
			"$set": bson.M{"UpdateAt": utils.Now(), "Balance": bill.Balance}}
		_, err := c.UpdateOne(ctx, find, update)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		billOid = queryBill[0].Oid
	}

	bill.Oid = billOid
	return nil
}
func NotifyUserWallet(uid string) {
	go func() {
		topic := game.Push
		wallet := QueryWallet(utils.ConvertOID(uid))
		msg := make(map[string]interface{})
		msg["Wallet"] = wallet
		msg["Action"] = "wallet"
		msg["GameType"] = game.All
		b, _ := json.Marshal(msg)
		sessionBean := gate.QuerySessionBean(uid)
		if sessionBean != nil {
			session, err := basegate.NewSession(common.App, sessionBean.Session)
			if err != nil {
				log.Error(err.Error())
			} else {
				if err := session.SendNR(topic, b); err != "" {
					log.Error(err)
				}
			}
		}
	}()
}
func AgentBalance2vnd(bill Bill) error {
	dbTransaction := common.NewDBTransaction()
	operateVndBalance := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = agentBalance2vnd(sctx, bill)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.")
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operateVndBalance); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}
func agentBalance2vnd(m mongo.SessionContext, bill Bill) error {
	db := m.Client().Database(common.MongoConfig.DbName)
	c := db.Collection(CWallet)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	oid, _ := primitive.ObjectIDFromHex(bill.Uid)
	findM := bson.M{"_id": oid}
	updateM := bson.M{
		"$inc": bson.M{"AgentBalance": bill.Amount, "VndBalance": -1 * bill.Amount},
		"$set": bson.M{"UpdateAt": utils.Now()},
	}
	if _, err := c.UpdateOne(ctx, findM, updateM); err != nil {
		log.Error(err.Error())
		return err
	}
	common.ExecQueueFunc(func() {
		var q Wallet
		common.GetMysql().First(&q, "oid=?", bill.Uid)
		q.Oid = utils.ConvertOID(bill.Uid)
		q.UpdateAt = utils.Now()
		wallet := QueryWallet(q.Oid)
		q.VndBalance = wallet.VndBalance
		q.AgentBalance = wallet.AgentBalance
		q.SafeBalance = wallet.SafeBalance
		common.GetMysql().Save(&q)
	})
	wallet := QueryWallet(oid)
	bill.Balance = wallet.AgentBalance
	c2 := db.Collection(CBill)
	if _, err := c2.InsertOne(ctx, bill); err != nil {
		return err
	}
	common.GetMysql().Create(&bill)
	bill.ID = 0
	bill.Oid = primitive.NewObjectID()
	bill.Amount = -1 * bill.Amount
	bill.Type = TypeIncome
	bill.Event = EventFromAgent
	bill.Balance = wallet.VndBalance + wallet.SafeBalance
	if _, err := c2.InsertOne(ctx, bill); err != nil {
		return err
	}
	common.GetMysql().Create(&bill)
	return nil
}
func SafeBalance2vnd(bill Bill) error {
	dbTransaction := common.NewDBTransaction()
	operateVndBalance := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = safeBalance2vnd(sctx, bill)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.")
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operateVndBalance); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}
func safeBalance2vnd(m mongo.SessionContext, bill Bill) error {
	db := m.Client().Database(common.MongoConfig.DbName)
	c := db.Collection(CWallet)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	oid, _ := primitive.ObjectIDFromHex(bill.Uid)
	findM := bson.M{"_id": oid}
	updateM := bson.M{
		"$inc": bson.M{"SafeBalance": bill.Amount, "VndBalance": -1 * bill.Amount},
		"$set": bson.M{"UpdateAt": utils.Now()},
	}
	if _, err := c.UpdateOne(ctx, findM, updateM); err != nil {
		log.Error(err.Error())
		return err
	}
	common.ExecQueueFunc(func() {
		var q Wallet
		common.GetMysql().First(&q, "oid=?", bill.Uid)
		q.Oid = utils.ConvertOID(bill.Uid)
		q.UpdateAt = utils.Now()
		wallet := QueryWallet(q.Oid)
		q.VndBalance = wallet.VndBalance
		q.AgentBalance = wallet.AgentBalance
		q.SafeBalance = wallet.SafeBalance
		common.GetMysql().Save(&q)
	})
	wallet := QueryWallet(oid)
	c2 := db.Collection(CBill)

	bill.ID = 0
	bill.Oid = primitive.NewObjectID()
	bill.Amount = -1 * bill.Amount
	if bill.Amount < 0{
		bill.Type = TypeIncome
	}else{
		bill.Type = TypeExpenses
	}
	bill.Event = EventSafeChange
	bill.Balance = wallet.VndBalance + wallet.SafeBalance
	if _, err := c2.InsertOne(ctx, bill); err != nil {
		return err
	}
	common.GetMysql().Create(&bill)
	return nil
}
// 余额不允许北扣成负数
func OperateVndBalanceV1(bill *Bill, cbs ...CallBack) (err error) {
	b := *bill
	dbTransaction := common.NewDBTransaction()
	operateVndBalance := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		db := sctx.Client().Database(common.MongoConfig.DbName)
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		err = newOperateWallet(sctx, b, false)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.")
			return err
		}
		for _, cb := range cbs {
			if err := cb.TransactionUnit(db, ctx); err != nil {
				_ = sctx.AbortTransaction(sctx)
				log.Info("caught exception during transaction, aborting.")
				return err
			}
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operateVndBalance); err != nil {
		log.Error(err.Error())
		return err
	}
	NotifyUserWallet(bill.Uid)
	return nil
}

// 允许余额成负数
func OperateVndBalanceV2(bill *Bill, cbs ...CallBack) (err error) {
	b := *bill
	dbTransaction := common.NewDBTransaction()
	operateVndBalance := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		db := sctx.Client().Database(common.MongoConfig.DbName)
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		err = newOperateWallet(sctx, b, true)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.")
			return err
		}
		for _, cb := range cbs {
			if err := cb.TransactionUnit(db, ctx); err != nil {
				_ = sctx.AbortTransaction(sctx)
				log.Info("caught exception during transaction, aborting.")
				return err
			}
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operateVndBalance); err != nil {
		log.Error(err.Error())
		return err
	}
	NotifyUserWallet(bill.Uid)
	return nil
}

func newOperateWallet(m mongo.SessionContext, bill Bill, noMoney bool) error {
	db := m.Client().Database(common.MongoConfig.DbName)
	c := db.Collection(CWallet)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	oid, _ := primitive.ObjectIDFromHex(bill.Uid)
	findM := bson.M{"_id": oid}
	if !noMoney {
		if bill.Amount < 0 {
			findM["VndBalance"] = bson.M{"$gte": utils.Abs(bill.Amount)}
		}
	}

	// log.Info("findM:%v", findM)
	updateM := bson.M{
		"$inc": bson.M{"VndBalance": bill.Amount},
		"$set": bson.M{"UpdateAt": utils.Now()},
	}
	uRet, err := c.UpdateOne(ctx, findM, updateM)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if uRet.ModifiedCount == 0 {
		err = fmt.Errorf("Not updated to wallet amount")
		log.Error(err.Error())
		return err
	}
	wallet := QueryWallet(oid)
	if wallet != nil {
		bill.Balance = wallet.VndBalance
	}
	if err := parseBillV1(ctx, db, bill); err != nil {
		return err
	}
	b := bill
	common.ExecQueueFunc(func() {
		var q Wallet
		common.GetMysql().First(&q, "oid=?", b.Uid)
		q.Oid = utils.ConvertOID(b.Uid)
		q.UpdateAt = utils.Now()
		wallet := QueryWallet(q.Oid)
		q.VndBalance = wallet.VndBalance
		q.AgentBalance = wallet.AgentBalance
		q.SafeBalance = wallet.SafeBalance
		common.GetMysql().Save(&q)
		common.GetMysql().Save(&b)
	})
	return nil
}

func parseBillV1(ctx context.Context, db *mongo.Database, bill Bill) error {
	c := db.Collection(CBill)
	//opts := options.Update().SetUpsert(true)
	var billOid primitive.ObjectID

	updateResult, err := c.InsertOne(ctx, bill)
	if err != nil {
		log.Error(err.Error())
		return err
	} else {
		billOid = utils.ConvertOID(updateResult.InsertedID.(primitive.ObjectID).Hex())
	}

	bill.Oid = billOid
	return nil
}

func operateWallet(m mongo.SessionContext, bill Bill) error {
	db := m.Client().Database(common.MongoConfig.DbName)
	c := db.Collection(CWallet)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	oid, _ := primitive.ObjectIDFromHex(bill.Uid)
	findM := bson.M{"_id": oid}
	updateM := bson.M{
		"$inc": bson.M{"VndBalance": bill.Amount},
		"$set": bson.M{"UpdateAt": utils.Now()},
	}
	if _, err := c.UpdateOne(ctx, findM, updateM); err != nil {
		log.Error(err.Error())
		return err
	}
	wallet := QueryWallet(oid)
	if wallet != nil {
		bill.Balance = wallet.VndBalance
	}
	if err := parseBill(ctx, db, bill); err != nil {
		return err
	}
	b := bill
	common.ExecQueueFunc(func() {
		var q Wallet
		common.GetMysql().First(&q, "oid=?", b.Uid)
		q.Oid = utils.ConvertOID(b.Uid)
		q.UpdateAt = utils.Now()
		wallet := QueryWallet(q.Oid)
		q.VndBalance = wallet.VndBalance
		common.GetMysql().Save(&q)
		var queryB Bill
		common.GetMysql().First(&queryB,
			"uid=? and type=? and event=? and event_id=?",
			b.Uid, b.Type, b.Event, b.EventId)
		b.ID = queryB.ID
		b.Amount += queryB.Amount
		common.GetMysql().Save(&b)
	})
	return nil
}

func operateWalletV1(db *mongo.Database, ctx context.Context, bill Bill) error {
	c := db.Collection(CWallet)
	oid, _ := primitive.ObjectIDFromHex(bill.Uid)
	findM := bson.M{"_id": oid}
	if bill.Amount < 0 {
		findM["VndBalance"] = bson.M{"$gte": utils.Abs(bill.Amount)}
	}
	log.Info("findM:%v", findM)
	updateM := bson.M{
		"$inc": bson.M{"VndBalance": bill.Amount},
		"$set": bson.M{"UpdateAt": utils.Now()},
	}
	uRet, err := c.UpdateOne(ctx, findM, updateM)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if uRet.ModifiedCount == 0 {
		err = fmt.Errorf("Not updated to wallet amount")
		log.Error(err.Error())
		return err
	}
	wallet := QueryWallet(oid)
	if wallet != nil {
		bill.Balance = wallet.VndBalance
	}
	if err := parseBillV1(ctx, db, bill); err != nil {
		return err
	}
	b := bill
	common.ExecQueueFunc(func() {
		var q Wallet
		common.GetMysql().First(&q, "oid=?", b.Uid)
		q.Oid = utils.ConvertOID(b.Uid)
		q.UpdateAt = utils.Now()
		wallet := QueryWallet(q.Oid)
		q.VndBalance = wallet.VndBalance
		common.GetMysql().Save(&q)
		common.GetMysql().Save(&b)
	})
	return nil
}

func operateWalletV2(m mongo.SessionContext, bill Bill) error {
	db := m.Client().Database(common.MongoConfig.DbName)
	c := db.Collection(CWallet)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	oid, _ := primitive.ObjectIDFromHex(bill.Uid)
	findM := bson.M{"_id": oid}
	updateM := bson.M{
		"$inc": bson.M{"VndBalance": bill.Amount},
		"$set": bson.M{"UpdateAt": utils.Now()},
	}
	if _, err := c.UpdateOne(ctx, findM, updateM); err != nil {
		log.Error(err.Error())
		return err
	}
	wallet := QueryWallet(oid)
	if wallet != nil {
		bill.Balance = wallet.VndBalance
	}
	if err := parseBill(ctx, db, bill); err != nil {
		return err
	}
	b := bill
	common.ExecQueueFunc(func() {
		var q Wallet
		common.GetMysql().First(&q, "oid=?", b.Uid)
		q.Oid = utils.ConvertOID(b.Uid)
		q.UpdateAt = utils.Now()
		wallet := QueryWallet(q.Oid)
		q.VndBalance = wallet.VndBalance
		common.GetMysql().Save(&q)
		var queryB Bill
		common.GetMysql().First(&queryB,
			"uid=? and type=? and event=? and event_id=?",
			b.Uid, b.Type, b.Event, b.EventId)
		b.ID = queryB.ID
		b.Amount += queryB.Amount
		common.GetMysql().Save(&b)
	})
	return nil
}
