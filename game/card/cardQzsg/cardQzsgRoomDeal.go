package cardQzsg

import (
	"encoding/json"
	"fmt"
	"github.com/robfig/cron"
	"math/rand"
	"sort"
	"strconv"
	"time"
	common2 "vn/common"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/cardStorage/cardQzsgStorage"
	"vn/storage/gameStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func (self *Room) RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return r.Int63n(max-min) + min
}

//生成桌子id
func (self *Room) GenerateTableID(tail string) string {
	//rand.Seed(time.Now().UnixNano())
	var tableID int
	for true {
		tableID = int(self.RandInt64(1, 1000000))
		if tableID < 100000 {
			tableID = tableID + 100000
		}
		ok := true
		if self.room.GetTable(strconv.Itoa(tableID)+"_"+tail) != nil {
			ok = false
			break
		}

		if ok {
			break
		}
	}
	return strconv.Itoa(tableID)
}
func (self *Room) CreateTable(tableID string) (table room.BaseTable, err error, id string) {
	tableHead := self.GenerateTableID(tableID) //服务器生成桌子id
	tableID = tableHead + "_" + tableID
	table, err = self.room.CreateById(self.App, tableID, self.NewTable)
	if err != nil {
		return nil, err, ""
	}
	self.tablesID.Store(tableID, tableID)
	return table, nil, tableID
}
func (self *Room) DestroyTable(tableID string) {
	self.room.DestroyTable(tableID)
	self.tablesID.Delete(tableID)
}
func (self *Room) NewTable(module module.App, tableId string) (room.BaseTable, error) {
	table := NewTable(
		self,
		module,
		tableId,
		room.TableId(tableId),
		room.Router(func(TableId string) string {
			return fmt.Sprintf("%v://%v/%v", self.GetType(), self.GetServerID(), tableId)
		}),
		room.Capaciity(2048),
		room.DestroyCallbacks(func(table room.BaseTable) error {
			log.Info("回收了房间: %v", table.TableId())
			_ = self.room.DestroyTable(table.TableId())
			return nil
		}),
	)
	return table, nil
}
func (self *Room) RoomRobotInit(baseScore int64) {
	baseRobotNum := 40
	if baseScore == 500 {
		baseRobotNum = 60
	} else if baseScore == 1000 {
		baseRobotNum = 100
	} else if baseScore == 2000 {
		baseRobotNum = 120
	} else if baseScore == 5000 {
		baseRobotNum = 140
	} else if baseScore == 10000 {
		baseRobotNum = 100
	} else if baseScore == 50000 {
		baseRobotNum = 35
	} else if baseScore == 100000 {
		baseRobotNum = 10
	} else {
		baseRobotNum = 4
	}
	conf := &cardQzsgStorage.RobotConf{
		HallType:     string(All),
		BaseScore:    baseScore,
		BaseRobotNum: baseRobotNum,
		MaxOffset:    baseRobotNum * 20 / 100,
	}
	cardQzsgStorage.InsertRoomRobotConf(conf)
}
func (self *Room) OnTimer60() {
	self.RoomRobotConf = cardQzsgStorage.GetRoomRobotConf()
}
func (self *Room) OnTimer5() {
	for _, v := range BaseScoreList {
		for _, v1 := range self.RoomRobotConf {
			if v1.HallType == string(All) && v1.BaseScore == v {
				offset := int(self.RandInt64(1, int64(v1.MaxOffset)/2))
				rand := self.RandInt64(1, 3)
				if rand == 1 {
					if self.HallOffsetPlayerNum[All][v]+offset > v1.MaxOffset {
						self.HallOffsetPlayerNum[All][v] -= offset
					} else {
						self.HallOffsetPlayerNum[All][v] += offset
					}
				} else {
					if self.HallOffsetPlayerNum[All][v]-offset < -v1.MaxOffset {
						self.HallOffsetPlayerNum[All][v] += offset
					} else {
						self.HallOffsetPlayerNum[All][v] -= offset
					}
				}
			}
		}
	}
}
func (self *Room) RoomInit() {
	gameStorage.UpsertGameReboot(game.CardQzsg, "false")

	gameConf := cardQzsgStorage.GetRoomConf()
	if gameConf == nil {
		gameConf = &cardQzsgStorage.Conf{
			ReadyTime:            5,
			QiangZhuangTime:      7,
			XiaZhuTime:           12,
			JieSuanTime:          7,
			ProfitPerThousand:    20,
			MinEnterTableOdds:    50,
			BotProfitPerThousand: 80,
		}
		cardQzsgStorage.InsertRoomConf(gameConf)
	}

	roomData := cardQzsgStorage.RoomData{
		TablesInfo: map[string]cardQzsgStorage.TableInfo{}, //
	}
	cardQzsgStorage.UpsertTablesInfo(roomData.TablesInfo)
	self.InitHallInfo()
	gameRobotConf := cardQzsgStorage.GetRoomRobotConf()
	for _, v := range BaseScoreList {
		if gameRobotConf == nil {
			self.RoomRobotInit(v)
		}
		self.CreateTable(strconv.FormatInt(v, 10) + "_" + strconv.FormatInt(3, 10) + "_6")
		time.Sleep(time.Millisecond * 100)
		self.CreateTable(strconv.FormatInt(v, 10) + "_" + strconv.FormatInt(4, 10) + "_6")
		time.Sleep(time.Millisecond * 100)
		self.CreateTable(strconv.FormatInt(v, 10) + "_" + strconv.FormatInt(5, 10) + "_6")
		time.Sleep(time.Millisecond * 100)
		self.CreateTable(strconv.FormatInt(v, 10) + "_" + strconv.FormatInt(5, 10) + "_6")
		time.Sleep(time.Millisecond * 100)
	}
	//self.CreateTable("1000_5_6" )
	self.tablesID.Range(func(key, value interface{}) bool { //启动table队列
		self.room.GetTable(value.(string))
		return true
	})
	go func() {
		//c := cron.New()
		//c.AddFunc("*/1 * * * * ?",self.OnTimer)
		//c.Start()
		self.RoomRobotConf = cardQzsgStorage.GetRoomRobotConf()
		c1 := cron.New()
		c1.AddFunc("*/5 * * * * ?", self.OnTimer5)
		c1.AddFunc("*/60 * * * * ?", self.OnTimer60)
		c1.Start()
	}()
}
func (self *Room) Enter(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	//	table_id := msg["table_id"].(string)
	reboot := gameStorage.QueryGameReboot(game.CardQzsg)
	if "true" == reboot { //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	isNotAllow := lobbyStorage.QueryLobbyGameLayoutByName(game.CardQzsg).IsNotAllowPlay
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	if isNotAllow == 1 && user.Type != userStorage.TypeNormal {
		return errCode.PlayAccountNotAllow.GetI18nMap(), nil
	}
	if msg["BaseScore"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	baseScore, _ := utils.ConvertInt(msg["BaseScore"])
	var tableID string
	var table room.BaseTable
	tableList := make([]room.BaseTable, 0)
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
			if myTable.BaseScore == baseScore && myTable.GetTablePlayerNum() < myTable.TotalPlayerNum {
				tableList = append(tableList, myTable)
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table != nil {
		table.PutQueue(protocol.Enter, session)
		return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
	}
	if len(tableList) == 0 {
		return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
	}
	tableIdx := self.RandInt64(1, int64(len(tableList)+1)) - 1
	table = tableList[tableIdx]
	table.PutQueue(protocol.Enter, session)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) AutoEnter(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	//	table_id := msg["table_id"].(string)
	reboot := gameStorage.QueryGameReboot(game.CardQzsg)
	if "true" == reboot { //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	isNotAllow := lobbyStorage.QueryLobbyGameLayoutByName(game.CardQzsg).IsNotAllowPlay
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	if isNotAllow == 1 && user.Type != userStorage.TypeNormal {
		return errCode.PlayAccountNotAllow.GetI18nMap(), nil
	}
	gameConf := cardQzsgStorage.GetRoomConf()
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))

	baseScore := int64(0)
	for i := len(BaseScoreList) - 1; i >= 0; i-- {
		if wallet.VndBalance/BaseScoreList[i] > int64(gameConf.MinEnterTableOdds) { //
			baseScore = BaseScoreList[i]
			break
		}
	}
	if baseScore == 0 {
		return errCode.BalanceNotEnough.GetI18nMap(), nil
	}

	var tableID string
	var table room.BaseTable
	tableList := make([]room.BaseTable, 0)
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
			if myTable.BaseScore == baseScore && myTable.TotalPlayerNum == 6 && myTable.GetTablePlayerNum() < myTable.TotalPlayerNum {
				tableList = append(tableList, myTable)
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table != nil {
		table.PutQueue(protocol.Enter, session)
		return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
	}
	if len(tableList) == 0 {
		return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
	}
	tableIdx := self.RandInt64(1, int64(len(tableList)+1)) - 1
	table = tableList[tableIdx]
	table.PutQueue(protocol.Enter, session)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) QuitTable(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	userID := session.GetUserID()
	var tableID string
	var table room.BaseTable
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table != nil {
		table.PutQueue(protocol.QuitTable, userID)
		return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
	}
	return errCode.Success("").GetMap(), nil
}
func (self *Room) Ready(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	//	table_id := msg["table_id"].(string)
	reboot := gameStorage.QueryGameReboot(game.CardQzsg)
	if "true" == reboot { //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	var table room.BaseTable
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	table.PutQueue(protocol.Ready, userID)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) AutoReady(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	if msg["AutoReady"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	var table room.BaseTable
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	table.PutQueue(protocol.AutoReady, session, msg)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) MasterStartGame(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	//	table_id := msg["table_id"].(string)
	reboot := gameStorage.QueryGameReboot(game.CardQzsg)
	if "true" == reboot { //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	var table room.BaseTable
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	table.PutQueue(protocol.MasterStartGame, session)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) GrabDealer(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	if msg["GrabDealer"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	var table room.BaseTable
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	table.PutQueue(protocol.GrabDealer, userID, msg)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) XiaZhu(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	if msg["betV"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	var table room.BaseTable
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	table.PutQueue(protocol.XiaZhu, userID, msg)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) Info(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	res := make(map[string]interface{}, 2)
	res["ServerId"] = self.BaseModule.GetServerID()
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) InitHallInfo() {
	self.HallInfo = map[HallType]map[int64]HallConfig{}
	self.HallInfo[All] = map[int64]HallConfig{}
	self.HallOffsetPlayerNum = map[HallType]map[int64]int{}
	self.HallOffsetPlayerNum[All] = map[int64]int{}
	self.HallInfo[All][500] = HallConfig{
		PlayerNum: 6,
		BaseScore: 500,
		BaseNum:   60,
		MaxOffset: 10,
		StepNum:   1,
	}

	self.HallInfo[All][1000] = HallConfig{
		PlayerNum: 6,
		BaseScore: 1000,
		BaseNum:   100,
		MaxOffset: 10,
		StepNum:   1,
	}

	self.HallInfo[All][5000] = HallConfig{
		PlayerNum: 6,
		BaseScore: 5000,
		BaseNum:   130,
		MaxOffset: 10,
		StepNum:   1,
	}

	self.HallInfo[All][10000] = HallConfig{
		PlayerNum: 6,
		BaseScore: 10000,
		BaseNum:   130,
		MaxOffset: 10,
		StepNum:   1,
	}

	self.HallInfo[All][50000] = HallConfig{
		PlayerNum: 6,
		BaseScore: 50000,
		BaseNum:   130,
		MaxOffset: 10,
		StepNum:   1,
	}

	self.HallInfo[All][100000] = HallConfig{
		PlayerNum: 6,
		BaseScore: 100000,
		BaseNum:   130,
		MaxOffset: 10,
		StepNum:   1,
	}
	self.HallInfo[All][500000] = HallConfig{
		PlayerNum: 6,
		BaseScore: 500000,
		BaseNum:   130,
		MaxOffset: 10,
		StepNum:   1,
	}
	self.HallInfo[All][1000000] = HallConfig{
		PlayerNum: 6,
		BaseScore: 1000000,
		BaseNum:   130,
		MaxOffset: 10,
		StepNum:   1,
	}
}
func (self *Room) GetHallInfo(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	if msg["Type"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	//hallType := HallType(msg["Type"].(string))
	res := make([]HallConfig, 0)

	info := map[HallType]map[int64]int{}
	info[All] = map[int64]int{}
	self.tablesID.Range(func(key, value interface{}) bool {
		table := self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.TotalPlayerNum == 4 {
				info[All][myTable.BaseScore] += myTable.GetTablePlayerNum()
			}
		}
		return true
	})

	four := self.HallInfo[All]
	for _, v := range four {
		v.CurNum = info[All][v.BaseScore]
		for _, v1 := range self.RoomRobotConf {
			if v1.HallType == string(All) && v1.BaseScore == v.BaseScore {
				v.CurNum += v1.BaseRobotNum + self.HallOffsetPlayerNum[All][v.BaseScore]
			}
		}
		res = append(res, v)
	}
	sort.Slice(res, func(i, j int) bool { //升序排序
		if res[i].BaseScore < res[j].BaseScore {
			return true
		} else if res[i].BaseScore == res[j].BaseScore && res[i].PlayerNum < res[j].PlayerNum {
			return true
		}
		return false
	})

	return errCode.Success(res).GetMap(), nil
}
func (self *Room) GetEnterData(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	var tableID string
	userID := session.GetUserID()
	self.tablesID.Range(func(key, value interface{}) bool {
		table := self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table := self.room.GetTable(tableID) //(self.curTableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	erro := table.PutQueue(protocol.GetEnterData, session, msg)
	if erro != nil {
		log.Info("--------------- table.PutQueue error---tableID = %s", tableID, "---error = %s", erro)
		return errCode.ServerError.GetI18nMap(), nil
	}
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}

func (self *Room) onDisconnect(userID string) (map[string]interface{}, error) {
	var tableID string
	self.tablesID.Range(func(key, value interface{}) bool {
		table := self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table := self.room.GetTable(tableID)
	if table == nil {
		//log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	if table != nil {
		table.PutQueue(protocol.QuitTable, userID)
		return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
	}
	return errCode.Success(nil).GetMap(), nil
}
func (self *Room) GetWinLoseRank(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	res := make(map[string]interface{}, 2)
	res["RankList"] = gameStorage.GetGameWinLoseRank(game.CardQzsg, 20)
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) CheckPlayerInGame(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	userID := session.GetUserID()
	res := make(map[string]interface{}, 2)
	res["InGame"] = false
	self.tablesID.Range(func(key, value interface{}) bool {
		table := self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				res["InGame"] = true
				return false
			}
		}
		return true
	})
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) GameInvite(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	var tableID string
	userID := session.GetUserID()
	self.tablesID.Range(func(key, value interface{}) bool {
		table := self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table := self.room.GetTable(tableID) //(self.curTableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	record := gameStorage.GameInviteRecord{
		GameType:        game.CardQzsg,
		GameName:        common2.I18str(string(game.CardQzsg)),
		InvitorNickName: user.NickName,
		RoomId:          myTable.tableID,
		BaseScore:       myTable.BaseScore,
		ServerId:        myTable.module.GetServerID(),
		UpdateAt:        utils.Now(),
	}
	self.NotifyGameInviteToOnlineUsers(record)
	return errCode.Success(nil).GetMap(), nil
}
func (self *Room) NotifyGameInviteToOnlineUsers(record gameStorage.GameInviteRecord) {
	sessionBeans := vGate.QuerySessionByPage("HallScene")
	isNotAllow := lobbyStorage.QueryLobbyGameLayoutByName(game.CardQzsg).IsNotAllowPlay
	for _, v := range *sessionBeans {
		uid := v.Oid.Hex()
		user := userStorage.QueryUserId(utils.ConvertOID(uid))
		if isNotAllow == 1 && user.Type != userStorage.TypeNormal {
			continue
		}
		record.BeInvitedUid = uid
		gameStorage.UpsertGameInviteRecord(record)
		res := make(map[string]interface{}, 3)
		res["Data"] = gameStorage.QueryGameInviteRecord(uid)
		res["Action"] = protocol.GameInviteRecord
		res["GameType"] = game.Lobby
		b, _ := json.Marshal(res)
		self.onlinePush.SendCallBackMsgNR([]string{v.SessionId}, game.Push, b)
		self.onlinePush.ExecuteCallBackMsg(self.onlinePush.TraceSpan)
	}
}
func (self *Room) InviteEnter(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	//	table_id := msg["table_id"].(string)
	reboot := gameStorage.QueryGameReboot(game.CardQzsg)
	userID := session.GetUserID()
	if msg["RoomId"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	roomId, _ := msg["RoomId"].(string)
	gameStorage.RemoveGameInviteRecord(userID, roomId)
	if "true" == reboot { //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	isNotAllow := lobbyStorage.QueryLobbyGameLayoutByName(game.CardQzsg).IsNotAllowPlay
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	if isNotAllow == 1 && user.Type != userStorage.TypeNormal {
		return errCode.PlayAccountNotAllow.GetI18nMap(), nil
	}
	var tableID string
	var table room.BaseTable
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID) {
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table != nil {
		table.PutQueue(protocol.InviteEnter, session)
		return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
	}
	tableID, _ = msg["RoomId"].(string)
	table = self.room.GetTable(tableID)
	if table == nil {
		res := make(map[string]interface{}, 3)
		res["Action"] = protocol.InviteEnter
		res["GameType"] = game.Lobby
		res["Code"] = errCode.RoomPlayerNumLimit.Code
		b, _ := json.Marshal(res)
		self.onlinePush.SendCallBackMsgNR([]string{session.GetSessionID()}, game.Push, b)
		self.onlinePush.ExecuteCallBackMsg(self.onlinePush.TraceSpan)
		return errCode.RoomPlayerNumLimit.GetI18nMap(), nil
	}
	table.PutQueue(protocol.InviteEnter, session)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
