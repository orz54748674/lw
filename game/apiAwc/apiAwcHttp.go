package apiAwc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/activity"
	"vn/storage/activityStorage"
	"vn/storage/apiStorage"
	"vn/storage/gameStorage"
	"vn/storage/walletStorage"
)

type HttpFuncMap map[string]func(w http.ResponseWriter, r *http.Request, msg map[string]interface{})

var (
	httpFuncs = HttpFuncMap{
		"getBalance":       getBalance,
		"bet":              placeBet,
		"cancelBet":        cancelBet,
		"settle":           settle,
		"adjustBet":        adjustBet,
		"voidBet":          voidBet,
		"unvoidBet":        unvoidBet,
		"refund":           refund,
		"unsettle":         unsettle,
		"voidSettle":       voidSettle,
		"unvoidSettle":     unvoidSettle,
		"betNSettle":       betNSettle,
		"cancelBetNSettle": cancelBetNSettle,
		"give":             give,
	}
)

type AwcHttp struct {
}

func (a *AwcHttp) Entrance(w http.ResponseWriter, r *http.Request) {
	log.Debug("AwcHttp.Entrance %d", time.Now().Unix())
	// log.Debug("AwcHttp.Entrance key:%s", r.FormValue("key"))
	// log.Debug("AwcHttp.Entrance message:%s", r.FormValue("message"))
	if r.FormValue("key") != cert {
		log.Error("awc Entrance key err")
		failed(w)
		return
	}
	params := map[string]interface{}{}
	if err := parse(w, r, &params); err != nil {
		return
	}
	log.Debug("awc Entrance params:%v", params)
	action, ok := params["action"]
	if !ok {
		log.Error("awc Entrance not find action")
		failed(w)
		return
	}
	log.Debug("awc Entrance action:%s", action)
	httpFunc, ok := httpFuncs[action.(string)]
	if !ok {
		log.Error("awc Entrance not find httpFunc")
		failed(w)
		return
	}
	httpFunc(w, r, params)
}

// 取得余额
func getBalance(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("awc getBalance %d", time.Now().Unix())
	params := &struct {
		UserId string `json:"userId"`
	}{}
	if err := parse(w, r, params); err != nil {
		return
	}
	log.Debug("awc getBalance params:%v", params)
	mApiUser := &apiStorage.ApiUser{}
	err := mApiUser.GetApiUserByAccount(params.UserId, apiType)
	if err != nil {
		log.Error("awc getBalance GetApiUserByAccount err:%s", err.Error())
		failed(w)
		return
	}

	data := getComResp(nil, mApiUser.Uid, params.UserId)
	response(w, data)
}

// 下单
func placeBet(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("awc placeBet %d", time.Now().Unix())
	params := &struct {
		Txns []*apiStorage.AwcBetRecord `json:"txns"`
	}{}
	if err := parse(w, r, params); err != nil {
		log.Error("awc placeBet map to struct err:%s", err.Error())
		return
	}
	if len(params.Txns) == 0 {
		log.Error("awc placeBet bet number:0")
		failed(w)
		return
	}
	account := params.Txns[0].Account

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(account, apiType); err != nil {
		log.Error("GetUserBalance GetApiUserByAccount err:%s", err.Error())
		failed(w)
		return
	}
	m := &apiStorage.AwcCancelBetRecord{}
	for _, txn := range params.Txns {
		if count, err := m.GetCancelRecord(txn.PlatformTxID); err != nil || count > 0 {
			log.Debug("PlatformTxID:%s is cancel", txn.PlatformTxID)
			failed(w)
			return
		}
		txn.Oid = primitive.NewObjectID()
		txn.Uid = mApiUser.Uid
		txn.BetAmount = txn.BetAmount * scale
		txn.SetTransactionUnits(apiStorage.AddAwcRecord)
		bill := walletStorage.NewBill(mApiUser.Uid, walletStorage.TypeExpenses, walletStorage.EventGameAwc, txn.Oid.Hex(), -1*int64(txn.BetAmount))
		if err := walletStorage.OperateVndBalanceV1(bill, txn); err != nil {
			log.Error("wallet pay bet _id:%s err:%s", txn.Oid.Hex(), err.Error())
			failed(w)
			return
		}
		activityStorage.UpsertGameDataInBet(txn.Uid, game.ApiAwc, 1)
	}
	data := getComResp(nil, mApiUser.Uid, account)
	response(w, data)
}

