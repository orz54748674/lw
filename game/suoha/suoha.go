package suoha

import (
	"fmt"
	"math/rand"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/framework/mqant/server"
	"vn/game"
	"vn/storage/gameStorage"
	"vn/storage/suohaStorage"
	"vn/storage/walletStorage"

	"github.com/yireyun/go-queue"
)

var Module = func() module.Module {
	this := new(suohaRoom)
	return this
}

type suohaRoom struct {
	basemodule.BaseModule
	room        *room.Room
	uid2TableID map[string]string
	gameConf    suohaStorage.Conf
	tableInfos  []suohaStorage.TableInfo
}

const (
	tableId                   = "1"
	actionEnterRoom           = "HD_enterRoom"
	actionGetHistory          = "HD_getHistory"
	actionBet                 = "HD_bet"
	actionPlayerLeave         = "HD_playerLeave"
	actionGetState            = "HD_getState"
	actionGetRecord           = "HD_getRecord"
	actionGetBigRewardRanking = "HD_getBigRewardRank"
	actionReady               = "HD_ready"
	actionPlayerAction        = "HD_playerAction"
	actionCreateTable         = "HD_createTable"
	actionEnterCreateTable    = "HD_enterCreateTable"
	actionQuickStart          = "HD_quickStart"
	actionGetTableInfos       = "HD_getTableInfos"
	actionJoinTable           = "HD_joinTable"
)

func (s *suohaRoom) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.SuoHa)
}
func (s *suohaRoom) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}
func (s *suohaRoom) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings,
		server.RegisterInterval(15*time.Second),
		server.RegisterTTL(30*time.Second),
	)
	s.room = room.NewRoom(s.App)
	s.uid2TableID = make(map[string]string)

	hook := game.NewHook(s.GetType())
	hook.RegisterAndCheckLogin(s.GetServer(), actionEnterRoom, s.joinTable)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetHistory, s.getHistory)
	hook.RegisterAndCheckLogin(s.GetServer(), actionBet, s.UserQueue)
	hook.RegisterAndCheckLogin(s.GetServer(), actionPlayerLeave, s.UserQueue)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetState, s.playerGetState)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetRecord, s.UserQueue)
	hook.RegisterAndCheckLogin(s.GetServer(), protocol.SendShortCut, s.sendShortCut)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetBigRewardRanking, s.getBigRewardRanking)
	hook.RegisterAndCheckLogin(s.GetServer(), actionReady, s.UserQueue)
	hook.RegisterAndCheckLogin(s.GetServer(), actionCreateTable, s.createTable)
	hook.RegisterAndCheckLogin(s.GetServer(), actionEnterCreateTable, s.enterCreateTable)
	hook.RegisterAndCheckLogin(s.GetServer(), actionQuickStart, s.quickStart)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetTableInfos, s.getTableInfos)
	hook.RegisterAndCheckLogin(s.GetServer(), actionJoinTable, s.joinTable)

	s.GetServer().RegisterGO("/suoha/onDisconnect", s.onDisconnect)
	common.AddListener(s.GetServerID(), common.EventDisconnect, "/suoha/onDisconnect")

	s.gameConf = suohaStorage.GetGameConf()
}

func (s *suohaRoom) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", s.GetType())
	gameStorage.UpsertGameReboot(game.SuoHa, "false")
	<-closeSig
}

func (s *suohaRoom) OnDestroy() {
	//一定别忘了继承
	s.BaseModule.OnDestroy()
}

