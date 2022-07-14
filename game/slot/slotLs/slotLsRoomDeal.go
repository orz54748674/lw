package slotLs

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	common2 "vn/game/common"
	"vn/storage/gameStorage"
	"vn/storage/slotStorage"
	"vn/storage/slotStorage/slotLsStorage"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func (self *Room) RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return r.Int63n(max-min) + min
}

//func (self *Room) OnTimer(){
//	for k,v := range CoinNum {
//		rand := self.RandInt64(50,300)
//		rand = rand * v / 50
//		slotLsStorage.IncJackpot(k,rand,rand / 10)
//	}
//}
func (self *Room) OnTimer10() {
	for k, v := range CoinNum {
		rand := self.RandInt64(50, 300) * 10
		rand = rand * v / 50
		slotLsStorage.IncJackpot(k, rand, rand/10)
	}
	for k, v := range CoinNum {
		goldJackpot, silverJackpot := slotLsStorage.GetJackpot()
		maxGoldJackpot := int64(1200000000)
		maxSilverJackpot := int64(120000000)
		rand := self.RandInt64(1, 60*60*24)
		if rand == 1 || goldJackpot[k] >= maxGoldJackpot {
			robot := common2.RandBotN(1, r)
			rand = self.RandInt64(1, int64(len(CoinValue))+1) - 1
			betV := v * CoinValue[rand]
			if len(robot) > 0 {
				slotStorage.InsertJackpotRecord(robot[0].Oid.Hex(), robot[0].NickName, betV, goldJackpot[k], game.SlotLs, "goldJackpot", "", nil)
				for k1, _ := range CoinNum {
					val := -goldJackpot[k1] + InitGoldJackpot[k1]
					slotLsStorage.IncJackpot(k1, val, 0)
				}
			}
		}

		rand = self.RandInt64(1, 60*60*24)
		if rand == 1 || silverJackpot[k] >= maxSilverJackpot {
			robot := common2.RandBotN(1, r)
			rand = self.RandInt64(1, int64(len(CoinValue))+1) - 1
			betV := v * CoinValue[rand]
			if len(robot) > 0 {
				slotStorage.InsertJackpotRecord(robot[0].Oid.Hex(), robot[0].NickName, betV, silverJackpot[k], game.SlotLs, "silverJackpot", "", nil)
				for k1, _ := range CoinNum {
					val := -silverJackpot[k1] + InitSilverJackpot[k1]
					slotLsStorage.IncJackpot(k1, 0, val)
				}
			}
		}
	}
}

