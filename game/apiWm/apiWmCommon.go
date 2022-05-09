package apiWm

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
)

var (
	host         = ""
	vendorId     = ""
	signature    = ""
	vendorIdDev  = "j18swapi"
	signatureDev = "dd16bb12d9ddf0c3cdc7ea710582eef2"
	// currency     = "VND"
	language   = 0
	scale      = 1000.0
	defaultEnv = "dev"
)

const (
	SuccessCode int = 0
	UserExists  int = 104
)

type Resp struct {
	ErrorCode    int         `json:"errorCode"`
	ErrorMassage string      `json:"errorMessage"`
	Result       interface{} `json:"result"`
}

func InitEnv(env string) {
	if env == "dev" {
		host = "https://api.a45.me/api/wallet/Gateway.php"
		vendorId = vendorIdDev
		signature = signatureDev
	} else if env == "prod" {
		defaultEnv = "prod"
		host = "https://api.a45.me/api/wallet/Gateway.php"
		vendorId = vendorIdDev
		signature = signatureDev
	} else {
		language = 3
		defaultEnv = "release"
	}
}

func CreateUser(account, pwd string) (int, error) {
	limitType := storage.QueryConf(storage.KWmBetLimit)
	data := map[string]interface{}{
		"cmd":       "MemberRegister",
		"vendorId":  vendorId,
		"signature": signature,
		"user":      fmt.Sprintf("%s_%s", vendorId, account),
		"password":  pwd,
		"username":  account,
		"limitType": limitType,
		"timestamp": time.Now().Unix(),
	}
	res, err := httpPost("", data)
	if err != nil {
		return -1, err
	}
	return res.ErrorCode, err
}

func Login(account, pwd string) (*Resp, error) {
	data := map[string]interface{}{
		"cmd":       "LoginGame",
		"vendorId":  vendorId,
		"signature": signature,
		"user":      fmt.Sprintf("%s_%s", vendorId, account),
		"password":  pwd,
		"lang":      language,
		"timestamp": time.Now().Unix(),
	}
	res, err := httpPost("", data)
	if err != nil {
		return nil, err
	}
	return res, err
}

func httpPost(urlStr string, requestBody map[string]interface{}) (res *Resp, err error) {
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
	res = &Resp{}
	log.Info("res: %s", string(body))
	err = json.Unmarshal(body, res)
	if err != nil {
		return
	}
	return
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

func getResponse() *Resp {
	return &Resp{
		ErrorCode:    0,
		ErrorMassage: "",
	}
}
