package apiSaBa

import (
	"encoding/json"
	"fmt"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/storage/apiStorage"
	"vn/storage/gameStorage"
	"vn/storage/userStorage"

	"github.com/mitchellh/mapstructure"
)

var (
	actionRpcEnter   = "/apiSaBa/enter"
	base             = "/saba"
	placebet         = base + "/placebet"
	placeBetParlay   = base + "/placebetparlay"
	confirmBet       = base + "/confirmbet"
	confirmBetParlay = base + "/confirmbetparlay"
	cancelBet        = base + "/cancelbet"
	settle           = base + "/settle"
	resettle         = base + "/resettle"
	unsettle         = base + "/unsettle"
	getBalance       = base + "/getbalance"
	cashOut          = base + "/cashout"
	cashOutResettle  = base + "/cashoutresettle"
	placeBet3rd      = base + "/placebet3rd"
	confirmBet3rd    = base + "/confirmbet3rd"
	updateBet        = base + "/updatebet"
)

// http://www.athena-demo-online.com/SingleWallet_integration_ch.html
type SaBa struct {
	basemodule.BaseModule
	rpcServer *SaBaRpc
}

var Module = func() module.Module {
	this := new(SaBa)
	return this
}

func (m *SaBa) GetType() string {
	return "apiSaBa"
}

func (m *SaBa) Version() string {
	return "1.0.0"
}

func (m *SaBa) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)
	apiStorage.InitApiConfig(m)
	mongoIncDataExpireDay := int64(m.App.GetSettings().Settings["mongoIncDataExpireDay"].(float64))
	apiStorage.InitSabaBetRecord(mongoIncDataExpireDay)
	m.GetServer().RegisterGO(actionRpcEnter, m.enter)
	registerLock := make(chan bool, 1)
	go utils.RegisterRpcToHttp(m, placebet, placebet, m.rpcServer.placeBet, registerLock)
	go utils.RegisterRpcToHttp(m, placeBetParlay, placeBetParlay, m.rpcServer.placeBetParlay, registerLock)
	go utils.RegisterRpcToHttp(m, confirmBet, confirmBet, m.rpcServer.confirmBet, registerLock)
	go utils.RegisterRpcToHttp(m, confirmBetParlay, confirmBetParlay, m.rpcServer.confirmBetParlay, registerLock)
	go utils.RegisterRpcToHttp(m, cancelBet, cancelBet, m.rpcServer.cancelBet, registerLock)
	go utils.RegisterRpcToHttp(m, settle, settle, m.rpcServer.settle, registerLock)
	go utils.RegisterRpcToHttp(m, resettle, resettle, m.rpcServer.resettle, registerLock)
	go utils.RegisterRpcToHttp(m, unsettle, unsettle, m.rpcServer.unsettle, registerLock)
	go utils.RegisterRpcToHttp(m, getBalance, getBalance, m.rpcServer.getBalance, registerLock)
	go utils.RegisterRpcToHttp(m, cashOut, cashOut, m.rpcServer.cashOut, registerLock)
	go utils.RegisterRpcToHttp(m, cashOutResettle, cashOutResettle, m.rpcServer.cashOutResettle, registerLock)
	go utils.RegisterRpcToHttp(m, placeBet3rd, placeBet3rd, m.rpcServer.placeBet3rd, registerLock)
	go utils.RegisterRpcToHttp(m, confirmBet3rd, confirmBet3rd, m.rpcServer.confirmBet3rd, registerLock)
	go utils.RegisterRpcToHttp(m, updateBet, updateBet, m.rpcServer.updateBet, registerLock)
	go m.getGameDetail()
	go m.SetGameData()
}

func (m *SaBa) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", m.GetType())
	<-closeSig
	log.Info("%v模块已停止...", m.GetType())
}

func (m *SaBa) OnDestroy() {
	//一定别忘了继承
	m.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", m.GetType())
}

func (m *SaBa) InitApiCfg(cfg *apiStorage.ApiConfig) {
	apiConfig = cfg
	cfg.ApiType = apiStorage.SaBaType
	cfg.ApiTypeName = string(game.ApiSaBa)
	cfg.Env = m.App.GetSettings().Settings["env"].(string)
	cfg.Module = m.GetType()
	cfg.GameType = apiStorage.Sports
	cfg.GameTypeName = apiStorage.SportsName
	cfg.Topic = actionRpcEnter[1:]
	cfg.Extends = `{"OType":2,"skincolor":"bl001"}`
	InitEnv(cfg.Env)
}

