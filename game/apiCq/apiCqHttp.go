package apiCq

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/storage/apiCqStorage"
	"vn/storage/apiStorage"
	"vn/storage/walletStorage"
)

type CqHttp struct {
}

func (h *CqHttp) Balance(w http.ResponseWriter, r *http.Request) {
	account := strings.Replace(r.URL.Path, "/transaction/balance/", "", 1)

	var status RepStatus
	status.DateTime = time.Now().Format(time.RFC3339)
	data := make(map[string]interface{})

	mApiUser := &apiStorage.ApiUser{}
	err := mApiUser.GetApiUserByAccount(account, ApiType)
	if err != nil {
		log.Error("GetUserBalance GetApiUserByAccount err:%s", err.Error())
		status.Message = "no this player"
		status.Code = "1006"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		status.Message = "success"
		status.Code = "0"
		data["currency"] = "VND"
		data["balance"] = float64(wallet.VndBalance)
	}

	ret := map[string]interface{}{
		"status": status,
		"data":   data,
	}

	h.response(w, ret)
}

func (h *CqHttp) Record(w http.ResponseWriter, r *http.Request) {
	mtcode := strings.Replace(r.URL.Path, "/transaction/record/", "", 1)

	var status RepStatus
	status.Message = "success"
	status.Code = "0"
	record, err := apiCqStorage.GetRecordByMTCode(mtcode)
	if err != nil {
		status.Code = "1014"
		status.Message = "Record not found"
	}

	status.DateTime = time.Now().Format(time.RFC3339)
	ret := map[string]interface{}{
		"status": status,
		"data":   record.Data,
	}

	h.response(w, ret)
}

func (h *CqHttp) CheckPlayer(w http.ResponseWriter, r *http.Request) {
	account := strings.Replace(r.URL.Path, "/player/check/", "", 1)
	mApiUser := &apiStorage.ApiUser{}

	status := map[string]interface{}{
		"code":     "0",
		"message":  "Success",
		"datatime": time.Now().Format(time.RFC3339),
	}
	res := map[string]interface{}{}
	res["data"] = true
	if err := mApiUser.GetApiUserByAccount(account, ApiType); err != nil {
		log.Error("CheckPlayer GetApiUserByAccount err:%s", err.Error())
		res["data"] = false
	}
	res["status"] = status

	h.response(w, res)
}

func (h *CqHttp) Wins(w http.ResponseWriter, r *http.Request) {
	log.Debug("Wins %d", time.Now().Unix())
	var wins apiCqStorage.Wins
	paramMap := make(map[string]interface{})
	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	if err := h.parse(w, r, &paramMap, &wins); err != nil {
		statusMap["code"] = "1003"
		statusMap["message"] = "参数解析失败:" + err.Error()
		resMap["data"] = dataMap
		resMap["status"] = statusMap
		h.response(w, resMap)
		return
	}

	resMap = apiCqStorage.WinsHandle(wins, paramMap)
	h.response(w, resMap)
}

func (h *CqHttp) Amends(w http.ResponseWriter, r *http.Request) {
	log.Debug("Amends %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	var amends apiCqStorage.Amends
	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &amends); err != nil {
		//statusMap["code"] = "1003"
		//statusMap["message"] = "参数解析失败:" + err.Error()
		//resMap["data"] = dataMap
		//resMap["status"] = statusMap
		h.response(w, h.GetRespData("1003", "参数解析失败:" + err.Error(), dataMap, statusMap))
		return
	}

	resMap = apiCqStorage.AmendsHandle(amends, paramMap)
	h.response(w, resMap)
}

