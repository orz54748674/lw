package cardCddS

import (
	"sync"
	"time"
	"vn/common"
	"vn/common/protocol"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/framework/mqant/server"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/cardStorage/cardCddSStorage"
	"vn/storage/gameStorage"
)

var Module = func() module.Module {
	this := new(Room)
	return this
}

type Room struct {
	basemodule.BaseModule
	room                *room.Room
	app                 module.App
	tablesID            sync.Map
	curTableID          string
	HallInfo            map[HallType]map[int64]HallConfig
	RoomRobotConf       []cardCddSStorage.RobotConf
	HallOffsetPlayerNum map[HallType]map[int64]int
	onlinePush          *vGate.OnlinePush
}

func (self *Room) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.CardCddS)
}
func (self *Room) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}
func (self *Room) OnInit(app module.App, settings *conf.ModuleSettings) {
	self.BaseModule.OnInit(self, app, settings,
		server.RegisterInterval(15*time.Second),
		server.RegisterTTL(30*time.Second),
	)
	self.room = room.NewRoom(app)
	self.app = app
	self.onlinePush = &vGate.OnlinePush{
		TraceSpan: log.CreateRootTrace(),
		App:       app,
	}
	self.onlinePush.OnlinePushInit(nil, 128)
	//self.GetServer().RegisterGO("/slotLs/onLogin", self.onLogin)
	//common.AddListener(self.GetServerID(),common.EventLogin,"/slotLs/onLogin")
	self.GetServer().RegisterGO("/cardCddS/onDisconnect", self.onDisconnect)
	common.AddListener(self.GetServerID(), common.EventDisconnect, "/cardCddS/onDisconnect")

	hook := game.NewHook(self.GetType())

	//需要队列
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Enter, self.Enter)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.AutoEnter, self.AutoEnter)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetEnterData, self.GetEnterData)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.QuitTable, self.QuitTable)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Ready, self.Ready)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.AutoReady, self.AutoReady)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.MasterStartGame, self.MasterStartGame)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.PutPoker, self.PutPoker)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.CheckPoker, self.CheckPoker)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.SortPoker, self.SortPoker)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.HintPoker, self.HintPoker)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.InviteEnter, self.InviteEnter)
	//直接request
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Info, self.Info)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetHallInfo, self.GetHallInfo)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetWinLoseRank, self.GetWinLoseRank)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.CheckPlayerInGame, self.CheckPlayerInGame)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GameInvite, self.GameInvite)
}

func (self *Room) Run(closeSig chan bool) {
	gameStorage.UpsertGameReboot(game.CardCddS, "false")
	log.Info("%v模块运行中...", self.GetType())
	self.RoomInit()
	<-closeSig
}

func (self *Room) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
}
