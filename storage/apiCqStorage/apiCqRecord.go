package apiCqStorage

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
	"vn/game"
	"vn/storage/gameStorage"
	"vn/storage/walletStorage"
)

var (
	cCqMtcode      = "cqMtcode"
	cCqRecord      = "cqRecord"
	ApiType   int8 = 2
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

func InsertGameRecord(uid, gameNo, betDetails, gameResult string, betAmount, curBalance, inCome int64, isSettled bool) {
	var recordParams gameStorage.BetRecordParam
	recordParams.Uid = uid
	recordParams.GameNo = gameNo
	recordParams.BetAmount = betAmount
	recordParams.BotProfit = 0
	recordParams.SysProfit = 0
	recordParams.BetDetails = betDetails
	recordParams.GameResult = gameResult
	recordParams.CurBalance = curBalance
	recordParams.GameType = game.ApiCq
	recordParams.Income = inCome
	recordParams.IsSettled = isSettled
	gameStorage.InsertBetRecord(recordParams)
}

func operateBatchBets(m mongo.SessionContext, bets Bets, uid string) error {
	var beforeBalance int64
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	beforeBalance = wallet.VndBalance

	for _, v := range bets.Data {
		bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventGameSport, "sport-bet-"+strconv.Itoa(rand.Intn(1000000)), -int64(v.Amount))
		err := walletStorage.OperateVndBalanceV1(bill)
		if err != nil {
			err = fmt.Errorf("%s", "1005")
			return err
		}
	}
	var balance int64
	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance = wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
	betRecords.Data.DataID = betRecords.Oid.Hex()
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
		event.Amount = int64(v.Amount)
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
		tmpMTCode.RecordOid = betRecords.Oid.Hex()
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

func operateBet(bet Bet, uid string) error {
	var beforeBalance int64
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	beforeBalance = wallet.VndBalance

	bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventGameCq9, "cq9-bet-"+strconv.Itoa(rand.Intn(1000000)), -int64(bet.Amount))
	err := walletStorage.OperateVndBalanceV1(bill)
	if err != nil {
		err = fmt.Errorf("%s", "1005")
		return err
	}
	var balance int64
	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance = wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
	betRecords.Data.DataID = betRecords.Oid.Hex()
	betRecords.Data.Action = "bet"
	betRecords.Data.Before = beforeBalance
	betRecords.Data.Balance = balance
	betRecords.Data.Currency = "VND"
	betRecords.Data.Target.Account = bet.Account
	betRecords.Data.Status.CreateTime = time.Now().Format(time.RFC3339)
	betRecords.Data.Status.Status = "success"
	betRecords.Data.Status.Message = "success"
	betRecords.Data.Status.EndTime = time.Now().Format(time.RFC3339)
	var event EventInfo
	event.Amount = int64(bet.Amount)
	event.EventTime = bet.EventTime
	event.Mtcode = bet.Mtcode
	betRecords.Data.Events = append(betRecords.Data.Events, event)
	betRecords.Status.Message = "success"
	betRecords.Status.Code = "0"
	betRecords.Status.Datatime = time.Now().Format(time.RFC3339)

	err = insertBetRecord(&betRecords)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	var tmpMTCode MTCode
	tmpMTCode.Account = bet.Account
	tmpMTCode.Mtcode = bet.Mtcode
	tmpMTCode.RecordOid = betRecords.Oid.Hex()
	err = insertMTCode(&tmpMTCode)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	return nil
}