//生成桌子id
func (self *Room) GenerateTableID() string {
	//rand.Seed(time.Now().UnixNano())
	var tableID int
	for true {
		tableID = int(self.RandInt64(1, 1000000))
		if tableID < 100000 {
			tableID = tableID + 100000
		}
		ok := true
		if self.room.GetTable(strconv.Itoa(tableID)) != nil {
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
	if tableID == "" {
		tableID = self.GenerateTableID() //服务器生成桌子id
	}
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
func (self *Room) Enter(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	//	table_id := msg["table_id"].(string)
	reboot := gameStorage.QueryGameReboot(game.SlotLs)
	if "true" == reboot { //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	if msg["modeType"] == nil {
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
	if table != nil {
		table.PutQueue(protocol.Enter, session, msg)
		return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
	}

	table, _, tableID = self.CreateTable("")
	if table == nil {
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	table = self.room.GetTable(tableID)
	table.PutQueue(protocol.Enter, session, msg)
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
		table.PutQueue(protocol.QuitTable, session)
		return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
	}
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) Spin(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	reboot := gameStorage.QueryGameReboot(game.SlotLs)
	if "true" == reboot { //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}

	if msg["CoinNum"] == nil || msg["CoinValue"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	CoinNum, _ := utils.ConvertInt(msg["CoinNum"])
	CoinValue, _ := utils.ConvertInt(msg["CoinValue"])
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil {
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
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	tbl := make(map[string]interface{}, 2)
	tbl["CoinNum"] = CoinNum
	tbl["CoinValue"] = CoinValue
	table.PutQueue(protocol.Spin, session, tbl)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) SpinFree(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil {
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
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	tbl := make(map[string]interface{}, 2)
	table.PutQueue(protocol.SpinFree, session, tbl)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) SpinTrial(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	if msg["CoinNum"] == nil || msg["CoinValue"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	CoinNum, _ := utils.ConvertInt(msg["CoinNum"])
	CoinValue, _ := utils.ConvertInt(msg["CoinValue"])
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil {
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
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	tbl := make(map[string]interface{}, 2)
	tbl["CoinNum"] = CoinNum
	tbl["CoinValue"] = CoinValue
	table.PutQueue(protocol.SpinTrial, session, tbl)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) SpinTrialFree(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil {
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
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	tbl := make(map[string]interface{}, 2)
	table.PutQueue(protocol.SpinTrialFree, session, tbl)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) GetJackpot(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	reboot := gameStorage.QueryGameReboot(game.SlotLs)
	if "true" == reboot { //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	if msg["CoinNum"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	coinNum, _ := utils.ConvertInt(msg["CoinNum"])
	res := make(map[string]interface{}, 2)
	goldJackpot, silverJackpot := slotLsStorage.GetJackpot()
	find := false
	for k, v := range CoinNum {
		if v == coinNum {
			res["GoldJackpot"] = goldJackpot[k]
			res["SilverJackpot"] = silverJackpot[k]
			find = true
			break
		}
	}
	if !find {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	return errCode.Success(res).GetMap(), nil
}

func (self *Room) Info(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	res := make(map[string]interface{}, 2)
	res["slotLsServerId"] = self.BaseModule.GetServerID()
	goldJackpot, silverJackpot := slotLsStorage.GetJackpot()
	res["GoldJackpot"] = goldJackpot
	res["SilverJackpot"] = silverJackpot
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
func (self *Room) SelectFreeGame(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	if msg["FreeType"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil {
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
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	if myTable.JieSuanDataFree.FreeGameTimes <= 0 {
		log.Info("---------------  not free game---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}

	freeType := FreeType(msg["FreeType"].(string))
	res := make(map[string]interface{}, 2)
	myTable.FreeType = freeType
	if freeType == PURPLE { //紫色随机倍数和次数
		timesIdx := self.RandInt64(1, int64(len(FreeGameRandConfig.Times)+1)) - 1
		res["Times"] = FreeGameRandConfig.Times[timesIdx]
		numOfTimesIdx := self.RandInt64(1, int64(len(FreeGameRandConfig.NumOfTimes)+1)) - 1
		res["NumOfTimes"] = FreeGameRandConfig.NumOfTimes[numOfTimesIdx]

		myTable.FreeGameConf.Times = FreeGameRandConfig.Times[timesIdx]
		myTable.FreeGameConf.NumOfTimes = FreeGameRandConfig.NumOfTimes[numOfTimesIdx]
		myTable.JieSuanDataFree.FreeRemainTimes = FreeGameRandConfig.NumOfTimes[numOfTimesIdx]
	} else {
		res["Times"] = FreeSelectList[freeType].Times
		res["NumOfTimes"] = FreeSelectList[freeType].NumOfTimes

		myTable.FreeGameConf.Times = FreeSelectList[freeType].Times
		myTable.FreeGameConf.NumOfTimes = FreeSelectList[freeType].NumOfTimes
		myTable.JieSuanDataFree.FreeRemainTimes = FreeSelectList[freeType].NumOfTimes
	}
	if myTable.JieSuanData.FreeGameTimes > 0 {
		myTable.JieSuanData.FreeGameTimes--
	}
	myTable.JieSuanDataFree.FreeGameTimes--

	tableInfoRet := myTable.GetTableInfo()
	res["TableInfo"] = tableInfoRet
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) SelectTrialFreeGame(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	if msg["FreeType"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil {
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
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	if myTable.JieSuanDataTrialFree.FreeGameTimes <= 0 {
		log.Info("---------------  not free game---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}

	freeType := FreeType(msg["FreeType"].(string))
	res := make(map[string]interface{}, 2)
	myTable.FreeType = freeType
	if freeType == PURPLE { //紫色随机倍数和次数
		timesIdx := self.RandInt64(1, int64(len(FreeGameRandConfig.Times)+1)) - 1
		res["Times"] = FreeGameRandConfig.Times[timesIdx]
		numOfTimesIdx := self.RandInt64(1, int64(len(FreeGameRandConfig.NumOfTimes)+1)) - 1
		res["NumOfTimes"] = FreeGameRandConfig.NumOfTimes[numOfTimesIdx]

		myTable.FreeGameConf.Times = FreeGameRandConfig.Times[timesIdx]
		myTable.FreeGameConf.NumOfTimes = FreeGameRandConfig.NumOfTimes[numOfTimesIdx]
		myTable.JieSuanDataTrialFree.FreeRemainTimes = FreeGameRandConfig.NumOfTimes[numOfTimesIdx]
	} else {
		res["Times"] = FreeSelectList[freeType].Times
		res["NumOfTimes"] = FreeSelectList[freeType].NumOfTimes

		myTable.FreeGameConf.Times = FreeSelectList[freeType].Times
		myTable.FreeGameConf.NumOfTimes = FreeSelectList[freeType].NumOfTimes
		myTable.JieSuanDataTrialFree.FreeRemainTimes = FreeSelectList[freeType].NumOfTimes
	}
	if myTable.JieSuanDataTrial.FreeGameTimes > 0 {
		myTable.JieSuanDataTrial.FreeGameTimes--
	}
	myTable.JieSuanDataTrialFree.FreeGameTimes--

	tableInfoRet := myTable.GetTableInfo()
	res["TableInfo"] = tableInfoRet
	return errCode.Success(res).GetMap(), nil
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
	myTable := table.(*MyTable)
	if !myTable.IsInFreeGame() {
		myTable.PutQueue(protocol.ClearTable)
	}
	log.Info("----------------slot disconnect----------")
	return errCode.Success(nil).GetMap(), nil
}
func (self *Room) SwitchMode(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	if msg["modeType"] == nil {
		return errCode.ServerError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil {
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
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s", tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	modeType := ModeType(msg["modeType"].(string))
	myTable.ModeType = modeType
	if modeType == TRIAL {
		myTable.TrialModeConf = TrialModeConf{
			VndBalance:    200000000,
			GoldJackpot:   []int64{100000000, 200000000, 400000000, 600000000, 1000000000},
			SilverJackpot: []int64{10000000, 20000000, 40000000, 60000000, 100000000},
		}
		myTable.JieSuanDataTrial = JieSuanData{}
		myTable.TrialData = TrialData{
			GoldJackpot:   myTable.TrialModeConf.GoldJackpot,
			SilverJackpot: myTable.TrialModeConf.SilverJackpot,
			VndBalance:    myTable.TrialModeConf.VndBalance,
		}
		res := make(map[string]interface{}, 3)
		res["TrialConf"] = myTable.TrialModeConf
		res["ReelsListTrial"] = myTable.ReelsListTrial
		res["ReelsListTrialFree"] = myTable.ReelsListTrialFree
		return errCode.Success(res).GetMap(), nil
	} else if modeType == NORMAL {
		myTable.ModeType = NORMAL
	}

	return errCode.Success(nil).GetMap(), nil
}
func (self *Room) GetJackpotRecord(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	_, err := utils.CheckParams2(params, []string{"Offset", "PageSize"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	Offset, _ := utils.ConvertInt(params["Offset"])
	PageSize, _ := utils.ConvertInt(params["PageSize"])
	res := make(map[string]interface{}, 2)
	record := slotStorage.QueryJackpotRecord(int(Offset), int(PageSize), game.SlotLs)
	res["JackpotRecord"] = record
	res["TotalNum"] = slotStorage.QueryJackpotRecordTotal(game.SlotLs)
	return errCode.Success(res).GetMap(), nil
}
