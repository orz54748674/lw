package apiCmd

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/activity"
	"vn/storage/activityStorage"
	"vn/storage/apiCmdStorage"
	"vn/storage/apiStorage"
	"vn/storage/gameStorage"
	"vn/storage/walletStorage"
)

type CmdHttp struct {
}

func (h *CmdHttp) CheckParams(r *http.Request, params []string) bool {
	for _, v := range params {
		tmp := r.URL.Query().Get(v)
		if tmp == "" {
			return false
		}
	}
	return true
}

func (h *CmdHttp) VerifyToken(w http.ResponseWriter, r *http.Request) {
	type Authenticate struct {
		XMLName xml.Name `xml:"authenticate"`
		MemberID    string      `xml:"member_id"`
		StatusCode  int  `xml:"status_code"`
		Message string `xml:"message"`
	}

	var rep Authenticate
	tmpToken := r.URL.Query().Get("token")
	if tmpToken == "" {
		rep.StatusCode = 1
		rep.Message = "Fail"
	} else {
		bExist, account := apiCmdStorage.CheckToken(tmpToken)
		if !bExist {
			rep.StatusCode = 2
			rep.Message = "Fail"
		} else {
			rep.StatusCode = 0
			rep.MemberID = account
			rep.Message = "Success"
		}
	}

	fmt.Println("------VerifyToken--------", rep)
	data, _ := xml.MarshalIndent(&rep, "", "  ")
	//加入XML头
	headerBytes := []byte(xml.Header)
	//拼接XML头和实际XML内容
	xmlData := append(headerBytes, data...)
	if _, err := w.Write(xmlData); err != nil {
		log.Error("response: ioWrite json format err:%s", err.Error())
		return
	}
}

func (h *CmdHttp) encryptResponse(w http.ResponseWriter, rep interface{}) {
	repByte, _ := json.Marshal(rep)
	fmt.Println("cmd response:", string(repByte))
	tmpStr, _ := utils.EncryptByAes(repByte, PartnerKey)
	if _, err := w.Write([]byte(tmpStr)); err != nil {
		log.Error("response: ioWrite json format err:%s", err.Error())
		return
	}
}

func (h *CmdHttp) GetBalance(w http.ResponseWriter, r *http.Request) {
	tmpByte, _ := json.Marshal(r.URL.Query())
	log.Info("GetBalance request info:", string(tmpByte))

	rep := struct {
		StatusCode int	`json:"StatusCode"`
		StatusMessage string	`json:"StatusMessage"`
		PackageId string	`json:"PackageId"`
		Balance float64	`json:"Balance"`
		DateReceived int64 `json:"DateReceived"`
		DateSent int64	`json:"DateSent"`
	}{}
	if !h.CheckParams(r, []string{"method", "balancePackage", "packageId", "dateSent"}) {
		rep.StatusCode = 0
		rep.StatusMessage = "GetBalance err:wrong request params"
		h.encryptResponse(w, rep)
	}
	method := r.URL.Query().Get("method")
	if method != "GetBalance" {
		rep.StatusCode = 0
		rep.StatusMessage = "GetBalance err:method not GetBalance"
		h.encryptResponse(w, rep)
	}
	balancePackage := r.URL.Query().Get("balancePackage")
	packageId := r.URL.Query().Get("packageId")
	dateSentStr := r.URL.Query().Get("dateSent")
	dataSent, _ := strconv.ParseInt(dateSentStr, 10, 64)
	tmpByte, _ = utils.DecryptByAes(balancePackage, PartnerKey)
	param := &struct {
		ActionId   int    `json:"ActionId"`
		SourceName string `json:"SourceName"`
	}{}
	json.Unmarshal(tmpByte, param)
	if param.ActionId == 1000 {
		mApiUser := &apiStorage.ApiUser{}
		if err := mApiUser.GetApiUserByAccount(param.SourceName, ApiType); err != nil {
			rep.StatusCode = 0
			rep.StatusMessage = "GetBalance err:not this player " + param.SourceName
			h.encryptResponse(w, rep)
			return
		}
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		rep.StatusCode = 100
		rep.StatusMessage = "Success"
		rep.PackageId = packageId
		rep.DateReceived = dataSent
		rep.DateSent = dataSent
		rep.Balance = float64(wallet.VndBalance)/1000
		h.encryptResponse(w, rep)
	} else {
		rep.StatusCode = 0
		rep.StatusMessage = "GetBalance err:ActionId not 1000"
		h.encryptResponse(w, rep)
		return
	}
}

