package apiWm

import (
	"fmt"
	"strconv"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/game/activity"
	"vn/storage/activityStorage"
	"vn/storage/apiStorage"
	"vn/storage/gameStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"

	"github.com/mitchellh/mapstructure"
)

type Wm struct {
	basemodule.BaseModule
}

var (
	actionRpcEnter = "/apiWm/enter"
	userMap        = make(map[string]string)
)

var Module = func() module.Module {
	this := new(Wm)
	return this
}

func (self *Wm) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "apiWm"
}

func (self *Wm) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (self *Wm) OnInit(app module.App, settings *conf.ModuleSettings) {
	self.BaseModule.OnInit(self, app, settings)
	apiStorage.InitApiConfig(self)

	self.GetServer().RegisterGO(actionRpcEnter, self.enter)
}

func (self *Wm) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	go self.GetDateTimeReport()
	<-closeSig
	log.Info("%v模块已停止...", self.GetType())
}

func (self *Wm) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}

func (self *Wm) enter(data map[string]interface{}) (resp string, err error) {
	log.Debug("api wm enter:%d", time.Now().Unix())
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
	log.Debug("params:%v", params)
	uid := params.Uid
	mApiUser := &apiStorage.ApiUser{}
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	eCode, err := CreateUser(user.Account, uid)
	if err != nil {
		return "", err
	} else if eCode == SuccessCode || eCode == UserExists {
		err = mApiUser.GetApiUser(uid, apiStorage.WmType)
		if err == mongo.ErrNoDocuments {
			mApiUser.Account = fmt.Sprintf("%s_%s", vendorId, user.Account)
			mApiUser.Type = apiStorage.WmType
			mApiUser.Uid = uid
			if err = mApiUser.Save(); err != nil {
				return "", err
			}
		} else if err != nil {
			return "", err
		}
		loginInfo, err := Login(user.Account, uid)
		if err != nil {
			log.Error("ApiLoginErr err:%s", err.Error())
			return "", err
		}
		if loginInfo.ErrorCode != SuccessCode {
			return "", fmt.Errorf("ApiLoginErr err ErrorCode:%d", loginInfo.ErrorCode)
		}
		return loginInfo.Result.(string), nil
	}
	return "", fmt.Errorf("CreateUser errCode:%d", eCode)
}

func (self *Wm) InitApiCfg(cfg *apiStorage.ApiConfig) {
	cfg.ApiType = apiStorage.WmType
	cfg.ApiTypeName = string(game.ApiWm)
	cfg.Env = self.App.GetSettings().Settings["env"].(string)
	cfg.Module = self.GetType()
	cfg.GameType = apiStorage.Live
	cfg.GameTypeName = "Live"
	cfg.Topic = actionRpcEnter[1:]
	cfg.ScreenType = 1
	InitEnv(cfg.Env)
	mongoIncDataExpireDay := int64(self.App.GetSettings().Settings["mongoIncDataExpireDay"].(float64))
	apiStorage.InitWmBetRecord(mongoIncDataExpireDay)
	apiStorage.InitWmBillRecord(mongoIncDataExpireDay)
}

func (self *Wm) GetDateTimeReport() {
	ticker := time.NewTicker(time.Second * 60)

	data := map[string]interface{}{
		"cmd":       "GetDateTimeReport",
		"vendorId":  vendorId,
		"signature": signature,
		"datatype":  2,
	}
	for {
		log.Debug("GetDateTimeReport time:%d", time.Now().Unix())
		<-ticker.C
		data["startTime"] = time.Now().Add(-time.Second * time.Duration(600)).Format("20060102150405")
		data["timestamp"] = time.Now().Unix()
		res, err := httpPost("", data)
		if err != nil || res.ErrorCode == 10418 {
			continue
		}
		if res == nil {
			continue
		}
		log.Debug("GetDateTimeReport res :%v", res)
		if res.ErrorCode == 0 && res.Result != nil {
			result := res.Result.([]interface{})
			for _, iItem := range result {
				item := iItem.(map[string]interface{})
				betAmount, err := strconv.ParseFloat(item["bet"].(string), 64)
				if err != nil {
					log.Error("GetDateTimeReport betAmount err:%s", err.Error())
					continue
				}
				validbet, err := strconv.ParseFloat(item["validbet"].(string), 64)
				if err != nil {
					log.Error("GetDateTimeReport betAmount err:%s", err.Error())
					continue
				}
				water, err := strconv.ParseFloat(item["water"].(string), 64)
				if err != nil {
					log.Error("GetDateTimeReport water err:%s", err.Error())
					continue
				}
				waterbet, err := strconv.ParseFloat(item["waterbet"].(string), 64)
				if err != nil {
					log.Error("GetDateTimeReport waterbet err:%s", err.Error())
					continue
				}
				winLoss, err := strconv.ParseFloat(item["winLoss"].(string), 64)
				if err != nil {
					log.Error("GetDateTimeReport winLoss err:%s", err.Error())
					continue
				}
				record := &apiStorage.WmBetRecord{
					Account:        item["user"].(string),
					BetId:          item["betId"].(string),
					BetTime:        item["betTime"].(string),
					BetAmount:      betAmount * scale,
					Validbet:       validbet * scale,
					Water:          water * scale,
					Result:         item["result"].(string),
					BetCode:        item["betCode"].(string),
					BetResult:      item["betResult"].(string),
					Waterbet:       waterbet * scale,
					WinLoss:        winLoss * scale,
					Gid:            item["gid"].(string),
					Event:          item["event"].(string),
					EventChild:     item["eventChild"].(string),
					TableId:        item["tableId"].(string),
					GameResult:     item["gameResult"].(string),
					GName:          item["gname"].(string),
					BetWalletId:    item["betwalletid"].(string),
					ResultWalletId: item["resultwalletid"].(string),
					Commission:     item["commission"].(string),
					Reset:          item["reset"].(string),
					SetTime:        item["settime"].(string),
				}
				if record.IsExists() {
					continue
				}
				uid, ok := userMap[record.Account]
				if !ok {
					apiUser := &apiStorage.ApiUser{}
					err := apiUser.GetApiUserByAccount(record.Account, apiStorage.WmType)
					if err != nil {
						log.Error("GetDateTimeReport GetApiUserByAccount err:%s", err.Error())
						continue
					} else {
						uid = apiUser.Uid
						userMap[record.Account] = uid
					}
				}
				record.Uid = uid
				record.Oid = primitive.NewObjectID()
				if err := record.AddWmBetRecord(); err != nil {
					log.Error("GetDateTimeReport record.AddWmBetRecord err:%s", err.Error())
					continue
				}
				wallet := walletStorage.QueryWallet(utils.ConvertOID(record.Uid))
				params := gameStorage.BetRecordParam{
					Uid:        record.Uid,
					GameType:   game.ApiWm,
					Income:     int64(record.WinLoss),
					BetAmount:  int64(record.BetAmount),
					CurBalance: wallet.VndBalance + wallet.SafeBalance,
					SysProfit:  0,
					BotProfit:  0,
					BetDetails: record.BetResult,
					GameId:     record.Oid.Hex(),
					GameNo:     fmt.Sprintf("%s-%s", record.Event, record.EventChild),
					GameResult: record.GameResult,
					IsSettled:  true,
				}
				gameStorage.InsertBetRecord(params)
				activityStorage.UpsertGameDataInBet(params.Uid, game.ApiWm, -1)
				activity.CalcEncouragementFunc(record.Uid)
				log.Debug("record:%v", record)
			}
		}

	}
}
