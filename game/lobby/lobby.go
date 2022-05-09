package lobby

import (
	"time"
	"vn/common"
	"vn/common/protocol"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/game/lobby/lobbyImpl"
	gate2 "vn/gate"
	"vn/storage"
	"vn/storage/agentStorage"
	"vn/storage/botStorage"
	"vn/storage/chatStorage"
	"vn/storage/gameStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/slotStorage"
	"vn/storage/walletStorage"
)

var Module = func() module.Module {
	this := new(Lobby)
	return this
}

type Lobby struct {
	basemodule.BaseModule
	push *gate2.OnlinePush
}

func (self *Lobby) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.Lobby)
}
func (self *Lobby) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

const (
	actionLogin           = "HD_login"
	actionGetUserInfo           = "HD_GetUserInfo"
	actionInfo            = "HD_info"
	actionPage            = "HD_page"
	actionAgentInfo       = "HD_agentInfo"
	actionBindPhone       = "HD_bindPhone"
	actionSetNickName     = "HD_SetNickName"
	actionSetAvatar       = "HD_SetAvatar"
	actionModifyPassword  = "HD_ModifyPassword"
	actionAgentConf       = "HD_agentConf"
	actionQueryBetRecord  = "HD_QueryBetRecord"
	actionAdminMailSend   = "HD_adminMailSend"
	actionGetMailList     = "HD_GetMailList"
	actionUpdateMailState = "HD_UpdateMailState"
	actionDeleteMail      = "HD_DeleteMail"
	actionWallet          = "HD_Wallet"
	actionNotice          = "HD_Notice"
	actionPrizePool       = "HD_prizePool"
	actionApiConf         = "HD_apiConf"
	actionGetWindowVnd    = "HD_GetWindowVnd"
	actionGetRankList     = "HD_GetWinRankList"
	actionSetSafeStatus     = "HD_SetSafeStatus"
	actionGetMaxJackpotAll     = "HD_GetMaxJackpotAll"
)

