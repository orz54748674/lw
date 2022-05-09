package apiSbo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/log"
)

const (
	SuccessCode int = 0
	UserExists  int = 4103
)

var (
	settleStatusRuning  = "running"
	settleStatusSettled = "settled"
	settleStatusVoid    = "void"
)

var (
	host          = "https://ex-api-demo-yy.568win.com"
	companyKey    = "5284A54E919E4138A0A2517DA417DF7E"
	Agent         = "sboJ18agentPeter"
	serverId      = "j18"
	devCompanyKey = "5284A54E919E4138A0A2517DA417DF7E"
	devServerId   = "j18"
	currency      = "VND"
	language      = "zh-cn"
	scale         = 1000.0
	defaultEnv    = "dev"
	accountPrefix = "sbo"
)

func InitEnv(env string) {
	if env == "dev" {
		host = "https://ex-api-demo-yy.568win.com"
		companyKey = devCompanyKey
		serverId = devServerId
		Agent = "sboJ18agentPeter"
	} else if env == "prod" {
		defaultEnv = "prod"
		host = "https://ex-api-demo-yy.568win.com"
		companyKey = devCompanyKey
		serverId = devServerId
		Agent = "sboJ18agentPeter"
	} else {
		language = "vn"
		defaultEnv = "release"
	}
}

func CreateUser(account string) (code int, err error) {
	// betLimit := storage.QueryConf(storage.KAwcBetLimit)
	reqData := map[string]interface{}{
		"CompanyKey": companyKey,
		"ServerId":   serverId,
		"Username":   fmt.Sprintf("%s%s", accountPrefix, account),
		"Agent":      Agent,
		// "UserGroup":  "SEXYBCRT",
	}
	res, err := httpPost("/web-root/restricted/player/register-player.aspx", reqData)
	if err != nil {
		return
	}
	log.Debug("sbo CreateUser res：%v", res)
	iErrorInfo, ok := res["error"]
	if !ok {
		err = fmt.Errorf("CreateUser response not find errorInfo")
		return
	}
	errorInfo := iErrorInfo.(map[string]interface{})
	code = int(errorInfo["id"].(float64))
	if code == SuccessCode || code == UserExists {
		return
	}

	err = fmt.Errorf("%v", errorInfo["msg"])
	return
}

func CreateAgent() (code int, err error) {
	reqData := map[string]interface{}{
		"CompanyKey":       companyKey,
		"ServerId":         serverId,
		"Username":         Agent,
		"Password":         Agent,
		"Currency":         currency,
		"Min":              10,
		"Max":              10000,
		"MaxPerMatch":      20000,
		"CasinoTableLimit": 3,
		// "UserGroup":  "SEXYBCRT",
	}
	res, err := httpPost("/web-root/restricted/agent/register-agent.aspx", reqData)
	if err != nil {
		return
	}
	log.Debug("sbo CreateAgent res：%v", res)
	iErrorInfo, ok := res["error"]
	if !ok {
		err = fmt.Errorf("CreateAgent response not find errorInfo")
		return
	}
	errorInfo := iErrorInfo.(map[string]interface{})
	code = int(errorInfo["id"].(float64))
	if code == SuccessCode {
		return
	}

	err = fmt.Errorf("%v", errorInfo["msg"])
	return
}

func Login(account, productType string) (data map[string]interface{}, err error) {
	reqData := map[string]interface{}{
		"CompanyKey":  companyKey,
		"ServerId":    serverId,
		"Username":    account,
		"Portfolio":   productType, //SportsBook / Casino / Games / VirtualSports / SeamlessGame / ThirdPartySportsBook
		"IsWapSports": false,
		// "UserGroup":  "SEXYBCRT",
	}
	res, err := httpPost("/web-root/restricted/player/login.aspx", reqData)
	if err != nil {
		return
	}
	log.Debug("sbo Login res：%v", res)
	iErrorInfo, ok := res["error"]
	if !ok {
		err = fmt.Errorf("sbo Login response not find errorInfo")
		return
	}
	errorInfo := iErrorInfo.(map[string]interface{})
	code := int(errorInfo["id"].(float64))
	data = make(map[string]interface{})
	if code == SuccessCode {
		data["url"] = res["url"]
		return
	}
	err = fmt.Errorf("sbo Login err:%v code:%d", errorInfo["msg"], code)
	return
}

func getBetListByModifyDate(startDate, endDate, productType string) (data []interface{}, err error) {
	reqData := map[string]interface{}{
		"CompanyKey":    companyKey,
		"ServerId":      serverId,
		"Language":      language,
		"StartDate":     startDate,
		"EndDate":       endDate,
		"Portfolio":     productType, //SportsBook / Casino / Games / VirtualSports / SeamlessGame / ThirdPartySportsBook
		"IsGetDownline": true,
		// "UserGroup":  "SEXYBCRT",
	}
	res, err := httpPost("/web-root/restricted/report/v2/get-bet-list-by-modify-date.aspx", reqData)
	if err != nil {
		return
	}
	iErrorInfo, ok := res["error"]
	if !ok {
		err = fmt.Errorf("sbo Login response not find errorInfo")
		return
	}
	errorInfo := iErrorInfo.(map[string]interface{})
	code := int(errorInfo["id"].(float64))
	data = []interface{}{}
	if code == SuccessCode {
		data = res["result"].([]interface{})
		return
	}
	log.Debug("res:%v", res)
	return
}

func httpPost(urlStr string, requestBody map[string]interface{}) (data map[string]interface{}, err error) {
	client := utils.GetHttpClient(defaultEnv, common.App.GetSettings().Settings["devProxy"].(string))
	requestUrl := host + urlStr
	params, err := json.Marshal(requestBody)
	if err != nil {
		return
	}
	log.Info("requestUrl:%s \r\nbody: %s", requestUrl, string(params))
	request, _ := http.NewRequest("POST", requestUrl, strings.NewReader(string(params)))
	request.Header.Add("Content-Type", "application/json")
	body, err := utils.Request(client, request)
	if err != nil {
		return
	}
	if err = json.Unmarshal(body, &data); err != nil {
		return
	}
	return
}
