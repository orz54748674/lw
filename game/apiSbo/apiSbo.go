package apiSbo

import (
	"encoding/json"
	"fmt"
	"time"
	"vn/common/utils"
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

type Sbo struct {
	basemodule.BaseModule
	rpcServer *SboRpc
}

var (
	actionRpcEnter      = "/apiSbo/enter"
	basePath            = "/sbo"
	getBalance          = basePath + "/GetBalance"
	deduct              = basePath + "/Deduct"
	settle              = basePath + "/Settle"
	rollback            = basePath + "/Rollback"
	cancel              = basePath + "/Cancel"
	tip                 = basePath + "/Tip"
	bonus               = basePath + "/Bonus"
	returnStake         = basePath + "/ReturnStake"
	getBetStatus        = basePath + "/GetBetStatus"
	liveCoinTransaction = basePath + "/LiveCoinTransaction"
)
var Module = func() module.Module {
	this := new(Sbo)
	return this
}

func (m *Sbo) GetType() string {
	return "apiSbo"
}

func (m *Sbo) Version() string {
	return "1.0.0"
}

func (m *Sbo) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)
	if code, err := CreateAgent(); err != nil {
		log.Debug("Sbo CreateAgent code：%v, err:%v", code, err)
	}
	apiStorage.InitApiConfig(m)
	m.GetServer().RegisterGO(actionRpcEnter, m.enter)

	registerLock := make(chan bool, 1)
	go utils.RegisterRpcToHttp(m, getBalance, getBalance, m.rpcServer.GetBalance, registerLock)
	go utils.RegisterRpcToHttp(m, deduct, deduct, m.rpcServer.Deduct, registerLock)
	go utils.RegisterRpcToHttp(m, settle, settle, m.rpcServer.Settle, registerLock)
	go utils.RegisterRpcToHttp(m, rollback, rollback, m.rpcServer.Rollback, registerLock)
	go utils.RegisterRpcToHttp(m, cancel, cancel, m.rpcServer.Cancel, registerLock)
	go utils.RegisterRpcToHttp(m, tip, tip, m.rpcServer.Tip, registerLock)
	go utils.RegisterRpcToHttp(m, bonus, bonus, m.rpcServer.Bonus, registerLock)
	go utils.RegisterRpcToHttp(m, returnStake, returnStake, m.rpcServer.ReturnStake, registerLock)
	go utils.RegisterRpcToHttp(m, getBetStatus, getBetStatus, m.rpcServer.GetBetStatus, registerLock)
	go utils.RegisterRpcToHttp(m, liveCoinTransaction, liveCoinTransaction, m.rpcServer.LiveCoinTransaction, registerLock)
	go m.loadBetRecord()

}

func (m *Sbo) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", m.GetType())
	<-closeSig
	log.Info("%v模块已停止...", m.GetType())
}

func (m *Sbo) OnDestroy() {
	//一定别忘了继承
	m.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", m.GetType())
}

func (m *Sbo) enter(data map[string]interface{}) (resp string, err error) {
	params := &struct {
		Token       string `json:"token"`
		GameType    string `json:"gameType"`
		Action      string `json:"action"`
		Uid         string `json:"uid"`
		ProductType string `json:"productType"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		log.Error("rpc awc enter err:%s", err.Error())
		return
	}
	uid := params.Uid
	mApiUser := &apiStorage.ApiUser{}
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	eCode, err := CreateUser(user.Account)
	if err != nil {
		return "", err
	} else if eCode == UserExists || eCode == SuccessCode {
		err = mApiUser.GetApiUser(uid, apiStorage.SboType)
		if err == mongo.ErrNoDocuments {
			mApiUser.Account = fmt.Sprintf("%s%s", accountPrefix, user.Account)
			mApiUser.Type = apiStorage.SboType
			mApiUser.Uid = uid
			if err = mApiUser.Save(); err != nil {
				return "", err
			}
		} else if err != nil {
			return "", err
		}
		loginInfo, err := Login(mApiUser.Account, params.ProductType)
		if err != nil {
			log.Error("ApiLoginErr err:%s", err.Error())
			return "", err
		}
		url, ok := loginInfo["url"]
		if !ok {
			log.Error("ApiLogin LoginUrl not find")
			return "", err
		}

		return url.(string), nil
	}
	return "", fmt.Errorf("CreateUser errCode:%d", eCode)
}

func (m *Sbo) InitApiCfg(cfg *apiStorage.ApiConfig) {
	cfg.ApiType = apiStorage.SboType
	cfg.ApiTypeName = string(game.ApiSbo)
	cfg.Env = m.App.GetSettings().Settings["env"].(string)
	cfg.Module = m.GetType()
	cfg.GameType = apiStorage.Sports
	cfg.GameTypeName = "Sports"
	cfg.Topic = actionRpcEnter[1:]
	cfg.ProductType = "SportsBook"

	apiStorage.AddApiConfig(cfg)

	cfg.GameType = apiStorage.Live
	cfg.GameTypeName = "Live"
	cfg.ProductType = "Casino"
	InitEnv(cfg.Env)
}

func (m *Sbo) loadBetRecord() {
	log.Debug("Sbo loadBetRecord")
	ticker := time.NewTicker(time.Second * 120)
	mRecordCheckTime := &apiStorage.ApiRecordCheckTime{}
	var cstZone = time.FixedZone("UTC", -4*3600)

	var maxTimeInterval time.Duration = 1800 //30 分钟
	var dayCount time.Duration = 30          // 没有记录去30天内记录
	timeFormat := "2006-01-02T15:04:05"
	for {
		<-ticker.C
		t, err := mRecordCheckTime.GetLastTime(apiStorage.SboType)
		if err != nil {
			log.Error("Sbo loadBetRecord mRecordCheckTime.GetLastTime err:%s", err.Error())
			t.Time = time.Now().In(cstZone).Add(-time.Hour * 24 * dayCount)
		}
		startTime := t.Time.In(cstZone).Format(timeFormat)
		objEndTime := time.Now()
		if (time.Now().In(cstZone).Unix() - t.Time.In(cstZone).Unix()) > int64(maxTimeInterval) {
			objEndTime = t.Time.Add(time.Second * maxTimeInterval)

		}
		endTime := objEndTime.In(cstZone).Format(timeFormat)
		log.Debug("Sbo loadBetRecord startTime:%v", startTime)
		log.Debug("Sbo loadBetRecord   endTime:%v", endTime)
		mRecordCheckTime.Time = objEndTime
		res, err := getBetListByModifyDate(startTime, endTime, "SportsBook")
		if err != nil {
			log.Error("Sbo loadBetRecord mRecordCheckTime.UpdateTime err:%s", err.Error())
			continue
		}
		log.Debug("res：%v", res)
		for _, iRecord := range res {
			record := iRecord.(map[string]interface{})
			refNo := record["refNo"].(string)
			btData, err := json.Marshal(record)
			if err != nil {
				log.Error("Sbo loadBetRecord json.Marshal:%v", string(btData))
				continue
			}
			gameStorage.UpdateRecord(refNo, string(btData))
		}

		err = mRecordCheckTime.UpdateTime(apiStorage.SboType)
		if err != nil {
			log.Error("Sbo loadBetRecord mRecordCheckTime.UpdateTime err:%s", err.Error())
			continue
		}
		log.Debug("Sbo loadBetRecord now time:%d,time.Now().In(cstZone):%d", time.Now().Unix(), time.Now().In(cstZone).Unix())
	}
}