func (s *suohaRoom) NewTable(module module.App, tableId string) (room.BaseTable, error) {
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

func (s *suohaRoom) containSelectBase(base int64) bool {
	for _, v := range s.gameConf.BaseConf {
		if v.Base == base {
			return true
		}
	}
	return false
}

func (s *suohaRoom) judgeCarryGolds(base, carryGolds int64) bool {
	for _, v := range s.gameConf.BaseConf {
		if v.Base == base {
			if carryGolds >= v.MinEnter && carryGolds <= v.MaxEnter {
				return true
			}
		}
	}
	return false
}

func (s *suohaRoom) getMinEnter(base int64) int64 {
	minEnter := int64(0)
	for _, v := range s.gameConf.BaseConf {
		if v.Base == base {
			minEnter = v.MinEnter
			break
		}
	}
	return minEnter
}

func (s *suohaRoom) getMaxEnter(base int64) int64 {
	maxEnter := int64(0)
	for _, v := range s.gameConf.BaseConf {
		if v.Base == base {
			maxEnter = v.MaxEnter
			break
		}
	}
	return maxEnter
}

func (s *suohaRoom) getTableInfos(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	if uid == "" {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	tableInfos := suohaStorage.GetSuoHaTableInfo()
	s.tableInfos = tableInfos
	var tmp []suohaStorage.TableInfo
	for _, v := range tableInfos {
		if !v.IsCreateTable {
			tmp = append(tmp, v)
		}
	}

	return errCode.Success(tmp).GetI18nMap(), nil
}

func (s *suohaRoom) joinTable(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	if uid == "" {
		return errCode.ErrParams.GetI18nMap(), nil
	}

	deskId, ok := msg["deskId"].(string)
	if !ok {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	base := int64(0)
	for _, v := range s.tableInfos {
		if v.Oid.Hex() == deskId {
			base = v.BaseScore
		}
	}

	param2, ok := msg["carryGolds"].(float64)
	if !ok {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	carryGolds := int64(param2)
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	if wallet.VndBalance < carryGolds {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	if carryGolds < s.getMinEnter(base) {
		return errCode.ErrParams.GetI18nMap(), nil
	}

	for _, info := range s.tableInfos {
		if info.BaseScore == base && !info.IsCreateTable {
			table := s.room.GetTable(info.Oid.Hex())
			suohaTable := table.(*Table)
			if gameState, err := suohaTable.SitDown(session, base, carryGolds); err != nil {
				continue
			} else {
				info.CurPlayer = info.CurPlayer + 1
				suohaStorage.UpsertSuoHaTableInfo(info)
				s.uid2TableID[uid] = info.Oid.Hex()
				return errCode.Success(gameState).GetI18nMap(), nil
			}
		}
	}

	tmpOid := primitive.NewObjectID()
	if table, err := s.room.CreateById(s.App, tmpOid.Hex(), s.NewTable); err != nil {
		return errCode.ServerBusy.GetI18nMap(), nil
	} else {
		suohaTable := table.(*Table)
		if gameState, err := suohaTable.SitDown(session, base, carryGolds); err != nil {
			return errCode.ServerBusy.GetI18nMap(), nil
		} else {
			var tmpTableInfo suohaStorage.TableInfo
			tmpTableInfo.Oid = tmpOid
			tmpTableInfo.BaseScore = base
			if err = suohaStorage.InsertSuoHaTableInfo(tmpTableInfo); err != nil {
				return errCode.ServerBusy.GetI18nMap(), nil
			}
			s.uid2TableID[uid] = tmpOid.Hex()
			return errCode.Success(gameState).GetI18nMap(), nil
		}
	}
}

func (s *suohaRoom) getBigRewardRanking(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	param1, ok1 := msg["Offset"].(float64)
	offset := int(param1)
	if !ok1 {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	param2, ok2 := msg["Limit"].(float64)
	if !ok2 {
		return errCode.ErrParams.GetI18nMap(), nil

	}
	limit := int(param2)

	ranking := suohaStorage.GetSuoHaRanking(offset, limit)
	return errCode.Success(ranking).GetI18nMap(), nil
}

func (s *suohaRoom) getBaseAndCarryGolds(uid string) (int64, int64) {
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	if wallet.VndBalance < 1000 {
		return 0, 0
	}

	idx := 0
	if wallet.VndBalance > s.gameConf.BaseConf[len(s.gameConf.BaseConf)-1].MaxEnter {
		idx = len(s.gameConf.BaseConf) - 1
	} else {
		for k, v := range s.gameConf.BaseConf {
			if v.MinEnter <= wallet.VndBalance {
				idx = k - 1
			}
		}
	}
	if idx < 0 {
		idx = 0
	}

	carryGolds := s.gameConf.BaseConf[idx].MaxEnter
	if carryGolds > wallet.VndBalance {
		carryGolds = wallet.VndBalance
	}

	return s.gameConf.BaseConf[idx].Base, carryGolds
}

func (s *suohaRoom) quickStart(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	base, carryGolds := s.getBaseAndCarryGolds(uid)
	if base <= 0 {
		return errCode.BalanceNotEnough.GetI18nMap(), nil
	}

	tableInfos := suohaStorage.GetSuoHaTableInfo()
	for _, info := range tableInfos {
		if info.BaseScore == base {
			table := s.room.GetTable(info.Oid.Hex())
			if table != nil {
				suohaTable := table.(*Table)
				if gameState, err := suohaTable.SitDown(session, base, carryGolds); err != nil {
					continue
				} else {
					s.uid2TableID[uid] = info.Oid.Hex()
					return errCode.Success(gameState).GetI18nMap(), nil
				}
			}
		}
	}

	tmpOid := primitive.NewObjectID()
	if table, err := s.room.CreateById(s.App, tmpOid.Hex(), s.NewTable); err != nil {
		return errCode.ServerBusy.GetI18nMap(), nil
	} else {
		suohaTable := table.(*Table)
		if gameState, err := suohaTable.SitDown(session, base, carryGolds); err != nil {
			return errCode.ServerBusy.GetI18nMap(), nil
		} else {
			var tmpTableInfo suohaStorage.TableInfo
			tmpTableInfo.Oid = tmpOid
			tmpTableInfo.BaseScore = base
			if err = suohaStorage.InsertSuoHaTableInfo(tmpTableInfo); err != nil {
				return errCode.ServerBusy.GetI18nMap(), nil
			}
			s.uid2TableID[uid] = tmpOid.Hex()
			return errCode.Success(gameState).GetI18nMap(), nil
		}
	}
}

func (s *suohaRoom) createTable(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	if uid == "" {
		return errCode.ErrParams.GetI18nMap(), nil
	}

	param1, ok := msg["base"].(float64)
	if !ok {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	base := int64(param1)
	bContain := false
	carryGolds := int64(0)
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	for _, v := range s.gameConf.BaseConf {
		if v.Base == base {
			bContain = true
			carryGolds = v.MaxEnter
			if carryGolds > wallet.VndBalance {
				carryGolds = wallet.VndBalance
			}
			break
		}
	}
	if !bContain {
		return errCode.ErrParams.GetI18nMap(), nil
	}

	tmpBase, _ := s.getBaseAndCarryGolds(uid)
	if base > tmpBase {
		return errCode.Illegal.GetI18nMap(), nil
	}

	param2, ok := msg["password"].(float64)
	if !ok {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	password := int(param2)
	if password < 100000 || password > 999999 {
		return errCode.ErrParams.GetI18nMap(), nil
	}

	tmpOid := primitive.NewObjectID()
	if table, err := s.room.CreateById(s.App, tmpOid.Hex(), s.NewTable); err != nil {
		return errCode.ServerBusy.GetI18nMap(), nil
	} else {
		suohaTable := table.(*Table)
		if gameState, err := suohaTable.SitDown(session, base, carryGolds); err != nil {
			return errCode.ServerBusy.GetI18nMap(), nil
		} else {
			var tmpTableInfo suohaStorage.TableInfo
			tmpTableInfo.Oid = tmpOid
			tmpTableInfo.BaseScore = base
			tmpTableInfo.IsCreateTable = true
			tmpTableInfo.Password = password
			tmpTableInfo.TableNo = int(utils.RandInt64(100000, 999999, rand.New(rand.NewSource(time.Now().UnixNano()))))
			if err = suohaStorage.InsertSuoHaTableInfo(tmpTableInfo); err != nil {
				return errCode.ServerBusy.GetI18nMap(), nil
			}
			s.uid2TableID[uid] = tmpOid.Hex()
			return errCode.Success(gameState).GetI18nMap(), nil
		}
	}
}

//进入已创建房间
func (s *suohaRoom) enterCreateTable(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()

	param1, ok := msg["TableNo"].(float64)
	if !ok {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	tableNo := int(param1)

	param2, ok := msg["Password"].(float64)
	if !ok {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	password := int(param2)

	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	tableInfos := suohaStorage.GetSuoHaTableInfo()
	for _, info := range tableInfos {
		if info.IsCreateTable && tableNo == info.TableNo && password == info.Password {
			if wallet.VndBalance < s.getMinEnter(info.BaseScore) {
				return errCode.BalanceNotEnough.GetI18nMap(), nil
			}

			carryGolds := s.getMaxEnter(info.BaseScore)
			if carryGolds > wallet.VndBalance {
				carryGolds = wallet.VndBalance
			}

			table := s.room.GetTable(info.Oid.Hex())
			suohaTable := table.(*Table)
			if gameState, err := suohaTable.SitDown(session, info.BaseScore, carryGolds); err != nil {
				return errCode.NotAvailableSeat.GetI18nMap(), nil
			} else {
				s.uid2TableID[uid] = info.Oid.Hex()
				return errCode.Success(gameState).GetI18nMap(), nil
			}
		}
	}

	return errCode.ErrParams.GetI18nMap(), nil
}

func (s *suohaRoom) playerGetState(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	table := s.room.GetTable(tableId)
	if table == nil {
		table, _ = s.room.CreateById(s.App, tableId, s.NewTable)
	}

	suohaTable := table.(*Table)
	res := suohaTable.GetState()

	return errCode.Success(res).GetI18nMap(), nil
}

func (s *suohaRoom) getHistory(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	table := s.room.GetTable(tableId)
	if table == nil {
		table, _ = s.room.CreateById(s.App, tableId, s.NewTable)
	}

	suohaTable := table.(*Table)
	res := suohaTable.GetHistory()

	return errCode.Success(res).GetI18nMap(), nil
}

func (s *suohaRoom) sendShortCut(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	table := s.room.GetTable(tableId)
	if table == nil {
		table, _ = s.room.CreateById(s.App, tableId, s.NewTable)
	}

	suohaTable := table.(*Table)
	suohaTable.SendShortCut(session, params)

	return nil, nil
}

func (s *suohaRoom) UserQueue(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	action, ok := msg["action"].(string)
	if !ok {
		return errCode.ErrParams.GetI18nMap(), nil
	}

	uid := session.GetUserID()
	tableID := s.uid2TableID[uid]

	table := s.room.GetTable(tableID)
	if table == nil {
		return errCode.Illegal.GetI18nMap(), nil
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
			queue: queue.NewQueue(1024),
		}
		userQueue.Set(uid, exec)
		return exec
	}
}

func (s *suohaRoom) onDisconnect(userID string) (interface{}, error) {
	table := s.room.GetTable(tableId)
	if table != nil {
		suohaTable := (table.(interface{})).(*Table)
		suohaTable.leaveHandle(userID)
	}
	return nil, nil
}
