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
)

const (
	SuccessCode int = 0
	UserExists  int = 2217
)

var (
	host           = ""
	merchantId     = "LUCKYWIN"
	merchantDevId  = "LUCKYWIN"
	merchantKey    = "2789bb10-a348-11eb-9f06-0242c0a87003"
	merchantDevKey = "1cbb6e50d256b483f030596aba208df532f4ca4ed47cc0825062dc53725419504b34985ad374f3b2e1461f21d05c1d2600c55b4a0e6a65201877bbb238d6d753"
	currency       = "VND"
	language       = "zh_CN"
	b64MerchantKey = ""
)

type Rep struct {
	ErrorCode int         `json:"code"`
	Message   string      `json:"msg"`
	Data      interface{} `json:"data"`
}

func InitEnv(env string) {
	if env == "dev" {
		host = "https://api.aegaming-global.games"
		merchantId = merchantDevId
		merchantKey = merchantDevKey
	} else {
		host = "https://api.aegaming-global.cc"
	}
	b64MerchantKey = utils.B64Encode(merchantKey)
}

func CreateUser(account string) (int, error) {
	currentTime := getCurrentTime()
	username := fmt.Sprintf("%s_%s", merchantId, account)
	log.Debug("sign:%s", fmt.Sprintf("%s|%s|%s|%s|%s", merchantId, currency, currentTime, username, b64MerchantKey))
	data := map[string]interface{}{
		"merchantId":  merchantId,
		"currency":    currency,
		"currentTime": currentTime,
		"username":    username,
		"sign":        utils.MD5(fmt.Sprintf("%s%s%s%s%s", merchantId, currency, currentTime, username, b64MerchantKey)),
		"language":    language,
		"brandCode":   "",
	}

	res, err := httpPost("/api/register", data)
	if err != nil {
		return -1, err
	}
	return res.ErrorCode, nil
}

func Login(account, gameId string, playMode, device int8) (data map[string]interface{}, err error) {
	currentTime := getCurrentTime()
	username := fmt.Sprintf("%s_%s", merchantId, account)
	md5Data := fmt.Sprintf("%s%s%s%d%d%s%s", currency, currentTime, username, playMode, device, gameId, language)
	log.Debug("Login md5Data", fmt.Sprintf("%s|%s|%s|%d|%d|%s|%s", currency, currentTime, username, playMode, device, gameId, language))
	log.Debug("Login sign", fmt.Sprintf(getSignFmt(), md5Data))
	sign := utils.MD5(fmt.Sprintf(getSignFmt(), md5Data))
	reqData := map[string]interface{}{
		"merchantId":   merchantId,
		"currency":     currency,
		"currentTime":  currentTime,
		"username":     fmt.Sprintf("%s_%s", merchantId, account),
		"device":       device,
		"gameId":       gameId,
		"playmode":     playMode,
		"merchHomeUrl": "",
		"sign":         sign,
		"language":     language,
	}
	res, err := httpPost("/api/login", reqData)
	if err != nil {
		return nil, err
	}
	if res.ErrorCode == SuccessCode {
		data = res.Data.(interface{}).(map[string]interface{})
		return
	}

	err = fmt.Errorf("%v", res.Message)
	return
}

func httpPost(urlStr string, requestBody map[string]interface{}) (res *Rep, err error) {
	// proxyUrl := "http://127.0.0.1:10800"
	// proxy, _ := url.Parse(proxyUrl)
	// tr := &http.Transport{
	// 	Proxy: http.ProxyURL(proxy),
	// 	//TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// }
	client := http.Client{
		// Transport: tr,
		Timeout: 5 * time.Second,
	}
	requestUrl := host + urlStr
	btRequestBody, _ := json.Marshal(requestBody)
	log.Info("requestUrl:%s \r\nbody: %s", requestUrl, string(btRequestBody))
	request, _ := http.NewRequest("POST", requestUrl, strings.NewReader(string(btRequestBody)))
	//增加header选项
	request.Header.Add("Content-Type", "application/json")
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

func getCurrentTime() string {
	return fmt.Sprint(time.Now().UnixNano())[0:13]
}

func getSignFmt() string {
	return merchantId + "%s" + b64MerchantKey
}
