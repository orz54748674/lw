package pay

import (
	"runtime"
	"time"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/game/pay/payWay"
	gate2 "vn/gate"
	"vn/storage/agentStorage"
	"vn/storage/payStorage"
)

var Module = func() module.Module {
	this := new(Pay)
	return this
}

type Pay struct {
	basemodule.BaseModule
	impl *Impl
	push *gate2.OnlinePush
}

func (self *Pay) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "pay"
}
func (self *Pay) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

var (
	actionPayInfo            = "HD_payInfo"
	actionGiftCode           = "HD_giftCode"
	actionCharge             = "HD_charge"
	actionChargeLog          = "HD_chargeLog"
	actionDouDou             = "HD_douDou"
	actionDouDouLog          = "HD_douDouLog"
	actionDouDouBtList       = "HD_douDouBtList"
	actionAdminOrder         = "HD_adminOrder"
	actionAdminAddOrder      = "HD_adminAddOrder"
	actionAdminDouDou        = "HD_adminDouDou"
	actionVGBankList         = "HD_vgBankList"
	actionAgentIncome2wallet = "HD_agentIncome2wallet"
	actionSafe2wallet        = "HD_safe2wallet"
	actionBindBtCard         = "HD_BindBtCard"
)

func (s *Pay) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings)
	s.impl = &Impl{
		App:      app,
		Settings: settings,
	}
	//self.GetServer().RegisterGO("/listener/onLogin", self.onLogin)
	//common.AddListener(self.GetServerID(),common.EventLogin,"/listener/onLogin")
	//self.GetServer().RegisterGO("/listener/onDisconnect", self.onDisconnect)
	//common.AddListener(self.GetServerID(),common.EventDisconnect,"/listener/onDisconnect")

	hook := game.NewHook(s.GetType())
	hook.RegisterAndCheckLogin(s.GetServer(), actionPayInfo, s.impl.payInfo)
	hook.RegisterAndCheckLogin(s.GetServer(), actionCharge, s.impl.charge)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGiftCode, s.giftCode)
	hook.RegisterAndCheckLogin(s.GetServer(), actionChargeLog, s.impl.chargeLog)
	hook.RegisterAndCheckLogin(s.GetServer(), actionDouDou, s.impl.douDou)
	hook.RegisterAndCheckLogin(s.GetServer(), actionDouDouLog, s.impl.doudouLog)
	hook.RegisterAndCheckLogin(s.GetServer(), actionVGBankList, s.impl.getVGBankList)
	hook.RegisterAndCheckLogin(s.GetServer(), actionAgentIncome2wallet, s.impl.agentIncome2wallet)
	hook.RegisterAndCheckLogin(s.GetServer(), actionSafe2wallet, s.impl.safe2wallet)
	hook.RegisterAndCheckLogin(s.GetServer(), actionBindBtCard, s.impl.bindBtCard)
	hook.RegisterAndCheckLogin(s.GetServer(), actionDouDouBtList, s.impl.douDouBtList)

	hook.RegisterAdminInterface(s.GetServer(), actionAdminOrder, s.impl.adminOrder)
	hook.RegisterAdminInterface(s.GetServer(), actionAdminAddOrder, s.impl.adminAddOrder)
	hook.RegisterAdminInterface(s.GetServer(), actionAdminDouDou, s.impl.adminDouDou)

	s.push = &gate2.OnlinePush{
		TraceSpan: log.CreateRootTrace(),
		App:       app,
	}
	s.push.OnlinePushInit(nil, 128)

	incDataExpireDay := time.Duration(
		app.GetSettings().Settings["mongoIncDataExpireDay"].(float64)) * 24 * time.Hour

	payStorage.InitPay(incDataExpireDay)
	payStorage.InitVGPay()
	payWay.AutoCheck()
	s.impl.initPayMethod()

	//go s.OnZeroRefreshData() //凌晨刷新数据
}
func (s *Pay) OnZeroRefreshData() {
	defer func() {
		if r := recover(); r != nil {
			buff := make([]byte, 1024)
			runtime.Stack(buff, false)
			log.Error("RefreshData panic(%v)\n info:%s", r, string(buff))
		}
	}()
	all := agentStorage.QueryAllAgentMemberData()
	agentStorage.OnUpdateAgentMemberData(all)
	for {
		now := time.Now()
		// 计算下一个零点
		next := now.Add(time.Hour * 24)
		next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
		t := time.NewTimer(next.Sub(now))
		<-t.C
		//以下为定时执行的操作

		all = agentStorage.QueryAllAgentMemberData()
		agentStorage.OnUpdateAgentMemberData(all)
	}
}
func (self *Pay) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	self.push.Run(100 * time.Millisecond)
	<-closeSig
	log.Info("%v模块已停止...", self.GetType())
}

func (self *Pay) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}

//func (s *Pay)onLogin(uid string)(interface{}, error)  {
//	return nil, nil
//}
//func (s *Pay)onDisconnect(uid string)(interface{}, error)  {
//	return nil, nil
//}
