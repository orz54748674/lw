package apiAe

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/storage/apiStorage"
	"vn/storage/walletStorage"
)

type AeHttp struct {
}

func (a *AeHttp) GetBalance(w http.ResponseWriter, r *http.Request) {
	log.Debug("AeHttp.GetBalance %d", time.Now().Unix())
	params := &struct {
		CurrentTime string `json:"currentTime"`
		AccountID   string `json:"accountId"`
		MerchantID  string `json:"merchantId"`
		Sign        string `json:"sign"`
		Currency    string `json:"currency"`
	}{}
	if err := a.parse(w, r, params); err != nil {
		log.Error("AeHttp.GetBalance parse params err:%s", err.Error())
		a.failed(w, "单一钱包不存在或无法取得", 9100)
		return
	}
	sign := utils.MD5(fmt.Sprintf("%s%s%s%s%s", merchantId, params.CurrentTime, params.AccountID, currency, b64MerchantKey))
	if sign != params.Sign {
		log.Debug("AeHttp.GetBalance sign err")
		a.failed(w, "单一钱包不存在或无法取得", 9100)
		return
	}

	accountInfo := strings.Split(params.AccountID, "_")
	account := strings.Join(accountInfo[1:], "_")
	mApiUser := &apiStorage.ApiUser{}
	err := mApiUser.GetApiUserByAccount(account, apiType)
	if err != nil {
		log.Error("GetUserBalance GetApiUserByAccount err:%s", err.Error())
		a.failed(w, "单一钱包不存在或无法取得", 9100)
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(mApiUser.Uid))
	data := map[string]interface{}{
		"currency":     currency,
		"balance":      float64(wallet.VndBalance),
		"bonusBalance": 0,
	}
	a.response(w, data)
}

func (a *AeHttp) Transfer(w http.ResponseWriter, r *http.Request) {
	log.Debug("AeHttp.Transfer %d", time.Now().Unix())
	params := &struct {
		CurrentTime string  `json:"currentTime"`
		GameId      string  `json:"gameId"`
		AccountID   string  `json:"accountId"`
		Amount      float64 `json:"amount"`
		BetAmount   float64 `json:"betAmount"`
		WinAmount   float64 `json:"winAmount"`
		MerchantID  string  `json:"merchantId"`
		Currency    string  `json:"currency"`
		Sign        string  `json:"sign"`
		TxnTypeID   int64   `json:"txnTypeId"`
		TxnID       string  `json:"txnId"`
	}{}
	if err := a.parse(w, r, params); err != nil {
		a.failed(w, "钱包余额过低", 1201)
		return
	}
	md5Data := fmt.Sprintf("%s%f%s%s%s%d%s", params.CurrentTime, params.Amount, params.AccountID, currency, params.TxnID, params.TxnTypeID, params.GameId)
	sign := utils.MD5(fmt.Sprintf(getSignFmt(), md5Data))
	if sign != params.Sign {
		a.failed(w, "钱包余额过低", 1201)
		return
	}

}

func (a *AeHttp) Query(w http.ResponseWriter, r *http.Request) {
	log.Debug("AeHttp.Query %d", time.Now().Unix())
	params := &struct {
		AccountID   string  `json:"accountId"`
		Currency    string  `json:"currency"`
		TxnID       string  `json:"txnId"`
		GameId      string  `json:"gameId"`
		Amount      float64 `json:"amount"`
		TxnTypeID   int64   `json:"txnTypeId"`
		CurrentTime string  `json:"currentTime"`
		MerchantID  string  `json:"merchantId"`
		Sign        string  `json:"sign"`
	}{}
	if err := a.parse(w, r, params); err != nil {
		a.failed(w, "单一钱包余额查询发生错误", 9203)
		return
	}
	md5Data := fmt.Sprintf("%s%f%s%s%s%d%s", params.CurrentTime, params.Amount, params.AccountID, currency, params.TxnID, params.TxnTypeID, params.GameId)
	sign := utils.MD5(fmt.Sprintf(getSignFmt(), md5Data))
	if sign != params.Sign {
		a.failed(w, "单一钱包余额查询发生错误", 9203)
		return
	}
}

func (a *AeHttp) parse(w http.ResponseWriter, r *http.Request, param interface{}) (err error) {
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
	return
}

func (a *AeHttp) failed(w http.ResponseWriter, msg string, code int) {
	data := map[string]interface{}{
		"msg":  msg,
		"code": code,
	}
	a.response(w, data)
}

func (a *AeHttp) response(w http.ResponseWriter, data map[string]interface{}) {
	btData, _ := json.Marshal(data)
	if _, err := w.Write([]byte(btData)); err != nil {
		log.Error("response: ioWrite  json format err:%s", err.Error())
		return
	}
}
