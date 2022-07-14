package apiCmdStorage

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/mongo/readconcern"
	"vn/framework/mongo-driver/mongo/writeconcern"
	"vn/framework/mqant/log"
	"vn/storage/walletStorage"
)

var (
	cCqMtcode = "cqMtcode"
	cCqRecord = "cqRecord"
)

func insertBetRecord(records *BetRecords) error {
	c := common.GetMongoDB().C(cCqRecord)
	if err := c.Insert(records); err != nil {
		log.Info("Insert cq record error: %s", err)
		return err
	}
	return nil
}

func insertMTCode(mtcode *MTCode) error {
	c := common.GetMongoDB().C(cCqMtcode)
	if err := c.Insert(mtcode); err != nil {
		log.Info("Insert cq mtcode error: %s", err)
		return err
	}
	return nil
}

func ConfirmNoThisMtcode(mtcode string) error {
	c := common.GetMongoDB().C(cCqMtcode)
	query := bson.M{"mtcode": mtcode}
	var mtcodeMsg MTCode
	err := c.Find(query).One(&mtcodeMsg)
	if err == nil {
		err = fmt.Errorf("2009")
		return err
	}
	if err != mongo.ErrNoDocuments {
		err = fmt.Errorf("1100")
		return err
	}
	return nil
}

func operateBatchBets(m mongo.SessionContext, bets Bets, uid string) error {
	var beforeBalance int64
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	beforeBalance = wallet.VndBalance

	for _, v := range bets.Data {
		var err error

		if v.Amount < 0 {
			err = fmt.Errorf("%s", "1003")
			return err
		}

		if err = ConfirmNoThisMtcode(v.MTCode); err != nil {
			return err
		}

		bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventGameSport, "sport-bet-"+strconv.Itoa(rand.Intn(1000000)), -v.Amount)
		err = walletStorage.OperateVndBalanceV1(bill)
		if err != nil {
			return err
		}
	}
	var balance int64
	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance = wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
	betRecords.Data._Id = betRecords.Oid.String()
	betRecords.Data.Action = "bets"
	betRecords.Data.Before = beforeBalance
	betRecords.Data.Balance = balance
	betRecords.Data.Currency = "VND"
	betRecords.Data.Target.Account = bets.Account
	betRecords.Data.Status.CreateTime = bets.CreateTime
	betRecords.Data.Status.Status = "success"
	betRecords.Data.Status.Message = "success"
	betRecords.Data.Status.EndTime = time.Now().Format(time.RFC3339)
	for _, v := range bets.Data {
		var event EventInfo
		event.Amount = v.Amount
		event.EventTime = v.EventTime
		event.Mtcode = v.MTCode
		betRecords.Data.Events = append(betRecords.Data.Events, event)
	}
	betRecords.Status.Message = "success"
	betRecords.Status.Code = "0"
	betRecords.Status.Datatime = time.Now().Format(time.RFC3339)

	err := insertBetRecord(&betRecords)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	for _, v := range bets.Data {
		var tmpMTCode MTCode
		tmpMTCode.Account = bets.Account
		tmpMTCode.RecordOid = betRecords.Oid.String()
		tmpMTCode.Mtcode = v.MTCode
		err = insertMTCode(&tmpMTCode)
		if err != nil {
			err = fmt.Errorf("1100")
			return err
		}
	}

	return nil
}

func BatchBetsHandle(bets Bets, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operateBets := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateBatchBets(sctx, bets, uid)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.", err.Error())
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operateBets); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func GetMTCodeData(mtcode string) (err error, data MTCode) {
	cMTCode := common.GetMongoDB().C(cCqMtcode)
	query := bson.M{"mtcode": mtcode}
	err = cMTCode.Find(query).One(&mtcode)
	if err != nil {
		err = fmt.Errorf("1014")
	}
	return err, data
}