func (s *Lobby) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings)
	impl := &Impl{
		app:      app,
		settings: settings,
	}
	//self.GetServer().RegisterGO("/say/hi", self.say) //handler
	//hdLogin := &game.Hook{Fun:impl.HdLogin}
	//self.GetServer().RegisterGO("HD_login", hdLogin.NoLoginHook)
	hook := game.NewHook(s.GetType())
	hook.RegisterAndNoLogin(s.GetServer(), actionLogin, impl.Login)
	hook.RegisterAndNoLogin(s.GetServer(), actionGetUserInfo, impl.GetUserInfo)
	hook.RegisterAndCheckLogin(s.GetServer(), actionInfo, impl.Info)
	hook.RegisterAndCheckLogin(s.GetServer(), actionPage, impl.Page)
	hook.RegisterAndCheckLogin(s.GetServer(), actionPrizePool, impl.PrizePool)
	hook.RegisterAndCheckLogin(s.GetServer(), actionAgentInfo, impl.AgentInfo)
	hook.RegisterAndCheckLogin(s.GetServer(), actionBindPhone, impl.BindPhone)
	hook.RegisterAndCheckLogin(s.GetServer(), "HD_test", impl.TestPush)
	hook.RegisterAndCheckLogin(s.GetServer(), actionSetNickName, impl.SetNickName)
	hook.RegisterAndCheckLogin(s.GetServer(), actionSetAvatar, impl.SetAvatar)
	hook.RegisterAndCheckLogin(s.GetServer(), actionModifyPassword, impl.ModifyPassword)
	hook.RegisterAndCheckLogin(s.GetServer(), actionAgentConf, impl.agentConf)
	hook.RegisterAndCheckLogin(s.GetServer(), actionQueryBetRecord, impl.QueryBetRecord)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetMailList, impl.GetMailList)
	hook.RegisterAndCheckLogin(s.GetServer(), actionUpdateMailState, s.UpdateMailState)
	hook.RegisterAndCheckLogin(s.GetServer(), actionDeleteMail, impl.DeleteMail)
	hook.RegisterAndCheckLogin(s.GetServer(), actionWallet, impl.wallet)
	hook.RegisterAndCheckLogin(s.GetServer(), actionApiConf, impl.ApiConf)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetWindowVnd, impl.GetWindowVnd)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetRankList, impl.GetRankList)
	hook.RegisterAndCheckLogin(s.GetServer(), protocol.GameInviteRecord, impl.GameInviteRecord)
	hook.RegisterAndCheckLogin(s.GetServer(), actionSetSafeStatus, impl.SetSafeStatus)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetMaxJackpotAll, impl.GetMaxJackpotAll)

	hook.RegisterAdminInterface(s.GetServer(), actionNotice, s.AdminNotice)
	hook.RegisterAdminInterface(s.GetServer(), actionAdminMailSend, s.AdminMailSend)

	s.GetServer().RegisterGO("/lobby/onDisconnect", s.onDisconnect)
	common.AddListener(s.GetServerID(), common.EventDisconnect, "/lobby/onDisconnect")
	//s.GetServer().RegisterGO("HD_login", impl.Login)
	//s.GetServer().RegisterGO("HD_info", impl.Info)
	//s.GetServer().RegisterGO("HD_test", impl.TestPush)
	s.push = &gate2.OnlinePush{
		TraceSpan: log.CreateRootTrace(),
		App:       app,
	}
	s.push.OnlinePushInit(nil, 128)

	game.InitShortCut()
	incDataExpireDay := time.Duration(
		app.GetSettings().Settings["mongoIncDataExpireDay"].(float64)) * 24 * time.Hour
	botStorage.InitBot()
	chatStorage.InitChatBotMsgList()
	gameStorage.InitGameProfit()
	gameStorage.InitGameProfitLog(incDataExpireDay)
	gameStorage.InitGameProfitByUser()
	gameStorage.InitGameProfitLogByUser(incDataExpireDay)
	storage.InitGlobal()
	walletStorage.Init(incDataExpireDay)
	agentStorage.InitAgent(incDataExpireDay)
	storage.InitCustomerConf()
	smsCodeExpireSecond := time.Duration(app.GetSettings().Settings["smsCodeExpireSecond"].(float64))
	lobbyStorage.InitSms(smsCodeExpireSecond * time.Second)
	gameStorage.InitBetRecord(incDataExpireDay)
	gameStorage.InitMail()
	gameStorage.InitMailRecord(incDataExpireDay)
	//gameStorage.InitActivityRecord(incDataExpireDay)
	gameStorage.InitChannel()
	gameStorage.InitGameOverview()
	gameStorage.InitUserOverview()
	gameStorage.InitGameReboot()
	lobbyStorage.InitNotice()

	gameStorage.InitGameReconnect()
	gameStorage.RemoveAllReconnect()

	slotStorage.InitJackpotRecord(30 * 24 * time.Hour)
	lobbyStorage.InitLobbyGameLayout()

	gameStorage.InitGameCommon()
	gameStorage.RemoveAllGameCommonData()
	initPage()

	lobbyStorage.InitLobbyBubble()
	gameStorage.UpsertGameReboot(game.All, "false")

	gameStorage.InitGameWinLoseRecord()
	gameStorage.InitGameInviteRecord()
	gameStorage.RemoveAllGameInviteRecord()
}

func (s *Lobby) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", s.GetType())
	s.push.Run(100 * time.Millisecond)
	go func() {
		broad := &lobbyImpl.Broadcast{
			App:      s.App,
			Settings: s.GetModuleSettings(),
		}
		broad.Init()
		broad.Run()
	}()
	<-closeSig
	log.Info("%v模块已停止...", s.GetType())
}

func (self *Lobby) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}
func (s *Lobby) onDisconnect(uid string) (interface{}, error) {
	OnUserPageOffline(uid)
	return nil, nil
}
