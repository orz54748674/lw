package apiCq

import (
	"fmt"
)

const (
	SuccessCode int = 0
	UserExists  int = 8
)

type Rep struct {
	Data   interface{} `json:"data"`
	Status RepStatus   `json:"status"`
}

type RepStatus struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	DateTime  string `json:"datatime"`
	TraceCode string `json:"traceCode"`
}

func Login(account, gameCode, driveType string) (url string, err error) {
	plat := "web"
	app := "N"
	if driveType == "0" {
		plat = "mobile"
		app = "Y"
	}
	params := []Param{
		NewParam("account", account),
		NewParam("gamehall", "cq9"),
		NewParam("gamecode", gameCode),
		NewParam("gameplat", plat),
		NewParam("lang", "zh-cn"),
		NewParam("app", app),
	}
	cqUrl := "/gameboy/player/sw/gamelink"
	res, err := httpPost(cqUrl, params)
	if err != nil {
		return
	}
	if res.Status.Code == "0" {
		data := res.Data.(interface{}).(map[string]interface{})
		url = data["url"].(string)
		return
	}

	err = fmt.Errorf("%v", res.Status.Message)
	return
}

func GetGameHall() {
	url := "/gameboy/game/halls"
	res, err := httpGet(url)
	if err != nil {
		return
	}

	err = fmt.Errorf("%v", res.Status.Message)
	fmt.Println("res.........", res)
	return
}

func GetGameCode() {
	url := "/gameboy/game/list/cq9"
	res, err := httpGet(url)
	if err != nil {
		return
	}

	err = fmt.Errorf("%v", res.Status.Message)
	fmt.Println("res.........", res)
	return
}


//func GetBetRecordTiming() {
//	for {
//		url := "/gameboy/order/view"
//		params := []Param{
//			NewParam("starttime", account),
//			NewParam("endtime", "cq9"),
//			NewParam("page", gameCode),
//			NewParam("account", plat),
//			NewParam("pagesize", "zh-cn"),
//		}
//		res, err := httpGet2(url)
//		if err != nil || res.Code != 0 {
//			if err != nil {
//				log.Error("get api cmd bet record err", err.Error())
//			} else {
//				log.Error("get api cmd bet record err", res.Code)
//			}
//			time.Sleep(time.Second * 30)
//			continue
//		}
//
//		dataArr := res.Data.([]interface{})
//		maxID := conf.VersionID
//		referenceMap := make(map[string]int)
//		for _, v := range dataArr {
//			record := v.(map[string]interface{})
//			fmt.Println("CMD GetBetRecordTiming record............", record)
//
//			curID := int(record["Id"].(float64))
//			if int64(curID) > maxID {maxID = int64(curID)}
//			curReferenceNo := record["ReferenceNo"].(string)
//			account := record["SourceName"].(string)
//			uid := apiCmdStorage.GetUidByAccount(account)
//			if uid == "" {
//				log.Error("upsert api cmd bet record err:player no found")
//				continue
//			}
//
//			record["AwayTeamName"] = GetTeamLeagueName(0, int(record["AwayTeamId"].(float64)))
//			record["HomeTeamName"] = GetTeamLeagueName(0, int(record["HomeTeamId"].(float64)))
//			record["LeagueName"] = GetTeamLeagueName(1, int(record["LeagueId"].(float64)))
//
//			if num, ok := referenceMap[curReferenceNo]; !ok || num < curID {
//				tmpID := apiCmdStorage.GetSelectBetRecordId(curReferenceNo)
//				if tmpID < curID {
//					apiCmdStorage.UpsertApiCmdBetRecord(record)
//
//					tmpBytes, _ := json.Marshal(record)
//					recordDetails := string(tmpBytes)
//					var recordParams gameStorage.BetRecordParam
//					recordParams.Uid = uid
//					recordParams.GameNo = curReferenceNo
//					recordParams.GameType = game.ApiCmd
//					recordParams.GameResult = recordDetails
//					recordParams.BetDetails = recordDetails
//					if record["IsCashOut"].(bool) {
//						recordParams.Income = int64(record["CashOutWinLoseAmount"].(float64) * 1000)
//						wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
//						recordParams.CurBalance = wallet.VndBalance + wallet.SafeBalance
//					}
//					referenceMap[curReferenceNo] = curID
//					if record["TransType"].(string) == "PAR" {
//						socTransID := int(record["SocTransId"].(float64))
//						tmpMap := make(map[string]interface{})
//						if err := json.Unmarshal(tmpBytes, &tmpMap); err == nil {
//							tmpData := GetParlayRecord(socTransID)
//							for k, v := range tmpData {
//								data := v.(map[string]interface{})
//								data = GetDataTeamLeagueName(data)
//								tmpData[k] = data
//							}
//							tmpMap["ParData"] = tmpData
//							bb, _ := json.Marshal(tmpMap)
//							recordParams.GameResult = string(bb)
//							recordParams.BetDetails = string(bb)
//						} else {
//							log.Error("cmd json.Unmarshal err", err.Error())
//						}
//					} else if record["IsCashOut"].(bool) {
//						socTransID := int(record["SocTransId"].(float64))
//						tmpMap := make(map[string]interface{})
//						if err := json.Unmarshal(tmpBytes, &tmpMap); err == nil {
//							tmpData := GetCashOutRecord(socTransID)
//							for k, v := range tmpData {
//								data := v.(map[string]interface{})
//								data = GetDataTeamLeagueName(data)
//								tmpData[k] = data
//							}
//							tmpMap["CashOutData"] = tmpData
//							bb, _ := json.Marshal(tmpMap)
//							recordParams.GameResult = string(bb)
//							recordParams.BetDetails = string(bb)
//						} else {
//							log.Error("cmd json.Unmarshal err", err.Error())
//						}
//					}
//					gameStorage.UpdateApiCmdBetRecord(recordParams, 6)
//				}
//			}
//		}
//
//		apiCmdStorage.UpdateApiCmdConf(int(maxID))
//		time.Sleep(time.Second * 60)
//	}
//}