func (h *CmdHttp) DeductBalance(w http.ResponseWriter, r *http.Request) {
	data := &struct {
		Method string `json:"Method"`
		BalancePackage string `json:"balancePackage"`
		PackageId string `json:"packageId"`
		DateSent string `json:"dateSent"`
	}{}

	rep := struct {
		StatusCode int	`json:"StatusCode"`
		StatusMessage string	`json:"StatusMessage"`
		PackageId string	`json:"PackageId"`
		Balance float64	`json:"Balance"`
		DateReceived int64 `json:"DateReceived"`
		DateSent int64	`json:"DateSent"`
	}{}

	if err := h.parse(r, data); err != nil {
		log.Error("deductBalance parse data err:", err.Error())
		rep.StatusCode = 0
		rep.StatusMessage = "DeductBalance err:wrong request params"
		h.encryptResponse(w, rep)
		return
	}

	enEscapeUrl, _ := url.QueryUnescape(data.BalancePackage)
	tmpByte, err := utils.DecryptByAes(enEscapeUrl, PartnerKey)
	if err != nil {
		log.Error("deductBalance DecryptByAes err:", err.Error())
		rep.StatusCode = 0
		rep.StatusMessage = "DeductBalance DecryptByAes err:" + err.Error()
		h.encryptResponse(w, rep)
		return
	}

	param := &struct {
		ActionId   int    `json:"ActionId"`
		SourceName string `json:"SourceName"`
		TransactionAmount float64 `json:"TransactionAmount"`
		ReferenceNo string `json:"ReferenceNo"`
	}{}
	json.Unmarshal(tmpByte, param)
	fmt.Println("DeductBalance........", param)

	if param.ActionId == 1003 {
		mApiUser := &apiStorage.ApiUser{}
		if err := mApiUser.GetApiUserByAccount(param.SourceName, ApiType); err != nil {
			log.Error("GetUserBalance GetApiUserByAccount err:%s", err.Error())
			rep.StatusCode = 0
			h.encryptResponse(w, rep)
			return
		}
		score := int64(param.TransactionAmount * 1000)
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		if wallet.VndBalance + score < 0 {
			rep.StatusCode = 0
			h.encryptResponse(w, rep)
			return
		}
		eventID := param.ReferenceNo
		bill := walletStorage.NewBill(mApiUser.Uid, walletStorage.TypeExpenses, walletStorage.EventApiCmd, eventID, score)
		walletStorage.OperateVndBalanceV1(bill)
		activityStorage.UpsertGameDataInBet(mApiUser.Uid, game.ApiCmd,1)

		var recordParams gameStorage.BetRecordParam
		recordParams.Uid = mApiUser.Uid
		recordParams.GameNo = eventID
		recordParams.BetAmount = int64(math.Abs(param.TransactionAmount*1000))
		recordParams.BotProfit = 0
		recordParams.SysProfit = 0
		recordParams.BetDetails = ""
		recordParams.GameResult = ""
		recordParams.CurBalance = wallet.VndBalance + score + wallet.SafeBalance

		recordParams.GameType = game.ApiCmd
		recordParams.Income = 0
		recordParams.IsSettled = false
		gameStorage.InsertBetRecord(recordParams)

		var referenceMsg apiCmdStorage.ReferenceMsg
		referenceMsg.ReferenceNo = eventID
		referenceMsg.BetAmount = recordParams.BetAmount
		referenceMsg.Uid = recordParams.Uid
		apiCmdStorage.InsertReferenceRecord(recordParams.Uid, recordParams.GameNo, recordParams.BetAmount)

		tmpTime, _ := strconv.ParseInt(data.DateSent, 10, 64)
		rep.StatusCode = 100
		rep.StatusMessage = param.SourceName
		rep.PackageId = data.PackageId
		rep.DateReceived = tmpTime
		rep.DateSent = tmpTime
		rep.Balance = float64(wallet.VndBalance)/1000
		h.encryptResponse(w, rep)
	} else {
		rep.StatusCode = 0
		h.encryptResponse(w, rep)
	}
}

