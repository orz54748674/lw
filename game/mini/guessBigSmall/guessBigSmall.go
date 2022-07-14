package guessBigSmall

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/framework/mqant/server"
	"vn/game"
	common2 "vn/game/common"
	"vn/storage/gameStorage"
	"vn/storage/gbsStorage"

	"github.com/yireyun/go-queue"
)

var Module = func() module.Module {
	this := new(guessBigSmall)
	return this
}

type guessBigSmall struct {
	basemodule.BaseModule
	room *room.Room

	botBalance int64
	coinConf   []int64
	gameConf   []gbsStorage.GameConf

	syncTableMap sync.Map
}

const (
	actionGetRecord           = "HD_getRecord"
	actionSelectBigOrSmall    = "HD_selectBigOrSmall"
	actionStart               = "HD_start"
	actionStop                = "HD_stop"
	actionUpdatePoolVal       = "HD_updatePoolVal"
	actionEnter               = "HD_enter"
	actionClose               = "HD_close"
	actionGetPoolRewardRecord = "HD_getPoolRewardRecord"
	actionGetServerTime       = "HD_getServerTime"
)

func (s *guessBigSmall) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.GuessBigSmall)
}
func (s *guessBigSmall) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}
func (s *guessBigSmall) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings,
		server.RegisterInterval(15*time.Second),
		server.RegisterTTL(30*time.Second),
	)
	s.room = room.NewRoom(s.App)
	hook := game.NewHook(s.GetType())
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetRecord, s.GetRecord)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetPoolRewardRecord, s.GetPoolRewardRecord)
	hook.RegisterAndCheckLogin(s.GetServer(), actionSelectBigOrSmall, s.UserQueue)
	hook.RegisterAndCheckLogin(s.GetServer(), actionStart, s.UserQueue)
	hook.RegisterAndCheckLogin(s.GetServer(), actionStop, s.UserQueue)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetServerTime, s.getServerTime)
	hook.RegisterAndCheckLogin(s.GetServer(), actionEnter, s.enter)
	hook.RegisterAndCheckLogin(s.GetServer(), actionClose, s.onClose)

	s.GetServer().RegisterGO("UpdatePoolVal", s.updatePoolVal)

	s.GetServer().RegisterGO("/guess/onDisconnect", s.onDisconnect)
	common.AddListener(s.GetServerID(), common.EventDisconnect, "/guess/onDisconnect")

	s.gameConf = gbsStorage.GetGameConf()
	gbsStorage.InitGbsStorage()
	go s.updatePoolTiming()
	go s.robotReward()
}

func (s *guessBigSmall) updatePoolTiming() {
	rateMap := map[int][]int{
		0: {0, 39},
		1: {40, 59},
		2: {60, 79},
		3: {80, 94},
		4: {90, 99},
	}
	for {
		sleepTime := rand.Intn(5) + 5
		time.Sleep(time.Second * time.Duration(sleepTime))

		randNum := rand.Intn(100)
		tmpIdx := 0
		for k, v := range rateMap {
			if v[0] <= randNum && v[1] >= randNum {
				tmpIdx = k
				break
			}
		}
		chip := s.gameConf[tmpIdx].Chip
		gbsStorage.UpsertPoolVal(chip, chip/1000*3)
		s.updatePoolVal(chip, chip/1000*3)
	}
}

func (s *guessBigSmall) robotReward() {
	rateMap := map[int][]int{
		0: {0, 39},
		1: {40, 59},
		2: {60, 79},
		3: {80, 94},
		4: {90, 99},
	}
	for {
		sleepTime := rand.Intn(300) + 300
		time.Sleep(time.Second * time.Duration(sleepTime))

		randNum := rand.Intn(100)
		tmpIdx := 0
		for k, v := range rateMap {
			if v[0] <= randNum && v[1] >= randNum {
				tmpIdx = k
				break
			}
		}

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		robotArr := common2.RandBotN(1, r)
		if len(robotArr) <= 0 {
			continue
		}
		robot := robotArr[0]
		s.gameConf = gbsStorage.GetGameConf()
		reward := s.gameConf[tmpIdx].PoolVal

		var record gbsStorage.GbsPoolRewardRecord
		record.CreateTime = time.Now().Unix()
		record.SelectChip = s.gameConf[tmpIdx].Chip
		record.Nickname = robot.NickName
		record.Reward = reward
		gbsStorage.InsertPoolRewardRecord(record)
		updatePoolVal := -reward + 20*record.SelectChip
		gbsStorage.UpsertPoolVal(record.SelectChip, updatePoolVal)
		s.updatePoolVal(record.SelectChip, updatePoolVal)
	}
}

