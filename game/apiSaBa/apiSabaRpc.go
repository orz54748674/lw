package apiSaBa

import (
	"encoding/json"
	"fmt"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/activity"
	"vn/storage/activityStorage"
	"vn/storage/apiStorage"
	"vn/storage/gameStorage"
	"vn/storage/walletStorage"
)

type SaBaRpc struct {
}

var (
	waitingStatus         = "waiting"
	runingStatus          = "runing"
	settleStatus          = "settle"
	cashOutStatus         = "cashout"
	cashOutResettleStatus = "cashoutResettle"
	UpdateBetStatus       = "updateBetStatus"
	cancelStatus          = "cancel"
)

// 下注细节
func (s *SaBaRpc) placeBet(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc placeBet data:%v", time.Now().Unix())
	params := &apiStorage.SabaBetRecord{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		s.failed(resp)
		return
	}

	mApiUser := &apiStorage.ApiUser{}
	if e := mApiUser.GetApiUserByAccount(params.UserId, apiStorage.SaBaType); e != nil {
		log.Error("placeBet GetApiUserByAccount err:%s", e.Error())
		s.failed(resp)
		return
	}
	params.BetAmount *= scale
	params.ActualAmount *= scale
	params.CreditAmount *= scale
	params.DebitAmount *= scale
	params.SettleStatus = waitingStatus

	params.Uid = mApiUser.Uid
	params.Oid = primitive.NewObjectID()
	bill := walletStorage.NewBill(mApiUser.Uid, walletStorage.TypeExpenses, walletStorage.EventGameSaba, params.Oid.Hex(), -1*int64(params.DebitAmount))
	bill.Remark = "SaBa placeBet"
	params.SetTransactionUnits(apiStorage.AddSabaBetRecord)
	if e := walletStorage.OperateVndBalanceV1(bill, params); e != nil {
		log.Error("wallet pay bet _id:%s err:%s", params.Oid.Hex(), e.Error())
		s.failed(resp)
		return
	}
	respData := map[string]interface{}{
		"refId":        params.RefId,
		"licenseeTxId": params.Oid,
	}
	s.success(resp, respData)
	return
}

