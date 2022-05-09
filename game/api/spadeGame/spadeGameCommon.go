package spadeGame

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/log"
)

type commonRespone struct {
	MerchantCode string `json:"merchantCode"`
	Code         int    `json:"code"`
	Msg          string `json:"msg"`
	SerialNo     string `json:"serialNo"`
}

type sGame struct {
	GameCode    string `json:"gameCode"`
	GameName    string `json:"gameName"`
	Jackpot     bool   `json:"jackpot"`
	Thumbnail   string `json:"thumbnail"`
	Screenshot  string `json:"screenshot"`
	Mthumbnail  string `json:"mthumbnail"`
	JackpotCode string `json:"jackpotCode"`
	JackpotName string `json:"jackpotName"`
}

const (
	SuccessCode int = 0
)

var (
	imagePath       = "./static/spade/"
	imgHost         = "https://api-egame-staging.sgplay.net"
	host            = fmt.Sprintf("%s/api", imgHost)
	merchantCode    = "SGLUCKY"
	devMerchantCode = "SGLUCKY"
	currency        = "VND"
	scale           = 1000.0
	language        = "zh_CN"
	defaultEnv      = "dev"
	accountPrefix   = "spade_game_"
	tokenSecret     = "zen2pct74a5Q7n0GmCGhe1dtE0KxYrHG"
	iv              = "tYs3NWAfRY45sZGGR2NWCymYSJW5iw7t"
	gameUrl         = "https://lobby-egame-staging.sgplay.net/SGLUCKY/auth/?"
)

func InitEnv(env string) {
	if env == "dev" {
		host = fmt.Sprintf("%s/api", imgHost)
		merchantCode = devMerchantCode
	} else if env == "prod" {
		defaultEnv = "prod"
		host = fmt.Sprintf("%s/api", imgHost)
		merchantCode = devMerchantCode
	} else {
		language = "vi_VN"
		defaultEnv = "release"
	}
}

func createToken(acctId string) (err error) {
	reqData := map[string]interface{}{
		"acctId":       acctId,
		"merchantCode": merchantCode,
		"action":       "ticketLog",
		"serialNo":     createSerialNo(),
	}
	res, err := httpPost("createToken", reqData)
	if err != nil {
		return
	}
	log.Debug("createToken res:%v", res)
	return
}

type gameListRep struct {
	commonRespone
	Games []*sGame `json:"games"`
}

func getGameList() (gameList []*sGame, err error) {
	reqData := map[string]interface{}{
		"merchantCode": merchantCode,
		"serialNo":     createSerialNo(),
	}

	res, err := httpPost("getGames", reqData)
	if err != nil {
		log.Debug("SpadeGame getGameList err:%s", err.Error())
		return
	}
	repData := &gameListRep{}
	err = json.Unmarshal(res, repData)
	if err != nil {
		log.Debug("SpadeGame getGameList Unmarshal err:%s", err.Error())
		return
	}
	if repData.Code != SuccessCode {
		log.Debug("SpadeGame getGameList Unmarshal failed:%s", repData.Msg)
		return
	}
	gameList = repData.Games
	return
}

func httpPost(urlStr string, requestBody map[string]interface{}) (data []byte, err error) {
	client := utils.GetHttpClient(defaultEnv, common.App.GetSettings().Settings["devProxy"].(string))
	requestUrl := host
	params, err := json.Marshal(requestBody)
	if err != nil {
		return
	}
	log.Info("requestUrl:%s \r\nbody: %s", requestUrl, string(params))
	request, _ := http.NewRequest("POST", requestUrl, strings.NewReader(string(params)))
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("DataType", "JSON")
	request.Header.Add("API", urlStr)
	data, err = utils.Request(client, request)
	return
}

func createSerialNo() (serialNo string) {
	return fmt.Sprint(time.Now().UnixNano())
}