func BetHandle(bet Bet, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operate := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateBet(bet, uid)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.", err.Error())
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operate); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func operateRollout(rollout Rollout, uid string) error {
	var beforeBalance int64
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	beforeBalance = wallet.VndBalance

	bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventGameCq9, "cq9-rollout-"+strconv.Itoa(rand.Intn(1000000)), -int64(rollout.Amount))
	err := walletStorage.OperateVndBalanceV1(bill)
	if err != nil {
		err = fmt.Errorf("%s", "1005")
		return err
	}
	var balance int64
	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance = wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
	betRecords.Data.DataID = betRecords.Oid.Hex()
	betRecords.Data.Action = "rollout"
	betRecords.Data.Before = beforeBalance
	betRecords.Data.Balance = balance
	betRecords.Data.Currency = "VND"
	betRecords.Data.Target.Account = rollout.Account
	betRecords.Data.Status.CreateTime = time.Now().Format(time.RFC3339)
	betRecords.Data.Status.Status = "success"
	betRecords.Data.Status.Message = "success"
	betRecords.Data.Status.EndTime = time.Now().Format(time.RFC3339)
	var event EventInfo
	event.Amount = int64(rollout.Amount)
	event.EventTime = rollout.EventTime
	event.Mtcode = rollout.Mtcode
	betRecords.Data.Events = append(betRecords.Data.Events, event)
	betRecords.Status.Message = "success"
	betRecords.Status.Code = "0"
	betRecords.Status.Datatime = time.Now().Format(time.RFC3339)

	err = insertBetRecord(&betRecords)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	var tmpMTCode MTCode
	tmpMTCode.Account = rollout.Account
	tmpMTCode.Mtcode = rollout.Mtcode
	tmpMTCode.RecordOid = betRecords.Oid.Hex()
	err = insertMTCode(&tmpMTCode)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	return nil
}

func RolloutHandle(rollout Rollout, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operate := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateRollout(rollout, uid)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.", err.Error())
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operate); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func operateRollin(amount int64, mtcode, account, uid, eventTime, actionType string) error {
	var beforeBalance int64
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	beforeBalance = wallet.VndBalance

	bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventGameCq9, "cq9-"+actionType+"-"+strconv.Itoa(rand.Intn(1000000)), amount)
	err := walletStorage.OperateVndBalanceV1(bill)
	if err != nil {
		err = fmt.Errorf("%s", "1005")
		return err
	}

	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance := wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
	betRecords.Data.DataID = betRecords.Oid.Hex()
	betRecords.Data.Action = actionType
	betRecords.Data.Before = beforeBalance
	betRecords.Data.Balance = balance
	betRecords.Data.Currency = "VND"
	betRecords.Data.Target.Account = account
	betRecords.Data.Status.CreateTime = time.Now().Format(time.RFC3339)
	betRecords.Data.Status.Status = "success"
	betRecords.Data.Status.Message = "success"
	betRecords.Data.Status.EndTime = time.Now().Format(time.RFC3339)
	var event EventInfo
	event.Amount = beforeBalance
	event.EventTime = eventTime
	event.Mtcode = mtcode
	betRecords.Data.Events = append(betRecords.Data.Events, event)
	betRecords.Status.Message = "success"
	betRecords.Status.Code = "0"
	betRecords.Status.Datatime = time.Now().Format(time.RFC3339)

	err = insertBetRecord(&betRecords)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	var tmpMTCode MTCode
	tmpMTCode.Account = account
	tmpMTCode.Mtcode = mtcode
	tmpMTCode.RecordOid = betRecords.Oid.Hex()
	err = insertMTCode(&tmpMTCode)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	return nil
}

func RollinHandle(amount int64, mtcode, account, uid, eventTime, actionType string) error {
	dbTransaction := common.NewDBTransaction()
	operate := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateRollin(amount, mtcode, account, uid, eventTime, actionType)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.", err.Error())
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operate); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func operateTakeAll(takeall TakeAll, uid string) error {
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	beforeBalance := wallet.VndBalance
	bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventGameCq9, "cq9-takeall-"+strconv.Itoa(rand.Intn(1000000)), -int64(wallet.VndBalance))
	err := walletStorage.OperateVndBalanceV1(bill)
	if err != nil {
		err = fmt.Errorf("%s", "1005")
		return err
	}

	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance := wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
	betRecords.Data.DataID = betRecords.Oid.Hex()
	betRecords.Data.Action = "rollout"
	betRecords.Data.Before = beforeBalance
	betRecords.Data.Balance = balance
	betRecords.Data.Currency = "VND"
	betRecords.Data.Target.Account = takeall.Account
	betRecords.Data.Status.CreateTime = time.Now().Format(time.RFC3339)
	betRecords.Data.Status.Status = "success"
	betRecords.Data.Status.Message = "success"
	betRecords.Data.Status.EndTime = time.Now().Format(time.RFC3339)
	var event EventInfo
	event.Amount = beforeBalance
	event.EventTime = takeall.EventTime
	event.Mtcode = takeall.Mtcode
	betRecords.Data.Events = append(betRecords.Data.Events, event)
	betRecords.Status.Message = "success"
	betRecords.Status.Code = "0"
	betRecords.Status.Datatime = time.Now().Format(time.RFC3339)

	err = insertBetRecord(&betRecords)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	var tmpMTCode MTCode
	tmpMTCode.Account = takeall.Account
	tmpMTCode.Mtcode = takeall.Mtcode
	tmpMTCode.RecordOid = betRecords.Oid.Hex()
	err = insertMTCode(&tmpMTCode)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	return nil
}

