package apiCmd

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/storage/apiCmdStorage"
	"vn/storage/gameStorage"
	"vn/storage/walletStorage"
)

type Rep struct {
	SerialKey string `json:"SerialKey"`
	Timestamp int64 `json:"Timestamp"`
	Code int `json:"Code"`
	Message string `json:"Message"`
	Data   interface{} `json:"data"`
}

type RepStatus struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	DateTime  string `json:"datatime"`
	TraceCode string `json:"traceCode"`
}

func IsUserExist(account string) (exist bool, err error) {
	params := []Param{
		NewParam("Method", "exist"),
		NewParam("PartnerKey", PartnerKey),
		NewParam("UserName", account),
	}
	fmt.Println("partnerkey....", PartnerKey, account)
	queryStr := HttpBuildQuery(params)
	url := "SportsApi.aspx?" + queryStr
	res, err := httpGet(url)
	if err != nil {
		return false, err
	}
	fmt.Println("res.............", res)
	exist = res.Data.(bool)
	return
}

func CreateToken(account string) (string, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	tmp := utils.RandomString(32,r)
	err := apiCmdStorage.UpsertCmdUserToken(account, tmp)
	if err != nil {
		return "", err
	}
	return tmp, nil
}

func Login(account, loginUrl string) (url string, err error) {
	exist, err := IsUserExist(account)
	if !exist {
		params := []Param{
			NewParam("Method", "createmember"),
			NewParam("PartnerKey", PartnerKey),
			NewParam("UserName", account),
			NewParam("Currency", "VD"),
		}
		queryStr := HttpBuildQuery(params)
		requestUrl := "SportApi.aspx?" + queryStr
		res, err := httpGet(requestUrl)
		if err != nil {
			return "", err
		}
		if res.Code != 0 {
			err = fmt.Errorf("cmd create account err:%s", res.Message)
			return "", err
		}
	}

	tmpToken, err := CreateToken(account)
	if err != nil {
		return
	}
	params := []Param{
		NewParam("lang", Lang),
		NewParam("user", account),
		NewParam("token", tmpToken),
		NewParam("currency", "VD"),
		NewParam("templatename", "green"),
		NewParam("view", "v1"),
	}

	queryStr := HttpBuildQuery(params)
	url = loginUrl + "auth.aspx?" + queryStr

	fmt.Println("url", url)
	return url, nil
}

func GetParlayRecord(socTransID int) []interface{} {
	params := []Param{
		NewParam("Method", "parlaybetrecord"),
		NewParam("PartnerKey", PartnerKey),
		NewParam("SocTransId", strconv.Itoa(socTransID)),
	}
	queryStr := HttpBuildQuery(params)
	requestUrl := "SportApi.aspx?" + queryStr
	res, err := httpGet(requestUrl)
	if err != nil || res.Code != 0 {
		log.Error("get api cmd bet record err")
	}

	dataArr := res.Data.([]interface{})
	return dataArr
}

func GetCashOutRecord(socTransID int) []interface{} {
	params := []Param{
		NewParam("Method", "cashoutbetrecord"),
		NewParam("PartnerKey", PartnerKey),
		NewParam("SocTransId", strconv.Itoa(socTransID)),
	}
	queryStr := HttpBuildQuery(params)
	requestUrl := "SportApi.aspx?" + queryStr
	res, err := httpGet(requestUrl)
	if err != nil || res.Code != 0 {
		log.Error("get api cmd bet record err")
	}

	dataArr := res.Data.([]interface{})
	return dataArr
}

func HttpGetLanguageInfo(reqType, reqID int) string {
	params := []Param{
		NewParam("Method", "languageinfo"),
		NewParam("PartnerKey", PartnerKey),
		NewParam("Type", strconv.Itoa(reqType)),
		NewParam("ID", strconv.Itoa(reqID)),
	}
	queryStr := HttpBuildQuery(params)
	requestUrl := "SportApi.aspx?" + queryStr
	res, err := httpGet(requestUrl)
	if err != nil || res.Code != 0 {
		log.Error("get api cmd bet record err", err.Error())
		time.Sleep(time.Second * 30)
		return ""
	}
	if tmpByte, err := json.Marshal(res.Data); err != nil {
		return ""
	} else {
		return string(tmpByte)
	}
}

func GetTeamLeagueName(infoType, infoID int) string {
	var str string
	info, err := apiCmdStorage.GetCmdTeamLeagueInfo(infoType, infoID)
	if err != nil {
		str = HttpGetLanguageInfo(infoType, infoID)
		if str != "" {
			apiCmdStorage.InsertCmdTeamLeagueInfo(infoType, infoID, str)
		}
	} else {
		str = info.InfoName
	}

	if str != "" {
		tmpMap := make(map[string]interface{})
		json.Unmarshal([]byte(str), &tmpMap)
		ret := tmpMap[Lang].(string)
		return ret
	}
	return ""
}

