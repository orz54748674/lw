package apiSbo

import (
	"encoding/json"
	"fmt"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/activity"
	"vn/storage/activityStorage"
	"vn/storage/apiStorage"
	"vn/storage/gameStorage"
	"vn/storage/walletStorage"
)

type SboRpc struct {
}

// 获取玩家余额
func (s *SboRpc) GetBalance(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("Sbo getBalance data:%v", time.Now().Unix())
	params := &struct {
		CompanyKey  string `json:"CompanyKey"`
		Username    string `json:"Username"`
		ProductType int    `json:"ProductType"`
		GameType    int    `json:"GameType"`
	}{}
	resp = map[string]interface{}{}
	if e := s.parse(data, resp, params); e != nil {
		return
	}
	if !s.auth(resp, params.CompanyKey) {
		return
	}
	apiUser, e := s.getApiUser(resp, params.Username)
	if e != nil {
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
	resp["Body"] = map[string]interface{}{
		"AccountName":  params.Username,
		"Balance":      float64(wallet.VndBalance) / scale,
		"ErrorCode":    0,
		"ErrorMessage": "No Error",
	}
	return
}

// 扣除投注金额
func (s *SboRpc) Deduct(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("Sbo Deduct data:%v", time.Now().Unix())
	params := &apiStorage.SboBetRecord{}
	resp = map[string]interface{}{}
	if e := s.parse(data, resp, params); e != nil {
		return
	}
	params.Oid = primitive.NewObjectID()
	eventId := params.Oid.Hex()
	if !s.auth(resp, params.CompanyKey) {
		return
	}
	log.Debug("Sbo Deduct Params:%v", params)
	apiUser, e := s.getApiUser(resp, params.Username)
	if e != nil {
		return
	}
	params.Uid = apiUser.Uid
	log.Debug("Sbo Deduct ProductType:%v", params.ProductType)
	params.Amount *= scale
	betAmount := params.Amount
	raise := false
	if params.ProductType == 1 || params.ProductType == 5 { //SportsBook
		if params.TransferCodeExists() {
			s.response(resp, 4401, "TransferCode already exists")
			return
		}
	} else if params.ProductType == 7 || params.ProductType == 3 || params.ProductType == 9 {
		res, e := params.GetRecords(params.TransferCode)
		if err != nil {
			log.Error("Sbo Deduct GetRecords err：%s ", e.Error())
			s.response(resp, 8, e.Error())
			return
		}
		if len(res) >= 2 {
			s.response(resp, 4401, "TransferCode already exists")
			return
		} else if len(res) == 1 {
			if res[0].Status == settleStatusSettled {
				s.response(resp, 2001, "Bet Already Settled")
				return
			} else if res[0].Status == settleStatusVoid {
				s.response(resp, 2002, "Bet Already Canceled")
				return
			}
			if params.ProductType == 9 {
				if res[0].TransactionId == params.TransactionId {
					s.response(resp, 8, "TransactionId repeat")
					return
				}
			} else {
				if res[0].Amount >= params.Amount {
					s.response(resp, 8, "Amount err")
					return
				}
				betAmount -= res[0].Amount
				raise = true
			}
			eventId = res[0].Oid.Hex()
		}
	}
	params.Status = "running"
	wallet := walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
	if wallet.VndBalance < int64(params.Amount) {
		s.response(resp, 5, "Not enough balance")
		return
	}
	bill := walletStorage.NewBill(apiUser.Uid, walletStorage.TypeExpenses, walletStorage.EventGameSbo, eventId, -int64(betAmount))
	bill.Remark = "Sbo Deduct"
	if raise {
		params.SetTransactionUnits(apiStorage.AddSboBetAmount)
	} else {
		params.SetTransactionUnits(apiStorage.AddSboBetRecord)
	}
	if e = walletStorage.OperateVndBalanceV1(bill, params); e != nil {
		log.Error("wallet pay bet _id:%s err:%s", params.Oid.Hex(), e.Error())
		s.response(resp, 7, e.Error())
		return
	}
	if !raise {
		activityStorage.UpsertGameDataInBet(apiUser.Uid, game.ApiSbo, 1)
	}
	if params.ProductType == 1 {
		betInfo := map[string]interface{}{
			"ProductType": params.ProductType,
			"GameType":    params.GameType,
			"OrderDetail": params.OrderDetail,
			"Gpid":        params.Gpid,
		}
		btBetInfo, _ := json.Marshal(betInfo)
		betRecordData := gameStorage.BetRecordParam{
			Uid:        params.Uid,
			GameType:   game.ApiSbo,
			Income:     0,
			BetAmount:  int64(params.Amount),
			CurBalance: wallet.VndBalance + wallet.SafeBalance - int64(params.Amount),
			SysProfit:  0,
			BotProfit:  0,
			BetDetails: string(btBetInfo),
			GameId:     params.Oid.Hex(),
			GameNo:     params.TransferCode,
			GameResult: "",
			IsSettled:  false,
		}
		gameStorage.InsertBetRecord(betRecordData)
	}
	wallet = walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
	resp["Body"] = map[string]interface{}{
		"AccountName":  params.Username,
		"BetAmount":    params.Amount / scale,
		"Balance":      float64(wallet.VndBalance) / scale,
		"ErrorCode":    0,
		"ErrorMessage": "No Error",
	}
	return
}

// 结算投注
func (s *SboRpc) Settle(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("Sbo Settle data:%v", time.Now().Unix())
	params := &apiStorage.SboBetRecord{}
	resp = map[string]interface{}{}
	if e := s.parse(data, resp, params); e != nil {
		return
	}
	resp["AccountName"] = params.Username
	if !s.auth(resp, params.CompanyKey) {
		return
	}

	apiUser, e := s.getApiUser(resp, params.Username)
	if e != nil {
		return
	}
	record, e := params.GetTransferCodeStatus(params.TransferCode)
	if e != nil {
		log.Error("Sbo Settle GetTransferCodeStatus err:%s", e.Error())
		s.response(resp, 7, "Internal Error")
		return
	}
	if record.Status == settleStatusSettled {
		s.response(resp, 2001, "Bet Already Settled")
		return
	} else if record.Status == settleStatusVoid {
		s.response(resp, 2002, "Bet Already Canceled")
		return
	}
	params.WinLoss *= scale
	if params.WinLoss > 0 {
		bill := walletStorage.NewBill(apiUser.Uid, walletStorage.TypeIncome, walletStorage.EventGameSbo, record.Oid.Hex(), int64(params.WinLoss))
		bill.Remark = "Sbo Settle"
		params.SetTransactionUnits(apiStorage.SettleSboBetRecord)
		if err = walletStorage.OperateVndBalanceV1(bill, params); err != nil {
			log.Error("wallet pay bet _id:%s err:%s", params.Oid.Hex(), err.Error())
			s.response(resp, 7, err.Error())
			return
		}
	}
	activityStorage.UpsertGameDataInBet(apiUser.Uid, game.ApiSbo, -1)
	activity.CalcEncouragementFunc(apiUser.Uid)
	wallet := walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
	betInfo := map[string]interface{}{
		"ProductType": params.ProductType,
		"GameType":    params.GameType,
		"OrderDetail": record.OrderDetail,
		"Gpid":        params.Gpid,
	}
	resInfo := map[string]interface{}{
		"ResultType":  params.ResultType,
		"ResultTime":  params.ResultTime,
		"GameResult":  params.GameResult,
		"ProductType": params.ProductType,
		"GameType":    params.GameType,
		"Gpid":        params.Gpid,
	}
	btBetInfo, _ := json.Marshal(betInfo)
	btResInfo, _ := json.Marshal(resInfo)
	gameId := record.Oid.Hex()
	if params.ProductType == 9 {
		gameId = params.TransferCode
	}
	income := int64(params.WinLoss - record.Amount)
	betRecordData := gameStorage.BetRecordParam{
		Uid:        record.Uid,
		GameType:   game.ApiSbo,
		Income:     income,
		BetAmount:  int64(record.Amount),
		CurBalance: wallet.VndBalance + wallet.SafeBalance,
		SysProfit:  0,
		BotProfit:  0,
		BetDetails: string(btBetInfo),
		GameId:     gameId,
		GameNo:     params.TransferCode,
		GameResult: string(btResInfo),
		IsSettled:  true,
	}
	if params.ProductType != 1 {
		gameStorage.InsertBetRecord(betRecordData)
	} else {
		gameStorage.UpdateSboBetRecord(record.Oid.Hex(), record.Uid, record.TransferCode, string(btResInfo), game.ApiSbo, income, wallet.VndBalance+wallet.SafeBalance, int64(record.Amount))
	}

	resp["Body"] = map[string]interface{}{
		"AccountName":  params.Username,
		"Balance":      float64(wallet.VndBalance) / scale,
		"ErrorCode":    0,
		"ErrorMessage": "No Error",
	}
	return
}

// 回滚
func (s *SboRpc) Rollback(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("Sbo Rollback data:%v", time.Now().Unix())
	params := &apiStorage.SboBetRecord{}
	resp = map[string]interface{}{}
	if e := s.parse(data, resp, params); e != nil {
		return
	}

	resp["AccountName"] = params.Username
	if !s.auth(resp, params.CompanyKey) {
		return
	}

	apiUser, e := s.getApiUser(resp, params.Username)
	if e != nil {
		return
	}
	record := &apiStorage.SboBetRecord{}
	var eventId string
	if params.ProductType == 9 {
		res, e := params.GetRecords(params.TransferCode)
		if e != nil {
			log.Error("Sbo Rollback GetTransferCodeStatus err:%s", e.Error())
			s.response(resp, 7, "Internal Error")
			return
		}
		rollback := false
		if len(res) == 1 {
			eventId = res[0].Oid.Hex()
		} else {
			eventId = params.TransferCode
		}
		for _, r := range res {
			if !r.Rollback {
				record.Amount += r.Amount
				record.WinLoss = r.WinLoss
				r.SetTransactionUnits(apiStorage.RollbackSboBetRecord)
				record.Status = r.Status
				// eventId = r.TransferCode
				record.GameType = r.GameType
				record.TransferCode = r.TransferCode
				record.ProductType = r.ProductType
				record.CancelBeforeStatus = r.CancelBeforeStatus
				record.Rollback = r.Rollback
				rollback = true
			}
		}
		if !rollback {
			s.response(resp, 2003, "Bet Already Rollback")
			return
		}
	} else {
		record, e = params.GetTransferCodeStatus(params.TransferCode)
		if e != nil {
			log.Error("Sbo Rollback GetTransferCodeStatus err:%s", e.Error())
			s.response(resp, 7, "Internal Error")
			return
		}
		eventId = record.Oid.Hex()
	}

	log.Debug("record.CancelBeforeStatus:%v settleStatusRuning：%v ", record.CancelBeforeStatus, settleStatusRuning)
	log.Debug(" record.Rollback:%v ", record.Rollback)
	if record.Rollback {
		s.response(resp, 2003, "Bet no Rollback")
		return
	}
	log.Debug("record.Status:%v  settleStatusVoid%v ", record.Status, settleStatusVoid)
	if record.Status == settleStatusVoid {
		bill := walletStorage.NewBill(apiUser.Uid, walletStorage.TypeExpenses, walletStorage.EventGameSbo, eventId, -int64(record.Amount))
		bill.Remark = fmt.Sprintf("Sbo Rollback status:%v", record.Status)
		record.SetTransactionUnits(apiStorage.RollbackSboBetRecord)
		if e = walletStorage.OperateVndBalanceV2(bill, record); e != nil {
			log.Error("Sbo Rollback wallet pay bet _id:%s err:%s", eventId, e.Error())
			s.response(resp, 7, "Internal Error")
			return
		}
	} else {
		if record.WinLoss > 0 {
			bill := walletStorage.NewBill(apiUser.Uid, walletStorage.TypeExpenses, walletStorage.EventGameSbo, eventId, -int64(record.WinLoss))
			bill.Remark = fmt.Sprintf("Sbo Rollback status:%v", record.Status)
			record.SetTransactionUnits(apiStorage.RollbackSboBetRecord)
			if e = walletStorage.OperateVndBalanceV2(bill, record); e != nil {
				log.Error("Sbo Rollback wallet pay bet _id:%s err:%s", eventId, e.Error())
				s.response(resp, 7, "Internal Error")
				return
			}
		} else {
			if e := record.RollbackSboBetRecord(); e != nil {
				log.Error("Sbo Rollback RollbackSboBetRecord id:%s err:%s", eventId, e.Error())
				s.response(resp, 7, "Internal Error")
				return
			}
		}
	}
	if record.Status == settleStatusSettled || record.CancelBeforeStatus == settleStatusSettled {
		activityStorage.UpsertGameDataInBet(apiUser.Uid, game.ApiSbo, 1)
		gameStorage.RefundSboBetRecord(record.Oid.Hex(), fmt.Sprintf("Rollback status:%v, CancelBeforeStatus:%v", record.Status, record.CancelBeforeStatus), game.ApiSbo)
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
	resp["Body"] = map[string]interface{}{
		"AccountName":  params.Username,
		"Balance":      float64(wallet.VndBalance) / scale,
		"ErrorCode":    0,
		"ErrorMessage": "No Error",
	}
	return
}

// 取消投注
func (s *SboRpc) Cancel(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("Sbo Cancel data:%v", time.Now().Unix())
	params := &apiStorage.SboBetRecord{}
	resp = map[string]interface{}{}
	if e := s.parse(data, resp, params); e != nil {
		return
	}

	resp["AccountName"] = params.Username
	if !s.auth(resp, params.CompanyKey) {
		return
	}
	apiUser, e := s.getApiUser(resp, params.Username)
	if e != nil {
		return
	}
	var betAmount, winLoss float64
	var records []walletStorage.CallBack
	var evenId, status string
	if params.IsCancelAll {
		res, e := params.GetRecords(params.TransferCode)
		if err != nil && err != mongo.ErrNoDocuments {
			log.Error("Sbo Cancel GetRecords err:%s", e.Error())
			s.response(resp, 7, "Internal Error")
			return
		}
		cancel := false
		if len(res) == 1 {
			evenId = res[0].Oid.Hex()
		} else {
			evenId = params.TransferCode
		}
		for _, r := range res {
			if r.Status != settleStatusVoid {
				betAmount += r.Amount
				winLoss = r.WinLoss
				r.SetTransactionUnits(apiStorage.CancelSboBetRecord)
				status = r.Status
				records = append(records, r)
				cancel = true
			}
		}
		if !cancel {
			s.response(resp, 2002, "Bet Already Canceled")
			return
		}
	} else {
		record, e := params.GetRecord(params.TransferCode, params.TransactionId)
		if e != nil {
			log.Error("Sbo Cancel GetRecord err:%s", e.Error())
			s.response(resp, 7, "Internal Error")
			return
		}
		if record.Status == settleStatusVoid {
			s.response(resp, 2002, "Bet Already Canceled")
			return
		}
		evenId = record.Oid.Hex()
		status = record.Status
		betAmount = record.Amount
		record.SetTransactionUnits(apiStorage.CancelSboBetRecord)
		records = append(records, record)
	}
	log.Debug("Sbo Cancel betAmount:%v winLoss:%v", betAmount, winLoss)
	reIncome := betAmount - winLoss
	billType := walletStorage.TypeExpenses
	if reIncome > 0 {
		billType = walletStorage.TypeIncome
	}
	bill := walletStorage.NewBill(apiUser.Uid, billType, walletStorage.EventGameSbo, evenId, int64(reIncome))
	bill.Remark = fmt.Sprintf("Sbo Cancel %v", status)
	if e := walletStorage.OperateVndBalanceV1(bill, records...); e != nil {
		log.Error("wallet pay bet _id:%s err:%s", evenId, e.Error())
		s.response(resp, 7, err.Error())
		return
	}
	if status == settleStatusSettled {
		activityStorage.UpsertGameDataInBet(apiUser.Uid, game.ApiSbo, 1)
	} else {
		activityStorage.UpsertGameDataInBet(apiUser.Uid, game.ApiSbo, -1)
	}
	gameStorage.RefundSboBetRecord(evenId, fmt.Sprintf("Cancel status:%v", status), game.ApiSbo)
	wallet := walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
	resp["Body"] = map[string]interface{}{
		"AccountName":  params.Username,
		"Balance":      float64(wallet.VndBalance) / scale,
		"ErrorCode":    0,
		"ErrorMessage": "No Error",
	}
	return
}

// 小费
func (s *SboRpc) Tip(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("Sbo Tip data:%v", time.Now().Unix())
	params := &struct {
		CompanyKey    string    `json:"CompanyKey"`
		Username      string    `json:"Username"`
		Amount        float64   `json:"Amount"`
		TipTime       time.Time `json:"TipTime"`
		ProductType   int       `json:"ProductType"`
		GameType      int       `json:"GameType"`
		TransferCode  string    `json:"TransferCode"`
		TransactionId string    `json:"TransactionId"`
		Gpid          int       `json:"Gpid"`
	}{}
	resp = map[string]interface{}{}
	if err = s.parse(data, resp, params); err != nil {
		return
	}
	if params.ProductType != 9 { //SportsBook
		s.response(resp, 8, "ProductType Untreated")
		return
	}
	if !s.auth(resp, params.CompanyKey) {
		return
	}
	apiUser, e := s.getApiUser(resp, params.Username)
	if e != nil {
		return
	}
	tipRecord := &apiStorage.SboBetRecord{
		Oid:           primitive.NewObjectID(),
		Username:      params.Username,
		Amount:        params.Amount * scale,
		BetTime:       params.TipTime,
		ProductType:   params.ProductType,
		GameType:      params.GameType,
		TransferCode:  params.TransferCode,
		TransactionId: params.TransactionId,
		Gpid:          params.Gpid,
	}
	if tipRecord.TransferCodeExists() {
		s.response(resp, 4401, "TransferCode already exists")
		return
	}
	bill := walletStorage.NewBill(apiUser.Uid, walletStorage.TypeExpenses, walletStorage.EventGameSbo, tipRecord.Oid.Hex(), -int64(tipRecord.Amount))
	bill.Remark = "Sbo Tip"
	tipRecord.SetTransactionUnits(apiStorage.AddSboBetRecord)
	if e = walletStorage.OperateVndBalanceV1(bill, tipRecord); e != nil {
		log.Error("Sbo Tip wallet pay bet _id:%s err:%s", tipRecord.Oid.Hex(), e.Error())
		s.response(resp, 7, e.Error())
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
	resp["Body"] = map[string]interface{}{
		"AccountName":  params.Username,
		"Balance":      float64(wallet.VndBalance) / scale,
		"ErrorCode":    0,
		"ErrorMessage": "No Error",
	}
	return
}

// 红利
func (s *SboRpc) Bonus(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("Sbo Bonus data:%v", time.Now().Unix())
	params := &struct {
		CompanyKey    string    `json:"CompanyKey"`
		Username      string    `json:"Username"`
		Amount        float64   `json:"Amount"`
		BonusTime     time.Time `json:"BonusTime"`
		ProductType   int       `json:"ProductType"`
		GameType      int       `json:"GameType"`
		TransferCode  string    `json:"TransferCode"`
		TransactionId string    `json:"TransactionId"`
		Gpid          int       `json:"Gpid"`
	}{}
	resp = map[string]interface{}{}
	if err = s.parse(data, resp, params); err != nil {
		return
	}
	if params.ProductType != 9 { //SportsBook
		s.response(resp, 8, "ProductType Untreated")
		return
	}
	if !s.auth(resp, params.CompanyKey) {
		return
	}
	apiUser, e := s.getApiUser(resp, params.Username)
	if e != nil {
		return
	}
	bonusRecord := &apiStorage.SboBetRecord{
		Oid:           primitive.NewObjectID(),
		Username:      params.Username,
		Amount:        params.Amount * scale,
		BetTime:       params.BonusTime,
		ProductType:   params.ProductType,
		GameType:      params.GameType,
		TransferCode:  params.TransferCode,
		TransactionId: params.TransactionId,
		Gpid:          params.Gpid,
	}
	if bonusRecord.TransferCodeExists() {
		s.response(resp, 4401, "TransferCode already exists")
		return
	}
	bill := walletStorage.NewBill(apiUser.Uid, walletStorage.TypeIncome, walletStorage.EventGameSbo, bonusRecord.Oid.Hex(), int64(bonusRecord.Amount))
	bill.Remark = "Sbo Tip"
	bonusRecord.SetTransactionUnits(apiStorage.AddSboBetRecord)
	if e := walletStorage.OperateVndBalanceV1(bill, bonusRecord); e != nil {
		log.Error("Sbo Tip wallet pay bet _id:%s err:%s", bonusRecord.Oid.Hex(), e.Error())
		s.response(resp, 7, e.Error())
		return
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
	resp["Body"] = map[string]interface{}{
		"AccountName":  params.Username,
		"Balance":      float64(wallet.VndBalance) / scale,
		"ErrorCode":    0,
		"ErrorMessage": "No Error",
	}
	return
}

// 归还注额
func (s *SboRpc) ReturnStake(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("Sbo Bonus data:%v", time.Now().Unix())
	params := &struct {
		CompanyKey      string    `json:"CompanyKey"`
		Username        string    `json:"Username"`
		CurrentStake    float64   `json:"CurrentStake"`
		ReturnStakeTime time.Time `json:"ReturnStakeTime"`
		ProductType     int       `json:"ProductType"`
		GameType        int       `json:"GameType"`
		TransferCode    string    `json:"TransferCode"`
		TransactionId   string    `json:"TransactionId"`
	}{}
	resp = map[string]interface{}{}
	if err = s.parse(data, resp, params); err != nil {
		return
	}
	if params.ProductType != 1 { //SportsBook
		s.response(resp, 8, "ProductType Untreated")
		return
	}
	if !s.auth(resp, params.CompanyKey) {
		return
	}
	return
}

// 取得投注状态
func (s *SboRpc) GetBetStatus(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("Sbo Bonus data:%v", time.Now().Unix())
	params := &struct {
		CompanyKey    string `json:"CompanyKey"`
		Username      string `json:"Username"`
		ProductType   int    `json:"ProductType"`
		GameType      int    `json:"GameType"`
		TransferCode  string `json:"TransferCode"`
		TransactionId string `json:"TransactionId"`
		Gpid          int    `json:"Gpid"`
	}{}
	resp = map[string]interface{}{}
	if e := s.parse(data, resp, params); e != nil {
		return
	}

	if !s.auth(resp, params.CompanyKey) {
		return
	}

	mSbo := &apiStorage.SboBetRecord{}
	record, e := mSbo.GetStatus(params.ProductType, params.GameType, params.TransferCode, params.TransactionId)
	body := map[string]interface{}{
		"TransferCode":  params.TransferCode,
		"TransactionId": params.TransactionId,
		"ErrorCode":     0,
		"ErrorMessage":  "No Error",
		"Status":        "",
		"WinLoss":       0,
		"Stake":         0,
	}
	if e != nil {
		log.Error("SboRpc GetBetStatus GetStatus err:%s", e.Error())
		body["ErrorCode"] = 6
		body["ErrorMessage"] = "Bet not exists"
	} else {
		body["Status"] = record.Status
		body["WinLoss"] = record.WinLoss
		body["Stake"] = record.Stake
	}
	resp["Body"] = body
	return

}

// LiveCoin购买
func (s *SboRpc) LiveCoinTransaction(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("Sbo LiveCoinTransaction data:%v", time.Now().Unix())
	params := &struct {
		CompanyKey      string    `json:"CompanyKey"`
		Username        string    `json:"Username"`
		Amount          float64   `json:"Amount"`
		TranscationTime time.Time `json:"TranscationTime"`
		ProductType     int       `json:"ProductType"`
		GameType        int       `json:"GameType"`
		TransferCode    string    `json:"TransferCode"`
		TransactionId   string    `json:"TransactionId"`
	}{}
	resp = map[string]interface{}{}
	if e := s.parse(data, resp, params); e != nil {
		return
	}
	if !s.auth(resp, params.CompanyKey) {
		return
	}

	apiUser, e := s.getApiUser(resp, params.Username)
	if e != nil {
		return
	}
	mSbo := &apiStorage.SboBetRecord{
		Oid:           primitive.NewObjectID(),
		Uid:           apiUser.Uid,
		Username:      params.Username,
		Amount:        params.Amount * scale,
		BetTime:       params.TranscationTime,
		ProductType:   params.ProductType,
		GameType:      params.GameType,
		TransferCode:  params.TransferCode,
		TransactionId: params.TransactionId,
	}
	if mSbo.TransferCodeExists() {
		s.response(resp, 4401, "TransferCode already exists")
		return
	}
	bill := walletStorage.NewBill(apiUser.Uid, walletStorage.TypeExpenses, walletStorage.EventGameSbo, mSbo.Oid.Hex(), -int64(mSbo.Amount))
	bill.Remark = "Sbo LiveCoinTransaction"
	mSbo.SetTransactionUnits(apiStorage.AddSboBetRecord)
	if e = walletStorage.OperateVndBalanceV1(bill, mSbo); e != nil {
		log.Error("wallet pay bet _id:%s err:%s", mSbo.Oid.Hex(), e.Error())
		s.response(resp, 7, e.Error())
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
	resp["Body"] = map[string]interface{}{
		"AccountName":  params.Username,
		"Balance":      float64(wallet.VndBalance) / scale,
		"ErrorCode":    0,
		"ErrorMessage": "No Error",
	}
	return
}

func (s *SboRpc) parse(data, resp map[string]interface{}, params interface{}) (err error) {
	body := data["Body"].(string)
	log.Debug("body:%v", body)
	err = json.Unmarshal([]byte(body), params)
	if err != nil {
		log.Debug("SboRpc parse err:%s", err.Error())
		s.response(resp, 2, "Invalid request format")
	}
	return
}

func (s *SboRpc) getApiUser(resp map[string]interface{}, user string) (apiUser *apiStorage.ApiUser, err error) {
	if len(user) == 0 {
		s.response(resp, 3, "find user err")
		err = fmt.Errorf("Username empty")
		return
	}
	apiUser = &apiStorage.ApiUser{}
	err = apiUser.GetApiUserByAccount(user, apiStorage.SboType)
	if err != nil {
		log.Error("SboRpc GetApiUserByAccount err:%s", err.Error())
		s.response(resp, 1, "find user err")
	}
	return
}

func (s *SboRpc) auth(resp map[string]interface{}, cKey string) (ok bool) {
	if cKey != companyKey {
		s.response(resp, 4, "CompanyKey Error")
		return
	}
	ok = true
	return
}

func (s *SboRpc) response(resp map[string]interface{}, code int, msg string) {
	body := map[string]interface{}{
		"ErrorCode":    code,
		"ErrorMessage": msg,
	}
	resp["Body"] = body
}
