package apiWm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

type WmHttp struct {
}

type HttpFuncMap map[string]func(w http.ResponseWriter, r *http.Request)

var (
	httpFuncs = HttpFuncMap{
		"CallBalance":      getBalance,
		"PointInout":       pointInout,
		"TimeoutBetReturn": timeoutBetReturn,
		"SendMemberReport": sendMemberReport,
	}
)

func (m *WmHttp) Entrance(w http.ResponseWriter, r *http.Request) {
	log.Debug("WmHttp.Entrance %d", time.Now().Unix())
	sign := r.FormValue("signature")
	if sign != signature {
		failed(w, "signature error")
		return
	}
	cmd := r.FormValue("cmd")
	httpFunc, ok := httpFuncs[cmd]
	if !ok {
		log.Error("wm Entrance not find httpFunc")
		failed(w, "cmd error")
		return
	}
	httpFunc(w, r)
}

func getBalance(w http.ResponseWriter, r *http.Request) {
	log.Debug("WmHttp getBalance %d", time.Now().Unix())

	apiUser, err := getApiUser(w, r.FormValue("user"))
	if err != nil {
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(apiUser.Uid))
	rsp := getResponse()
	rsp.Result = map[string]interface{}{
		"user":         apiUser.Account,
		"money":        wallet.VndBalance,
		"responseDate": time.Now().Format("2006-01-02 15:04:05"),
	}
	response(w, rsp)
}