func TakeAllHandle(takeall TakeAll, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operate := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateTakeAll(takeall, uid)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.", err.Error())
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operate); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func operateEndRound(endRound EndRound, uid string) error {
	var beforeBalance int64
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	beforeBalance = wallet.VndBalance

	for _, v := range endRound.Data {
		bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventGameSport, "cq9-endround-"+strconv.Itoa(rand.Intn(1000000)), int64(v.Amount))
		err := walletStorage.OperateVndBalanceV1(bill)
		if err != nil {
			err = fmt.Errorf("%s", "1005")
			return err
		}
	}
	var balance int64
	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance = wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
	betRecords.Data.DataID = betRecords.Oid.Hex()
	betRecords.Data.Action = "endround"
	betRecords.Data.Before = beforeBalance
	betRecords.Data.Balance = balance
	betRecords.Data.Currency = "VND"
	betRecords.Data.Target.Account = endRound.Account
	betRecords.Data.Status.CreateTime = endRound.CreateTime
	betRecords.Data.Status.Status = "success"
	betRecords.Data.Status.Message = "success"
	betRecords.Data.Status.EndTime = time.Now().Format(time.RFC3339)
	for _, v := range endRound.Data {
		var event EventInfo
		event.Amount = int64(v.Amount)
		event.EventTime = v.EventTime
		event.Mtcode = v.Mtcode
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

	for _, v := range endRound.Data {
		var tmpMTCode MTCode
		tmpMTCode.Account = endRound.Account
		tmpMTCode.RecordOid = betRecords.Oid.Hex()
		tmpMTCode.Mtcode = v.Mtcode
		err = insertMTCode(&tmpMTCode)
		if err != nil {
			err = fmt.Errorf("1100")
			return err
		}
	}

	return nil
}

func EndRoundHandle(endRound EndRound, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operate := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateEndRound(endRound, uid)
		if err != nil {
			_ = sctx.AbortTransaction(sctx)
			log.Info("caught exception during transaction, aborting.", err.Error())
			return err
		}
		return d.Commit(sctx)
	}
	if err := dbTransaction.Exec(common.GetMongoDB().Client(), operate); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func GetMTCodeData(mtcode string) (err error, data MTCode) {
	cMTCode := common.GetMongoDB().C(cCqMtcode)
	query := bson.M{"mtcode": mtcode}
	err = cMTCode.Find(query).One(&data)
	if err != nil {
		err = fmt.Errorf("1014")
	}
	return err, data
}

func operateRefunds(m mongo.SessionContext, mtcodes []string, uid string) error {
	cRecords := common.GetMongoDB().C(cCqRecord)
	for _, v := range mtcodes {
		err, _ := GetMTCodeData(v)
		if err != nil {
			log.Error("operateRefunds GetMTCodeData err:%s", err.Error())
			err = fmt.Errorf("1014")
			return err
		}

		betRecord, _ := GetRecordByMTCode(v)
		var amount int64
		for k, val := range betRecord.Data.Events {
			if val.Mtcode == v {
				if val.Status == "refund" {
					err = fmt.Errorf("1015")
					return err
				} else {
					amount = val.Amount
					betRecord.Data.Events[k].Status = "refund"
				}
			}
		}

		bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventGameSport, "sport-refund-"+strconv.Itoa(rand.Intn(1000000)), amount)
		err = walletStorage.OperateVndBalanceV1(bill)
		if err != nil {
			return err
		}

		query := bson.M{"_id": betRecord.Oid}
		err = cRecords.Update(query, &betRecord)
		if err != nil {
			return err
		}
	}
	return nil
}

