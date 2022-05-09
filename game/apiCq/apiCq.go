package apiCq

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/storage/apiStorage"
	"vn/storage/userStorage"
)

var Module = func() module.Module {
	this := new(Cq)
	return this
}

type Cq struct {
	basemodule.BaseModule
	room *room.Room
}

func (self *Cq) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.ApiCq)
}
func (self *Cq) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

var (
	ActionRpcEnter      = "/apiCq/enter"
	ApiType        int8 = 2
	CqSport             = "BPUP2019"
	CqBjl               = "Bjl"
	LoginHost           = ""
)

func InitHost(env string) {
	if env != "release" {
		LoginHost = "http://api.cqgame.games"
	} else {
		LoginHost = "https://agent.x-gaming.bet/api/keno-api/"
	}
}

func (s *Cq) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings)
	apiStorage.InitApiConfig(s)
	s.GetServer().RegisterGO(ActionRpcEnter, s.enter)
}

func (s *Cq) GetGameCode(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	url := "/gameboy/game/list/cq9"
	res, err := httpGet(url)
	if err != nil {
		return errCode.ServerError.GetI18nMap(), nil
	}

	err = fmt.Errorf("%v", res.Status.Message)
	fmt.Println("res.........", res)
	return errCode.Success(res).GetI18nMap(), nil
}

func (s *Cq) enter(data map[string]interface{}) (url string, err error) {
	params := &struct {
		Token     string `json:"token"`
		GameType  string `json:"gameType"`
		Action    string `json:"action"`
		DriveType string `json:"driveType"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		log.Error("rpc awc enter err:%s", err.Error())
		return
	}
	playerToken := userStorage.QueryToken(params.Token)
	if playerToken == nil {
		err = fmt.Errorf("user not login")
		return
	}

	uid := playerToken.Oid.Hex()
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
	url, err = Login(mApiUser.Account, CqBjl, params.DriveType)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (self *Cq) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	<-closeSig
	log.Info("%v模块已停止...", self.GetType())

}

func (self *Cq) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}

func (s *Cq) onLogin(uid string) (interface{}, error) {
	log.Info("serverId: %s, uid: %s", s.GetServerID(), uid)
	return nil, nil
}

func (s *Cq) onDisconnect(uid string) (interface{}, error) {
	log.Info("onDisconnect serverId: %s, uid: %s", s.GetServerID(), uid)
	return nil, nil
}

func (s *Cq) InitApiCfg(cfg *apiStorage.ApiConfig) {
	cfg.ApiType = ApiType
	cfg.ApiTypeName = string(game.ApiCq)
	cfg.Env = s.App.GetSettings().Settings["env"].(string)
	cfg.Module = s.GetType()
	cfg.GameType = apiStorage.Sports
	cfg.GameTypeName = "Sports"
	cfg.Topic = ActionRpcEnter[1:]

	InitHost(cfg.Env)
}