func GetDataTeamLeagueName(record map[string]interface{}) map[string]interface{} {
	if _, ok := record["AwayId"]; !ok {
		record["AwayTeamName"] = ""
	} else {
		record["AwayTeamName"] = GetTeamLeagueName(0, int(record["AwayId"].(float64)))
	}
	if _, ok := record["HomeId"]; !ok {
		record["HomeTeamName"] = ""
	} else {
		record["HomeTeamName"] = GetTeamLeagueName(0, int(record["HomeId"].(float64)))
	}
	if _, ok := record["LeagueId"]; !ok {
		record["LeagueName"] = ""
	} else {
		record["LeagueName"] = GetTeamLeagueName(1, int(record["LeagueId"].(float64)))
	}
	return record
}

func GetBetRecordTiming() {
	for {
		conf, _ := apiCmdStorage.GetApiCmdConf()
		versionID := conf.VersionID
		params := []Param{
			NewParam("Method", "betrecord"),
			NewParam("PartnerKey", PartnerKey),
			NewParam("Version", strconv.FormatInt(versionID, 10)),
		}
		queryStr := HttpBuildQuery(params)
		requestUrl := "SportApi.aspx?" + queryStr
		res, err := httpGet(requestUrl)
		if err != nil || res.Code != 0 {
			if err != nil {
				log.Error("get api cmd bet record err", err.Error())
			} else {
				log.Error("get api cmd bet record err", res.Code)
			}
			time.Sleep(time.Second * 30)
			continue
		}

		dataArr := res.Data.([]interface{})
		maxID := conf.VersionID
		referenceMap := make(map[string]int)
		for _, v := range dataArr {
			record := v.(map[string]interface{})
			fmt.Println("CMD GetBetRecordTiming record............", record)

			curID := int(record["Id"].(float64))
			if int64(curID) > maxID {maxID = int64(curID)}
			curReferenceNo := record["ReferenceNo"].(string)
			account := record["SourceName"].(string)
			uid := apiCmdStorage.GetUidByAccount(account)
			if uid == "" {
				log.Error("upsert api cmd bet record err:player no found")
				continue
			}

			record["AwayTeamName"] = GetTeamLeagueName(0, int(record["AwayTeamId"].(float64)))
			record["HomeTeamName"] = GetTeamLeagueName(0, int(record["HomeTeamId"].(float64)))
			record["LeagueName"] = GetTeamLeagueName(1, int(record["LeagueId"].(float64)))

			if num, ok := referenceMap[curReferenceNo]; !ok || num < curID {
				tmpID := apiCmdStorage.GetSelectBetRecordId(curReferenceNo)
				if tmpID < curID {
					apiCmdStorage.UpsertApiCmdBetRecord(record)

					tmpBytes, _ := json.Marshal(record)
					recordDetails := string(tmpBytes)
					var recordParams gameStorage.BetRecordParam
					recordParams.Uid = uid
					recordParams.GameNo = curReferenceNo
					recordParams.GameType = game.ApiCmd
					recordParams.GameResult = recordDetails
					recordParams.BetDetails = recordDetails
					if record["IsCashOut"].(bool) {
						recordParams.Income = int64(record["CashOutWinLoseAmount"].(float64) * 1000)
						wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
						recordParams.CurBalance = wallet.VndBalance + wallet.SafeBalance
					}
					referenceMap[curReferenceNo] = curID
					if record["TransType"].(string) == "PAR" {
						socTransID := int(record["SocTransId"].(float64))
						tmpMap := make(map[string]interface{})
						if err := json.Unmarshal(tmpBytes, &tmpMap); err == nil {
							tmpData := GetParlayRecord(socTransID)
							for k, v := range tmpData {
								data := v.(map[string]interface{})
								data = GetDataTeamLeagueName(data)
								tmpData[k] = data
							}
							tmpMap["ParData"] = tmpData
							bb, _ := json.Marshal(tmpMap)
							recordParams.GameResult = string(bb)
							recordParams.BetDetails = string(bb)
						} else {
							log.Error("cmd json.Unmarshal err", err.Error())
						}
					} else if record["IsCashOut"].(bool) {
						socTransID := int(record["SocTransId"].(float64))
						tmpMap := make(map[string]interface{})
						if err := json.Unmarshal(tmpBytes, &tmpMap); err == nil {
							tmpData := GetCashOutRecord(socTransID)
							for k, v := range tmpData {
								data := v.(map[string]interface{})
								data = GetDataTeamLeagueName(data)
								tmpData[k] = data
							}
							tmpMap["CashOutData"] = tmpData
							bb, _ := json.Marshal(tmpMap)
							recordParams.GameResult = string(bb)
							recordParams.BetDetails = string(bb)
						} else {
							log.Error("cmd json.Unmarshal err", err.Error())
						}
					}
					gameStorage.UpdateApiCmdBetRecord(recordParams, 6)
				}
			}
		}

		apiCmdStorage.UpdateApiCmdConf(int(maxID))
		time.Sleep(time.Second * 60)
	}
}