func operateRefund(mtcode string, uid string) error {
	cRecords := common.GetMongoDB().C(cCqRecord)
	err, _ := GetMTCodeData(mtcode)
	if err != nil {
		log.Error("operateRefunds GetMTCodeData err:%s", err.Error())
		err = fmt.Errorf("1014")
		return err
	}

	betRecord, _ := GetRecordByMTCode(mtcode)
	var amount int64
	for k, val := range betRecord.Data.Events {
		if val.Mtcode == mtcode {
			if val.Status == "refund" {
				err = fmt.Errorf("1015")
				return err
			} else {
				amount = val.Amount
				betRecord.Data.Events[k].Status = "refund"
			}
		}
	}

	bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventGameSport, "cq9-refund-"+strconv.Itoa(rand.Intn(1000000)), amount)
	err = walletStorage.OperateVndBalanceV1(bill)
	if err != nil {
		return err
	}

	query := bson.M{"_id": betRecord.Oid}
	err = cRecords.Update(query, &betRecord)
	if err != nil {
		return err
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

func RefundHandle(mtcode string, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operateBets := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateRefund(mtcode, uid)
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
	cRecords := common.GetMongoDB().C(cCqRecord)
	for _, v := range mtcodes {
		err, _ := GetMTCodeData(v)
		if err != nil {
			err = fmt.Errorf("1014")
			return err
		}

		betRecord, _ := GetRecordByMTCode(v)
		var amount int64
		for k, val := range betRecord.Data.Events {
			if val.Mtcode == v {
				if val.Status != "refund" {
					err = fmt.Errorf("1015")
					return err
				} else {
					amount = val.Amount
					betRecord.Data.Events[k].Status = "cancel"
				}
			}
		}

		bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventGameSport, "sport-cancel-"+strconv.Itoa(rand.Intn(1000000)), -amount)
		err = walletStorage.OperateVndBalanceV1(bill)
		if err != nil {
			return err
		}

		query := bson.M{"_id": betRecord.Oid}
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
	if amend.Action == "debit" && beforeBalance < int64(amend.Amount) {
		err = fmt.Errorf("1005")
	}

	billType := walletStorage.TypeIncome
	amount := int64(amend.Amount)
	if amend.Action == "debit" {
		billType = walletStorage.TypeExpenses
		amount = -amount
	}
	bill := walletStorage.NewBill(uid, billType, walletStorage.EventGameSport, "sport-bet-"+strconv.Itoa(rand.Intn(1000000)), amount)
	err = walletStorage.OperateVndBalanceV1(bill)
	if err != nil {
		err = fmt.Errorf("1005")
		return err
	}

	var balance int64
	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance = wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
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
		event.Amount = int64(v.Amount)
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

	for _, v := range amend.Data {
		var tmpMTCode MTCode
		tmpMTCode.Account = amend.Account
		tmpMTCode.RecordOid = betRecords.Oid.Hex()
		tmpMTCode.Mtcode = v.MTCode
		err = insertMTCode(&tmpMTCode)
		if err != nil {
			err = fmt.Errorf("1100")
			return err
		}
	}

	return nil
}

func AmendHandle(amend Amend, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operateBets := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateAmend(sctx, amend, uid)
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

func operateWinsPlayer(m mongo.SessionContext, data WinsData, uid string) error {
	var beforeBalance int64
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	beforeBalance = wallet.VndBalance
	var err error
	var eventList []EventInfo
	for _, v := range data.Event {
		var event EventInfo
		event.Amount = int64(v.Amount)
		event.EventTime = v.EventTime
		event.Mtcode = v.MTCode

		bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventGameSport, "sport-wins-"+strconv.Itoa(rand.Intn(1000000)), int64(v.Amount))
		err = walletStorage.OperateVndBalanceV1(bill)
		if err != nil {
			err = fmt.Errorf("1100")
			return err
		}
		eventList = append(eventList, event)
	}

	var balance int64
	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance = wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
	betRecords.Data.DataID = betRecords.Oid.Hex()
	betRecords.Data.Action = "wins"
	betRecords.Data.Before = beforeBalance
	betRecords.Data.Balance = balance
	betRecords.Data.Currency = "VND"
	betRecords.Data.Target.Account = data.Account
	betRecords.Data.Status.CreateTime = data.EventTime
	betRecords.Data.Status.Status = "success"
	betRecords.Data.Status.Message = "success"
	betRecords.Data.Status.EndTime = time.Now().Format(time.RFC3339)
	betRecords.Data.Events = eventList
	betRecords.Status.Message = "success"
	betRecords.Status.Code = "0"
	betRecords.Status.Datatime = time.Now().Format(time.RFC3339)

	if err = insertBetRecord(&betRecords); err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	for _, v := range eventList {
		var tmpMTCode MTCode
		tmpMTCode.Account = data.Account
		tmpMTCode.RecordOid = betRecords.Oid.Hex()
		tmpMTCode.Mtcode = v.Mtcode
		err = insertMTCode(&tmpMTCode)
		if err != nil {
			err = fmt.Errorf("1100")
			return err
		}
	}

	return nil
}

func WinsPlayerHandle(data WinsData, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operateBets := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateWinsPlayer(sctx, data, uid)
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

func CheckParams(paramList []string, paramMap map[string]interface{}) (errCode string, err error) {
	for _, v := range paramList {
		if _, ok := paramMap[v]; !ok {
			return "1003", fmt.Errorf("重要参数缺失！%s不能为空", v)
		}
		if v == "amount" {
			if int64(paramMap[v].(float64)) < 0 {
				return "1003", fmt.Errorf("金额不能为负")
			}
		}
		if v == "eventtime" || v == "createTime" {
			if _, err = time.ParseInLocation(time.RFC3339, paramMap[v].(string), time.Local); err != nil {
				return "1004", fmt.Errorf("时间戳解析失败！时：%s", paramMap[v].(string))
			}
		}
	}
	return "", nil
}

func WinsHandle(wins Wins, paramMap map[string]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	type failMsg struct {
		Account string `json:"account"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Ucode   string `json:"ucode"`
	}
	type succMsg struct {
		Account  string  `json:"account"`
		Currency string  `json:"currency"`
		Balance  float64 `json:"balance"`
		Ucode    string  `json:"ucode"`
	}
	var failArr []failMsg
	var succArr []succMsg

	statusMap["code"] = "0"
	statusMap["message"] = "success"
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	tmpList := paramMap["list"].([]interface{})
	for k, data := range wins.List {
		tmpMap := tmpList[k].(map[string]interface{})
		if errCode, err := CheckParams([]string{"account", "eventtime", "ucode"}, tmpMap); err != nil {
			var tmpFail failMsg
			tmpFail.Account = data.Account
			tmpFail.Code = errCode
			tmpFail.Message = err.Error()
			tmpFail.Ucode = data.UCode
			failArr = append(failArr, tmpFail)
			continue
		}
		tmpEvent := tmpMap["event"].([]interface{})
		sucFlg := true
		for _, v := range tmpEvent {
			aa := v.(map[string]interface{})
			if errCode, err := CheckParams([]string{"mtcode", "amount", "validbet", "roundid", "eventtime", "gamecode", "gamehall"}, aa); err != nil {
				var tmpFail failMsg
				tmpFail.Account = data.Account
				tmpFail.Code = errCode
				tmpFail.Message = err.Error()
				tmpFail.Ucode = data.UCode
				failArr = append(failArr, tmpFail)
				sucFlg = false
				break
			}
			if err := ConfirmNoThisMtcode(aa["mtcode"].(string)); err != nil {
				var tmpFail failMsg
				tmpFail.Account = data.Account
				tmpFail.Code = "2009"
				tmpFail.Message = "混合码已经存在了！"
				tmpFail.Ucode = data.UCode
				failArr = append(failArr, tmpFail)
				sucFlg = false
				break
			}
		}
		if !sucFlg {
			continue
		}

		apiUser, existFlg := FindUserByAccount(data.Account, ApiType)
		if !existFlg {
			var tmpFail failMsg
			tmpFail.Account = data.Account
			tmpFail.Code = "1006"
			tmpFail.Message = data.Account + " not found"
			tmpFail.Ucode = data.UCode
			failArr = append(failArr, tmpFail)
			continue
		}

		err := WinsPlayerHandle(data, apiUser.Uid)
		if err != nil {
			var tmpFail failMsg
			tmpFail.Account = data.Account
			tmpFail.Code = err.Error()
			tmpFail.Message = "fail"
			tmpFail.Ucode = data.UCode
			failArr = append(failArr, tmpFail)
		} else {
			var balance int64
			wallet := walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
			balance = wallet.VndBalance
			var tmpSucc succMsg
			tmpSucc.Ucode = data.UCode
			tmpSucc.Account = data.Account
			tmpSucc.Currency = "VND"
			tmpSucc.Balance = float64(balance)
			for _, v := range data.Event {
				tmpSucc.Balance = tmpSucc.Balance + v.Amount - float64(int64(v.Amount))
			}
			succArr = append(succArr, tmpSucc)
		}
	}

	dataMap["failed"] = failArr
	dataMap["success"] = succArr
	res["data"] = dataMap
	res["status"] = statusMap
	return res
}

func operateAmendsPlayer(m mongo.SessionContext, data AmendsData, uid string) error {
	var beforeBalance int64
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	beforeBalance = wallet.VndBalance

	var eventList []EventInfo
	var err error
	for _, v := range data.Event {
		var event EventInfo
		event.Amount = int64(v.Amount)
		event.EventTime = v.EventTime
		event.Mtcode = v.MTCode

		billType := walletStorage.TypeIncome
		amount := int64(v.Amount)
		if v.Action == "debit" {
			billType = walletStorage.TypeExpenses
			amount = -amount
		}

		bill := walletStorage.NewBill(uid, billType, walletStorage.EventGameSport, "sport-wins-"+strconv.Itoa(rand.Intn(1000000)), amount)
		err = walletStorage.OperateVndBalanceV1(bill)
		if err != nil {
			err = fmt.Errorf("1100")
			return err
		}
		eventList = append(eventList, event)
	}

	var balance int64
	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance = wallet.VndBalance

	var betRecords BetRecords
	betRecords.Oid = primitive.NewObjectID()
	betRecords.Data.DataID = betRecords.Oid.Hex()
	betRecords.Data.Action = "amends"
	betRecords.Data.Before = beforeBalance
	betRecords.Data.Balance = balance
	betRecords.Data.Currency = "VND"
	betRecords.Data.Target.Account = data.Account
	betRecords.Data.Status.CreateTime = data.EventTime
	betRecords.Data.Status.Status = "success"
	betRecords.Data.Status.Message = "success"
	betRecords.Data.Status.EndTime = time.Now().Format(time.RFC3339)
	betRecords.Data.Events = eventList
	betRecords.Status.Message = "success"
	betRecords.Status.Code = "0"
	betRecords.Status.Datatime = time.Now().Format(time.RFC3339)

	err = insertBetRecord(&betRecords)
	if err != nil {
		err = fmt.Errorf("1100")
		return err
	}

	for _, v := range eventList {
		var tmpMTCode MTCode
		tmpMTCode.Account = data.Account
		tmpMTCode.RecordOid = betRecords.Oid.Hex()
		tmpMTCode.Mtcode = v.Mtcode
		err = insertMTCode(&tmpMTCode)
		if err != nil {
			err = fmt.Errorf("1100")
			return err
		}
	}

	return nil
}

func AmendsPlayerHandle(data AmendsData, uid string) error {
	dbTransaction := common.NewDBTransaction()
	operateBets := func(sctx mongo.SessionContext, d common.DBTransaction) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}
		err = operateAmendsPlayer(sctx, data, uid)
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

func AmendsHandle(amends Amends, paramMap map[string]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	type failMsg struct {
		Account string `json:"account"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Ucode   string `json:"ucode"`
	}
	type succMsg struct {
		Account  string  `json:"account"`
		Currency string  `json:"currency"`
		Before   float64 `json:"before"`
		Balance  float64 `json:"balance"`
		Ucode    string  `json:"ucode"`
	}
	var failArr []failMsg
	var succArr []succMsg
	statusMap["code"] = "0"
	statusMap["message"] = "success"
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	tmpList := paramMap["list"].([]interface{})
	for k, data := range amends.List {
		tmpMap := tmpList[k].(map[string]interface{})
		if errCode, err := CheckParams([]string{"eventtime", "account", "amount", "action", "ucode"}, tmpMap); err != nil {
			var tmpFail failMsg
			tmpFail.Account = data.Account
			tmpFail.Code = errCode
			tmpFail.Message = err.Error()
			tmpFail.Ucode = data.UCode
			failArr = append(failArr, tmpFail)
			continue
		}
		tmpEvent := tmpMap["event"].([]interface{})
		sucFlg := true
		for _, v := range tmpEvent {
			aa := v.(map[string]interface{})
			if errCode, err := CheckParams([]string{"mtcode", "amount", "validbet", "action", "roundid", "eventtime", "gamecode"}, aa); err != nil {
				var tmpFail failMsg
				tmpFail.Account = data.Account
				tmpFail.Code = errCode
				tmpFail.Message = err.Error()
				tmpFail.Ucode = data.UCode
				failArr = append(failArr, tmpFail)
				sucFlg = false
				break
			}
			if err := ConfirmNoThisMtcode(aa["mtcode"].(string)); err != nil {
				var tmpFail failMsg
				tmpFail.Account = data.Account
				tmpFail.Code = "2009"
				tmpFail.Message = "混合码已经存在了！"
				tmpFail.Ucode = data.UCode
				failArr = append(failArr, tmpFail)
				sucFlg = false
				break
			}
		}
		if !sucFlg {
			continue
		}

		apiUser, existFlg := FindUserByAccount(data.Account, ApiType)
		if !existFlg {
			var tmpFail failMsg
			tmpFail.Account = data.Account
			tmpFail.Code = "1006"
			tmpFail.Message = data.Account + " not found"
			tmpFail.Ucode = data.UCode
			failArr = append(failArr, tmpFail)
			continue
		}

		uid := apiUser.Uid
		var beforeBalance int64
		wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
		beforeBalance = wallet.VndBalance
		if data.Action == "debit" && beforeBalance-int64(data.Amount) < 0 {
			var tmpFail failMsg
			tmpFail.Code = "1005"
			tmpFail.Message = "余额不足"
			tmpFail.Account = data.Account
			tmpFail.Ucode = data.UCode
			failArr = append(failArr, tmpFail)
			continue
		}

		err := AmendsPlayerHandle(data, uid)
		if err != nil {
			var tmpFail failMsg
			tmpFail.Account = data.Account
			tmpFail.Code = err.Error()
			tmpFail.Message = "fail"
			tmpFail.Ucode = data.UCode
			failArr = append(failArr, tmpFail)
			continue
		} else {
			var balance int64
			wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
			balance = wallet.VndBalance
			var tmpSucc succMsg
			tmpSucc.Ucode = data.UCode
			tmpSucc.Account = data.Account
			tmpSucc.Currency = "VND"
			tmpSucc.Before = float64(beforeBalance)
			tmpSucc.Balance = float64(balance)
			if data.Action == "debit" {
				tmpSucc.Balance = tmpSucc.Balance - data.Amount + float64(int64(data.Amount))
			} else {
				tmpSucc.Balance = tmpSucc.Balance + data.Amount - float64(int64(data.Amount))
			}
			succArr = append(succArr, tmpSucc)
		}
	}

	dataMap["failed"] = failArr
	dataMap["success"] = succArr
	res["data"] = dataMap
	res["status"] = statusMap
	return res
}