// 已结帐派彩
func settle(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("awc settle %d", time.Now().Unix())
	params := &struct {
		Txns []*apiStorage.AwcBetRecord `json:"txns"`
	}{}
	if err := parse(w, r, params); err != nil {
		log.Error("awc settle map to struct err:%s", err.Error())
		return
	}
	if len(params.Txns) == 0 {
		log.Error("awc settle bet number:0")
		failed(w)
		return
	}
	platformTxIds := []string{}
	for _, v := range params.Txns {
		platformTxIds = append(platformTxIds, v.PlatformTxID)
	}
	log.Debug("awc settle platformTxIds:%v", platformTxIds)
	mAwcBetRecord := &apiStorage.AwcBetRecord{}
	records, err := mAwcBetRecord.GetRecords(platformTxIds)
	if err != nil {
		log.Error("awc settle mAwcBetRecord.GetRecords err:%s", err.Error())
		failed(w)
		return
	}
	if len(records) == 0 {
		log.Debug("awc settle No need to settle")
		data := getComResp(nil)
		response(w, data)
		return
	}
	settleMap := map[string]*apiStorage.AwcBetRecord{}
	for _, v := range params.Txns {
		settleMap[v.PlatformTxID] = v
	}

	var vndBalance int64
	for _, record := range records {
		record.SetTransactionUnits(apiStorage.SettleAwcRecord)
		settleItem := settleMap[record.PlatformTxID]
		settleItem.WinAmount = settleItem.WinAmount * scale
		record.WinAmount = settleItem.WinAmount
		bill := walletStorage.NewBill(record.Uid, walletStorage.TypeIncome, walletStorage.EventGameAwc, record.Oid.Hex(), int64(settleItem.WinAmount))
		if err := walletStorage.OperateVndBalanceV1(bill, record); err != nil {
			log.Error("wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
			failed(w)
			return
		}
		wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))

		profit := int64(settleItem.WinAmount - record.BetAmount)
		betInfo := map[string]interface{}{
			"betType":   record.BetType,
			"betAmount": record.BetAmount,
			"gameType":  record.GameType,
			"gameCode":  record.GameCode,
			"gameName":  record.GameName,
			"profit":    profit,
		}
		btBetInfo, _ := json.Marshal(betInfo)
		betDetails := string(btBetInfo)
		gameRes, _ := json.Marshal(settleItem.GameInfo)
		betRecordData := gameStorage.BetRecordParam{
			Uid:        record.Uid,
			GameType:   game.ApiAwc,
			Income:     profit,
			BetAmount:  int64(record.BetAmount),
			CurBalance: wallet.VndBalance + wallet.SafeBalance,
			SysProfit:  0,
			BotProfit:  0,
			BetDetails: betDetails,
			GameId:     record.Oid.Hex(),
			GameNo:     record.RoundId,
			GameResult: string(gameRes),
			IsSettled:  false,
		}
		gameStorage.InsertBetRecord(betRecordData)
		activityStorage.UpsertGameDataInBet(record.Uid, game.ApiAwc, -1)
		activity.CalcEncouragementFunc(record.Uid)
		vndBalance = wallet.AgentBalance
	}
	data := getComResp(map[string]interface{}{"balance": float64(vndBalance) / scale}, "", params.Txns[0].Account)
	response(w, data)
}

