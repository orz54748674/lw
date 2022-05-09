package slotLs

import (
	"github.com/robfig/cron"
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
	"vn/storage/slotStorage/slotLsStorage"
)

var Module = func() module.Module {
	this := new(Room)
	return this
}
type Room struct {
	basemodule.BaseModule
	room *room.Room
	app module.App
	tablesID sync.Map
	curTableID string
}
func (self *Room) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.SlotLs)
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
	self.GetServer().RegisterGO("/slotLs/onDisconnect", self.onDisconnect)
	common.AddListener(self.GetServerID(),common.EventDisconnect,"/slotLs/onDisconnect")

	hook := game.NewHook(self.GetType())

	//需要队列
	//hook.RegisterAndCheckLogin(self.GetServer(),protocol.XiaZhu,self.TableQueue)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.Enter,self.Enter)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.Spin,self.Spin)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.SpinFree,self.SpinFree)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.SpinTrial,self.SpinTrial)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.SpinTrialFree,self.SpinTrialFree)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.QuitTable,self.QuitTable)
	//直接request
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.GetJackpot,self.GetJackpot)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.Info,self.Info)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.SelectFreeGame,self.SelectFreeGame)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.SelectTrialFree,self.SelectTrialFreeGame)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.SwitchMode,self.SwitchMode)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.CheckPlayerInGame,self.CheckPlayerInGame)
	hook.RegisterAndCheckLogin(self.GetServer(),protocol.GetJackpotRecord,self.GetJackpotRecord)
}

func (self *Room) Run(closeSig chan bool) {
	gameStorage.UpsertGameReboot(game.SlotLs,"false")
	log.Info("%v模块运行中...", self.GetType())
	gameConf := slotLsStorage.GetRoomConf()
	if gameConf == nil{
		gameConf = &slotLsStorage.Conf{
			InitGoldJackpot:   InitGoldJackpot,
			InitSilverJackpot: InitSilverJackpot,
			PoolScaleThousand: InitPoolScaleThousand,
			BotProfitPerThousand: 20,
			FreeGameMinTimes: 20,
		}
		slotLsStorage.InsertRoomConf(gameConf)
	}

	roomData := slotLsStorage.GetRoomData()
	if roomData == nil{
		roomData := slotLsStorage.RoomData{
			GoldJackpot: gameConf.InitGoldJackpot,
			SilverJackpot: gameConf.InitSilverJackpot,
		}
		slotLsStorage.InsertRoomData(&roomData)
	}
	go func() {
		//c := cron.New()
		//c.AddFunc("*/1 * * * * ?",self.OnTimer)
		//c.Start()

		c1 := cron.New()
		c1.AddFunc("*/5 * * * * ?",self.OnTimer10)
		c1.Start()
	}()
	<-closeSig
}

func (self *Room) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
}
