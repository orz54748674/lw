package sms

import (
	"vn/common/utils"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
)

var (
	actionGetSmsInfo = "HD_getSmsInfo"
	httpSmsBind      = "/sms/bind"
)

var Module = func() module.Module {
	this := new(smsGate)
	return this
}

type smsGate struct {
	basemodule.BaseModule
	ctrl *smsController
}

func (s *smsGate) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "sms_gate"
}

func (s *smsGate) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (s *smsGate) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings)

	hook := game.NewHook(s.GetType())

	hook.RegisterAndCheckLogin(s.GetServer(), actionGetSmsInfo, s.ctrl.getSmsInfo)

	registerLock := make(chan bool, 1)
	go utils.RegisterRpcToHttp(s, httpSmsBind, httpSmsBind, s.ctrl.smsBind, registerLock)

}

func (s *smsGate) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", s.GetType())
	<-closeSig
	log.Info("%v模块已停止...", s.GetType())
}

func (s *smsGate) OnDestroy() {
	//一定别忘了继承
	s.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", s.GetType())
}
