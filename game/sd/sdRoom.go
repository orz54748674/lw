package sd

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
)

var Module = func() module.Module {
	this := new(Room)
	return this
}

type Room struct {
	basemodule.BaseModule
	room     *room.Room
	app      module.App
	tablesID sync.Map
}

func (self *Room) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.SeDie)
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
	self.GetServer().RegisterGO("/sd/onLogin", self.onLogin)
	common.AddListener(self.GetServerID(), common.EventLogin, "/sd/onLogin")
	self.GetServer().RegisterGO("/sd/onDisconnect", self.onDisconnect)
	common.AddListener(self.GetServerID(), common.EventDisconnect, "/sd/onDisconnect")

	hook := game.NewHook(self.GetType())

	//需要队列
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.XiaZhu, self.TableQueue)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.LastXiaZhu, self.TableQueue)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.DoubleXiaZhu, self.TableQueue)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Enter, self.Enter)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.QuitTable, self.QuitTable)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetShortCutList, self.GetShortCutList)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.SendShortCut, self.SendShortCut)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetEnterData, self.GetEnterData)

	//直接request
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetPlayerList, self.GetPlayerList)

	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetTableList, self.GetTableList)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetBaseScoreList, self.GetBaseScoreList)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.CreateTableReq, self.CreateTableReq)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetWinLoseRank, self.GetWinLoseRank)
	//hook.RegisterAndCheckLogin(self.GetServer(),protocol.Info,self.Info)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.CheckPlayerInGame, self.CheckPlayerInGame)
}

func (self *Room) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	self.RoomInit()
	<-closeSig
}

func (self *Room) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
}
