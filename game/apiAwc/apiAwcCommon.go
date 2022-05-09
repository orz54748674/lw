package apiAwc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/storage"
	"vn/storage/walletStorage"
)

const (
	SuccessCode string = "0000"
	UserExists  string = "1001"
)

var (
	host       = "https://api.onlinegames22.com"
	agentId    = "luckywagpr"
	agentDevId = "lkwinagt"
	cert       = "fCGKKpp3OdbsyEjHaPH"
	devCert    = "7TceAzZnVwoZIJ63N6R"
	currency   = "VND"
	language   = "cn"
	scale      = 1000.0
	defaultEnv = "dev"
)

type Rep map[string]interface{}

func InitEnv(env string) {
	if env == "dev" {
		host = "https://tttint.onlinegames22.com"
		agentId = agentDevId
		cert = devCert
	} else if env == "prod" {
		defaultEnv = "prod"
		host = "https://tttint.onlinegames22.com"
		agentId = agentDevId
		cert = devCert
	} else {
		language = "vn"
		// host = "https://api.aegaming-global.cc"
		defaultEnv = "release"
	}
}

func CreateUser(account string) (string, error) {
	betLimit := storage.QueryConf(storage.KAwcBetLimit)
	data := map[string]interface{}{
		"cert":     cert,
		"agentId":  agentId,
		"userId":   account,
		"currency": currency,
		"language": language,
		"betLimit": betLimit.(string),
	}

	res, err := httpPost("/wallet/createMember", data)
	if err != nil {
		return "", err
	}
	status, ok := res["status"]
	if !ok {
		return "", fmt.Errorf("CreateUser response not find status")
	}
	return status.(string), nil
}

func Login(account string) (data map[string]interface{}, err error) {
	reqData := map[string]interface{}{
		"cert":     cert,
		"agentId":  agentId,
		"userId":   account,
		"language": language,
	}
	res, err := httpPost("/wallet/login", reqData)
	if err != nil {
		return nil, err
	}
	status, ok := res["status"]
	if !ok {
		return nil, fmt.Errorf("Login response not find status")
	}
	if status == SuccessCode {
		data = res
		return
	}

	err = fmt.Errorf("%v", res)
	return
}

func DoLoginAndLaunchGame(account string) (data map[string]interface{}, err error) {
	betLimit := storage.QueryConf(storage.KAwcBetLimit)
	reqData := map[string]interface{}{
		"cert":     cert,
		"agentId":  agentId,
		"userId":   account,
		"language": language,
		"platform": "SEXYBCRT",
		"gameType": "LIVE",
		"gameCode": "MX-LIVE-001",
		"betLimit": betLimit.(string),
	}
	res, err := httpPost("/wallet/doLoginAndLaunchGame", reqData)
	if err != nil {
		return nil, err
	}
	status, ok := res["status"]
	if !ok {
		return nil, fmt.Errorf("DoLoginAndLaunchGame response not find status")
	}
	if status == SuccessCode {
		data = res
		return
	}

	err = fmt.Errorf("%v", res)
	return
}

func httpPost(urlStr string, requestBody map[string]interface{}) (res Rep, err error) {
	client := utils.GetHttpClient(defaultEnv, common.App.GetSettings().Settings["devProxy"].(string))
	requestUrl := host + urlStr
	params := HttpBuildQuery(requestBody)
	log.Info("requestUrl:%s \r\nbody: %s", requestUrl, params)
	request, _ := http.NewRequest("POST", requestUrl, strings.NewReader(params))
	//增加header选项
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Accept", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	log.Info("res: %s", string(body))
	err = json.Unmarshal(body, &res)
	if err != nil {
		return
	}
	return
}

func getComResp(data map[string]interface{}, params ...string) map[string]interface{} {
	resp := map[string]interface{}{
		"status":    "0000",
		"balanceTs": time.Now().Format("2006-01-02T15:04:05.999Z07:00"),
	}
	for k, v := range data {
		resp[k] = v
	}
	paramsLen := len(params)
	if paramsLen > 0 {
		if len(params[0]) != 0 {
			wallet := walletStorage.QueryWallet(utils.ConvertOID(params[0]))
			resp["balance"] = float64(wallet.VndBalance) / scale
		}
	} else if paramsLen > 1 {
		if len(params[1]) != 0 {
			resp["userId"] = params[1]
		}
	}
	return resp
}

func HttpBuildQuery(params map[string]interface{}) (param_str string) {
	params_arr := make([]string, 0, len(params))
	for k, v := range params {
		params_arr = append(params_arr, fmt.Sprintf("%s=%v", k, v))
	}
	//fmt.Println(params_arr)
	param_str = strings.Join(params_arr, "&")
	return param_str
}
