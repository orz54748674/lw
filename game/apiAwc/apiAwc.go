package apiAwc

import (
	"fmt"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/storage/apiStorage"
	"vn/storage/userStorage"

	"github.com/mitchellh/mapstructure"
)

type Awc struct {
	basemodule.BaseModule
}

var (
	actionEnter         = "HD_enter"
	actionRpcEnter      = "/apiAwc/enter"
	apiType        int8 = 3
)

var Module = func() module.Module {
	this := new(Awc)
	return this
}

func (self *Awc) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "apiAwc"
}

func (self *Awc) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (self *Awc) OnInit(app module.App, settings *conf.ModuleSettings) {
	self.BaseModule.OnInit(self, app, settings)
	apiStorage.InitApiConfig(self)
	hook := game.NewHook(self.GetType())
	hook.RegisterAndCheckLogin(self.GetServer(), actionEnter, self.Enter)
	self.GetServer().RegisterGO(actionRpcEnter, self.enter)
}

func (self *Awc) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	<-closeSig
	log.Info("%v模块已停止...", self.GetType())
}

func (self *Awc) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}

func (self *Awc) Enter(session gate.Session, data map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	mApiUser := &apiStorage.ApiUser{}
	err := mApiUser.GetApiUser(uid, apiType)
	if err == mongo.ErrNoDocuments {
		user := userStorage.QueryUserId(utils.ConvertOID(uid))
		if eCode, err := CreateUser(user.Account); err != nil {
			return errCode.ApiCreateUserErr.GetI18nMap(), err
		} else if eCode == UserExists || eCode == SuccessCode {
			mApiUser.Account = user.Account
			mApiUser.Type = apiType
			mApiUser.Uid = uid
			if err = mApiUser.Save(); err != nil {
				return errCode.ServerError.GetI18nMap(), err
			}
		} else {
			return errCode.ApiErr.GetI18nMap(), nil
		}
	} else if err != nil {
		return errCode.ServerError.GetI18nMap(), err
	}
	loginInfo, err := DoLoginAndLaunchGame(mApiUser.Account)
	if err != nil {
		log.Error("ApiLoginErr err:%s", err.Error())
		return errCode.ApiLoginErr.GetI18nMap(), err
	}
	url, ok := loginInfo["url"]
	if !ok {
		log.Error("ApiLogin LoginUrl not find")
		return errCode.ApiLoginErr.GetI18nMap(), err
	}
	respData := map[string]interface{}{
		"LoginUrl": url,
		"Token":    "",
	}
	return errCode.Success(respData).GetI18nMap(), nil
}

func (self *Awc) enter(data map[string]interface{}) (resp string, err error) {
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
	eCode, err := CreateUser(user.Account)
	if err != nil {
		return "", err
	} else if eCode == UserExists || eCode == SuccessCode {
		err = mApiUser.GetApiUser(uid, apiType)
		if err == mongo.ErrNoDocuments {
			mApiUser.Account = user.Account
			mApiUser.Type = apiType
			mApiUser.Uid = uid
			if err = mApiUser.Save(); err != nil {
				return "", err
			}
		} else if err != nil {
			return "", err
		}
		loginInfo, err := DoLoginAndLaunchGame(mApiUser.Account)
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
	return "", fmt.Errorf("CreateUser errCode:%s", eCode)
}

func (self *Awc) InitApiCfg(cfg *apiStorage.ApiConfig) {
	cfg.ApiType = apiStorage.AwcType
	cfg.ApiTypeName = string(game.ApiAwc)
	cfg.Env = self.App.GetSettings().Settings["env"].(string)
	cfg.Module = self.GetType()
	cfg.GameType = apiStorage.Live
	cfg.GameTypeName = "Live"
	cfg.Topic = actionRpcEnter[1:]

	InitEnv(cfg.Env)
	apiStorage.InitAwcCancelBetRecord()
	mongoIncDataExpireDay := int64(self.App.GetSettings().Settings["mongoIncDataExpireDay"].(float64))
	apiStorage.InitAwcGiveRecord(mongoIncDataExpireDay)
	apiStorage.InitAwcBetRecord(mongoIncDataExpireDay)
}
