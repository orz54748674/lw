package apiXg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
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

type XgHttp struct {
}

type SettleItem struct {
	User          string  `json:"user"`
	Currency      string  `json:"currency"`
	Amount        float64 `json:"amount"`
	TransactionId string  `json:"transactionId"`
	WagerId       int64   `json:"wagerId"`
}

func (h *XgHttp) Init(env string) {
	InitHost(env)
	go h.run()
}

func (h *XgHttp) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	log.Debug("GetUserBalance %d", time.Now().Unix())
	params := &struct {
		RequestId string `json:"requestId"`
		User      string `json:"user"`
	}{}
	if err := h.parse(w, r, params); err != nil {
		return
	}
	mApiUser := &apiStorage.ApiUser{}
	err := mApiUser.GetApiUserByAccount(params.User, apiStorage.XgType)
	if err != nil {
		log.Error("GetUserBalance GetApiUserByAccount err:%s", err.Error())
		h.failed(w)
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
	data := map[string]interface{}{
		"requestId": params.RequestId,
		"status":    "ok",
		"user":      params.User,
		"currency":  "VND",
		"balance":   float64(wallet.VndBalance) / scale,
	}
	h.response(w, data)
}

func (h *XgHttp) AddBet(w http.ResponseWriter, r *http.Request) {
	log.Debug("AddBet %d", time.Now().Unix())
	params := &struct {
		RequestId     string  `json:"requestId"`
		User          string  `json:"user"`
		Currency      string  `json:"currency"`
		Amount        float64 `json:"amount"`
		GameType      string  `json:"gameType"`
		Table         string  `json:"table"`
		Round         int64   `json:"round"`
		Run           int     `json:"run"`
		Bet           string  `json:"bet"`
		BetTime       string  `json:"betTime"`
		TransactionId string  `json:"transactionId"`
	}{}
	if err := h.parse(w, r, params); err != nil {
		return
	}
	//params.JsonTime
	t, err := utils.StrFormatTime("yyyy/M/d HH:mm:ss", params.BetTime)
	if err != nil {
		log.Error("AddBet StrFormatTime err:", err.Error())
		h.failed(w)
		return
	}

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(params.User, apiStorage.XgType); err != nil {
		log.Error("GetUserBalance GetApiUserByAccount err:%s", err.Error())
		h.failed(w)
		return
	}
	params.Amount = params.Amount * scale
	mRecord := &apiStorage.XgBetRecord{
		Oid:           primitive.NewObjectID(),
		Uid:           mApiUser.Uid,
		RequestId:     params.RequestId,
		User:          params.User,
		Currency:      params.Currency,
		Amount:        params.Amount,
		GameType:      params.GameType,
		Table:         params.Table,
		Round:         params.Round,
		Run:           params.Run,
		Bet:           params.Bet,
		TransactionId: params.TransactionId,
		BetTime:       t,
	}
	bill := walletStorage.NewBill(mApiUser.Uid, walletStorage.TypeExpenses, walletStorage.EventGameXg, mRecord.Oid.Hex(), -1*int64(params.Amount))
	mRecord.SetTransactionUnits(apiStorage.AddXgRecord)
	if err := walletStorage.OperateVndBalanceV1(bill, mRecord); err != nil {
		log.Error("wallet pay bet _id:%s err:%s", mRecord.Oid.Hex(), err.Error())
		h.failed(w)
		return
	}
	activityStorage.UpsertGameDataInBet(mRecord.Uid, game.Xg, 1)
	wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
	data := map[string]interface{}{
		"requestId":     params.RequestId,
		"status":        "ok",
		"user":          params.User,
		"currency":      "VND",
		"balance":       float64(wallet.VndBalance) / scale,
		"transactionId": params.TransactionId,
	}
	h.response(w, data)
}