// 取消已结帐派彩
func unsettle(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("awc unsettle %d", time.Now().Unix())
	params := &struct {
		Txns []*apiStorage.AwcBetRecord `json:"txns"`
	}{}
	if err := parse(w, r, params); err != nil {
		log.Error("awc unsettle map to struct err:%s", err.Error())
		return
	}
	if len(params.Txns) == 0 {
		log.Error("awc unsettle bet number:0")
		failed(w)
		return
	}
	platformTxIds := []string{}
	for _, v := range params.Txns {
		platformTxIds = append(platformTxIds, v.PlatformTxID)
	}
	log.Debug("awc unsettle platformTxIds:%v", platformTxIds)
	mAwcBetRecord := &apiStorage.AwcBetRecord{}
	records, err := mAwcBetRecord.GetSettledRecords(platformTxIds)
	if err != nil {
		log.Error("awc unsettle mAwcBetRecord.GetSettledRecords err:%s", err.Error())
		failed(w)
		return
	}
	if len(records) == 0 {
		log.Debug("awc unsettle No need to unsettle")
		response(w, getComResp(nil))
		return
	}
	settleMap := map[string]*apiStorage.AwcBetRecord{}
	for _, v := range params.Txns {
		settleMap[v.PlatformTxID] = v
	}

	var vndBalance int64
	log.Debug("records:%d", len(records))
	for _, record := range records {
		record.SetTransactionUnits(apiStorage.UnsettleAwcRecord)
		log.Debug("record.WinAmount :%v", record.WinAmount)
		bill := walletStorage.NewBill(record.Uid, walletStorage.TypeExpenses, walletStorage.EventGameAwc, record.Oid.Hex(), -int64(record.WinAmount))
		if err := walletStorage.OperateVndBalanceV2(bill, record); err != nil {
			log.Error("wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
			failed(w)
			return
		}
		wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))

		gRes := map[string]interface{}{"SettleStatus": apiStorage.Colse}
		btGameRes, _ := json.Marshal(gRes)

		gameStorage.RefundAwcBetRecord(record.Oid.Hex(), string(btGameRes), game.ApiAwc)
		activityStorage.UpsertGameDataInBet(record.Uid, game.ApiAwc, 1)
		vndBalance = wallet.AgentBalance
	}
	data := getComResp(map[string]interface{}{"balance": float64(vndBalance) / scale}, "", params.Txns[0].Account)
	response(w, data)
}