func (s *guessBigSmall) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", s.GetType())
	gameStorage.UpsertGameReboot(game.Bjl, "false")
	<-closeSig
}

func (s *guessBigSmall) OnDestroy() {
	//一定别忘了继承
	s.BaseModule.OnDestroy()
}

func (s *guessBigSmall) NewTable(module module.App, tableId string) (room.BaseTable, error) {
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

func (s *guessBigSmall) enter(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()

	gameConf := gbsStorage.GetGameConf()
	var gameState gbsStorage.GameState
	table := s.room.GetTable(uid)
	if table != nil {
		gbsTable := (table.(interface{})).(*Table)
		if gbsTable.IsInGame {
			gameState = gbsTable.GameState
			gameState.CurTime = time.Now().Unix()
		}
	}

	info := struct {
		GameConf  []gbsStorage.GameConf `json:"gameConf"`
		GameState gbsStorage.GameState  `json:"gameState"`
	}{
		GameConf:  gameConf,
		GameState: gameState,
	}

	return errCode.Success(info).GetI18nMap(), nil
}

func (s *guessBigSmall) UserQueue(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	action, ok := msg["action"].(string)
	if !ok {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	uid := session.GetUserID()

	table := s.room.GetTable(uid)
	if table == nil {
		table, _ = s.room.CreateById(s.App, uid, s.NewTable)
		s.syncTableMap.Store(uid, uid)
	}

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
			queue: queue.NewQueue(6),
		}
		userQueue.Set(uid, exec)
		return exec
	}
}

func (s *guessBigSmall) updatePoolVal(selectChip, val int64) (interface{}, error) {
	s.syncTableMap.Range(func(k, v interface{}) bool {
		tmpV := v.(string)
		table := s.room.GetTable(tmpV)
		guessTable := (table.(interface{})).(*Table)
		guessTable.UpdatePoolVal(selectChip, val)
		return true
	})
	return nil, nil
}

func (s *guessBigSmall) onClose(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	table := s.room.GetTable(uid)
	if table != nil {
		gbsTable := (table.(interface{})).(*Table)
		if !gbsTable.IsInGame {
			s.room.DestroyTable(uid)
			s.syncTableMap.Delete(uid)
		}
	}
	return errCode.Success("").GetI18nMap(), nil
}

func (s *guessBigSmall) onDisconnect(uid string) (interface{}, error) {
	table := s.room.GetTable(uid)
	if table != nil {
		gbsTable := (table.(interface{})).(*Table)
		if !gbsTable.IsInGame {
			s.room.DestroyTable(uid)
			s.syncTableMap.Delete(uid)
		}
	}
	return nil, nil
}

func (s *guessBigSmall) GetRecord(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	if uid == "" {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	param1, ok1 := msg["offset"].(float64)
	offset := int(param1)
	if !ok1 {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	param2, ok2 := msg["limit"].(float64)
	if !ok2 {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	limit := int(param2)

	records := gbsStorage.GetGbsRecord(uid, offset, limit)
	return errCode.Success(records).GetI18nMap(), nil
}

func (s *guessBigSmall) getServerTime(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	if uid == "" {
		return errCode.ErrParams.GetI18nMap(), nil
	}

	info := struct {
		ServerTime int64 `json:"serverTime"`
	}{
		ServerTime: time.Now().Unix(),
	}
	return errCode.Success(info).GetI18nMap(), nil
}

func (s *guessBigSmall) GetPoolRewardRecord(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	if uid == "" {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	param1, ok1 := msg["offset"].(float64)
	offset := int(param1)
	if !ok1 {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	param2, ok2 := msg["limit"].(float64)
	if !ok2 {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	limit := int(param2)

	records := gbsStorage.GetGbsPoolRewardRecord(offset, limit)
	return errCode.Success(records).GetI18nMap(), nil
}