func (h *XgHttp) Settle(w http.ResponseWriter, r *http.Request) {
	log.Debug("Settle %d", time.Now().Unix())
	params := &struct {
		RequestId   string        `json:"requestId"`
		SettleItems []*SettleItem `json:"settleItems"`
	}{}
	if err := h.parse(w, r, params); err != nil {
		return
	}
	transactionIds := []string{}
	for _, v := range params.SettleItems {
		transactionIds = append(transactionIds, v.TransactionId)
	}
	log.Debug("transactionIds:%v", transactionIds)
	mXgRecord := &apiStorage.XgBetRecord{}
	records, err := mXgRecord.GetRecords(transactionIds)
	if err != nil {
		log.Error("mXgRecord.GetRecords err:%s", err.Error())
		h.failed(w)
		return
	}
	settleMap := map[string]*SettleItem{}
	for _, v := range params.SettleItems {
		settleMap[v.TransactionId] = v
	}
	userWalletMap := map[string]*walletStorage.Wallet{}
	if len(records) == 0 {
		log.Debug("No need to settle")
		h.failed(w)
		return
	}

	for _, record := range records {
		settleInfo := settleMap[record.TransactionId]
		record.SettleRequestId = params.RequestId
		record.WagerId = settleInfo.WagerId
		settleInfo.Amount = settleInfo.Amount * scale
		record.SettleAmount = settleInfo.Amount
		log.Debug("record.SettleAmount:%f record.Amount：%f", record.SettleAmount, record.Amount)
		betRecordData := gameStorage.BetRecordParam{
			Uid:        record.Uid,
			GameType:   game.Xg,
			Income:     int64(settleInfo.Amount) - int64(record.Amount),
			BetAmount:  int64(record.Amount),
			SysProfit:  0,
			BotProfit:  0,
			BetDetails: "",
			GameId:     record.Oid.Hex(),
			GameNo:     fmt.Sprint(settleInfo.WagerId),
			GameResult: "",
			IsSettled:  true,
		}
		if int64(record.SettleAmount) == 0 {
			err := record.SettleXgRecord()
			if err != nil {
				log.Error("SettleXgRecord  bet _id:%s err:%s", record.Oid.Hex(), err.Error())
			}
			wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
			userWalletMap[record.User] = wallet
			betRecordData.CurBalance = wallet.VndBalance + wallet.SafeBalance
			gameStorage.InsertBetRecord(betRecordData)
			continue
		}
		record.SetTransactionUnits(apiStorage.SettleXgRecord)
		bill := walletStorage.NewBill(record.Uid, walletStorage.TypeIncome, walletStorage.EventGameXg, record.Oid.Hex(), int64(settleInfo.Amount))
		record.SetTransactionUnits(apiStorage.SettleXgRecord)
		if err := walletStorage.OperateVndBalanceV1(bill, record); err != nil {
			log.Error("wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
			continue
		}
		//h.wagerInfos <- &winInfo{WagerId: record.WagerId, Uid: record.Uid, Oid: record.Oid}
		wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
		userWalletMap[record.User] = wallet
		betRecordData.CurBalance = wallet.VndBalance
		gameStorage.InsertBetRecord(betRecordData)
		activityStorage.UpsertGameDataInBet(record.Uid, game.Xg, -1)
		activity.CalcEncouragementFunc(record.Uid)
	}

	settleUserBalanceList := []map[string]interface{}{}
	for _, v := range params.SettleItems {
		log.Debug("settle balance:%f", float64(userWalletMap[v.User].VndBalance)/scale)
		settleUserBalanceList = append(settleUserBalanceList, map[string]interface{}{
			"user":     v.User,
			"currency": "VND",
			"balance":  float64(userWalletMap[v.User].VndBalance) / scale,
		})
	}
	data := map[string]interface{}{
		"requestId":             params.RequestId,
		"status":                "ok",
		"settleUserBalanceList": settleUserBalanceList,
	}
	h.response(w, data)
}

func (h *XgHttp) Rollback(w http.ResponseWriter, r *http.Request) {
	log.Debug("Rollback %d", time.Now().Unix())
	params := &struct {
		RequestId     string `json:"requestId"`
		User          string `json:"user"`
		TransactionId string `json:"transactionId"`
	}{}
	if err := h.parse(w, r, params); err != nil {
		return
	}
	mXgRecord := &apiStorage.XgBetRecord{}
	record, err := mXgRecord.GetRecord(params.TransactionId, params.User)
	if err != nil {
		log.Error("mXgRecord.GetRecord err:%s", err.Error())
		h.failed(w)
		return
	}
	record.SettleRequestId = params.RequestId
	bill := walletStorage.NewBill(record.Uid, walletStorage.TypeExpenses, walletStorage.EventGameXg, record.Oid.Hex(), int64(record.Amount))
	record.SetTransactionUnits(apiStorage.RollbackXgRecord)
	if err := walletStorage.OperateVndBalanceV1(bill, record); err != nil {
		log.Error("wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
		h.failed(w)
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
	data := map[string]interface{}{
		"requestId":     params.RequestId,
		"status":        "ok",
		"user":          params.User,
		"currency":      "VND",
		"balance":       float64(wallet.VndBalance),
		"transactionId": params.TransactionId,
	}
	h.response(w, data)
}

func (h *XgHttp) run() {
	go h.getBetRecordByTime()
	go h.getReplenishmentByTime()
}

func (h *XgHttp) getBetRecordByTime() {
	mXgRecord := &apiStorage.XgBetRecord{}
	t := time.NewTicker(time.Second * 90)
	defer t.Stop()
	url := "xg-casino/GetBetRecordByTime"
	var cstZone = time.FixedZone("UTC", -5*3600)
	for {
		records, err := mXgRecord.GetNoReadRecords()
		log.Debug("mXgRecord.GetNoReadRecords records:%v err：%v", records, err)
		if err == nil {
			if len(records) == 0 {
				log.Debug("getBetRecordByTime not find")
				<-t.C
				continue
			}
			page := 1
			pageLimit := 100
			for {
				params := []Param{
					NewParam("AgentId", agentId),
					NewParam("StartTime", records[len(records)-1].BetTime.In(cstZone).Format("2006-01-02T15:04:05")),
					NewParam("EndTime", records[0].BetTime.In(cstZone).Add(610*time.Second).Format("2006-01-02T15:04:05")),
					NewParam("Page", fmt.Sprint(page)),
					NewParam("PageLimit", fmt.Sprint(pageLimit)),
				}
				queryStr := HttpBuildQuery(params)
				params = append(params, NewParam("Key", getSign(queryStr)))
				res, err := httpPost(url, params)
				log.Debug("getBetRecordByTime httpPost res:%v, err：%v", res, err)
				if err != nil {
					log.Error("%s err:%s", url, err.Error())
					break
				}

				if res.ErrorCode == SuccessCode {
					data, _ := res.Data.(map[string]interface{})
					result := data["Result"].([]interface{})
					if len(result) > 0 {
						go h.statsBetRecordByTime(result, records)
					}
					if len(result) < pageLimit {
						break
					}
					page++
				} else {
					log.Debug("GetBetRecordByTime res.ErrorCode:%v", res.ErrorCode)
					break
				}
			}
		} else {
			log.Error("mXgRecord.GetRecords err:%s", err.Error())
		}
		<-t.C
	}
}

func (h *XgHttp) statsBetRecordByTime(datas []interface{}, records []*apiStorage.XgBetRecord) {
	settleMap := map[string][]string{}
	for _, iRecord := range datas {
		item := iRecord.(map[string]interface{})
		gameType := ""
		for key, gType := range GameTypes {
			if gType == item["GameType"].(string) {
				gameType = key
			}
		}
		if len(gameType) == 0 {
			continue
		}
		key := fmt.Sprintf("%v_%v", item["Account"], item["WagersId"])
		settleMap[key] = []string{fmt.Sprintf("%s|%v", gameType, item["GameResult"]), item["BetType"].(string)}
	}
	for _, record := range records {
		data, ok := settleMap[fmt.Sprintf("%s_%d", record.User, record.WagerId)]
		if ok {
			betInfo := fmt.Sprintf("%s|%s", record.Bet, data[1])
			gameStorage.RefundXgBetRecordInfo(record.Oid.Hex(), data[0], betInfo)
			record.SetWagerId(record.WagerId, 1)
		}
	}
}

func (h *XgHttp) auth(r *http.Request, params interface{}) error {
	apiKey := r.Header.Get("X-API-KEY")
	apiTOKEN := r.Header.Get("X-API-TOKEN")
	log.Debug("body map before:%v", params)
	paramMap := utils.StructToMap(params, "json")
	log.Debug("body sort:%s", h.mapToSortJson(paramMap))
	if apiKey != agentId {
		return fmt.Errorf("X-API-KEY err")
	}
	log.Debug("X-API-TOKEN:%s getToken:%s", apiTOKEN, h.getToken(paramMap))
	if apiTOKEN != h.getToken(paramMap) {
		return fmt.Errorf("X-API-TOKEN err")
	}
	return nil
}

func (h *XgHttp) failed(w http.ResponseWriter) {
	data := map[string]interface{}{
		"status":  "failed",
		"message": "參數錯誤",
	}
	h.response(w, data)
}

func (h *XgHttp) parse(w http.ResponseWriter, r *http.Request, param interface{}) (err error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("parse read param:%s", err.Error())
		h.failed(w)
		return
	}
	log.Debug("body:%s", string(body))
	err = json.Unmarshal(body, &param)
	if err != nil {
		log.Error("parse param to struct err:%s", err.Error())
		h.failed(w)
		return
	}
	if err = h.auth(r, param); err != nil {
		log.Error("parse auth err:%s", err.Error())
		h.failed(w)
		return
	}
	return
}

func (h *XgHttp) response(w http.ResponseWriter, data map[string]interface{}) {
	jsonData := h.mapToSortJson(data)
	log.Debug("mapToSortJson:%s", jsonData)
	w.Header().Set("X-API-KEY", agentId)
	w.Header().Set("X-API-TOKEN", h.getToken(data))
	if _, err := w.Write([]byte(jsonData)); err != nil {
		log.Error("response: ioWrite  json format err:%s", err.Error())
		return
	}
}

func (h *XgHttp) mapToSortJson(any interface{}) string {
	res := ""
	switch any.(type) {
	case float64:
		res = fmt.Sprintf("%.4f", any.(float64))
	case int64:
		res = strconv.FormatInt(any.(int64), 10)
	case int:
		res = strconv.Itoa(any.(int))
	case string:
		res = fmt.Sprintf("\"%s\"", any.(string))
	case map[string]interface{}:
		var tmp, keys []string
		data := any.(map[string]interface{})
		for k, _ := range data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			tmp = append(tmp, fmt.Sprintf("\"%s\":%s", k, h.mapToSortJson(data[k])))
		}
		res = fmt.Sprintf("{%s}", strings.Join(tmp, ","))
	case []map[string]interface{}:
		var tmp []string
		for _, v := range any.([]map[string]interface{}) {
			tmp = append(tmp, h.mapToSortJson(v))
		}
		res = fmt.Sprintf("[%s]", strings.Join(tmp, ","))

	case []interface{}:
		var tmp []string
		for _, v := range any.([]interface{}) {
			tmp = append(tmp, h.mapToSortJson(v))
		}
		res = fmt.Sprintf("[%s]", strings.Join(tmp, ","))
	}
	return res
}

func (h *XgHttp) getToken(paramMap map[string]interface{}) string {
	shaToken := utils.Sha1(fmt.Sprintf("%s%s%s", agentId, h.mapToSortJson(paramMap), agentKey))
	return strings.ToLower(base64.StdEncoding.EncodeToString([]byte(shaToken)))
}

func (h *XgHttp) getReplenishmentByTime() {
	log.Debug("getReplenishmentByTime")
	ticker := time.NewTicker(time.Second * 60)
	url := "xg-casino/GetReplenishmentByTime"
	mRecordCheckTime := &apiStorage.ApiRecordCheckTime{}
	var cstZone = time.FixedZone("UTC", -4*3600)
	for {
		t, err := mRecordCheckTime.GetLastTime(apiStorage.XgType)
		if err != nil {
			log.Error("getReplenishmentByTime mRecordCheckTime.GetLastTime err:%s", err.Error())
			t.Time = time.Now().In(cstZone).Add(-time.Hour * 24 * 30)
		}
		startTime := t.Time.In(cstZone).Format("2006-01-02T15:04:05")
		endTime := time.Now().In(cstZone).Format("2006-01-02T15:04:05")
		log.Debug("getReplenishmentByTime startTime:%v", startTime)
		log.Debug("getReplenishmentByTime   endTime:%v", endTime)
		mRecordCheckTime.Time = time.Now()
		for gameType, v := range GameTypes {
			go func(gameType, v string) {
				page := 1
				for {
					params := []Param{
						NewParam("AgentId", agentId),
						NewParam("StartTime", startTime),
						NewParam("EndTime", endTime),
						NewParam("GameType", v),
						NewParam("Page", fmt.Sprint(page)),
					}
					queryStr := HttpBuildQuery(params)
					params = append(params, NewParam("Key", getSign(queryStr)))
					res, err := httpPost(url, params)
					if err != nil {
						log.Error("%s err:%s", url, err.Error())
						time.Sleep(2 * time.Second)
						break
					}
					log.Debug("getReplenishmentByTime GameType:%v v:%v res:%v", gameType, v, res)
					if res.ErrorCode == SuccessCode {
						data, _ := res.Data.(map[string]interface{})
						result := data["Result"].([]interface{})
						pagination := data["Pagination"].(map[string]interface{})
						pageLimit := int(pagination["PageLimit"].(float64))
						if len(result) > 0 {
							go h.StatsReplenishmentByTime(result)
						}
						if len(result) < pageLimit {
							break
						}
						page++
					} else {
						log.Debug("getReplenishmentByTime GameType:%v v:%v  res.ErrorCode:%v", gameType, v, res.ErrorCode)
						break
					}
				}
			}(gameType, v)
		}
		err = mRecordCheckTime.UpdateTime(apiStorage.XgType)
		if err != nil {
			log.Error("getReplenishmentByTime mRecordCheckTime.UpdateTime err:%s", err.Error())
		}
		log.Debug("now time:%d,time.Now().In(cstZone):%d", time.Now().Unix(), time.Now().In(cstZone).Unix())
		<-ticker.C
	}
}

func (h *XgHttp) StatsReplenishmentByTime(datas []interface{}) {
	mXgBetRecord := &apiStorage.XgBetRecord{}
	for _, iItem := range datas {
		item := iItem.(map[string]interface{})
		modifiedStatus := item["ModifiedStatus"].(string)
		transactionId := item["TransactionId"].(string)
		record, err := mXgBetRecord.GetRecordByTransactionId(transactionId)
		if err != nil {
			if err != mongo.ErrNoDocuments {
				log.Error("mXgBetRecord.GetRecordByTransactionId:%v", err.Error())
			}
			continue
		}
		if record.ModifiedStatus == "Canceled" {
			continue
		}
		if modifiedStatus == "Canceled" {
			newSettleAmount := record.Amount - record.SettleAmount
			changeType := walletStorage.TypeExpenses
			if newSettleAmount > 0 {
				changeType = walletStorage.TypeIncome
			}
			bill := walletStorage.NewBill(record.Uid, changeType, walletStorage.EventGameXg, record.Oid.Hex(), int64(newSettleAmount))
			bill.Remark = fmt.Sprintf("XG modifiedStatus:%v changeType:%v WagerId:%v SettleAmount：%v", modifiedStatus, changeType, record.WagerId, newSettleAmount)
			record.ModifiedStatus = modifiedStatus
			record.SetTransactionUnits(apiStorage.SetModifiedStatus)
			if err := walletStorage.OperateVndBalanceV2(bill, record); err != nil {
				log.Error("xg StatsReplenishmentByTime wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
				continue
			}
			gameStorage.RefundXgBetRecord(record.Oid.Hex(), 0, 0)
		} else if modifiedStatus == "Modified" {
			settleAmount, ok := item["SettleAmount"]
			if !ok {
				log.Error("xg StatsReplenishmentByTime settleAmount not find")
				continue
			}
			newSettleAmount := settleAmount.(float64)
			difference := newSettleAmount - record.SettleAmount
			changeType := walletStorage.TypeExpenses
			if difference > 0 {
				changeType = walletStorage.TypeIncome
			}
			bill := walletStorage.NewBill(record.Uid, changeType, walletStorage.EventGameXg, record.Oid.Hex(), int64(difference))
			bill.Remark = fmt.Sprintf("XG modifiedStatus:%v changeType:%v WagerId:%v SettleAmount：%v", modifiedStatus, changeType, record.WagerId, difference)
			record.SettleAmount = newSettleAmount
			record.SetTransactionUnits(apiStorage.ChangeSettleXgRecord)
			if err := walletStorage.OperateVndBalanceV2(bill, record); err != nil {
				log.Error("xg StatsReplenishmentByTime wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
				return
			}
			go h.getBetRecordByUuid(record)
			gameStorage.RefundXgBetRecord(record.Oid.Hex(), int64(newSettleAmount-record.Amount), int64(record.Amount))
		}
	}
}

func (h *XgHttp) getBetRecordByUuid(record *apiStorage.XgBetRecord) {
	url := "xg-casino/GetBetRecordByUuid"
	params := []Param{
		NewParam("AgentId", agentId),
		NewParam("WagersId", fmt.Sprint(record.WagerId)),
	}
	queryStr := HttpBuildQuery(params)
	params = append(params, NewParam("Key", getSign(queryStr)))
	res, err := httpPost(url, params)
	if err != nil {
		log.Error("%s err:%s", url, err.Error())
		return
	}
	if res.ErrorCode == SuccessCode {
		data := res.Data.(map[string]interface{})
		iBetType, ok := data["BetType"]
		if !ok {
			log.Error("getBetRecordByUuid not find BetType")
			return
		}
		iGameResult, ok := data["GameResult"]
		if !ok {
			log.Error("getBetRecordByUuid not find GameResult")
			return
		}
		betInfo := fmt.Sprintf("%s|%s", record.Bet, iBetType.(string))
		gameStorage.RefundXgBetRecordInfo(record.Oid.Hex(), iGameResult.(string), betInfo)
	}
}
