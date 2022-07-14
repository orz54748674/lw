package cardLhd

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
	"vn/storage/cardStorage/cardLhdStorage"
	"vn/storage/gameStorage"
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
	return string(game.CardLhd)
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
	self.GetServer().RegisterGO("/cardLhd/onLogin", self.onLogin)
	common.AddListener(self.GetServerID(), common.EventLogin, "/cardLhd/onLogin")
	self.GetServer().RegisterGO("/cardLhd/onDisconnect", self.onDisconnect)
	common.AddListener(self.GetServerID(), common.EventDisconnect, "/cardLhd/onDisconnect")

	hook := game.NewHook(self.GetType())

	//需要队列
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.XiaZhu, self.TableQueue)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.LastXiaZhu, self.TableQueue)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.DoubleXiaZhu, self.TableQueue)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Enter, self.TableQueue)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.QuitTable, self.QuitTable)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetShortCutList, self.GetShortCutList)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.SendShortCut, self.SendShortCut)

	//直接request
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetPlayerList, self.GetPlayerList)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.GetResultsRecord, self.GetResultsRecord)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.Info, self.Info)
	hook.RegisterAndCheckLogin(self.GetServer(), protocol.CheckPlayerInGame, self.CheckPlayerInGame)
}

func (self *Room) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	gameStorage.UpsertGameReboot(game.CardLhd, "false")
	gameConf := cardLhdStorage.GetRoomConf()
	if gameConf == nil {
		gameConf = &cardLhdStorage.Conf{
			ProfitPerThousand:    20,                                                                 //系统抽水 2%
			BotProfitPerThousand: 80,                                                                 //机器人抽水 8%
			PlayerChipsList:      []int{1000, 5000, 10000, 50000, 100000, 500000, 1000000, 10000000}, //玩家筹码列表
			XiaZhuTime:           15,                                                                 //下注时间
			JieSuanTime:          9,                                                                  //结算时间
			ReadyGameTime:        3,                                                                  //摇盆时间
			KickRoomCnt:          5,                                                                  //连续三轮不下注，踢出房间
			ShortCutPrivate:      3,
			ShortCutInterval:     3,
			ShortYxbLimit:        20000,
			OddsList: map[cardLhdStorage.XiaZhuResult]int64{
				cardLhdStorage.LONG: 1,
				cardLhdStorage.HU:   1,
				cardLhdStorage.HE:   8,
			},
		}
		cardLhdStorage.InsertRoomConf(gameConf)
	}

	roomData := cardLhdStorage.GetRoomData()
	if roomData == nil || len(roomData.TablesInfo) == 0 {
		roomData := cardLhdStorage.RoomData{
			//Room: self.room,
			TablesInfo: make(map[string]cardLhdStorage.TableInfo), //map[string]yxxStorage.TableInfo{},
			CurTableID: "000000",
		}
		cardLhdStorage.InsertRoomData(&roomData)
	}
	self.CreateTable("")
	self.tablesID.Range(func(key, value interface{}) bool { //启动table队列
		self.room.GetTable(value.(string))
		return true
	})

	<-closeSig
}

func (self *Room) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
}
