package apiSaBa

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/storage/apiStorage"

	"github.com/goinggo/mapstructure"
)

const (
	SuccessCode int = 0
	UserExists  int = 6
)

var (
// settleStatusRuning  = "running"
// settleStatusSettled = "settled"
// settleStatusVoid    = "void"
)

var (
	host          = "http://r549api.bw6688.com/api"
	devHost       = "http://r549tsa.bw6688.com/api"
	vendorId      = "md2i1c14tu"
	operatorId    = "J18test"
	devVendorId   = "md2i1c14tu"
	devOperatorId = "J18test"
	currency      = 20
	language      = "cs"
	scale         = 1000.0
	defaultEnv    = "dev"
	suffix        = ""
	apiConfig     *apiStorage.ApiConfig
)

func InitEnv(env string) {
	if env == "dev" {
		host = devHost
		vendorId = devVendorId
		operatorId = devOperatorId
		suffix = "_test"
	} else if env == "prod" {
		defaultEnv = "prod"
		host = devHost
		vendorId = devVendorId
		operatorId = devOperatorId
		suffix = "_test"
	} else {
		language = "vn"
		currency = 51
		defaultEnv = "release"
	}
}

func CreateUser(account string) (code int, err error) {
	// betLimit := storage.QueryConf(storage.KAwcBetLimit)
	reqData := map[string]interface{}{
		"vendor_id":        vendorId,
		"OperatorId":       operatorId,
		"Vendor_Member_ID": account,
		"UserName":         account,
		"OddsType":         1,
		"Currency":         currency,
		"MinTransfer":      10000,
		"MaxTransfer":      10000000,
		// "UserGroup":  "SEXYBCRT",
	}
	res, err := httpPost("/CreateMember/", reqData)
	if err != nil {
		return
	}
	log.Debug("apiSaBa CreateUser res：%v", res)
	iCode, ok := res["error_code"]
	if !ok {
		err = fmt.Errorf("CreateUser response not find error_code")
		return
	}
	code = int(iCode.(float64))
	message := res["message"].(string)
	if code == SuccessCode || code == UserExists {
		return
	}

	err = fmt.Errorf("%v", message)
	return
}

func Login(account string) (data map[string]interface{}, err error) {
	reqData := map[string]interface{}{
		"vendor_id":        vendorId,
		"Vendor_Member_ID": account,
		// "UserGroup":  "SEXYBCRT",
	}
	res, err := httpPost("/Login/", reqData)
	if err != nil {
		return
	}
	log.Debug("apiSaBa Login res：%v", res)
	iCode, ok := res["error_code"]
	if !ok {
		err = fmt.Errorf("CreateUser response not find error_code")
		return
	}
	code := int(iCode.(float64))
	message := res["message"].(string)
	data = make(map[string]interface{})
	if code == SuccessCode {
		data["data"] = res["Data"].(string)
		return
	}
	err = fmt.Errorf("apiSaBa Login err:%v code:%d", message, code)
	return
}

func GetSabaUrl(account string) (data map[string]interface{}, err error) {
	reqData := map[string]interface{}{
		"vendor_id":        vendorId,
		"platform":         1,
		"Vendor_Member_ID": account,
		// "UserGroup":  "SEXYBCRT",
	}
	res, err := httpPost("/GetSabaUrl/", reqData)
	if err != nil {
		return
	}
	log.Debug("apiSaBa GetSabaUrl res：%v", res)
	iCode, ok := res["error_code"]
	if !ok {
		err = fmt.Errorf("apiSaBa GetSabaUrl response not find error_code")
		return
	}
	code := int(iCode.(float64))
	message := res["message"].(string)
	data = make(map[string]interface{})
	if code == SuccessCode {
		log.Debug("url:%v", data["url"])
		extends := &struct {
			OType     int    `json:"OType"`
			Skincolor string `json:"skincolor"`
		}{}
		data["url"] = fmt.Sprintf("%v&lang=%v", res["Data"], language)
		err = json.Unmarshal([]byte(apiConfig.Extends), extends)
		if err != nil {
			log.Error("apiSaBa GetSabaUrl parse apiConfig.Extends err:%s", err.Error())
		} else {
			data["url"] = fmt.Sprintf("%v&OType=%d&skincolor=%s", data["url"], extends.OType, extends.Skincolor)
		}
		return
	}
	err = fmt.Errorf("apiSaBa GetSabaUrl err:%v code:%d", message, code)
	return
}

func httpPost(urlStr string, requestBody map[string]interface{}) (data map[string]interface{}, err error) {
	client := utils.GetHttpClient(defaultEnv, common.App.GetSettings().Settings["devProxy"].(string))
	requestUrl := host + urlStr
	params := HttpBuildQuery(requestBody)
	log.Info("requestUrl:%s \r\nbody: %s", requestUrl, string(params))
	request, _ := http.NewRequest("POST", requestUrl, strings.NewReader(params))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	body, err := utils.Request(client, request)
	log.Info("response body: %s", string(body))
	if err != nil {
		log.Error("saba httpPost Request err:%s", err.Error())
		return
	}
	if err = json.Unmarshal(body, &data); err != nil {
		log.Error("saba httpPost Unmarshal err:%s", err.Error())
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

type GameData struct {
	HomeScore   int    `jpath:"home_score" json:"home_score"`
	AwayScore   int    `jpath:"away_score" json:"away_score"`
	HtHomeScore int    `jpath:"ht_home_score" json:"ht_home_score"`
	HtAwayScore int    `jpath:"ht_away_score" json:"ht_away_score"`
	GameStatus  string `jpath:"game_status" json:"game_status"`
	MatchId     int    `jpath:"match_id" json:"match_id"`
}

func getGameDetail(matchIds []string) (data []*GameData, err error) {
	data = []*GameData{}
	params := &struct {
		ErrorCode int         `jpath:"error_code" json:"error_code"`
		Message   string      `jpath:"message" json:"message"`
		Data      []*GameData `jpath:"Data" json:"Data"`
	}{}
	reqData := map[string]interface{}{
		"vendor_id": vendorId,
		"match_ids": strings.Join(matchIds, ","),
	}
	resData, err := httpPost("/GetGameDetail", reqData)
	if err != nil {
		return
	}
	if err = mapstructure.DecodePath(resData, params); err != nil {
		return
	}
	if params.ErrorCode == 0 {
		data = params.Data
	} else {
		err = fmt.Errorf(params.Message)
	}
	return
}
