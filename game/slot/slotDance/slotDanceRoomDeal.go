package slotDance

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
	"vn/storage/gameStorage"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return r.Int63n(max-min) + min
}

//生成桌子id
func (self *Room) GenerateTableID() string {
	//rand.Seed(time.Now().UnixNano())
	var tableID int
	for true {
		tableID = int(RandInt64(1, 1000000))
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
		myTable := (table.(interface{})).(*MyTable)
		if myTable.ModeType == TRIAL {
			myTable.PutQueue(protocol.ClearTable)
		} else {
			table.PutQueue(protocol.Enter, session, msg)
			return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
		}
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
func (self *Room) Info(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	res := make(map[string]interface{}, 2)
	res["slotDanceServerId"] = self.BaseModule.GetServerID()
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
		initSymbol := make([]int64, len(myTable.ReelsListTrial))
		for k, v := range myTable.ReelsListTrial {
			rand := myTable.RandInt64(1, int64(len(v)+1))
			rand = rand - 1
			initSymbol[k] = rand
		}
		myTable.TrialModeConf = TrialModeConf{
			VndBalance: 200000000,
		}
		myTable.JieSuanData = JieSuanData{}
		myTable.JieSuanData.TrialData = TrialData{
			VndBalance: myTable.TrialModeConf.VndBalance,
		}
		res := make(map[string]interface{}, 3)
		res["TrialConf"] = myTable.TrialModeConf
		res["ReelsListTrial"] = myTable.ReelsListTrial
		res["InitSymbol"] = initSymbol
		return errCode.Success(res).GetMap(), nil
	} else if modeType == NORMAL {
		myTable.ModeType = NORMAL
		myTable.JieSuanData = JieSuanData{}
		initSymbol := make([]int64, len(myTable.ReelsList))
		for k, v := range myTable.ReelsList {
			rand := myTable.RandInt64(1, int64(len(v)+1))
			rand = rand - 1
			initSymbol[k] = rand
		}
		res := make(map[string]interface{}, 1)
		res["InitSymbol"] = initSymbol
		return errCode.Success(res).GetMap(), nil
	}

	return errCode.Success(nil).GetMap(), nil
}
