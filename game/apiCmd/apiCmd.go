package apiCmd

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"vn/common/utils"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/storage/apiCmdStorage"
	"vn/storage/apiStorage"
	"vn/storage/userStorage"
)

var Module = func() module.Module {
	this := new(Cmd)
	return this
}

type Cmd struct {
	basemodule.BaseModule
	room *room.Room
}

func (self *Cmd) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.ApiCmd)
}

func (self *Cmd) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

var (
	ActionEnter         = "HD_enter"
	ActionRpcEnter      = "/apiCmd/enter"
	ApiType        int8 = 3
	LoginUrl       map[string]string
	ApiUrl         string
	PartnerKey     string
	Lang           string
)

func (s *Cmd) InitConf(env string) {
	conf, _ := apiCmdStorage.GetApiCmdConf()
	if env != "release" {
		LoginUrl["0"] = "https://luckymobile.1win888.net/"
		LoginUrl["1"] = "https://lucky.1win888.net/"
		ApiUrl = "http://api.1win888.net/"
		PartnerKey = conf.PartnerKey
		Lang = "zh-CN"
	} else {
		LoginUrl["0"] = "https://luckymobile.fts368.com/"
		LoginUrl["1"] = "https://lucky.fts368.com/"
		ApiUrl = "http://api.fts368.com/"
		PartnerKey = conf.PartnerKey
		Lang = "vi-VN"
	}
}

func (s *Cmd) GetLoginUrlByDeviceType(strType string) string {
	//"0"---手机端，"1"---pc端
	if strType == "0" || strType == "1" {
		return LoginUrl[strType]
	}
	return LoginUrl["0"]
}

func (s *Cmd) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings)
	LoginUrl = make(map[string]string)
	apiStorage.InitApiConfig(s)
	s.GetServer().RegisterGO(ActionRpcEnter, s.enter)
	apiCmdStorage.InitCmdStorage()

	go GetBetRecordTiming()

}

func (self *Cmd) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	<-closeSig
	log.Info("%v模块已停止...", self.GetType())
}

func (self *Cmd) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}

func (s *Cmd) onDisconnect(uid string) (interface{}, error) {
	log.Info("onDisconnect serverId: %s, uid: %s", s.GetServerID(), uid)
	return nil, nil
}

func (s *Cmd) enter(data map[string]interface{}) (resp string, err error) {
	params := &struct {
		Token     string `json:"token"`
		GameType  string `json:"gameType"`
		Action    string `json:"action"`
		DriveType string `json:"driveType"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		log.Error("rpc cmd enter err:%s", err.Error())
		return
	}
	token := userStorage.QueryToken(params.Token)
	if token == nil {
		err = fmt.Errorf("user not login")
		return
	}

	uid := token.Oid.Hex()
	mApiUser := &apiStorage.ApiUser{}
	err = mApiUser.GetApiUser(uid, ApiType)
	if err == mongo.ErrNoDocuments {
		user := userStorage.QueryUserId(utils.ConvertOID(uid))
		mApiUser.Account = user.Account
		mApiUser.Type = ApiType
		mApiUser.Uid = uid
		if err = mApiUser.Save(); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	url, err := Login(mApiUser.Account, s.GetLoginUrlByDeviceType(params.DriveType))
	if err != nil {
		return "", err
	}

	return url, nil
}

func (s *Cmd) InitApiCfg(cfg *apiStorage.ApiConfig) {
	cfg.ApiType = ApiType
	cfg.ApiTypeName = string(game.ApiCmd)
	cfg.Env = s.App.GetSettings().Settings["env"].(string)
	cfg.Module = s.GetType()
	cfg.GameType = apiStorage.Sports
	cfg.GameTypeName = "Sports"
	cfg.Topic = ActionRpcEnter[1:]

	s.InitConf(cfg.Env)
}
