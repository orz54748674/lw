package apiXg

import (
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/storage/apiStorage"
)

var Module = func() module.Module {
	this := new(Xg)
	return this
}

type Xg struct {
	basemodule.BaseModule
	user   *cApiUser
	xgHttp *XgHttp
}

func (self *Xg) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.Xg)
}
func (self *Xg) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

var (
	actionEnter    = "HD_enter"
	actionRpcEnter = "/apiXg/enter"
	GameTypes      = map[string]string{
		"Baccarat":    "1",
		"DragonTiger": "5",
		"Roulette":    "3",
	}
)

func (self *Xg) OnInit(app module.App, settings *conf.ModuleSettings) {
	self.BaseModule.OnInit(self, app, settings)
	apiStorage.InitApiConfig(self)
	hook := game.NewHook(self.GetType())
	hook.RegisterAndCheckLogin(self.GetServer(), actionEnter, self.user.Enter)
	self.GetServer().RegisterGO(actionRpcEnter, self.user.enter)

}

func (self *Xg) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	<-closeSig
	log.Info("%v模块已停止...", self.GetType())

}

func (self *Xg) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}

func (self *Xg) InitApiCfg(cfg *apiStorage.ApiConfig) {
	cfg.ApiType = apiStorage.XgType
	cfg.ApiTypeName = string(game.Xg)
	cfg.Env = self.App.GetSettings().Settings["env"].(string)
	cfg.Module = self.GetType()
	cfg.GameType = apiStorage.Live
	cfg.GameTypeName = "Live"
	cfg.Topic = actionRpcEnter[1:]

	InitHost(cfg.Env)
	self.xgHttp.Init(cfg.Env)
	mongoIncDataExpireDay := int64(self.App.GetSettings().Settings["mongoIncDataExpireDay"].(float64))
	apiStorage.InitXgBetRecord(mongoIncDataExpireDay)
}