func (h *CqHttp) Amend(w http.ResponseWriter, r *http.Request) {
	log.Debug("Amend %d", time.Now().Unix())
	var amend apiCqStorage.Amend
	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &amend); err != nil {
		return
	}

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	if errCode, err := h.CheckParams([]string{"account", "gamehall", "gamecode", "createTime", "action", "amount"}, paramMap); err != nil {
		//statusMap["code"] = errCode
		//statusMap["message"] = err.Error()
		//resMap["data"] = dataMap
		//resMap["status"] = statusMap
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	tmpData := paramMap["data"].([]interface{})
	for _, v := range tmpData {
		tmp := v.(map[string]interface{})
		if errCode, err := h.CheckParams([]string{"mtcode", "amount", "roundid", "eventtime", "action", "validbet"}, tmp); err != nil {
			h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
			return
		}
		if err := apiCqStorage.ConfirmNoThisMtcode(tmp["mtcode"].(string)); err != nil {
			//statusMap["code"] = err.Error()
			//statusMap["message"] = "重复mtcode"
			//resMap["data"] = dataMap
			//resMap["status"] = statusMap
			h.response(w, h.GetRespData(err.Error(), "重复mtcode", dataMap, statusMap))
			return
		}
	}

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(amend.Account, ApiType); err != nil {
		//statusMap["code"] = "1006"
		//statusMap["message"] = err.Error()
		//resMap["data"] = dataMap
		//resMap["status"] = statusMap
		h.response(w, h.GetRespData("1006", err.Error(), dataMap, statusMap))
		return
	}

	err := apiCqStorage.AmendHandle(amend, mApiUser.Uid)
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		tmpBalance := float64(wallet.VndBalance)
		if amend.Action == "debit" {
			dataMap["balance"] = tmpBalance - amend.Amount + float64(int64(amend.Amount))
		} else {
			dataMap["balance"] = tmpBalance + amend.Amount - float64(int64(amend.Amount))
		}
		dataMap["balance"] = tmpBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap

	h.response(w, resMap)
}

