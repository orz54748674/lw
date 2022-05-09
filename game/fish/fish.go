package fish

import (
	"fmt"
	"strconv"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/framework/mqant/server"
	"vn/game"
	"vn/storage/fishStorage"
)

var Module = func() module.Module {
	this := new(fishRoom)
	return this
}

type fishRoom struct {
	basemodule.BaseModule
	room        *room.Room
	tableIdArr  []string
	roomConf    fishStorage.GameConf
	uid2TableID map[string]string
	CurDate string
}

const (
	actionEnterRoom    = "HD_enterRoom"
	actionPlayerFire   = "HD_playerFire"
	actionKillFish     = "HD_killFish"
	actionChangeCannon = "HD_changeCannon"
	actionPlayerLeave  = "HD_playerLeave"
	actionSpecialKillFish = "HD_specialKillFish"
	actionChangeSeat = "HD_changeSeat"
)

func (s *fishRoom) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.Fish)
}
func (s *fishRoom) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (s *fishRoom) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings,
		server.RegisterInterval(15*time.Second),
		server.RegisterTTL(30*time.Second),
	)
	s.room = room.NewRoom(s.App)

	hook := game.NewHook(s.GetType())
	hook.RegisterAndCheckLogin(s.GetServer(),actionEnterRoom, s.enterRoom)
	hook.RegisterAndCheckLogin(s.GetServer(),actionPlayerFire, s.TableQueue)
	hook.RegisterAndCheckLogin(s.GetServer(),actionKillFish, s.TableQueue)
	hook.RegisterAndCheckLogin(s.GetServer(),actionChangeCannon, s.TableQueue)
	hook.RegisterAndCheckLogin(s.GetServer(),actionPlayerLeave, s.TableQueue)
	hook.RegisterAndCheckLogin(s.GetServer(),actionSpecialKillFish, s.TableQueue)
	hook.RegisterAndCheckLogin(s.GetServer(),actionChangeSeat, s.TableQueue)

	s.GetServer().RegisterGO("/fish/onDisconnect", s.onDisconnect)
	common.AddListener(s.GetServerID(), common.EventDisconnect, "/fish/onDisconnect")

	s.roomConf = fishStorage.GameConf{
		TableTypeList: []int{1, 2, 3},
		EnterLimit: map[int]int{
			1: 100,
			2: 500,
			3: 1000,
		},
	}
	s.CurDate = time.Now().Format("2006-01-02")

	s.uid2TableID = make(map[string]string)

	fishStorage.RemoveRoomData()

	fishStorage.GetFishSysBalance(0)
	go func() {
		for {
			time.Sleep(time.Second)
			nowDate := time.Now().Format("2006-01-02")
			if nowDate != s.CurDate {
				fishStorage.GetFishSysBalance(0)
			}
		}
	}()
}

//生成桌子id
func (s *fishRoom) GenerateTableID() string {
	var tableID string
	for true {
		tableID = strconv.Itoa(int(time.Now().Unix())) + strconv.Itoa(int(RandInt64(1, 10000)))
		if s.room.GetTable(tableID) == nil {
			break
		}
	}

	return tableID
}

func (s *fishRoom) enterRoom(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	param1, ok := msg["tableType"].(string)
	if !ok {
		return errCode.Illegal.GetI18nMap(), nil
	}
	tableType, _ := strconv.Atoi(param1)
	if tableType < 1 || tableType > 3 {
		return errCode.Illegal.GetI18nMap(), nil
	}

	userID := session.GetUserID()
	if userID == "" {
		return errCode.Illegal.GetI18nMap(), nil
	}

	//分配桌子（先找已存在能进入的桌子，没有创建桌子）
	tableInfos := fishStorage.GetTablesInfo()
	if tableInfos == nil {
		tableInfos = map[string]fishStorage.TableInfo{}
	} else {
		for _, v := range tableInfos {
			table := s.room.GetTable(v.TableID)
			if tableType == v.TableType && table != nil {
				fishTable := (table.(interface{})).(*Table)
				if fishTable.AllowJoin() {
					fishTable.SitDown(session, tableType)
					res := fishTable.GetState()
					s.uid2TableID[userID] = v.TableID
					return errCode.Success(res).GetI18nMap(), nil
				}
			}
		}
	}

	tableId := s.GenerateTableID()

	var tableInfo fishStorage.TableInfo
	tableInfo.TableID = tableId
	tableInfo.ServerID = s.GetServerID()
	tableInfo.TableType = tableType
	tableInfos[tableId] = tableInfo
	fishStorage.UpsertTablesInfo(tableInfos)

	table, _ := s.room.CreateById(s.App, tableId, s.NewTable)
	fishTable := (table.(interface{})).(*Table)
	fishTable.SitDown(session, tableType)
	res := fishTable.GetState()
	s.uid2TableID[userID] = tableId
	return errCode.Success(res).GetI18nMap(), nil
}

func (s *fishRoom) NewTable(module module.App, tableId string) (room.BaseTable, error) {
	table := NewTable(
		s,
		module,
		tableId,
		room.TableId(tableId),
		room.Router(func(TableId string) string {
			return fmt.Sprintf("%v://%v/%v", s.GetType(), s.GetServerID(), tableId)
		}),
		room.Capaciity(2048),
		room.DestroyCallbacks(func(table room.BaseTable) error {
			log.Info("回收了房间: %v", table.TableId())
			_ = s.room.DestroyTable(table.TableId())
			return nil
		}),
	)
	return table, nil
}

func (self *fishRoom) TableQueue(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	userID := session.GetUserID()
	tableID := self.uid2TableID[userID]
	action, ok := msg["action"].(string)
	if !ok {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	table := self.room.GetTable(tableID)

	if table == nil {
		log.Info("---table not exist---")
		return errCode.ServerError.GetI18nMap(), nil
	}
	err := table.PutQueue(action, session, msg)
	if err != nil {
		log.Info("---table.PutQueue error---tableID = %s", tableID, "---error = %s", err)
		return errCode.ServerError.GetI18nMap(), nil
	}
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}

func (s *fishRoom) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", s.GetType())
	<-closeSig
}

func (s *fishRoom) onDisconnect(userID string) (interface{}, error) {
	tableID := s.uid2TableID[userID]
	table := s.room.GetTable(tableID)
	if table != nil {
		fishTable := (table.(interface{})).(*Table)
		fishTable.PlayerDisconnect(userID)
	}
	delete(s.uid2TableID, userID)
	return nil, nil
}

func (s *fishRoom) OnDestroy() {
	//一定别忘了继承
	s.BaseModule.OnDestroy()
}
