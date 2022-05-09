package bjl

import (
	"fmt"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/framework/mqant/server"
	"vn/game"
	"vn/storage/gameStorage"

	"github.com/yireyun/go-queue"
)

var Module = func() module.Module {
	this := new(bjlRoom)
	return this
}

type bjlRoom struct {
	basemodule.BaseModule
	room *room.Room
}

const (
	tableId           = "1"
	actionEnterRoom   = "HD_enterRoom"
	actionGetHistory  = "HD_getHistory"
	actionBet         = "HD_bet"
	actionPlayerLeave = "HD_playerLeave"
	actionGetState    = "HD_getState"
	actionGetRecord   = "HD_getRecord"
)

func (s *bjlRoom) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.Bjl)
}
func (s *bjlRoom) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}
func (s *bjlRoom) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings,
		server.RegisterInterval(15*time.Second),
		server.RegisterTTL(30*time.Second),
	)
	s.room = room.NewRoom(s.App)

	hook := game.NewHook(s.GetType())
	hook.RegisterAndCheckLogin(s.GetServer(), actionEnterRoom, s.enterRoom)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetHistory, s.getHistory)
	hook.RegisterAndCheckLogin(s.GetServer(), actionBet, s.UserQueue)
	hook.RegisterAndCheckLogin(s.GetServer(), actionPlayerLeave, s.UserQueue)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetState, s.playerGetState)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetRecord, s.UserQueue)
	hook.RegisterAndCheckLogin(s.GetServer(), protocol.SendShortCut, s.sendShortCut)

	_, err := s.room.CreateById(s.App, tableId, s.NewTable)
	if err != nil {
		log.Error(err.Error())
	}

	s.GetServer().RegisterGO("/bjl/onDisconnect", s.onDisconnect)
	common.AddListener(s.GetServerID(), common.EventDisconnect, "/bjl/onDisconnect")
}

func (s *bjlRoom) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", s.GetType())
	gameStorage.UpsertGameReboot(game.Bjl, "false")
	<-closeSig
}

func (s *bjlRoom) OnDestroy() {
	//一定别忘了继承
	s.BaseModule.OnDestroy()
}

func (s *bjlRoom) NewTable(module module.App, tableId string) (room.BaseTable, error) {
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

func (s *bjlRoom) enterRoom(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	table := s.room.GetTable(tableId)
	if table == nil {
		table, _ = s.room.CreateById(s.App, tableId, s.NewTable)
	}

	bjlTable := table.(*Table)
	bjlTable.SitDown(session)
	res := bjlTable.GetState()

	return errCode.Success(res).GetI18nMap(), nil
}

func (s *bjlRoom) playerGetState(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	table := s.room.GetTable(tableId)
	if table == nil {
		table, _ = s.room.CreateById(s.App, tableId, s.NewTable)
	}

	bjlTable := table.(*Table)
	res := bjlTable.GetState()

	return errCode.Success(res).GetI18nMap(), nil
}

func (s *bjlRoom) getHistory(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	table := s.room.GetTable(tableId)
	if table == nil {
		table, _ = s.room.CreateById(s.App, tableId, s.NewTable)
	}

	bjlTable := table.(*Table)
	res := bjlTable.GetHistory()

	return errCode.Success(res).GetI18nMap(), nil
}

func (s *bjlRoom) sendShortCut(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	table := s.room.GetTable(tableId)
	if table == nil {
		table, _ = s.room.CreateById(s.App, tableId, s.NewTable)
	}

	bjlTable := table.(*Table)
	bjlTable.SendShortCut(session, params)

	return nil, nil
}

func (s *bjlRoom) UserQueue(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	action, ok := msg["action"].(string)
	if !ok {
		return errCode.ErrParams.GetI18nMap(), nil
	}

	table := s.room.GetTable(tableId)
	if table == nil {
		table, _ = s.room.CreateById(s.App, tableId, s.NewTable)
	}

	//if table == nil {
	//	log.Info("---table not exist---")
	//	return errCode.ServerError.GetI18nMap(), nil
	//}
	//err := table.PutQueue(action, session, msg)
	//if err != nil {
	//	//log.Info("---table.PutQueue error---tableID = %s", tableID, "---error = %s", err)
	//	return errCode.ServerError.GetI18nMap(), nil
	//}
	//return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil

	uid := session.GetUserID()
	userSyncExec := getQueue(uid)
	userSyncExec.queue.Put(func() {
		_ = utils.CallReflect(table, action, session, msg)
	})
	userSyncExec.exec()
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}

var userQueue = common.NewMapRWMutex()

type userSyncExec struct {
	queue   *queue.EsQueue
	running bool
}

func (s *userSyncExec) exec() {
	if s.running {
		return
	}
	s.running = true
	ok := true
	for ok {
		val, _ok, _ := s.queue.Get()
		if _ok {
			f := val.(func())
			f()
		}
		ok = _ok
	}
	s.running = false
}

func getQueue(uid string) *userSyncExec {
	q := userQueue.Get(uid)
	if q != nil {
		return q.(*userSyncExec)
	} else {
		exec := &userSyncExec{
			queue: queue.NewQueue(1024),
		}
		userQueue.Set(uid, exec)
		return exec
	}
}

func (s *bjlRoom) onDisconnect(userID string) (interface{}, error) {
	table := s.room.GetTable(tableId)
	if table != nil {
		bjlTable := (table.(interface{})).(*Table)
		bjlTable.leaveHandle(userID)
	}
	return nil, nil
}
