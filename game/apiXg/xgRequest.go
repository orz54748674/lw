package apiXg

import (
	"fmt"
	"vn/storage"
)

const (
	SuccessCode int = 0
	UserExists  int = 8
)

var (
	scale       = 1.0
	host        = ""
	agentId     = "2789bb10-a348-11eb-9f06-0242c0a87003"
	agentIdDev  = "8630e392-a347-11eb-8856-06b6fe12fbc4"
	agentKeyDev = "8630e4be-a347-11eb-9a78-06b6fe12fbc4"
	agentKey    = "dcaabe96-a658-11eb-b019-0242c0a87003"
	language    = "zh-CN"
	defaultEnv  = "dev"
)

type Rep struct {
	ErrorCode int         `json:"ErrorCode"`
	Message   string      `json:"Message"`
	Data      interface{} `json:"Data"`
	UUQUID    string      `json:"uuquid"`
}

func InitHost(env string) {
	if env == "dev" {
		host = "http://stage.api.agent.x-gaming.bet:8667/api/keno-api/"
		agentId = agentIdDev
		agentKey = agentKeyDev
	} else if env == "prod" {
		defaultEnv = "prod"
		host = "http://stage.api.agent.x-gaming.bet:8667/api/keno-api/"
		agentId = agentIdDev
		agentKey = agentKeyDev
	} else {
		defaultEnv = "release"
		language = "vn"
		host = "https://agent.x-gaming.bet/api/keno-api/"
	}
}

// url:/xg-casino/AccountExist?AgentId=公鑰(代理編號)&Account=GD9183&Key=驗證參數
func IsUserExist(account string) (exist bool, err error) {
	params := []Param{
		NewParam("AgentId", agentId),
		NewParam("Account", account),
	}
	queryStr := HttpBuildQuery(params)
	params = append(params, NewParam("Key", getSign(queryStr)))
	queryStr2 := HttpBuildQuery(params)
	url := "xg-casino/AccountExist?" + queryStr2
	res, err := httpGet(url)
	if err != nil {
		return false, err
	}
	exist = res.Data.(bool)
	return
}

func CreateUser(account string) (int, error) {
	betLimit := storage.QueryConf(storage.KXgBetLimit)
	params := []Param{
		NewParam("AgentId", agentId),
		NewParam("Account", account),
		NewParam("LimitStake", betLimit.(string)),
	}
	queryStr := HttpBuildQuery(params)
	params = append(params, NewParam("Key", getSign(queryStr)))
	url := "xg-casino/CreateMember"
	res, err := httpPost(url, params)
	if err != nil {
		return -1, err
	}
	return res.ErrorCode, nil
}

func Login(account, gameId string) (data map[string]interface{}, err error) {
	eCode, err := setBetLimit(account)
	if err != nil {
		return
	}
	if eCode != SuccessCode {
		err = fmt.Errorf("ServerBusy")
		return
	}
	params := []Param{
		NewParam("AgentId", agentId),
		NewParam("Lang", language),
		NewParam("GameId", gameId),
		NewParam("Account", account),
	}
	params = append(params, NewParam("Key", getSign(HttpBuildQuery(params))))
	queryStr := HttpBuildQuery(params)
	url := "xg-casino/Login?" + queryStr
	res, err := httpGet(url)
	if err != nil {
		return
	}
	if res.ErrorCode == SuccessCode {
		data = res.Data.(interface{}).(map[string]interface{})
		return
	}

	err = fmt.Errorf("%v", res.Message)
	return
}

func setBetLimit(account string) (int, error) {
	betLimit := storage.QueryConf(storage.KXgBetLimit)
	params := []Param{
		NewParam("AgentId", agentId),
		NewParam("Account", account),
		NewParam("Template", betLimit.(string)),
	}
	params = append(params, NewParam("Key", getSign(HttpBuildQuery(params))))
	url := "xg-casino/Template"
	res, err := httpPost(url, params)
	if err != nil {
		return -1, err
	}
	return res.ErrorCode, nil
}