func operateRefunds(m mongo.SessionContext, mtcodes []string, uid string) error {
	var err error
	cMTCode := common.GetMongoDB().C(cCqMtcode)
	cRecords := common.GetMongoDB().C(cCqRecord)
	for _, v := range mtcodes {
		query := bson.M{"mtcode": v}
		var mtcode MTCode
		err = cMTCode.Find(query).One(&mtcode)
		if err != nil || &mtcode == nil {
			err = fmt.Errorf("1014")
			return err
		}

		var betRecord BetRecords
		var amount int64
		query = bson.M{"recordOid": mtcode.RecordOid}
		err = cRecords.Find(query).One(&betRecord)
		for _, v := range betRecord.Data.Events {
			if v.Status != "" {
				err = fmt.Errorf("1015")
				return err
			}
			amount = v.Amount
			v.Status = "refund"
		}

		bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventGameSport, "sport-refund-"+strconv.Itoa(rand.Intn(1000000)), amount)
		err = walletStorage.OperateVndBalanceV1(bill)
		if err != nil {
			return err
		}

		err = cRecords.Update(query, &betRecord)
		if err != nil {
			return err
		}
	}
	return nil
}

func RefundsHandle(mtcodes []string, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operateBets := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateRefunds(sctx, mtcodes, uid)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.")
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operateBets); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func operateCancel(m mongo.SessionContext, mtcodes []string, uid string) error {
	var err error
	cMTCode := common.GetMongoDB().C(cCqMtcode)
	cRecords := common.GetMongoDB().C(cCqRecord)
	for _, v := range mtcodes {
		query := bson.M{"mtcode": v}
		var mtcode MTCode
		err = cMTCode.Find(query).One(&mtcode)
		if err != nil || &mtcode == nil {
			err = fmt.Errorf("1014")
			return err
		}

		var betRecord BetRecords
		var amount int64
		query = bson.M{"recordOid": mtcode.RecordOid}
		err = cRecords.Find(query).One(&betRecord)
		for _, v := range betRecord.Data.Events {
			if v.Status != "refund" {
				err = fmt.Errorf("1015")
				return err
			}
			amount = v.Amount
			v.Status = "cancel"
		}

		bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventGameSport, "sport-cancel-"+strconv.Itoa(rand.Intn(1000000)), -amount)
		err = walletStorage.OperateVndBalanceV1(bill)
		if err != nil {
			return err
		}

		err = cRecords.Update(query, &betRecord)
		if err != nil {
			return err
		}
	}
	return nil
}

func CancelHandle(mtcodes []string, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operateBets := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateCancel(sctx, mtcodes, uid)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.")
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operateBets); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func operateAmend(m mongo.SessionContext, amend Amend, uid string) error {
	var beforeBalance int64
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	beforeBalance = wallet.VndBalance

	var err error
	if amend.Action == "debit" && beforeBalance < amend.Amount {
		err = fmt.Errorf("1005")
	}

	for _, v := range amend.Data {
		if v.Amount < 0 {
			err = fmt.Errorf("1003")
			return err
		}

		c := common.GetMongoDB().C(cCqMtcode)
		query := bson.M{"mtcode": v.MTCode}
		var mtcode MTCode
		err = c.Find(query).One(&mtcode)
		if err != nil || &mtcode != nil {
			err = fmt.Errorf("2009")
			return err
		}
	}
	billType := walletStorage.TypeIncome
	amount := amend.Amount
	if amend.Action == "debit" {
		billType = walletStorage.TypeExpenses
		amount = -amount
	}
	bill := walletStorage.NewBill(uid, billType, walletStorage.EventGameSport, "sport-bet-"+strconv.Itoa(rand.Intn(1000000)), amount)
	err = walletStorage.OperateVndBalanceV1(bill)
	if err != nil {
		fmt.Errorf("1100")
		return err
	}

	var balance int64
	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance = wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
	betRecords.Data._Id = betRecords.Oid.String()
	betRecords.Data.Action = "amend"
	betRecords.Data.Before = beforeBalance
	betRecords.Data.Balance = balance
	betRecords.Data.Currency = "VND"
	betRecords.Data.Target.Account = amend.Account
	betRecords.Data.Status.CreateTime = amend.CreateTime
	betRecords.Data.Status.Status = "success"
	betRecords.Data.Status.Message = "success"
	betRecords.Data.Status.EndTime = time.Now().Format(time.RFC3339)
	for _, v := range amend.Data {
		var event EventInfo
		event.Amount = v.Amount
		event.EventTime = v.EventTime
		event.Mtcode = v.MTCode
		event.Action = amend.Action
		betRecords.Data.Events = append(betRecords.Data.Events, event)
	}
	betRecords.Status.Message = "success"
	betRecords.Status.Code = "0"
	betRecords.Status.Datatime = time.Now().Format(time.RFC3339)

	err = insertBetRecord(&betRecords)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	return nil
}