func (h *CmdHttp) HandleUpdateBalance(msg apiCmdStorage.UpdateBalanceMsg, msgStr string) {
	apiCmdStorage.SaveUpdateBalanceMsg(msgStr)
	fmt.Println("CMD HandleUpdateBalance Msg..............", msg)
	for _, v := range msg.TicketDetails {
		fmt.Println("CMD HandleUpdateBalance............", v)
		uid := apiCmdStorage.GetUidByAccount(v.SourceName)
		if uid == "" {
			continue
		}
		score := int64(v.TransactionAmount*1000)
		wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
		if wallet.VndBalance + score < 0 {
			continue
		}
		eventID := v.ReferenceNo
		billType := walletStorage.TypeIncome
		if score < 0 {
			billType = walletStorage.TypeExpenses
		}
		bill := walletStorage.NewBill(uid, billType, walletStorage.EventApiCmd, eventID, score)
		walletStorage.OperateVndBalanceV1(bill)
		activityStorage.UpsertGameDataInBet(uid, game.ApiCmd,-1)
		activity.CalcEncouragementFunc(uid)

		var recordParams gameStorage.BetRecordParam
		recordParams.Uid = uid
		recordParams.GameNo = eventID
		recordParams.BetAmount = 0
		recordParams.BotProfit = 0
		recordParams.SysProfit = 0
		recordParams.BetDetails = ""
		recordParams.GameResult = ""
		recordParams.CurBalance = wallet.VndBalance + score + wallet.SafeBalance
		recordParams.GameType = game.ApiCmd
		recordParams.Income = score
		recordParams.GameId = strconv.FormatInt(msg.MatchID, 10)
		if msg.ActionId == 2001 || msg.ActionId == 2002 || msg.ActionId == 6001 || msg.ActionId == 6002 {
			gameStorage.UpdateApiCmdBetRecord(recordParams,1)
		} else if msg.ActionId == 4001 || msg.ActionId == 4002 || msg.ActionId == 4003 {
			gameStorage.UpdateApiCmdBetRecord(recordParams,2)
		} else if msg.ActionId == 5001 || msg.ActionId == 5002 || msg.ActionId == 5003 {
			recordParams.Income = 0
			gameStorage.UpdateApiCmdBetRecord(recordParams,3)
		} else if msg.ActionId == 7001 || msg.ActionId == 7002 {
			betAmount := apiCmdStorage.GetReferenceBetAmount(uid, v.ReferenceNo)
			recordParams.BetAmount = betAmount
			gameStorage.UpdateApiCmdBetRecord(recordParams,4)
		} else if msg.ActionId == 9000 {
			gameStorage.UpdateApiCmdBetRecord(recordParams,5)
		}
	}
}

func (h *CmdHttp) UpdateBalance(w http.ResponseWriter, r *http.Request) {
	data := &struct {
		Method string `json:"Method"`
		BalancePackage string `json:"balancePackage"`
		PackageId string `json:"packageId"`
		DateSent string `json:"dateSent"`
	}{}
	rep := struct {
		StatusCode int	`json:"StatusCode"`
		StatusMessage string	`json:"StatusMessage"`
		PackageId string	`json:"PackageId"`
		DateReceived int64 `json:"DateReceived"`
		DateSent int64	`json:"DateSent"`
	}{}

	if err := h.parse(r, data); err != nil {
		log.Error("UpdateBalance parse data err:", err.Error())
		rep.StatusCode = 0
		rep.StatusMessage = "UpdateBalance err:" + err.Error()
		h.encryptResponse(w, rep)
		return
	}
	fmt.Println("UpdateBalance data.........", data)

	enEscapeUrl, _ := url.QueryUnescape(data.BalancePackage)
	tmpByte, err := utils.DecryptByAes(enEscapeUrl, PartnerKey)
	if err != nil {
		log.Error("UpdateBalance DecryptByAes err:", err.Error())
		rep.StatusCode = 0
		rep.StatusMessage = "UpdateBalance err:" + err.Error()
		h.encryptResponse(w, rep)
		return
	}

	var param apiCmdStorage.UpdateBalanceMsg
	json.Unmarshal(tmpByte, &param)

	go h.HandleUpdateBalance(param, string(tmpByte))

	tmpTime, _ := strconv.ParseInt(data.DateSent, 10, 64)
	rep.StatusCode = 100
	rep.StatusMessage = "Update Balance Succeed"
	rep.PackageId = data.PackageId
	rep.DateReceived = tmpTime
	rep.DateSent = tmpTime
	h.encryptResponse(w, rep)
}

func (h *CmdHttp) parse(r *http.Request, param interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(body, &param); err != nil {
		return err
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		return fmt.Errorf("Content-Type err")
	}

	return nil
}