func (m *SaBa) enter(data map[string]interface{}) (resp string, err error) {
	params := &struct {
		Token    string `json:"token"`
		GameType string `json:"gameType"`
		Action   string `json:"action"`
		Uid      string `json:"uid"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		log.Error("rpc awc enter err:%s", err.Error())
		return
	}

	uid := params.Uid
	mApiUser := &apiStorage.ApiUser{}
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	user.Account = fmt.Sprintf("%s_%s%s", operatorId, user.Account, suffix)
	eCode, err := CreateUser(user.Account)
	log.Debug("SaBa CreateUser eCode:%v", eCode)
	if err != nil {
		return "", err
	} else if eCode == UserExists || eCode == SuccessCode {
		err = mApiUser.GetApiUser(uid, apiStorage.SaBaType)
		log.Debug("SaBa mApiUser.GetApiUser err:%v", err)
		if err == mongo.ErrNoDocuments {
			mApiUser.Account = user.Account
			mApiUser.Type = apiStorage.SaBaType
			mApiUser.Uid = uid
			if err = mApiUser.Save(); err != nil {
				return "", err
			}
		} else if err != nil {
			return "", err
		}
		loginInfo, err := Login(mApiUser.Account)
		if err != nil {
			log.Error("ApiLoginErr err:%s", err.Error())
			return "", err
		}
		_, ok := loginInfo["data"]
		if !ok {
			log.Error("ApiLogin LoginUrl not data")
			return "", err
		}

		res, err := GetSabaUrl(mApiUser.Account)
		if err != nil {
			log.Error("GetSabaUrl err:%s", err.Error())
			return "", err
		}
		url, ok := res["url"]
		if !ok {
			log.Error("ApiLogin LoginUrl not find")
			return "", err
		}

		return url.(string), nil
	}
	return "", fmt.Errorf("CreateUser errCode:%d", eCode)
}

func (m *SaBa) getGameDetail() {
	log.Debug("getGameDetail")
	mBetRecord := &apiStorage.SabaBetRecord{}
	ticker := time.NewTicker(time.Second * 40)
	for {
		<-ticker.C
		records, err := mBetRecord.GetNoCompletedRecord()
		if err != nil && err != mongo.ErrNoDocuments {
			log.Error("GetNoCompletedRecord err:%s", err.Error())
			continue
		}
		if len(records) == 0 {
			continue
		}
		matchIds := []string{}
		for _, record := range records {
			matchId := fmt.Sprint(record.MatchId)
			if utils.StrInArray(matchId, matchIds) || matchId == "0" {
				continue
			}
			matchIds = append(matchIds, matchId)
		}
		if len(matchIds) == 0 {
			continue
		}
		gameDatas, err := getGameDetail(matchIds)
		if err != nil {
			log.Error("GetNoCompletedRecord err:%s", err.Error())
			continue
		}
		for _, gameData := range gameDatas {
			mBetRecord.HomeScore = gameData.HomeScore
			mBetRecord.AwayScore = gameData.HomeScore
			mBetRecord.HtAwayScore = gameData.HtAwayScore
			mBetRecord.HtHomeScore = gameData.HtHomeScore
			mBetRecord.GameStatus = gameData.GameStatus
			err := mBetRecord.SetScore(gameData.MatchId)
			if err != nil {
				log.Error("GetNoCompletedRecord mBetRecord.SetScore MatchId:%d err:%s", gameData.MatchId, err.Error())
				continue
			}
		}
	}

}

func (m *SaBa) SetGameData() {
	log.Debug("SetGameData")
	mBetRecord := &apiStorage.SabaBetRecord{}
	ticker := time.NewTicker(time.Second * 40)
	for {
		<-ticker.C
		records, err := mBetRecord.GetNoSetGameData(settleStatus)
		if err != nil && err != mongo.ErrNoDocuments {
			log.Error("SetGameData GetNoSetGameData err:%s", err.Error())
			continue
		}
		if len(records) == 0 {
			continue
		}

		for _, record := range records {
			GameResults := []map[string]interface{}{}
			if len(record.MatchOids) == 0 {
				GameResults = append(GameResults, m.createGameData(record))
			} else {
				oids := []primitive.ObjectID{}
				for _, v := range record.MatchOids {
					oids = append(oids, utils.ConvertOID(v))
				}
				res, err := mBetRecord.GetRecordByOids(oids)
				if err != nil {
					log.Error("SetGameData GetRecordByOids err:%s", err.Error())
					continue
				}
				for _, rd := range res {
					rd.SettleStatus = record.SettleStatus
					GameResults = append(GameResults, m.createGameData(record))
				}
			}
			btGameResult, _ := json.Marshal(GameResults)
			gameStorage.UpdateGameRes(record.Oid.Hex(), string(btGameResult))
		}
	}
}

func (m *SaBa) createGameData(record *apiStorage.SabaBetRecord) (gameData map[string]interface{}) {
	gameData = map[string]interface{}{
		"MatchId":      record.MatchId,
		"HomeScore":    record.HomeScore,
		"HtHomeScore":  record.HtHomeScore,
		"AwayScore":    record.AwayScore,
		"HTAwayScore":  record.HtAwayScore,
		"Status":       record.Status,
		"SettleStatus": record.SettleStatus,
	}
	return
}
