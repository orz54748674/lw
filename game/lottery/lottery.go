package lottery

import (
	"sync"
	"time"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	gate2 "vn/gate"
	"vn/storage/lotteryStorage"
)

var (
	actionAdminReloadPaly    = "HD_adminReloadPaly"
	actionAdminReloadLottery = "HD_adminReloadLottery"

	// 退款处理
	actionAdminNumberRefund = "HD_adminNumberRefund"
	actionAdminBatchRefund  = "HD_adminBatchRefund"

	actionAdminModifyBetInfo = "HD_adminModifyBetInfo"
	actionAdminOpen          = "HD_adminOpen"
)

type Lottery struct {
	basemodule.BaseModule
	impl *Impl
	push *gate2.OnlinePush
}

var Module = func() module.Module {
	return new(Lottery)
}

func (m *Lottery) Version() string {
	return "1.0.0"
}

func (m *Lottery) GetType() string {
	return "lottery"
}

func (m *Lottery) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)
	mongoIncDataExpireDay := int64(app.GetSettings().Settings["mongoIncDataExpireDay"].(float64))
	lotteryStorage.InitLottery()
	lotteryStorage.InitLotteryRecord()
	lotteryStorage.InitLotteryPlay()
	lotteryStorage.InitPlayMap()
	lotteryStorage.InitLotteryBetRecord(mongoIncDataExpireDay)
	m.push = &gate2.OnlinePush{
		TraceSpan: log.CreateRootTrace(),
		App:       app,
	}
	m.push.OnlinePushInit(nil, 2048)
	m.impl = &Impl{
		push:                m.push,
		app:                 app,
		clientPlayRWLock:    new(sync.RWMutex),
		clientLotteryRWLock: new(sync.RWMutex),
	}
	m.impl.initPlayMap()
	m.impl.initLotteryMap()
	hook := game.NewHook(m.GetType())
	hook.RegisterAndCheckLogin(m.GetServer(), actionInfo, m.impl.info)
	hook.RegisterAndCheckLogin(m.GetServer(), actionRecord, m.impl.record)
	hook.RegisterAndCheckLogin(m.GetServer(), actionAddBet, m.impl.addBet)
	hook.RegisterAndCheckLogin(m.GetServer(), actionGetBetRecordList, m.impl.getBetRecordList)
	hook.RegisterAndCheckLogin(m.GetServer(), actionBetMsgLog, m.impl.betMsgLog)

	hook.RegisterAdminInterface(m.GetServer(), actionAdminReloadPaly, m.impl.reloadPlay)
	hook.RegisterAdminInterface(m.GetServer(), actionAdminReloadLottery, m.impl.reloadLottery)

	hook.RegisterAdminInterface(m.GetServer(), actionAdminNumberRefund, m.impl.numberRefund)
	hook.RegisterAdminInterface(m.GetServer(), actionAdminBatchRefund, m.impl.batchRefund)

	hook.RegisterAdminInterface(m.GetServer(), actionAdminModifyBetInfo, m.impl.modifyBetInfo)
	hook.RegisterAdminInterface(m.GetServer(), actionAdminOpen, m.impl.open)

	m.GetServer().RegisterGO("/lottery/noticeOpen", m.impl.noticeOpen)

}

func (m *Lottery) Run(closeSig chan bool) {
	log.Info("%v 模块运行中...", m.GetType())
	go m.push.Run(100 * time.Millisecond)
	go m.impl.broadcastTime()
	<-closeSig
	log.Info("%v 模块已停止...", m.GetType())
}

func (m *Lottery) OnDestroy() {
	//一定别忘了继承
	m.BaseModule.OnDestroy()
	log.Info("%v 模块已回收...", m.GetType())
}