// 取消下单
func cancelBet(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("awc cancelBet %d", time.Now().Unix())
	params := &struct {
		Txns []*apiStorage.AwcBetRecord `json:"txns"`
	}{}
	if err := parse(w, r, params); err != nil {
		log.Error("awc cancelBet map to struct err:%s", err.Error())
		return
	}
	if len(params.Txns) == 0 {
		log.Error("awc cancelBet bet number:0")
		failed(w)
		return
	}
	platformTxIds := []string{}
	cancelDatas := []*apiStorage.AwcCancelBetRecord{}
	var account, uid string
	for _, v := range params.Txns {
		account = v.Account
		platformTxIds = append(platformTxIds, v.PlatformTxID)
		cancelDatas = append(cancelDatas, &apiStorage.AwcCancelBetRecord{
			PlatformTxID: v.PlatformTxID,
			Type:         1,
		})
	}
	m := &apiStorage.AwcCancelBetRecord{}
	log.Debug("m.AddMany(cancelDatas):%v", m.AddMany(cancelDatas))

	log.Debug("awc cancelBet platformTxIds:%v", platformTxIds)
	mAwcBetRecord := &apiStorage.AwcBetRecord{}
	records, err := mAwcBetRecord.GetRecords(platformTxIds)
	if err != nil {
		log.Error("awc cancelBet mAwcBetRecord.GetRecords err:%s", err.Error())
		response(w, getComResp(nil))
		return
	}
	if len(records) > 0 {
		for _, record := range records {
			uid = record.Uid
			record.SetTransactionUnits(apiStorage.CancelAwcRecord)
			bill := walletStorage.NewBill(record.Uid, walletStorage.TypeIncome, walletStorage.EventGameAwc, record.Oid.Hex(), int64(record.BetAmount))
			if err := walletStorage.OperateVndBalanceV1(bill, record); err != nil {
				log.Error("wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
				failed(w)
				return
			}
			activityStorage.UpsertGameDataInBet(record.Uid, game.ApiAwc, -1)
		}
	}
	if uid == "" {
		mApiUser := &apiStorage.ApiUser{}
		if err := mApiUser.GetApiUserByAccount(account, apiType); err != nil {
			log.Error("awc cancelBet GetApiUserByAccount err:%s", err.Error())
			failed(w)
			return
		}
		uid = mApiUser.Uid
	}

	data := getComResp(nil, uid)
	response(w, data)
}

// 调整订单
func adjustBet(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("AwcHttp.AdjustBet %d", time.Now().Unix())
}

// 交易作废
func voidBet(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("awc voidBet  %d", time.Now().Unix())
	params := &struct {
		Txns []*apiStorage.AwcBetRecord `json:"txns"`
	}{}
	if err := parse(w, r, params); err != nil {
		log.Error("awc voidBet map to struct err:%s", err.Error())
		return
	}
	if len(params.Txns) == 0 {
		log.Error("awc voidBet bet number:0")
		failed(w)
		return
	}
	platformTxIds := []string{}
	var account, uid string
	for _, v := range params.Txns {
		account = v.Account
		platformTxIds = append(platformTxIds, v.PlatformTxID)
	}
	log.Debug("awc voidBet platformTxIds:%v", platformTxIds)
	mAwcBetRecord := &apiStorage.AwcBetRecord{}
	records, err := mAwcBetRecord.GetRecords(platformTxIds)
	if err != nil {
		log.Error("awc voidBet mAwcBetRecord.GetRecords err:%s", err.Error())
		failed(w)
		return
	}
	log.Debug(" len(records):%d", len(records))
	if len(records) > 0 {
		settleMap := map[string]*apiStorage.AwcBetRecord{}
		for _, v := range params.Txns {
			settleMap[v.PlatformTxID] = v
		}
		for _, record := range records {
			uid = record.Uid
			settleItem := settleMap[record.PlatformTxID]
			record.SetTransactionUnits(apiStorage.VoidAwcRecord)
			record.VoidType = settleItem.VoidType
			record.UpdateTime = settleItem.UpdateTime
			settleItem.BetAmount = settleItem.BetAmount * scale
			bill := walletStorage.NewBill(record.Uid, walletStorage.TypeIncome, walletStorage.EventGameAwc, record.Oid.Hex(), int64(settleItem.BetAmount))
			log.Debug("walletStorage.TypeIncome:%s bill:%v", walletStorage.TypeIncome, bill)
			if err := walletStorage.OperateVndBalanceV1(bill, record); err != nil {
				log.Error("wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
				failed(w)
				return
			}
			log.Debug("UserId:%s voidBet end", uid)
		}
	}

	if uid == "" {
		mApiUser := &apiStorage.ApiUser{}
		if err := mApiUser.GetApiUserByAccount(account, apiType); err != nil {
			log.Error("awc voidBet GetApiUserByAccount err:%s", err.Error())
			failed(w)
			return
		}
		uid = mApiUser.Uid
	}

	data := getComResp(nil, uid)
	response(w, data)
}

// 取消交易作废
func unvoidBet(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("AwcHttp.UnvoidBet %d", time.Now().Unix())
}

// 返还金额
func refund(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("AwcHttp.Refund %d", time.Now().Unix())
}

// 结帐单转为无效
func voidSettle(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("awc voidSettle %d", time.Now().Unix())
	params := &struct {
		Txns []*apiStorage.AwcBetRecord `json:"txns"`
	}{}
	if err := parse(w, r, params); err != nil {
		log.Error("awc voidSettle map to struct err:%s", err.Error())
		return
	}
	if len(params.Txns) == 0 {
		log.Error("awc voidSettle bet number:0")
		failed(w)
		return
	}
	platformTxIds := []string{}
	for _, v := range params.Txns {
		platformTxIds = append(platformTxIds, v.PlatformTxID)
	}
	log.Debug("awc voidSettle platformTxIds:%v", platformTxIds)
	mAwcBetRecord := &apiStorage.AwcBetRecord{}
	records, err := mAwcBetRecord.GetSettledRecords(platformTxIds)
	if err != nil {
		log.Error("awc voidSettle mAwcBetRecord.GetSettledRecords err:%s", err.Error())
		failed(w)
		return
	}
	if len(records) == 0 {
		log.Debug("awc voidSettle No need to voidSettle")
		response(w, getComResp(nil))
		return
	}
	for _, record := range records {
		record.SetTransactionUnits(apiStorage.VoidSettleAwcRecord)
		amount := int64(record.BetAmount - record.WinAmount)
		changeType := walletStorage.TypeExpenses
		if amount > 0 {
			changeType = walletStorage.TypeIncome
		}
		bill := walletStorage.NewBill(record.Uid, changeType, walletStorage.EventGameAwc, record.Oid.Hex(), amount)
		if err := walletStorage.OperateVndBalanceV2(bill, record); err != nil {
			log.Error("awc voidSettle wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
			failed(w)
			return
		}
		gRes := map[string]interface{}{"SettleStatus": apiStorage.ViodSettle}
		btGameRes, _ := json.Marshal(gRes)
		gameStorage.RefundAwcBetRecord(record.Oid.Hex(), string(btGameRes), game.ApiAwc)
	}
	data := getComResp(nil)
	response(w, data)
}

// 无效单结账
func unvoidSettle(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("AwcHttp.UnvoidSettle %d", time.Now().Unix())
}

// 下注并直接结算
func betNSettle(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("AwcHttp.BetNSettle %d", time.Now().Unix())
}

// 取消结算并取消注单
func cancelBetNSettle(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("AwcHttp.CancelBetNSettle %d", time.Now().Unix())
}

// Promotion Bonus 活动派彩
func give(w http.ResponseWriter, r *http.Request, msg map[string]interface{}) {
	log.Debug("AwcHttp.Give %d", time.Now().Unix())
	params := &struct {
		Txns []*apiStorage.AwcGiveRecord `json:"txns"`
	}{}
	if err := parse(w, r, params); err != nil {
		log.Error("awc voidSettle map to struct err:%s", err.Error())
		return
	}
	mApiUser := &apiStorage.ApiUser{}
	var wallet *walletStorage.Wallet
	var uid primitive.ObjectID
	for _, txns := range params.Txns {
		txns.Oid = primitive.NewObjectID()
		if err := mApiUser.GetApiUserByAccount(txns.Account, apiStorage.AwcType); err != nil {
			log.Error("GetApiUserByAccount :%s err:%s", txns.Account, err.Error())
			failed(w)
			return
		}
		txns.Amount = txns.Amount * scale
		txns.Uid = mApiUser.Uid
		uid = utils.ConvertOID(txns.Uid)
		bill := walletStorage.NewBill(txns.Uid, walletStorage.TypeIncome, walletStorage.EventGameAwc, txns.Oid.Hex(), int64(txns.Amount))
		txns.SetTransactionUnits(apiStorage.AddAwcGiveRecord)
		if txns.IsExists() {
			continue
		}

		if err := walletStorage.OperateVndBalanceV1(bill, txns); err != nil {
			log.Error("wallet pay bet _id:%s err:%s", txns.Oid.Hex(), err.Error())
			failed(w)
			return
		}

		wallet = walletStorage.QueryWallet(utils.ConvertOID(txns.Uid))
		params := gameStorage.BetRecordParam{
			Uid:        txns.Uid,
			GameType:   game.ApiAwc,
			Income:     int64(txns.Amount),
			BetAmount:  0,
			CurBalance: wallet.VndBalance + wallet.SafeBalance,
			SysProfit:  0,
			BotProfit:  0,
			BetDetails: `{"name":"活动派彩"}`,
			GameId:     txns.Oid.Hex(),
			GameNo:     fmt.Sprintf("%s_%s", txns.PromotionTxId, txns.PromotionId),
			GameResult: `{"name":"活动派彩"}`,
			IsSettled:  true,
		}
		gameStorage.InsertBetRecord(params)
	}
	if wallet == nil {
		wallet = walletStorage.QueryWallet(uid)
	}
	data := getComResp(map[string]interface{}{"desc": "success", "balance": float64(wallet.VndBalance) / scale})
	response(w, data)
}

func parse(w http.ResponseWriter, r *http.Request, param interface{}) (err error) {
	body := r.FormValue("message")
	log.Debug("body:%s", string(body))
	err = json.Unmarshal([]byte(body), &param)
	if err != nil {
		log.Error("parse param to struct err:%s", err.Error())
		failed(w)
		return
	}
	return
}

func response(w http.ResponseWriter, data map[string]interface{}) {
	btData, err := json.Marshal(data)
	if err != nil {
		log.Error("AwcHttp response json.Marshal err:%s", err.Error())
		return
	}
	if _, err := w.Write([]byte(btData)); err != nil {
		log.Error("AwcHttp response: ioWrite  json format err:%s", err.Error())
		return
	}
}

func failed(w http.ResponseWriter) {
	response(w, map[string]interface{}{"status": "9999"})
}