func pointInout(w http.ResponseWriter, r *http.Request) {
	log.Debug("WmHttp pointInout %d", time.Now().Unix())
	var err error
	if err = parseForm(w, r); err != nil {
		return
	}
	log.Debug("form data:%v", r.Form)
	apiUser, err := getApiUser(w, r.FormValue("user"))
	if err != nil {
		return
	}
	money, err := strconv.ParseFloat(r.FormValue("money"), 64)
	if err != nil {
		log.Error("WmHttp ParseForm money err:%s", err.Error())
		failed(w, err.Error())
		return
	}

	requestDate, err := utils.StrToCnTime(r.FormValue("requestDate"))
	if err != nil {
		log.Error("WmHttp ParseForm requestDate err:%s", err.Error())
		failed(w, err.Error())
		return
	}
	code := r.FormValue("code")
	params := &apiStorage.WmBillRecord{
		Account:     apiUser.Account,
		Uid:         apiUser.Uid,
		BetAmount:   money,
		RequestDate: requestDate,
		Dealid:      r.FormValue("dealid"),
		Gtype:       r.FormValue("gtype"),
		Type:        r.FormValue("type"),
		BetDetail:   r.FormValue("betdetail"),
		GameNo:      r.FormValue("gameno"),
		Category:    r.FormValue("category"),
		Code:        code,
		BetId:       r.FormValue("betId"),
	}
	params.BetAmount *= scale
	log.Debug("WmHttp pointInout BetAmount：%f ", params.BetAmount)
	if code == "3" || code == "4" {
		params.Payout, err = strconv.ParseFloat(r.FormValue("payout"), 64)
		if err != nil {
			log.Error("WmHttp ParseForm payout err:%s", err.Error())
			failed(w, err.Error())
			return
		}
	}
	mRecord := &apiStorage.WmBillRecord{}
	_, err = mRecord.GetWmBillRecord(params.Dealid)
	if err == mongo.ErrNoDocuments {
		params.Oid = primitive.NewObjectID()
		if code == "1" || code == "5" {
			params.SetTransactionUnits(apiStorage.AddWmBillRecord)
			bill := walletStorage.NewBill(params.Uid, walletStorage.TypeIncome, walletStorage.EventGameWm, params.Oid.Hex(), int64(params.BetAmount))
			bill.Remark = fmt.Sprintf("Wm 加点,Code:%s", code)
			if err := walletStorage.OperateVndBalanceV1(bill, params); err != nil {
				log.Error("wallet pay bet _id:%s err:%s", params.Oid.Hex(), err.Error())
				failed(w, err.Error())
				return
			}
		} else if code == "2" {
			params.SetTransactionUnits(apiStorage.AddWmBillRecord)
			bill := walletStorage.NewBill(params.Uid, walletStorage.TypeExpenses, walletStorage.EventGameWm, params.Oid.Hex(), int64(params.BetAmount))
			bill.Remark = fmt.Sprintf("Wm 扣点,Code:%s", code)
			if err := walletStorage.OperateVndBalanceV1(bill, params); err != nil {
				log.Error("wallet pay bet _id:%s err:%s", params.Oid.Hex(), err.Error())
				failed(w, err.Error())
				return
			}
			activityStorage.UpsertGameDataInBet(params.Uid, game.ApiWm, 1)
		} else if code == "3" {
			if params.IsExists(params.Code, params.BetId) {
				log.Error("WmHttp pointInout params.BetId:%d Re-added", params.BetId)
				failed(w, err.Error())
				return
			}
			params.SetTransactionUnits(apiStorage.AddWmBillRecord)
			bill := walletStorage.NewBill(params.Uid, walletStorage.TypeIncome, walletStorage.EventGameWm, params.Oid.Hex(), int64(params.BetAmount))
			bill.Remark = fmt.Sprintf("Wm 重对加点,Code:%s", code)
			if err := walletStorage.OperateVndBalanceV1(bill, params); err != nil {
				log.Error("wallet pay bet _id:%s err:%s", params.Oid.Hex(), err.Error())
				failed(w, err.Error())
				return
			}
			gameStorage.RefundWmBetRecord(params.Oid.Hex(), int64(params.Payout))
		} else if code == "4" {
			if params.IsExists(params.Code, params.BetId) {
				log.Error("WmHttp pointInout params.BetId:%d Weighted down", params.BetId)
				failed(w, err.Error())
				return
			}
			params.SetTransactionUnits(apiStorage.AddWmBillRecord)
			bill := walletStorage.NewBill(params.Uid, walletStorage.TypeExpenses, walletStorage.EventGameWm, params.Oid.Hex(), int64(params.BetAmount))
			bill.Remark = fmt.Sprintf("Wm 重对扣点,Code:%s", code)
			if err := walletStorage.OperateVndBalanceV1(bill, params); err != nil {
				log.Error("wallet pay bet _id:%s err:%s", params.Oid.Hex(), err.Error())
				failed(w, err.Error())
				return
			}
			gameStorage.RefundWmBetRecord(params.Oid.Hex(), int64(params.Payout))
		} else {
			failed(w, fmt.Sprintf("code:%s err", code))
			return
		}
	} else if err != nil {
		failed(w, fmt.Sprintf("code:%s err", code))
		return
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(params.Uid))
	data := getResponse()
	data.Result = map[string]interface{}{
		"money":        fmt.Sprintf("%f", money),
		"responseDate": time.Now().Format("2006-01-02 15:04:05"),
		"dealid":       params.Dealid,
		"cash":         fmt.Sprintf("%f", float64(wallet.VndBalance)/scale),
	}
	log.Debug("params:%v", params)
	response(w, data)

}
func timeoutBetReturn(w http.ResponseWriter, r *http.Request) {
	log.Debug("WmHttp timeoutBetReturn %d", time.Now().Unix())
	if err := parseForm(w, r); err != nil {
		return
	}
	money, err := strconv.ParseFloat(r.FormValue("money"), 64)
	if err != nil {
		log.Error("WmHttp timeoutBetReturn ParseForm money err:%s", err.Error())
		failed(w, err.Error())
		return
	}
	requestDate, err := utils.StrToCnTime(r.FormValue("requestDate"))
	if err != nil {
		log.Error("WmHttp timeoutBetReturn ParseForm requestDate:%v err:%s", r.FormValue("requestDate"), err.Error())
		failed(w, err.Error())
		return
	}
	code := r.FormValue("code")
	dealid := r.FormValue("dealid")
	mRecord := &apiStorage.WmBillRecord{}
	record, err := mRecord.GetWmBillRecord(dealid)
	if err != nil {
		log.Error("WmHttp timeoutBetReturn mRecord.GetWmBillRecord err:%s", err.Error())
		failed(w, err.Error())
		return
	}
	if record.RollbackStatus == 1 {
		log.Error("WmHttp timeoutBetReturn Dealid Rolled back")
		failed(w, "Rolled back")
		return
	}
	record.RollbackStatus = 1
	record.RollbackTime = requestDate
	log.Debug("WmHttp timeoutBetReturn money:%v", money)
	if code == "1" {
		record.SetTransactionUnits(apiStorage.SetRollbackStatus)
		bill := walletStorage.NewBill(record.Uid, walletStorage.TypeExpenses, walletStorage.EventGameWm, record.Oid.Hex(), -int64(money*scale))
		bill.Remark = fmt.Sprintf("Wm 回滚扣点,Code:%s", code)
		if err := walletStorage.OperateVndBalanceV1(bill, record); err != nil {
			log.Error("wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
			failed(w, err.Error())
			return
		}
	} else if code == "2" {
		record.SetTransactionUnits(apiStorage.SetRollbackStatus)
		bill := walletStorage.NewBill(record.Uid, walletStorage.TypeIncome, walletStorage.EventGameWm, record.Oid.Hex(), -int64(money*scale))
		bill.Remark = fmt.Sprintf("Wm 回滚加点,Code:%s", code)
		if err := walletStorage.OperateVndBalanceV1(bill, record); err != nil {
			log.Error("wallet pay bet _id:%s err:%s", record.Oid.Hex(), err.Error())
			failed(w, err.Error())
			return
		}
	} else {
		failed(w, fmt.Sprintf("code err:%s", code))
		return
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
	data := getResponse()
	data.Result = map[string]interface{}{
		"money":        fmt.Sprintf("%f", money),
		"responseDate": time.Now().Format("2006-01-02 15:04:05"),
		"dealid":       dealid,
		"cash":         fmt.Sprintf("%f", float64(wallet.VndBalance)/scale),
	}
	response(w, data)
}

func sendMemberReport(w http.ResponseWriter, r *http.Request) {
	log.Debug("WmHttp sendMemberReport %d", time.Now().Unix())
	if err := parseForm(w, r); err != nil {
		return
	}
	data := make(map[string]interface{})
	paramFormValues(r.Form, data)
	result := data["result"].(map[string]interface{})
	for _, iItem := range result {
		item := iItem.(map[string]interface{})
		betAmount, err := strconv.ParseFloat(item["bet"].(string), 64)
		if err != nil {
			log.Error("WmHttp sendMemberReport betAmount err:%s", err.Error())
			continue
		}
		validbet, err := strconv.ParseFloat(item["validbet"].(string), 64)
		if err != nil {
			log.Error("WmHttp sendMemberReport betAmount err:%s", err.Error())
			continue
		}
		water, err := strconv.ParseFloat(item["water"].(string), 64)
		if err != nil {
			log.Error("WmHttp sendMemberReport water err:%s", err.Error())
			continue
		}
		waterbet, err := strconv.ParseFloat(item["waterbet"].(string), 64)
		if err != nil {
			log.Error("WmHttp sendMemberReport waterbet err:%s", err.Error())
			continue
		}
		winLoss, err := strconv.ParseFloat(item["winLoss"].(string), 64)
		if err != nil {
			log.Error("WmHttp sendMemberReport  winLoss err:%s", err.Error())
			continue
		}
		log.Debug("WmHttp sendMemberReport  1")
		record := &apiStorage.WmBetRecord{
			Account:        item["user"].(string),
			BetId:          item["betId"].(string),
			BetTime:        item["betTime"].(string),
			BetAmount:      betAmount * scale,
			Validbet:       validbet * scale,
			Water:          water * scale,
			Result:         item["result"].(string),
			BetResult:      item["betResult"].(string),
			Waterbet:       waterbet * scale,
			WinLoss:        winLoss * scale,
			Gid:            item["gid"].(string),
			Event:          item["event"].(string),
			EventChild:     item["eventChild"].(string),
			TableId:        item["tableId"].(string),
			GameResult:     item["gameResult"].(string),
			GName:          item["gname"].(string),
			BetWalletId:    item["betwalletid"].(string),
			ResultWalletId: item["resultwalletid"].(string),
			Commission:     item["commission"].(string),
			Reset:          item["reset"].(string),
			SetTime:        item["settime"].(string),
		}
		log.Debug("WmHttp sendMemberReport  2")
		if record.IsExists() {
			continue
		}
		log.Debug("WmHttp sendMemberReport  3")
		uid, ok := userMap[record.Account]
		if !ok {
			apiUser := &apiStorage.ApiUser{}
			err := apiUser.GetApiUserByAccount(record.Account, apiStorage.WmType)
			if err != nil {
				log.Error("WmHttp sendMemberReport GetApiUserByAccount err:%s", err.Error())
				continue
			} else {
				uid = apiUser.Uid
				userMap[record.Account] = uid
			}
		}
		log.Debug("WmHttp sendMemberReport  4")
		record.Uid = uid
		record.Oid = primitive.NewObjectID()
		if err := record.AddWmBetRecord(); err != nil {
			log.Error("WmHttp sendMemberReport record.AddWmBetRecord err:%s", err.Error())
			continue
		}
		log.Debug("WmHttp sendMemberReport  5")
		wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
		params := gameStorage.BetRecordParam{
			Uid:        record.Uid,
			GameType:   game.ApiWm,
			Income:     int64(record.WinLoss),
			BetAmount:  int64(record.BetAmount),
			CurBalance: wallet.VndBalance + wallet.SafeBalance + int64(record.WinLoss),
			SysProfit:  0,
			BotProfit:  0,
			BetDetails: record.BetResult,
			GameId:     record.Oid.Hex(),
			GameNo:     fmt.Sprintf("%s-%s", record.Event, record.EventChild),
			GameResult: record.GameResult,
			IsSettled:  true,
		}
		gameStorage.InsertBetRecord(params)
		activityStorage.UpsertGameDataInBet(params.Uid, game.ApiWm, -1)
		activity.CalcEncouragementFunc(record.Uid)
		log.Debug("WmHttp sendMemberReport  6")
	}
}

func response(w http.ResponseWriter, data *Resp) {
	btData, err := json.Marshal(data)
	if err != nil {
		log.Error("WmHttp response json.Marshal err:%s", err.Error())
		return
	}
	log.Debug("response data:%v", string(btData))
	if _, err := w.Write(btData); err != nil {
		log.Error("WmHttp response: ioWrite  json format err:%s", err.Error())
		return
	}
}

func failed(w http.ResponseWriter, err string) {
	rsp := getResponse()
	rsp.ErrorCode = 10000
	rsp.ErrorMassage = err
	response(w, rsp)
}

func parseForm(w http.ResponseWriter, r *http.Request) (err error) {
	if err = r.ParseForm(); err != nil {
		log.Error("WmHttp ParseForm err:%s", err.Error())
		failed(w, err.Error())
		return
	}
	return
}

func getApiUser(w http.ResponseWriter, user string) (apiUser *apiStorage.ApiUser, err error) {
	apiUser = &apiStorage.ApiUser{}
	err = apiUser.GetApiUserByAccount(user, apiStorage.WmType)
	if err != nil {
		log.Error("WmHttp GetApiUserByAccount err:%s", err.Error())
		failed(w, err.Error())
	}
	return
}

func paramFormValues(values url.Values, param map[string]interface{}) {
	for k, v := range values {
		if len(v) == 0 {
			continue
		}
		ks := strings.Split(k, "[")
		kLen := len(ks)
		if len(ks) == 1 {
			param[k] = v[0]
		} else if kLen > 1 {
			for i := 0; i < kLen; i++ {
				ks[i] = strings.TrimRight(ks[i], "]")
			}
			dataToMap(ks, param, v[0])
		}
	}
}
func dataToMap(keys []string, param interface{}, data interface{}) {
	if len(keys) == 1 {
		param.(map[string]interface{})[keys[0]] = data
	} else {
		if _, ok := param.(map[string]interface{})[keys[0]]; ok {
			dataToMap(keys[1:], param.(map[string]interface{})[keys[0]], data)
		} else {
			param.(map[string]interface{})[keys[0]] = make(map[string]interface{})
			dataToMap(keys[1:], param.(map[string]interface{})[keys[0]], data)
		}
	}
}