// 下注细节 仅支援欧洲盘
func (s *SaBaRpc) placeBetParlay(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc placeBetParlay data:%v", time.Now().Unix())
	params := &struct {
		OperationId           string    `bson:"OperationId" json:"operationId"`
		UserId                string    `bson:"UserId" json:"userId"`
		Currency              int       `bson:"Currency" json:"currency"`
		BetTime               time.Time `bson:"BetTime" json:"betTime"`
		TotalBetAmount        float64   `bson:"TotalBetAmount" json:"totalBetAmount"`
		BefoareDiscountAmount float64   `bson:"BeforeDiscountAmount" json:"beforeDiscountAmount"`
		UpdateTime            time.Time `bson:"UpdateTime" json:"updateTime"`
		Percentage            float64   `bson:"Percentage" json:"percentage"`
		IP                    string    `bson:"IP" json:"IP"`
		TsId                  string    `bson:"TsId" json:"tsId"`
		BetFrom               string    `bson:"BetFrom" json:"betFrom"`
		CreditAmount          float64   `bson:"CreditAmount" json:"creditAmount"`
		DebitAmount           float64   `bson:"DebitAmount" json:"debitAmount"`
		Txns                  []struct {
			RefId      string  `bson:"RefId" json:"refId"`
			BetAmount  float64 `bson:"BetAmount" json:"betAmount"`
			ParlayType string  `bson:"ParlayType" json:"parlayType"`
			Detail     []struct {
				MatchId  int     `bson:"MatchId" json:"matchId"`
				Type     int8    `json:"type"`
				Name     string  `json:"name"`
				BetCount int     `josn:"betCount"`
				Stake    float64 `json:"stake"`
			} `json:"detail"`
		} `json:"txns"`
		TicketDetail []*apiStorage.SabaBetRecord `json:"ticketDetail"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		s.failed(resp)
		return
	}
	if len(params.Txns) == 0 {
		s.failed(resp)
		return
	}
	mApiUser := &apiStorage.ApiUser{}
	if e := mApiUser.GetApiUserByAccount(params.UserId, apiStorage.SaBaType); e != nil {
		log.Error("SaBaRpc placeBetParlay GetApiUserByAccount err:%s", e.Error())
		s.failed(resp)
		return
	}
	var refIds []string
	for _, tx := range params.Txns {
		refIds = append(refIds, tx.RefId)
	}
	ticketMap := map[int]*apiStorage.SabaBetRecord{}
	oids := []string{}
	for _, tk := range params.TicketDetail {
		tk.Oid = primitive.NewObjectID()
		ticketMap[tk.MatchId] = tk
		tk.Uid = mApiUser.Uid
		tk.UserId = params.UserId
		tk.OperationId = params.OperationId
		tk.IP = params.IP
		tk.BetTime = params.BetTime
		tk.BetFrom = params.BetFrom
		tk.SettleStatus = runingStatus
		tk.AddSabaBetRecord()
		oids = append(oids, tk.Oid.Hex())
	}
	m := &apiStorage.SabaBetRecord{}
	records, e := m.GetRecordByRefIds(refIds, "")
	if e != nil || len(records) > 0 {
		log.Error("SaBaRpc placeBetParlay GetRecords err:%v,len(records):%v", e, len(records))
		s.failed(resp)
		return
	}
	respData := []map[string]interface{}{}
	for _, tx := range params.Txns {
		btDetail, e := json.Marshal(tx.Detail)
		if e != nil {
			log.Error("SaBaRpc placeBetParlay Marshal Detail err:%v", e.Error())
			continue
		}
		tx.BetAmount *= scale
		record := &apiStorage.SabaBetRecord{
			Oid:          primitive.NewObjectID(),
			OperationId:  params.OperationId,
			UserId:       params.UserId,
			Currency:     params.Currency,
			BetTime:      params.BetTime,
			UpdateTime:   params.UpdateTime,
			IP:           params.IP,
			Uid:          mApiUser.Uid,
			TsId:         params.TsId,
			BetFrom:      params.BetFrom,
			RefId:        tx.RefId,
			BetAmount:    tx.BetAmount,
			ParlayType:   tx.ParlayType,
			SettleStatus: waitingStatus,
			Detail:       string(btDetail),
			DebitAmount:  tx.BetAmount,
			ActualAmount: tx.BetAmount,
		}
		if tx.ParlayType == "SingleBet_ViaLucky" {
			tk := ticketMap[tx.Detail[0].MatchId]
			record.MatchId = tk.MatchId
			record.HomeId = tk.HomeId
			record.AwayId = tk.AwayId
			record.HomeName = tk.HomeName
			record.AwayName = tk.AwayName
			record.KickOffTime = tk.KickOffTime
			record.SportType = tk.SportType
			record.SportTypeName = tk.SportTypeName
			record.BetType = tk.BetType
			record.BetTypeName = tk.BetTypeName
			record.OddsId = tk.OddsId
			record.Odds = tk.Odds
			record.OddsType = tk.OddsType
			record.BetChoice = tk.BetChoice
			record.LeagueId = tk.LeagueId
			record.LeagueName = tk.LeagueName
			record.IsLive = tk.IsLive
			record.Point = tk.Point
			record.Point2 = tk.Point2
			record.BetTeam = tk.BetTeam
			record.HomeScore = tk.HomeScore
			record.AwayScore = tk.AwayScore
			record.BaStatus = tk.BaStatus
			record.Excluding = tk.Excluding
			record.LeagueNameEn = tk.LeagueNameEn
			record.SportTypeNameEn = tk.SportTypeNameEn
			record.HomeNameEn = tk.HomeNameEn
			record.AwayNameEn = tk.AwayNameEn
			record.BetTypeNameEn = tk.BetTypeNameEn
		} else {
			record.MatchOids = oids
		}
		record.SetTransactionUnits(apiStorage.AddSabaBetRecord)
		bill := walletStorage.NewBill(mApiUser.Uid, walletStorage.TypeExpenses, walletStorage.EventGameSaba, record.Oid.Hex(), -1*int64(record.DebitAmount))
		bill.Remark = "SaBa placeBetParlay"
		record.SetTransactionUnits(apiStorage.AddSabaBetRecord)
		if e := walletStorage.OperateVndBalanceV1(bill, record); e != nil {
			log.Error("SaBa placeBetParlay wallet pay bet _id:%s err:%s", record.Oid.Hex(), e.Error())
			continue
		}
		// records = append(records, record)
		respData = append(respData, map[string]interface{}{"refId": record.RefId, "licenseeTxId": record.Oid.Hex()})
	}
	s.success(resp, map[string]interface{}{"txns": respData})
	log.Debug("params:%v", params)
	return
}

func (s *SaBaRpc) confirmBet(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc confirmBet data:%v", time.Now().Unix())
	params := &struct {
		OperationId string    `json:"operationId"`
		UserId      string    `json:"userId"`
		UpdateTime  time.Time `json:"updateTime"`
		Txns        []*struct {
			RefId         string  `json:"refId"`
			TxId          int64   `json:"txId"`
			LicenseeTxId  string  `json:"licenseeTxId"`
			Odds          float64 `json:"odds"`
			OddsType      int     `json:"oddsType"`
			ActualAmount  float64 `json:"actualAmount"`
			IsOddsChanged bool    `json:"isOddsChanged"`
			CreditAmount  float64 `bson:"CreditAmount" json:"creditAmount"`
			DebitAmount   float64 `bson:"DebitAmount" json:"debitAmount"`
		} `json:"txns"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		s.failed(resp)
		return
	}
	if len(params.Txns) == 0 {
		s.failed(resp)
		return
	}
	oids := []primitive.ObjectID{}
	for _, tx := range params.Txns {
		oids = append(oids, utils.ConvertOID(tx.LicenseeTxId))
	}
	mApiUser := &apiStorage.ApiUser{}
	if e := mApiUser.GetApiUserByAccount(params.UserId, apiStorage.SaBaType); e != nil {
		log.Error("SaBaRpc confirmBet GetApiUserByAccount err:%s", e.Error())
		s.failed(resp)
		return
	}
	m := &apiStorage.SabaBetRecord{}
	records, e := m.GetRecordByOids(oids)
	if e != nil {
		log.Error("SaBaRpc confirmBet GetRecords err:%s", e.Error())
		s.failed(resp)
		return
	}
	recordMap := map[string]*apiStorage.SabaBetRecord{}
	for _, record := range records {
		recordMap[record.Oid.Hex()] = record
	}
	for _, tx := range params.Txns {
		log.Debug("tx.LicenseeTxId:%v", tx.LicenseeTxId)
		log.Debug("recordMap:%v", recordMap)
		record, ok := recordMap[tx.LicenseeTxId]
		if !ok {
			continue
		}
		if record.Status == "" {
			data := bson.M{"SettleStatus": runingStatus, "TxId": tx.TxId}
			tx.ActualAmount *= scale
			tx.CreditAmount *= scale
			record.TxId = tx.TxId
			oid := record.Oid.Hex()
			if tx.IsOddsChanged {
				record.ActualAmount = tx.ActualAmount
				record.CreditAmount = tx.CreditAmount
				record.Odds = tx.Odds
				record.OddsType = tx.OddsType

				data["ActualAmount"] = tx.ActualAmount
				data["CreditAmount"] = tx.CreditAmount
				data["Odds"] = tx.Odds
				data["OddsType"] = tx.OddsType

				bill := walletStorage.NewBill(mApiUser.Uid, walletStorage.TypeIncome, walletStorage.EventGameSaba, oid, int64(record.BetAmount))
				bill.Remark = "SaBa confirmBet"
				if e := walletStorage.OperateVndBalanceV1(bill); e != nil {
					log.Error("SaBa confirmBet wallet pay bet _id:%s err:%s", oid, e.Error())
					continue
				}
			}
			e := record.Update(data)
			if e != nil {
				log.Error("SaBaRpc confirmBet Update err:%s", e.Error())
				continue
			}
			wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
			btBetInfo, _ := json.Marshal(s.createBetInfo(record))
			betRecordData := gameStorage.BetRecordParam{
				Uid:        mApiUser.Uid,
				GameType:   game.ApiSaBa,
				Income:     0,
				BetAmount:  int64(tx.ActualAmount),
				CurBalance: wallet.VndBalance + wallet.SafeBalance,
				SysProfit:  0,
				BotProfit:  0,
				BetDetails: string(btBetInfo),
				GameId:     oid,
				GameNo:     fmt.Sprint(tx.TxId),
				GameResult: "",
				IsSettled:  false,
			}
			gameStorage.InsertBetRecord(betRecordData)
			activityStorage.UpsertGameDataInBet(mApiUser.Uid, game.ApiSaBa, 1)
		}
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
	respData := map[string]interface{}{
		"balance": float64(wallet.VndBalance) / scale,
	}
	s.success(resp, respData)
	return
}

func (s *SaBaRpc) confirmBetParlay(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc confirmBetParlay data:%v", time.Now().Unix())
	params := &struct {
		OperationId string    `bson:"OperationId" json:"operationId"`
		UserId      string    `bson:"UserId" json:"userId"`
		UpdateTime  time.Time `bson:"UpdateTime" json:"updateTime"`
		Txns        []struct {
			RefId         string  `bson:"RefId" json:"refId"`
			TxId          int64   `bson:"TxId" json:"txId"`
			LicenseeTxId  string  `json:"licenseeTxId"`
			IsOddsChanged bool    `json:"isOddsChanged"`
			ActualAmount  float64 `json:"actualAmount"`
			CreditAmount  float64 `bson:"CreditAmount" json:"creditAmount"`
			DebitAmount   float64 `bson:"DebitAmount" json:"debitAmount"`
		} `json:"txns"`
		TicketDetail []*apiStorage.SabaBetRecord `json:"ticketDetail"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		s.failed(resp)
		return
	}
	if len(params.Txns) == 0 {
		s.failed(resp)
		return
	}
	mApiUser := &apiStorage.ApiUser{}
	if e := mApiUser.GetApiUserByAccount(params.UserId, apiStorage.SaBaType); e != nil {
		log.Error("SaBaRpc confirmBetParlay GetApiUserByAccount err:%s", e.Error())
		s.failed(resp)
		return
	}
	var refIds []string
	for _, tx := range params.Txns {
		refIds = append(refIds, tx.RefId)
	}

	m := &apiStorage.SabaBetRecord{}
	records, e := m.GetRecordByRefIds(refIds, waitingStatus)
	if e != nil {
		log.Error("SaBaRpc confirmBetParlay GetRecords err:%s", e.Error())
		s.failed(resp)
		return
	}

	if len(records) == 0 {
		log.Error("SaBaRpc confirmBetParlay records: empty")
		s.failed(resp)
		return
	}
	recordMap := map[string]*apiStorage.SabaBetRecord{}
	oids := []primitive.ObjectID{}
	for _, record := range records {
		recordMap[record.Oid.Hex()] = record
		for _, v := range record.MatchOids {
			oids = append(oids, utils.ConvertOID(v))
		}
	}
	matchRecords, e := m.GetRecordByOids(oids)
	if e != nil {
		log.Error("SaBaRpc confirmBetParlay GetRecordByMatchIds err:%s", e.Error())
		s.failed(resp)
		return
	}
	tkMap := map[int]*apiStorage.SabaBetRecord{}
	for _, tk := range params.TicketDetail {
		tkMap[tk.MatchId] = tk
	}

	betInfos := []map[string]interface{}{}
	matchs := []*apiStorage.SabaBetRecord{}
	for _, matchRecord := range matchRecords {
		if mR, ok := tkMap[matchRecord.MatchId]; ok {
			matchRecord.BetType = mR.BetType
			matchRecord.SportType = mR.SportType
			matchRecord.OddsId = mR.OddsId
			matchRecord.Odds = mR.Odds
			matchRecord.OddsType = mR.OddsType
			matchRecord.LeagueId = mR.LeagueId
			matchRecord.IsLive = mR.IsLive
			matchRecord.IsOddsChanged = mR.IsOddsChanged
			tkMap[matchRecord.MatchId].Oid = matchRecord.Oid
			matchs = append(matchs, matchRecord)
		}
	}
	for _, match := range matchs {
		tkMap[match.MatchId] = match

		betInfos = append(betInfos, s.createBetInfo(match))
	}
	btBetInfos, _ := json.Marshal(betInfos)
	for _, tx := range params.Txns {
		record, ok := recordMap[tx.LicenseeTxId]
		if !ok {
			continue
		}

		data := bson.M{"SettleStatus": runingStatus, "TxId": tx.TxId}
		tx.ActualAmount *= scale
		tx.CreditAmount *= scale
		record.TxId = tx.TxId
		oid := record.Oid.Hex()
		if tx.IsOddsChanged {
			record.ActualAmount = tx.ActualAmount
			record.CreditAmount = tx.CreditAmount

			data["ActualAmount"] = tx.ActualAmount
			data["CreditAmount"] = tx.CreditAmount

			bill := walletStorage.NewBill(mApiUser.Uid, walletStorage.TypeIncome, walletStorage.EventGameSaba, oid, int64(record.BetAmount))
			bill.Remark = "SaBa confirmBetParlay"
			if e := walletStorage.OperateVndBalanceV1(bill); e != nil {
				log.Error("SaBa confirmBetParlay wallet pay bet _id:%s err:%s", oid, e.Error())
				continue
			}
		}
		e := record.Update(data)
		if e != nil {
			log.Error("SaBaRpc confirmBetParlay Update err:%s", e.Error())
			continue
		}
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		btBetInfo := []byte{}
		if record.ParlayType == "SingleBet_ViaLucky" {
			btBetInfo, _ = json.Marshal(s.createBetInfo(record))
		} else {
			btBetInfo = btBetInfos
		}

		betRecordData := gameStorage.BetRecordParam{
			Uid:        mApiUser.Uid,
			GameType:   game.ApiSaBa,
			Income:     0,
			BetAmount:  int64(tx.ActualAmount),
			CurBalance: wallet.VndBalance + wallet.SafeBalance,
			SysProfit:  0,
			BotProfit:  0,
			BetDetails: string(btBetInfo),
			GameId:     oid,
			GameNo:     fmt.Sprint(tx.TxId),
			GameResult: "",
			IsSettled:  false,
		}
		gameStorage.InsertBetRecord(betRecordData)
		activityStorage.UpsertGameDataInBet(mApiUser.Uid, game.ApiSaBa, 1)
	}
	for _, tk := range params.TicketDetail {
		if tk.IsOddsChanged {
			record := tkMap[tk.MatchId]
			data := bson.M{
				"SportType": tk.SportType,
				"BetType":   tk.BetType,
				"OddsId":    tk.OddsId,
				"Odds":      tk.Odds,
				"OddsType":  tk.OddsType,
				// "BetChoice": tk.BetChoice,
				"LeagueId": tk.LeagueId,
				"IsLive":   tk.IsLive,
			}
			e := record.Update(data)
			if e != nil {
				log.Error("SaBaRpc confirmBetParlay Update err:%s", e.Error())
				continue
			}
		}
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
	s.success(resp, map[string]interface{}{"balance": float64(wallet.VndBalance) / scale})
	return
}

func (s *SaBaRpc) cancelBet(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc cancelBet data:%v", time.Now().Unix())
	params := &struct {
		OperationId string    `json:"operationId"`
		UserId      string    `json:"userId"`
		UpdateTime  time.Time `json:"updateTime"`
		Txns        []*struct {
			RefId        string  `json:"refId"`
			CreditAmount float64 `json:"creditAmount"`
			DebitAmount  float64 `json:"debitAmount"`
		} `json:"txns"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		s.failed(resp)
		return
	}
	if len(params.Txns) == 0 {
		s.failed(resp)
		return
	}
	refId := []string{}
	for _, tx := range params.Txns {
		refId = append(refId, tx.RefId)
	}
	mBetRecord := &apiStorage.SabaBetRecord{}
	records, e := mBetRecord.GetRecordByRefIds(refId, waitingStatus)
	if e != nil {
		log.Error("SaBaRpc cancelBet GetRecords err:%s", e.Error())
		s.failed(resp)
		return
	}
	if len(records) == 0 {
		s.failed(resp)
		return
	}
	recordMap := map[string]*apiStorage.SabaBetRecord{}
	uid := ""
	for _, record := range records {
		recordMap[record.RefId] = record
		uid = record.Uid
	}

	for _, tx := range params.Txns {
		record, ok := recordMap[tx.RefId]
		if !ok {
			continue
		}
		record.CreditAmount = tx.CreditAmount * scale
		record.SettleStatus = cancelStatus
		bill := walletStorage.NewBill(record.Uid, walletStorage.TypeIncome, walletStorage.EventGameSaba, record.Oid.Hex(), int64(record.CreditAmount))
		bill.Remark = "SaBa cancelBet"
		record.SetTransactionUnits(apiStorage.CancelSabaBetRecord)
		if e := walletStorage.OperateVndBalanceV1(bill, record); e != nil {
			log.Error("SaBa cancelBet wallet pay bet _id:%s err:%s", record.Oid.Hex(), e.Error())
			continue
		}
	}
	if len(uid) == 0 {
		s.failed(resp)
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	s.success(resp, map[string]interface{}{"balance": float64(wallet.VndBalance) / scale})
	return
}

func (s *SaBaRpc) settle(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc settle data:%v", time.Now().Unix())
	params := &struct {
		OperationId string `json:"operationId"`
		Txns        []*struct {
			UserId       string    `json:"userId"`
			RefId        string    `json:"refId"`
			TxId         int64     `json:"txId"`
			UpdateTime   time.Time `json:"updateTime"`
			WinlostDate  time.Time `json:"winlostDate"`
			Status       string    `json:"status"`
			Payout       float64   `json:"payout"`
			CreditAmount float64   `json:"creditAmount"`
			DebitAmount  float64   `json:"debitAmount"`
		} `json:"txns"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		s.failed(resp)
		return
	}
	if len(params.Txns) == 0 {
		s.failed(resp)
		return
	}
	TxId := []int64{}
	for _, tx := range params.Txns {
		TxId = append(TxId, tx.TxId)
	}
	mBetRecord := &apiStorage.SabaBetRecord{}
	records, e := mBetRecord.GetRecordByTxIds(TxId, runingStatus)
	if e != nil {
		log.Error("SaBaRpc settle GetRecords err:%s", e.Error())
		s.failed(resp)
		return
	}

	if len(records) == 0 {
		s.failed(resp)
		return
	}
	recordMap := map[int64]*apiStorage.SabaBetRecord{}
	for _, record := range records {
		recordMap[record.TxId] = record
	}
	for _, tx := range params.Txns {
		record, ok := recordMap[tx.TxId]
		if !ok {
			continue
		}
		record.Payout = tx.Payout * scale
		record.WinlostDate = tx.WinlostDate
		record.UpdateTime = tx.UpdateTime
		record.Status = tx.Status
		if tx.Payout == 0 {
			info := map[string]interface{}{
				"Payout":       record.Payout,
				"WinlostDate":  record.WinlostDate,
				"UpdateTime":   record.UpdateTime,
				"Status":       record.Status,
				"SettleStatus": settleStatus,
			}
			if e := record.Update(info); e != nil {
				log.Error("SaBaRpc settle record.Update Err:%v", e.Error())
				continue
			}
		} else {
			bill := walletStorage.NewBill(record.Uid, walletStorage.TypeIncome, walletStorage.EventGameSaba, record.Oid.Hex(), int64(tx.Payout))
			bill.Remark = "SaBa settle"
			record.SetTransactionUnits(apiStorage.SettleSabaBetRecord)
			if e := walletStorage.OperateVndBalanceV1(bill, record); e != nil {
				log.Error("SaBa settle wallet pay bet _id:%s err:%s", record.Oid.Hex(), e.Error())
				continue
			}
		}

		// profit := record.Payout - record.CreditAmount
		profit := record.Payout - record.BetAmount
		gameRes := fmt.Sprintf(`{"Status":"%s","SettleStatus":"%s"}`, record.Status, record.SettleStatus)

		wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
		gameStorage.UpdateBetRecord(record.Oid.Hex(), record.Uid, fmt.Sprint(record.TxId), gameRes, game.ApiSaBa, int64(profit), wallet.VndBalance+wallet.SafeBalance, int64(record.BetAmount))
		activityStorage.UpsertGameDataInBet(record.Uid, game.ApiSaBa, -1)
		activity.CalcEncouragementFunc(record.Uid)

		log.Debug("record:%v", record)
	}
	s.success(resp, map[string]interface{}{})
	return
}

func (s *SaBaRpc) resettle(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc resettle data:%v", time.Now().Unix())
	params := &struct {
		OperationId string `json:"operationId"`
		Txns        []*struct {
			UserId       string    `json:"userId"`
			RefId        string    `json:"refId"`
			TxId         int64     `json:"txId"`
			UpdateTime   time.Time `json:"updateTime"`
			WinlostDate  time.Time `json:"winlostDate"`
			Status       string    `json:"status"`
			Payout       float64   `json:"payout"`
			CreditAmount float64   `json:"creditAmount"`
			DebitAmount  float64   `json:"debitAmount"`
		} `json:"txns"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		s.failed(resp)
		return
	}
	if len(params.Txns) == 0 {
		s.failed(resp)
		return
	}
	TxId := []int64{}
	for _, tx := range params.Txns {
		TxId = append(TxId, tx.TxId)
	}
	mBetRecord := &apiStorage.SabaBetRecord{}
	records, e := mBetRecord.GetRecordByTxIds(TxId, settleStatus)
	if e != nil {
		log.Error("SaBaRpc resettle GetRecords err:%s", e.Error())
		s.failed(resp)
		return
	}

	if len(records) == 0 {
		s.failed(resp)
		return
	}
	recordMap := map[int64]*apiStorage.SabaBetRecord{}
	for _, record := range records {
		recordMap[record.TxId] = record
	}
	for _, tx := range params.Txns {
		record, ok := recordMap[tx.TxId]
		if !ok {
			continue
		}
		tx.CreditAmount *= scale
		tx.DebitAmount *= scale
		record.Payout = tx.Payout * scale
		record.WinlostDate = tx.WinlostDate
		record.UpdateTime = tx.UpdateTime
		record.Status = tx.Status
		var bill *walletStorage.Bill
		if tx.CreditAmount > 0 {
			bill = walletStorage.NewBill(record.Uid, walletStorage.TypeIncome, walletStorage.EventGameSaba, record.Oid.Hex(), int64(tx.CreditAmount))
		} else {
			bill = walletStorage.NewBill(record.Uid, walletStorage.TypeExpenses, walletStorage.EventGameSaba, record.Oid.Hex(), -int64(tx.DebitAmount))
		}
		bill.Remark = "SaBa resettle"
		record.SetTransactionUnits(apiStorage.SettleSabaBetRecord)
		if e := walletStorage.OperateVndBalanceV1(bill, record); e != nil {
			log.Error("SaBa resettle wallet pay bet _id:%s err:%s", record.Oid.Hex(), e.Error())
			continue
		}

		// profit := record.Payout - record.CreditAmount
		profit := record.Payout - record.BetAmount
		gameRes := fmt.Sprintf(`{"Status":"%s","SettleStatus":"%s"}`, record.Status, record.SettleStatus)

		wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
		gameStorage.RefundBetRecord(record.Oid.Hex(), "", game.ApiSaBa, 0, int64(record.CreditAmount), 0)
		gameStorage.UpdateBetRecord(record.Oid.Hex(), record.Uid, fmt.Sprint(record.TxId), gameRes, game.ApiSaBa, int64(profit), wallet.VndBalance+wallet.SafeBalance, int64(record.BetAmount))
		activity.CalcEncouragementFunc(record.Uid)

		log.Debug("record:%v", record)
	}
	s.success(resp, map[string]interface{}{})
	return
}

func (s *SaBaRpc) unsettle(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc unsettle data:%v", time.Now().Unix())
	params := &struct {
		OperationId string `json:"operationId"`
		Txns        []*struct {
			UserId       string    `json:"userId"`
			RefId        string    `json:"refId"`
			TxId         int64     `json:"txId"`
			UpdateTime   time.Time `json:"updateTime"`
			CreditAmount float64   `json:"creditAmount"`
			DebitAmount  float64   `json:"debitAmount"`
		} `json:"txns"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		s.failed(resp)
		return
	}
	if len(params.Txns) == 0 {
		s.failed(resp)
		return
	}
	TxId := []int64{}
	for _, tx := range params.Txns {
		TxId = append(TxId, tx.TxId)
	}
	mBetRecord := &apiStorage.SabaBetRecord{}
	records, e := mBetRecord.GetRecordByTxIds(TxId, settleStatus)
	if e != nil {
		log.Error("SaBaRpc unsettle GetRecords err:%s", e.Error())
		s.failed(resp)
		return
	}
	if len(records) == 0 {
		s.failed(resp)
		return
	}
	recordMap := map[int64]*apiStorage.SabaBetRecord{}
	for _, record := range records {
		recordMap[record.TxId] = record
	}
	for _, tx := range params.Txns {
		record, ok := recordMap[tx.TxId]
		if !ok {
			continue
		}
		tx.CreditAmount *= scale
		tx.DebitAmount *= scale
		record.SettleStatus = runingStatus
		record.Status = "unsettle"
		if tx.CreditAmount > 0 {
			bill := walletStorage.NewBill(record.Uid, walletStorage.TypeExpenses, walletStorage.EventGameSaba, record.Oid.Hex(), -int64(tx.CreditAmount))
			bill.Remark = "SaBa unsettle"
			record.SetTransactionUnits(apiStorage.UnsettleSabaBetRecord)
			if e := walletStorage.OperateVndBalanceV1(bill, record); e != nil {
				log.Error("SaBa unsettle wallet pay bet _id:%s err:%s", record.Oid.Hex(), e.Error())
				continue
			}
		} else {
			info := map[string]interface{}{
				"Payout":       0,
				"WinlostDate":  record.WinlostDate,
				"UpdateTime":   record.UpdateTime,
				"Status":       record.Status,
				"SettleStatus": record.SettleStatus,
			}
			if e := record.Update(info); e != nil {
				log.Error("SaBaRpc unsettle record.Update Err:%v", e.Error())
				continue
			}
		}
		gameStorage.RefundBetRecord(record.Oid.Hex(), "", game.ApiSaBa, 0, int64(record.CreditAmount), 0)
		activityStorage.UpsertGameDataInBet(record.Uid, game.ApiSaBa, 1)

		log.Debug("record:%v", record)
	}
	s.success(resp, map[string]interface{}{})
	return
}

func (s *SaBaRpc) getBalance(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc getBalance time:%v", time.Now().Unix())
	params := &struct {
		Action string `json:"action"`
		UserId string `json:"userId"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		return
	}
	log.Debug("SaBaRpc getBalance params:%v", params)
	mApiUser := &apiStorage.ApiUser{}
	if e := mApiUser.GetApiUserByAccount(params.UserId, apiStorage.SaBaType); e != nil {
		log.Error("SaBaRpc getBalance GetApiUserByAccount err:%s", e.Error())
		return
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))

	resp["Body"] = map[string]interface{}{
		"status":    0,
		"userId":    params.UserId,
		"balance":   float64(wallet.VndBalance) / scale,
		"balanceTs": s.timeISO8601(),
		"msg":       "",
	}
	return
}

type txn struct {
	UserId                 string    `json:"userId"`
	RefId                  string    `json:"refId"`
	TxId                   int64     `json:"txId"`
	UpdateTime             time.Time `json:"updateTime"`
	AvailableCashOutAmount float64   `json:"availableCashOutAmount"`
	CashOutActualAmount    float64   `json:"CashOutActualAmount"`
	CashOutAmount          float64   `json:"CashOutAmount"`
	BuybackAmount          float64   `json:"buybackAmount"`
	TicketStatus           string    `json:"TicketStatus"`
	CreditAmount           float64   `json:"creditAmount"`
	DebitAmount            float64   `json:"debitAmount"`
	OriginalTicket         *struct {
		TxId  int64  `json:"txId"`
		RefId string `json:"refId"`
	} `json:"originalTicket"`
}

func (s *SaBaRpc) cashOut(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc cashOut data:%v", time.Now().Unix())
	params := &struct {
		OperationId string `bson:"OperationId" json:"operationId"`
		Txns        []*txn `json:"txns"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		return
	}
	log.Debug("SaBaRpc cashOut params:%v", params)

	refIds := []string{}
	ticketMap := map[string]*txn{}
	for _, item := range params.Txns {
		ticketMap[item.OriginalTicket.RefId] = item
		refIds = append(refIds, item.OriginalTicket.RefId)
	}
	mBetRecord := &apiStorage.SabaBetRecord{}
	records, e := mBetRecord.GetRecordByRefIds(refIds, runingStatus)
	if e != nil {
		log.Error("SaBaRpc cashOut GetRecordByRefIds err:%s", e.Error())
		s.failed(resp)
		return
	}
	for _, record := range records {
		if record.CashStatus == 1 {
			continue
		}
		tx := ticketMap[record.RefId]
		tx.DebitAmount *= scale
		bill := walletStorage.NewBill(record.Uid, walletStorage.TypeExpenses, walletStorage.EventGameSaba, record.Oid.Hex(), -int64(tx.DebitAmount))
		bill.Remark = "SaBa cashOut"
		record.UpdateTime = tx.UpdateTime
		record.Payout -= tx.DebitAmount
		record.Status = cashOutStatus
		record.SettleStatus = cashOutStatus
		record.CashStatus += 1
		record.SetTransactionUnits(apiStorage.CashOutSabaBetRecord)
		if e := walletStorage.OperateVndBalanceV1(bill, record); e != nil {
			log.Error("SaBa settle wallet pay bet _id:%s err:%s", record.Oid.Hex(), e.Error())
			continue
		}
		profit := record.Payout - record.BetAmount
		gameRes := fmt.Sprintf(`{"Status":"%s","SettleStatus":"%s"}`, record.Status, record.SettleStatus)
		wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
		gameStorage.UpdateBetRecord(record.Oid.Hex(), record.Uid, fmt.Sprint(record.TxId), gameRes, game.ApiSaBa, int64(profit), wallet.VndBalance+wallet.SafeBalance, int64(record.BetAmount))
		activityStorage.UpsertGameDataInBet(record.Uid, game.ApiSaBa, -1)
		activity.CalcEncouragementFunc(record.Uid)
	}
	s.success(resp, map[string]interface{}{})
	return
}

func (s *SaBaRpc) cashOutResettle(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc cashOutResettle data:%v", time.Now().Unix())
	params := &struct {
		OperationId string `bson:"OperationId" json:"operationId"`
		Txns        []*txn `json:"txns"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		return
	}
	log.Debug("SaBaRpc cashOutResettle params:%v", params)

	refIds := []string{}
	ticketMap := map[string]*txn{}
	for _, item := range params.Txns {
		ticketMap[item.OriginalTicket.RefId] = item
		refIds = append(refIds, item.OriginalTicket.RefId)
	}
	mBetRecord := &apiStorage.SabaBetRecord{}
	records, e := mBetRecord.GetRecordByRefIds(refIds, "")
	if e != nil {
		log.Error("SaBaRpc cashOutResettle GetRecordByRefIds err:%s", e.Error())
		s.failed(resp)
		return
	}
	for _, record := range records {
		tx := ticketMap[record.RefId]
		tx.DebitAmount *= scale
		bill := walletStorage.NewBill(record.Uid, walletStorage.TypeIncome, walletStorage.EventGameSaba, record.Oid.Hex(), int64(tx.CreditAmount))
		bill.Remark = "SaBa cashOutResettle"
		record.UpdateTime = tx.UpdateTime
		record.Payout = 0
		record.Status = cashOutResettleStatus
		record.SettleStatus = cashOutResettleStatus
		record.SetTransactionUnits(apiStorage.CashOutResettleSabaBetRecord)
		if e := walletStorage.OperateVndBalanceV1(bill, record); e != nil {
			log.Error("SaBa cashOutResettle wallet pay bet _id:%s err:%s", record.Oid.Hex(), e.Error())
			continue
		}
		profit := 0
		gameRes := fmt.Sprintf(`{"Status":"%s","SettleStatus":"%s"}`, record.Status, record.SettleStatus)
		wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
		gameStorage.UpdateBetRecord(record.Oid.Hex(), record.Uid, fmt.Sprint(record.TxId), gameRes, game.ApiSaBa, int64(profit), wallet.VndBalance+wallet.SafeBalance, int64(record.BetAmount))
	}
	s.success(resp, map[string]interface{}{})
	return
}

func (s *SaBaRpc) updateBet(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc updateBet data:%v", time.Now().Unix())
	params := &struct {
		OperationId string `bson:"OperationId" json:"operationId"`
		Txns        []struct {
			UserId       string    `json:"userId"`
			RefId        string    `json:"refId"`
			TxId         int64     `json:"txId"`
			UpdateTime   time.Time `json:"updateTime"`
			BetAmount    float64   `json:"betAmount"`
			ActualAmount float64   ` json:"actualAmount"`
			OddsType     int       `json:"oddsType"`
			Odds         float64   `json:"odds"`
			CreditAmount float64   `json:"creditAmount"`
			DebitAmount  float64   `json:"debitAmount"`
		} `json:"txns"`
	}{}
	resp = map[string]interface{}{}
	if err = s.auth(data, params); err != nil {
		return
	}
	log.Debug("SaBaRpc updateBet params:%v", params)

	refIds := []string{}
	for _, item := range params.Txns {
		refIds = append(refIds, item.RefId)
	}
	mBetRecord := &apiStorage.SabaBetRecord{}
	records, e := mBetRecord.GetRecordByRefIds(refIds, "")
	if e != nil {
		log.Error("SaBaRpc updateBet GetRecordByRefIds err:%s", e.Error())
		s.failed(resp)
		return
	}
	recordMap := map[string]*apiStorage.SabaBetRecord{}
	for _, record := range records {
		recordMap[record.RefId] = record
	}
	for _, tx := range params.Txns {
		record := recordMap[tx.RefId]
		if record.CashStatus == 2 {
			continue
		}
		tx.CreditAmount *= scale
		bill := walletStorage.NewBill(record.Uid, walletStorage.TypeIncome, walletStorage.EventGameSaba, record.Oid.Hex(), int64(tx.CreditAmount))
		bill.Remark = "SaBa updateBet"
		record.UpdateTime = tx.UpdateTime
		record.Payout += tx.CreditAmount
		record.Status = cashOutStatus
		record.SettleStatus = cashOutStatus
		record.CashStatus += 2
		record.SetTransactionUnits(apiStorage.CashOutSabaBetRecord)
		if e := walletStorage.OperateVndBalanceV1(bill, record); e != nil {
			log.Error("SaBa updateBet wallet pay bet _id:%s err:%s", record.Oid.Hex(), e.Error())
			continue
		}
		profit := record.Payout - record.BetAmount
		gameRes := fmt.Sprintf(`{"Status":"%s","SettleStatus":"%s"}`, record.Status, record.SettleStatus)
		wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
		gameStorage.UpdateBetRecord(record.Oid.Hex(), record.Uid, fmt.Sprint(record.TxId), gameRes, game.ApiSaBa, int64(profit), wallet.VndBalance+wallet.SafeBalance, int64(record.BetAmount))
	}
	s.success(resp, map[string]interface{}{})
	return
}

func (s *SaBaRpc) placeBet3rd(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc placeBet3rd data:%v", time.Now().Unix())
	return
}
func (s *SaBaRpc) confirmBet3rd(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SaBaRpc confirmBet3rd data:%v", time.Now().Unix())
	return
}
func (s *SaBaRpc) auth(reqData map[string]interface{}, params interface{}) (err error) {
	body := reqData["Body"].(string)
	log.Debug("SaBaRpc body:%v", body)
	data := &struct {
		Key     string      `json:"key"`
		Message interface{} `json:"message"`
	}{
		Key:     "",
		Message: params,
	}
	err = json.Unmarshal([]byte(body), data)
	if err != nil {
		log.Error("SaBaRpc auth json.Unmarshal err:%v", err.Error())
		return
	}
	if data.Key != vendorId {
		err = fmt.Errorf("saba key err")
		return
	}
	return
}

func (s *SaBaRpc) timeISO8601() string {
	return time.Now().In(time.FixedZone("UTC", -4*3600)).Format("2006-01-02T15:04:05.999Z07:00")
}

func (s *SaBaRpc) failed(resp map[string]interface{}) {
	resp["Body"] = map[string]interface{}{
		"status": "203",
		"msg":    "Account is not exist",
	}
}

func (s *SaBaRpc) success(resp map[string]interface{}, data map[string]interface{}) {
	resp["Body"] = map[string]interface{}{
		"status": "0",
	}
	for k, v := range data {
		resp["Body"].(map[string]interface{})[k] = v
	}
}

func (s *SaBaRpc) createBetInfo(record *apiStorage.SabaBetRecord) (betInfo map[string]interface{}) {
	betInfo = map[string]interface{}{
		"BetAmount":     record.BetAmount,
		"BetTime":       record.BetTime,
		"ActualAmount":  record.ActualAmount,
		"KickOffTime":   record.KickOffTime,
		"SportTypeName": record.SportTypeNameEn,
		"BetType":       record.BetType,
		"BetTypeName":   record.BetTypeNameEn,
		"Point":         record.Point,
		"Point2":        record.Point2,
		"BetTeam":       record.BetTeam,
		"BetFrom":       record.BetFrom,
		"HomeName":      record.HomeNameEn,
		"AwayName":      record.AwayNameEn,
		"LeagueId":      record.LeagueId,
		"LeagueName":    record.LeagueNameEn,
		"BetChoice":     record.BetChoiceEn,
		"OddsId":        record.OddsId,
		"Odds":          record.Odds,
		"OddsType":      record.OddsType,
		"MatchId":       record.MatchId,
		"IsLive":        record.IsLive,
		"HomeScore":     record.HomeScore,
		"HtHomeScore":   record.HtHomeScore,
		"AwayScore":     record.AwayScore,
		"HTAwayScore":   record.HtAwayScore,
		"Excluding":     record.Excluding,
	}
	return
}
