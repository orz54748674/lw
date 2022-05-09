package data

import (
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
)

var Module = func() module.Module {
	this := new(Data)
	return this
}

type Data struct {
	basemodule.BaseModule
}

func (self *Data) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "data"
}
func (self *Data) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}
func (s *Data) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings)
	s.GetServer().RegisterGO("/data/onLogin", s.onLogin)
	common.AddListener(s.GetServerID(), common.EventLogin, "/data/onLogin")
	s.GetServer().RegisterGO("/data/onDisconnect", s.onDisconnect)
	common.AddListener(s.GetServerID(), common.EventDisconnect, "/data/onDisconnect")
	//hook := game.NewHook(s.GetType())
	//hook.RegisterAndCheckLogin(s.GetServer(), "HD_hello", s.hello)
	r := &script{}
	r.start()
	Init()
}

func (s *Data) hello(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	log.Info("hello.................. %v, ip:%v", msg, session.GetIP())
	time.Sleep(500 * time.Millisecond)
	return errCode.Success(nil).GetI18nMap(), nil
}

func (self *Data) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	<-closeSig
	log.Info("%v模块已停止...", self.GetType())
}

func (self *Data) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}

func (s *Data) onLogin(uid string) (interface{}, error) {
	return nil, nil
}
func (s *Data) onDisconnect(uid string) (interface{}, error) {
	return nil, nil
}
