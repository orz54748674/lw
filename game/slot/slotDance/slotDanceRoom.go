package slotDance

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
	"vn/storage/gameStorage"
	"vn/storage/slotStorage/slotDanceStorage"
)

var Module = func() module.Module {
	this := new(Room)
	return this
}

type Room struct {
	basemodule.BaseModule
	room       *room.Room
	app        module.App
	tablesID   sync.Map
	curTableID string
}

func (self *Room) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.SlotDance)
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
	//self.GetServer().RegisterGO("/slotLs/onLogin", self.onLogin)
	//common.AddListener(self.GetServerID(),common.EventLogin,"/slotLs/onLogin")
	self.GetServer().RegisterGO("/slotDance/onDisconnect", self.onDisconnect)
	common.AddListener(self.GetServerID(), common.EventDisconnect, "/slotDance/onDisconnect")

	hook := game.NewHook(self.GetType())

	//需要队列
	//hook.RegisterAndCheckLogin(self.GetServer(),protocol.XiaZhu,self.TableQueue)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Enter, self.Enter)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Spin, self.Spin)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.QuitTable, self.QuitTable)
	//直接request
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Info, self.Info)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.SwitchMode, self.SwitchMode)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.CheckPlayerInGame, self.CheckPlayerInGame)
}

func (self *Room) Run(closeSig chan bool) {
	gameStorage.UpsertGameReboot(game.SlotDance, "false")
	log.Info("%v模块运行中...", self.GetType())
	gameConf := slotDanceStorage.GetRoomConf()
	if gameConf == nil {
		gameConf = &slotDanceStorage.Conf{
			BotProfitPerThousand: 20,
			FreeGameMinTimes:     20,
		}
		slotDanceStorage.InsertRoomConf(gameConf)
	}

	roomData := slotDanceStorage.GetRoomData()
	if roomData == nil {
		roomData := slotDanceStorage.RoomData{}
		slotDanceStorage.InsertRoomData(&roomData)
	}
	<-closeSig
}

func (self *Room) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
}
