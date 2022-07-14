package yxx

import (
	"fmt"
	"math/rand"
	"strconv"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/yxxStorage"
)

//生成桌子id
func (self *Room) GenerateTableID() string {
	//rand.Seed(time.Now().UnixNano())
	var tableID int
	roomData := yxxStorage.GetRoomData()
	for true {
		tableID = rand.Intn(1000000)
		if tableID < 100000 {
			tableID = tableID + 100000
		}
		_, ok := roomData.TablesInfo[strconv.Itoa(tableID)]
		if !ok {
			break
		}
	}
	roomData.CurTableID = strconv.Itoa(tableID)
	yxxStorage.UpsertRoomData(roomData)
	return strconv.Itoa(tableID)
}
func (self *Room) CreateTable(tableID string) (r string, err error) {
	if tableID == "" {
		tableID = "000000" //self.GenerateTableID() //服务器生成桌子id
	}
	_, err = self.room.CreateById(self.App, tableID, self.NewTable)
	if err != nil {
		return "", err
	}
	self.tablesID.Store(tableID, tableID)
	self.curTableID = tableID
	return "success", nil
}
func (self *Room) DestroyTable(tableID string) {
	self.room.DestroyTable(tableID)
	self.tablesID.Delete(tableID)
}
func (self *Room) NewTable(module module.App, tableId string) (room.BaseTable, error) {
	//roomData := yxxStorage.GetRoomData()
	//yxxStorage.UpsertTablesInfo(roomData)
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

func (self *Room) TableQueue(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	//	table_id := msg["table_id"].(string)
	action := msg["action"].(string)
	table := self.room.GetTable(self.curTableID) //

	if table == nil {
		log.Info("--------------- table not exist---")
		return errCode.ServerError.GetI18nMap(), nil
	}
	erro := table.PutQueue(action, session, msg)
	if erro != nil {
		log.Info("--------------- table.PutQueue error---tableID = %s", self.curTableID, "---error = %s", erro)
		return errCode.ServerError.GetI18nMap(), nil
	}
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) GetShortCutList(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
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
	erro := table.PutQueue(protocol.GetShortCutList, session, msg)
	if erro != nil {
		log.Info("--------------- table.PutQueue error---tableID = %s", tableID, "---error = %s", erro)
		return errCode.ServerError.GetI18nMap(), nil
	}
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) SendShortCut(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
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
	erro := table.PutQueue(protocol.SendShortCut, session, msg)
	if erro != nil {
		log.Info("--------------- table.PutQueue error---tableID = %s", tableID, "---error = %s", erro)
		return errCode.ServerError.GetI18nMap(), nil
	}
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *Room) QuitTable(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	//	table_id := msg["table_id"].(string)
	userID := session.GetUserID()
	table := self.room.GetTable(self.curTableID) //

	if table != nil {
		erro := table.PutQueue(protocol.QuitTable, session, userID)
		if erro != nil {
			log.Info("--------------- table.PutQueue error---tableID = %s", self.curTableID, "---error = %s", erro)
			return errCode.ServerError.GetI18nMap(), nil
		}
		return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
	} else {
		return errCode.Success("").GetMap(), nil
	}
}
func (self *Room) onLogin(uid string) (map[string]interface{}, error) {
	//	log.Info("--------------------------------------login success----------------")
	self.tablesID.Range(func(key, value interface{}) bool {
		table := self.room.GetTable(value.(string)) //
		if table != nil {
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(uid) {
				msg := make(map[string]interface{})
				sb := vGate.QuerySessionBean(uid)
				if sb == nil {
					log.Info("session is nil")
					return false
				}
				msg["ServerID"] = self.GetServerID()
				msg["TableID"] = myTable.tableID
				ret := myTable.DealProtocolFormat(msg, protocol.Reenter, nil)
				myTable.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
				myTable.onlinePush.ExecuteCallBackMsg(myTable.Trace())
				return false
			}
		}
		return true
	})

	return nil, nil
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
				res["TableInfo"] = myTable.GetTableInfo(true)
				res["SelfInfo"] = myTable.GetPlayerInfo(userID, true)
				res["PlayerInfo"] = myTable.PositionList
				return false
			}
		}
		return true
	})
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) onDisconnect(uid string) (map[string]interface{}, error) {
	//	log.Info("--------------------------------------disconnect----------------")
	return nil, nil
}

func (self *Room) GetPlayerList(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	roomData := yxxStorage.GetRoomData()
	table := self.room.GetTable(roomData.CurTableID)
	if table == nil {
		log.Info("--------------- table not exist---")
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	res, erro := myTable.GetPlayerList(session, msg)
	if res == nil {
		return erro, nil
	}
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) GetResultsRecord(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	roomData := yxxStorage.GetRoomData()
	table := self.room.GetTable(roomData.CurTableID)
	if table == nil {
		log.Info("--------------- table not exist---")
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	res, erro := myTable.GetResultsRecord(session, msg)
	if res == nil {
		return erro, nil
	}
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) GetPrizeRecord(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	roomData := yxxStorage.GetRoomData()
	table := self.room.GetTable(roomData.CurTableID)
	if table == nil {
		log.Info("--------------- table not exist---")
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	res, erro := myTable.GetPrizeRecord(session, msg)
	if res == nil {
		return erro, nil
	}
	return errCode.Success(res).GetMap(), nil
}

func (self *Room) Info(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	roomData := yxxStorage.GetRoomData()
	table := self.room.GetTable(roomData.CurTableID)
	if table == nil {
		log.Info("--------------- table not exist---")
		return errCode.ServerError.GetI18nMap(), nil
	}
	res := make(map[string]interface{}, 2)
	res["PrizePool"] = roomData.TablesInfo[roomData.CurTableID].PrizePool
	res["YxxServerId"] = roomData.TablesInfo[roomData.CurTableID].ServerID

	return errCode.Success(res).GetMap(), nil
}