func (h *CqHttp) Cancel(w http.ResponseWriter, r *http.Request) {
	mtcodes := struct {
		MtcodeArr []string `json:"mtcode"`
	}{}
	paramMap := make(map[string]interface{})

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	err := h.parse(w, r, &paramMap, &mtcodes)
	if err != nil {
		//statusMap["code"] = "1003"
		//statusMap["message"] = "参数解析失败"
		//resMap["data"] = dataMap
		//resMap["status"] = statusMap
		h.response(w, h.GetRespData("1003", "参数解析失败:" + err.Error(), dataMap, statusMap))

		return
	}
	if errCode, err := h.CheckParams([]string{"mtcode"}, paramMap); err != nil || len(mtcodes.MtcodeArr) <= 0 {
		//statusMap["code"] = "1003"
		//statusMap["message"] = "重要参数缺失！"
		//resMap["data"] = dataMap
		//resMap["status"] = statusMap
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	err, mtcodeData := apiCqStorage.GetMTCodeData(mtcodes.MtcodeArr[0])
	if err != nil {
		//statusMap["code"] = "1014"
		//statusMap["message"] = "mtcode not found"
		//resMap["data"] = dataMap
		//resMap["status"] = statusMap
		h.response(w, h.GetRespData("1014", "mtcode not found", dataMap, statusMap))
		return
	}
	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(mtcodeData.Account, ApiType); err != nil {
		//statusMap["code"] = "1006"
		//statusMap["message"] = err.Error()
		//resMap["data"] = dataMap
		//resMap["status"] = statusMap
		h.response(w, h.GetRespData("1006", err.Error(), dataMap, statusMap))
		return
	}

	err = apiCqStorage.CancelHandle(mtcodes.MtcodeArr, mApiUser.Uid)
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		dataMap["balance"] = wallet.VndBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap

	h.response(w, resMap)
}

func (h *CqHttp) Refunds(w http.ResponseWriter, r *http.Request) {
	mtcodes := struct {
		MtcodeArr []string `json:"mtcode"`
	}{}
	paramMap := make(map[string]interface{})

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	err := h.parse(w, r, &paramMap, &mtcodes)
	if err != nil {
		statusMap["code"] = "1003"
		statusMap["message"] = "参数解析失败"
		resMap["data"] = dataMap
		resMap["status"] = statusMap
		h.response(w, resMap)
		return
	}
	if _, err = h.CheckParams([]string{"mtcode"}, paramMap); err != nil || len(mtcodes.MtcodeArr) <= 0 {
		statusMap["code"] = "1003"
		statusMap["message"] = "重要参数缺失！"
		resMap["data"] = dataMap
		resMap["status"] = statusMap
		h.response(w, resMap)
		return
	}

	err, mtcodeData := apiCqStorage.GetMTCodeData(mtcodes.MtcodeArr[0])
	if err != nil {
		statusMap["code"] = "1014"
		statusMap["message"] = "mtcode no found"
		resMap["data"] = dataMap
		resMap["status"] = statusMap
		h.response(w, resMap)
		return
	}
	mApiUser := &apiStorage.ApiUser{}
	if err = mApiUser.GetApiUserByAccount(mtcodeData.Account, ApiType); err != nil {
		statusMap["code"] = "1006"
		statusMap["message"] = err.Error()
		resMap["data"] = dataMap
		resMap["status"] = statusMap
		h.response(w, resMap)
		return
	}

	err = apiCqStorage.RefundsHandle(mtcodes.MtcodeArr, mApiUser.Uid)
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		dataMap["balance"] = wallet.VndBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap

	h.response(w, resMap)
}

func (h *CqHttp) CheckParams(paramList []string, paramMap map[string]interface{}) (errCode string, err error) {
	for _, v := range paramList {
		if _, ok := paramMap[v]; !ok {
			return "1003", fmt.Errorf("重要参数缺失！%s不能为空", v)
		}
		if v == "amount" {
			if int64(paramMap[v].(float64)) < 0 {
				return "1003", fmt.Errorf("金额不能为负")
			}
		}
		if v == "eventtime" || v == "createTime" || v == "eventTime" {
			if _, err = time.ParseInLocation(time.RFC3339, paramMap[v].(string), time.Local); err != nil {
				return "1004", fmt.Errorf("时间戳解析失败！时：", paramMap[v].(string))
			}
		}
	}
	return "", nil
}

func (h *CqHttp) GetRespData(code, message string, dataMap, statusMap map[string]interface{}) map[string]interface{} {
	resMap := make(map[string]interface{})
	statusMap["code"] = code
	statusMap["message"] = message
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	return resMap
}

//视讯接口
func (h *CqHttp) Bet(w http.ResponseWriter, r *http.Request) {
	log.Debug("Bet %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	var bet apiCqStorage.Bet
	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &bet); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"account", "eventTime", "gamehall", "gamecode", "roundid", "amount", "mtcode"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	if err := apiCqStorage.ConfirmNoThisMtcode(paramMap["mtcode"].(string)); err != nil {
		h.response(w, h.GetRespData(err.Error(), "混合码已经存在了", dataMap, statusMap))
		return
	}

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(bet.Account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	err := apiCqStorage.BetHandle(bet, mApiUser.Uid)
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		tmpBalance := float64(wallet.VndBalance)
			tmpBalance = tmpBalance - bet.Amount + float64(int64(bet.Amount))
		dataMap["balance"] = tmpBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	h.response(w, resMap)
}

//视讯接口
func (h *CqHttp) EndRound(w http.ResponseWriter, r *http.Request) {
	log.Debug("EndRound %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	var endRound apiCqStorage.EndRound
	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &endRound); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"account", "gamehall", "gamecode", "roundid", "amount", "createTime"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	tmpData := paramMap["data"].([]interface{})
	for _, v := range tmpData {
		tmp := v.(map[string]interface{})
		if errCode, err := h.CheckParams([]string{"mtcode", "amount", "validbet", "eventtime"}, tmp); err != nil {
			h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
			return
		}
		if err := apiCqStorage.ConfirmNoThisMtcode(tmp["mtcode"].(string)); err != nil {
			h.response(w, h.GetRespData(err.Error(), "混合码已经存在了", dataMap, statusMap))
			return
		}
	}

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(endRound.Account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	err := apiCqStorage.EndRoundHandle(endRound, mApiUser.Uid)
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		tmpBalance := float64(wallet.VndBalance)
		for _, v := range endRound.Data {
			tmpBalance = tmpBalance - v.Amount + float64(int64(v.Amount))
		}
		dataMap["balance"] = tmpBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	h.response(w, resMap)
}

func (h *CqHttp) Rollout(w http.ResponseWriter, r *http.Request) {
	log.Debug("Rollout %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	var rollout apiCqStorage.Rollout
	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &rollout); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"account", "eventTime", "gamehall", "gamecode", "roundid", "amount", "mtcode"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	if err := apiCqStorage.ConfirmNoThisMtcode(paramMap["mtcode"].(string)); err != nil {
		h.response(w, h.GetRespData(err.Error(), "混合码已经存在了", dataMap, statusMap))
		return
	}

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(rollout.Account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	err := apiCqStorage.RolloutHandle(rollout, mApiUser.Uid)
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		dataMap["balance"] = wallet.VndBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	h.response(w, resMap)
}

func (h *CqHttp) Rollin(w http.ResponseWriter, r *http.Request) {
	log.Debug("Rollin %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &map[string]interface{}{}); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"account", "eventTime", "gamehall", "gamecode", "roundid", "validbet", "bet",
		"amount", "mtcode", "win", "mtcode", "createTime", "rake", "gametype"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	if err := apiCqStorage.ConfirmNoThisMtcode(paramMap["mtcode"].(string)); err != nil {
		h.response(w, h.GetRespData(err.Error(), "混合码已经存在了", dataMap, statusMap))
		return
	}

	amount := int64(paramMap["amount"].(float64))
	mtcode := paramMap["mtcode"].(string)
	account := paramMap["account"].(string)
	eventTime := paramMap["eventTime"].(string)

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	err := apiCqStorage.RollinHandle(amount, mtcode, account, mApiUser.Uid, eventTime, "rollin")
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		dataMap["balance"] = wallet.VndBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	h.response(w, resMap)
}

func (h *CqHttp) Debit(w http.ResponseWriter, r *http.Request) {
	log.Debug("Rollin %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &map[string]interface{}{}); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"account", "eventTime", "gamehall", "gamecode", "roundid",
		"amount", "mtcode"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	if err := apiCqStorage.ConfirmNoThisMtcode(paramMap["mtcode"].(string)); err != nil {
		h.response(w, h.GetRespData(err.Error(), "混合码已经存在了", dataMap, statusMap))
		return
	}

	amount := -int64(paramMap["amount"].(float64))
	mtcode := paramMap["mtcode"].(string)
	account := paramMap["account"].(string)
	eventTime := paramMap["eventTime"].(string)

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	err := apiCqStorage.RollinHandle(amount, mtcode, account, mApiUser.Uid, eventTime, "debit")
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		dataMap["balance"] = wallet.VndBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	h.response(w, resMap)
}

func (h *CqHttp) Credit(w http.ResponseWriter, r *http.Request) {
	log.Debug("Rollin %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &map[string]interface{}{}); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"account", "eventTime", "gamehall", "gamecode", "roundid",
		"amount", "mtcode"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	if err := apiCqStorage.ConfirmNoThisMtcode(paramMap["mtcode"].(string)); err != nil {
		h.response(w, h.GetRespData(err.Error(), "混合码已经存在了", dataMap, statusMap))
		return
	}

	amount := int64(paramMap["amount"].(float64))
	mtcode := paramMap["mtcode"].(string)
	account := paramMap["account"].(string)
	eventTime := paramMap["eventTime"].(string)

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	err := apiCqStorage.RollinHandle(amount, mtcode, account, mApiUser.Uid, eventTime, "credit")
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		dataMap["balance"] = wallet.VndBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	h.response(w, resMap)
}

func (h *CqHttp) Bonus(w http.ResponseWriter, r *http.Request) {
	log.Debug("Rollin %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &map[string]interface{}{}); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"account", "eventTime", "gamehall", "gamecode", "roundid",
		"amount", "mtcode"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	if err := apiCqStorage.ConfirmNoThisMtcode(paramMap["mtcode"].(string)); err != nil {
		h.response(w, h.GetRespData(err.Error(), "混合码已经存在了", dataMap, statusMap))
		return
	}

	amount := int64(paramMap["amount"].(float64))
	mtcode := paramMap["mtcode"].(string)
	account := paramMap["account"].(string)
	eventTime := paramMap["eventTime"].(string)

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	err := apiCqStorage.RollinHandle(amount, mtcode, account, mApiUser.Uid, eventTime, "bonus")
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		dataMap["balance"] = wallet.VndBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	h.response(w, resMap)
}

func (h *CqHttp) Payoff(w http.ResponseWriter, r *http.Request) {
	log.Debug("Payoff %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &map[string]interface{}{}); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"account", "eventTime", "amount", "mtcode"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	if err := apiCqStorage.ConfirmNoThisMtcode(paramMap["mtcode"].(string)); err != nil {
		h.response(w, h.GetRespData(err.Error(), "混合码已经存在了", dataMap, statusMap))
		return
	}

	amount := int64(paramMap["amount"].(float64))
	mtcode := paramMap["mtcode"].(string)
	account := paramMap["account"].(string)
	eventTime := paramMap["eventTime"].(string)

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	err := apiCqStorage.RollinHandle(amount, mtcode, account, mApiUser.Uid, eventTime, "payoff")
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		dataMap["balance"] = wallet.VndBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	h.response(w, resMap)
}

func (h *CqHttp) Refund(w http.ResponseWriter, r *http.Request) {
	log.Debug("Refund %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &map[string]interface{}{}); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"mtcode"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	mtcode := paramMap["mtcode"].(string)

	err, mtcodeData := apiCqStorage.GetMTCodeData(mtcode)
	if err != nil {
		statusMap["code"] = "1014"
		statusMap["message"] = "mtcode no found"
		resMap["data"] = dataMap
		resMap["status"] = statusMap
		h.response(w, resMap)
		return
	}

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(mtcodeData.Account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	err = apiCqStorage.RefundHandle(mtcode, mApiUser.Uid)
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		dataMap["balance"] = wallet.VndBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	h.response(w, resMap)
}

func (h *CqHttp) TakeAll(w http.ResponseWriter, r *http.Request) {
	log.Debug("TakeAll %d", time.Now().Unix())

	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	var takeall apiCqStorage.TakeAll
	paramMap := make(map[string]interface{})
	if err := h.parse(w, r, &paramMap, &takeall); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"account", "eventTime", "gamehall", "gamecode", "roundid", "mtcode"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}

	if err := apiCqStorage.ConfirmNoThisMtcode(paramMap["mtcode"].(string)); err != nil {
		h.response(w, h.GetRespData(err.Error(), "混合码已经存在了", dataMap, statusMap))
		return
	}

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(takeall.Account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
	tmpBalance := float64(wallet.VndBalance)
	err := apiCqStorage.TakeAllHandle(takeall, mApiUser.Uid)
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		dataMap["balance"] = 0
		dataMap["amount"] = tmpBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap
	h.response(w, resMap)
}

func (h *CqHttp) BatchBets(w http.ResponseWriter, r *http.Request) {
	log.Debug("BatchBets %d", time.Now().Unix())
	resMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	statusMap := make(map[string]interface{})
	statusMap["datatime"] = time.Now().Format(time.RFC3339)

	paramMap := make(map[string]interface{})
	var bets apiCqStorage.Bets
	if err := h.parse(w, r, &paramMap, &bets); err != nil {
		h.response(w, h.GetRespData("1003", "参数解析错误:" + err.Error(), dataMap, statusMap))
		return
	}

	if errCode, err := h.CheckParams([]string{"account", "gamehall", "gamecode", "createTime"}, paramMap); err != nil {
		h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
		return
	}
	tmpData := paramMap["data"].([]interface{})
	for _, v := range tmpData {
		tmp := v.(map[string]interface{})
		if errCode, err := h.CheckParams([]string{"mtcode", "amount", "roundid", "eventtime"}, tmp); err != nil {
			h.response(w, h.GetRespData(errCode, err.Error(), dataMap, statusMap))
			return
		}
		if err := apiCqStorage.ConfirmNoThisMtcode(tmp["mtcode"].(string)); err != nil {
			h.response(w, h.GetRespData(err.Error(), "混合码已经存在了", dataMap, statusMap))
			return
		}
	}

	mApiUser := &apiStorage.ApiUser{}
	if err := mApiUser.GetApiUserByAccount(bets.Account, ApiType); err != nil {
		h.response(w, h.GetRespData(err.Error(), "玩家资讯遗失！", dataMap, statusMap))
		return
	}

	err := apiCqStorage.BatchBetsHandle(bets, mApiUser.Uid)
	if err != nil {
		statusMap["code"] = err.Error()
		statusMap["message"] = "fail"
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
		tmpBalance := float64(wallet.VndBalance)
		for _, v := range bets.Data {
			tmpBalance = tmpBalance - v.Amount + float64(int64(v.Amount))
		}
		dataMap["balance"] = tmpBalance
		dataMap["currency"] = "VND"
		statusMap["code"] = "0"
		statusMap["message"] = "success"
	}
	resMap["data"] = dataMap
	resMap["status"] = statusMap

	h.response(w, resMap)
}


func (h *CqHttp) auth(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	apiToken := r.Header.Get("wtoken")
	if contentType != "application/json" {
		return fmt.Errorf("Content-Type err")
	}
	if apiToken != "cqtoken" {
		return fmt.Errorf("CQ-API-TOKEN err")
	}
	return nil
}

func (h *CqHttp) failed(w http.ResponseWriter) {
	data := map[string]interface{}{
		"status":  "failed",
		"message": "參數錯誤",
	}
	h.response(w, data)
}

func (h *CqHttp) parse(w http.ResponseWriter, r *http.Request, checkParam, param interface{}) (err error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("parse read param:%s", err.Error())
		return
	}
	log.Debug("body:%s", string(body))
	err = json.Unmarshal(body, &param)
	if err != nil {
		log.Error("parse param to struct err:%s", err.Error())
		return
	}
	err = json.Unmarshal(body, &checkParam)
	if err != nil {
		log.Error("parse param to struct err:%s", err.Error())
		return
	}
	if err = h.auth(r); err != nil {
		log.Error("parse auth err:%s", err.Error())
		return
	}
	return
}

func (h *CqHttp) response(w http.ResponseWriter, data map[string]interface{}) {
	repByte, _ := json.Marshal(data)
	fmt.Println("CQ9 response:", string(repByte))
	if _, err := w.Write(repByte); err != nil {
		log.Error("CQ9 w.Write err:%s", err.Error())
		return
	}
}

func (h *CqHttp) sha1(str string) string {
	sha := sha1.New()
	sha.Write([]byte(str))
	return string(sha.Sum(nil))
}

func (h *CqHttp) mapToSortJson(any interface{}) string {
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
	case []interface{}:
		var tmp []string
		for _, v := range any.([]interface{}) {
			tmp = append(tmp, fmt.Sprintf("%s", h.mapToSortJson(v)))
		}
		res = fmt.Sprintf("[%s]", strings.Join(tmp, ","))
	}
	return res
}

func (h *CqHttp) recvToJson(w http.ResponseWriter, r *http.Request, param interface{}) (err error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("parse read param:%s", err.Error())
		return
	}
	log.Debug("body:%s", string(body))
	err = json.Unmarshal(body, &param)
	if err != nil {
		log.Error("parse param to struct err:%s", err.Error())
		return
	}
	return nil
}
