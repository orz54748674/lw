package slotCs

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
	"vn/storage/slotStorage/slotCsStorage"
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
	return string(game.SlotCs)
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
	self.GetServer().RegisterGO("/slotCs/onDisconnect", self.onDisconnect)
	common.AddListener(self.GetServerID(), common.EventDisconnect, "/slotCs/onDisconnect")

	hook := game.NewHook(self.GetType())

	//需要队列
	//hook.RegisterAndCheckLogin(self.GetServer(),protocol.XiaZhu,self.TableQueue)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Enter, self.Enter)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Spin, self.Spin)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetResults, self.GetResults)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.SelectBonusSymbol, self.SelectBonusSymbol)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.EnterBonusGame, self.EnterBonusGame)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.EnterMiniGame, self.EnterMiniGame)

	hook.RegisterAndCheckLogin(self.GetServer(), protocol.SelectBonusTimes, self.SelectBonusTimes)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.SelectMiniSymbol, self.SelectMiniSymbol)

	hook.RegisterAndCheckLogin(self.GetServer(), protocol.QuitTable, self.QuitTable)
	//直接request
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetJackpot, self.GetJackpot)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Info, self.Info)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.SwitchMode, self.SwitchMode)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.CheckPlayerInGame, self.CheckPlayerInGame)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetJackpotRecord, self.GetJackpotRecord)

}

func (self *Room) Run(closeSig chan bool) {
	gameStorage.UpsertGameReboot(game.SlotCs, "false")
	log.Info("%v模块运行中...", self.GetType())
	gameConf := slotCsStorage.GetRoomConf()
	if gameConf == nil {
		gameConf = &slotCsStorage.Conf{
			InitJackpot:          InitJackpot,
			PoolScaleThousand:    InitPoolScaleThousand,
			BonusTimeOut:         12,
			ProfitPerThousand:    20,
			BotProfitPerThousand: 20,
			FreeGameMinTimes:     []int{10, 20, 30},
			BonusGameMinTimes:    []int{10, 20, 30},
		}
		slotCsStorage.InsertRoomConf(gameConf)
	}

	roomData := slotCsStorage.GetRoomData()
	if roomData == nil {
		roomData := slotCsStorage.RoomData{
			Jackpot: gameConf.InitJackpot,
		}
		slotCsStorage.InsertRoomData(&roomData)
	}
	go func() {
		//c := cron.New()
		//c.AddFunc("*/1 * * * * ?",self.OnTimer)
		//c.Start()

		c1 := cron.New()
		c1.AddFunc("*/5 * * * * ?", self.OnTimer10)
		c1.Start()
	}()
	<-closeSig
}

func (self *Room) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
}
