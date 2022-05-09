package slotSex

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
	"vn/storage/slotStorage/slotSexStorage"
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
//		slotSexStorage.IncJackpot(k,rand)
//	}
//}
func (self *Room) OnTimer10(){
	for k,v := range CoinNum {
		rand := self.RandInt64(50,300) * 5
		rand = rand * v / 50
		slotSexStorage.IncJackpot(k,rand)
	}
	rand := self.RandInt64(1,100)
	if rand == 5{
		baseRand := self.RandInt64(1,100)
		coinVal := int64(0)
		if baseRand < 60{
			coinVal = CoinNum[0]
		}else if baseRand < 80{
			coinVal = CoinNum[1]
		}else{
			coinVal = CoinNum[2]
		}
		jackpotRand := self.RandInt64(1,40)
		getJackpot := coinVal * 5000 + coinVal * 5000 * jackpotRand / 100
		robot := common2.RandBotN(1,r)
		if len(robot) > 0{
			slotStorage.InsertJackpotRecord(robot[0].Oid.Hex(),robot[0].NickName,coinVal,getJackpot,game.SlotSex,"","",nil)
			curJackpot := slotSexStorage.GetJackpot()
			for k1,_ := range CoinNum{
				val := -curJackpot[k1] + InitJackpot[k1]
				slotSexStorage.IncJackpot(k1,val)
			}
		}
	}
}
//生成桌子id
func (self *Room) GenerateTableID() string {
	//rand.Seed(time.Now().UnixNano())
	var tableID int
	for true {
		tableID = int(self.RandInt64(1,1000000))
		if tableID < 100000{
			tableID = tableID + 100000
		}
		ok := true
		if self.room.GetTable(strconv.Itoa(tableID)) != nil{
			ok = false
			break
		}

		if ok {
			break
		}
	}
	return strconv.Itoa(tableID)
}
func (self *Room) CreateTable(tableID string) (table room.BaseTable, err error,id string) {
	if tableID == ""{
		tableID = self.GenerateTableID() //服务器生成桌子id
	}
	table, err = self.room.CreateById(self.App, tableID,self.NewTable)
	if err != nil {
		return nil, err,""
	}
	self.tablesID.Store(tableID,tableID)
	return table, nil,tableID
}
func (self *Room) DestroyTable(tableID string){
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
func (self *Room) Enter(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	//	table_id := msg["table_id"].(string)
	reboot := gameStorage.QueryGameReboot(game.SlotSex)
	if "true" == reboot{ //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	var table room.BaseTable
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil{
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID){
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table != nil {
		myTable := (table.(interface{})).(*MyTable)
		if myTable.ModeType == TRIAL{
			myTable.PutQueue(protocol.ClearTable)
		}else{
			table.PutQueue(protocol.Enter,session,msg)
			return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
		}
	}
	table,_,tableID = self.CreateTable("")
	if table == nil{
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	table = self.room.GetTable(tableID)
	table.PutQueue(protocol.Enter,session,msg)
	return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}
func (self *Room) QuitTable(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	userID := session.GetUserID()
	var tableID string
	var table room.BaseTable
	self.tablesID.Range(func(key, value interface{}) bool {
		table = self.room.GetTable(value.(string)) //
		if table != nil{
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID){
				tableID = value.(string)
				return false
			}
		}
		return true
	})
	table = self.room.GetTable(tableID)
	if table != nil {
		table.PutQueue(protocol.QuitTable,session)
		return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
	}
	return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}
func (self *Room) Spin(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	reboot := gameStorage.QueryGameReboot(game.SlotSex)
	if "true" == reboot{ //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}

	if msg["CoinNum"] == nil || msg["CoinValue"] == nil{
		return errCode.ServerError.GetI18nMap(), nil
	}
	CoinNum,_ := utils.ConvertInt(msg["CoinNum"])
	if CoinNum > 20 || CoinNum <= 0{
		return errCode.ServerError.GetI18nMap(), nil
	}
	CoinValue,_ := utils.ConvertInt(msg["CoinValue"])
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil{
		self.tablesID.Range(func(key, value interface{}) bool {
			table := self.room.GetTable(value.(string)) //
			if table != nil{
				myTable := (table.(interface{})).(*MyTable)
				if myTable.PlayerIsTable(userID){
					tableID = value.(string)
					return false
				}
			}
			return true
		})
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	tbl := make(map[string]interface{},2)
	tbl["CoinNum"] = CoinNum
	tbl["CoinValue"] = CoinValue
	table.PutQueue(protocol.Spin,session,tbl)
	return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}


func (self *Room) GetJackpot(session gate.Session,msg map[string]interface{}) (map[string]interface{},error) {
	reboot := gameStorage.QueryGameReboot(game.SlotSex)
	if "true" == reboot{ //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	if msg["CoinNum"] == nil{
		return errCode.ServerError.GetI18nMap(), nil
	}
	coinNum,_ := utils.ConvertInt(msg["CoinNum"])
	res := make(map[string]interface{},2)
	goldJackpot := slotSexStorage.GetJackpot()
	find := false
	for k,v := range CoinNum {
		if v == coinNum{
			res["Jackpot"] = goldJackpot[k]
			find = true
			break
		}
	}
	if !find{
		return errCode.ErrParams.GetI18nMap(), nil
	}
	return  errCode.Success(res).GetMap(),nil
}

func (self *Room) Info(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	res := make(map[string]interface{},2)
	res["slotSexServerId"] = self.BaseModule.GetServerID()
	goldJackpot := slotSexStorage.GetJackpot()
	res["Jackpot"] = goldJackpot
	return errCode.Success(res).GetMap(), nil
}

func (self *Room) EnterBonusGame(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil{
		self.tablesID.Range(func(key, value interface{}) bool {
			table := self.room.GetTable(value.(string)) //
			if table != nil{
				myTable := (table.(interface{})).(*MyTable)
				if myTable.PlayerIsTable(userID){
					tableID = value.(string)
					return false
				}
			}
			return true
		})
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	if !myTable.JieSuanData.BonusGame{
		log.Info("---------------  not bonus game---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}

	res := make(map[string]interface{})
	n := len(myTable.BonusSymbolList)
	for i := 0;i < n;i++{
		idx := myTable.RandInt64(1,int64(n + 1)) - 1
		tmpIdx := myTable.RandInt64(1,int64(n + 1)) - 1
		tmp := myTable.BonusSymbolList[idx]
		myTable.BonusSymbolList[idx] = myTable.BonusSymbolList[tmpIdx]
		myTable.BonusSymbolList[tmpIdx] = tmp
	}
	res["BonusSymbolList"] = myTable.BonusSymbolList
	myTable.GameConf = slotSexStorage.GetRoomConf()
	myTable.CountDown = myTable.GameConf.BonusTimeOut
	myTable.BonusGameData.State = 1
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) SelectBonusSymbol(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil{
		self.tablesID.Range(func(key, value interface{}) bool {
			table := self.room.GetTable(value.(string)) //
			if table != nil{
				myTable := (table.(interface{})).(*MyTable)
				if myTable.PlayerIsTable(userID){
					tableID = value.(string)
					return false
				}
			}
			return true
		})
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	if !myTable.JieSuanData.BonusGame || myTable.BonusGameData.State != 1{
		return errCode.ServerError.GetI18nMap(), nil
	}
	table.PutQueue(protocol.SelectBonusSymbol,session)
	return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}

func (self *Room) SelectMiniSymbol(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	if msg["Serial"] == nil{
		return errCode.ServerError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil{
		self.tablesID.Range(func(key, value interface{}) bool {
			table := self.room.GetTable(value.(string)) //
			if table != nil{
				myTable := (table.(interface{})).(*MyTable)
				if myTable.PlayerIsTable(userID){
					tableID = value.(string)
					return false
				}
			}
			return true
		})
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	if myTable.MiniGameData.State != 1{
		return errCode.ServerError.GetI18nMap(), nil
	}
	table.PutQueue(protocol.SelectMiniSymbol,session,msg)
	return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}
func (self *Room) EnterMiniGame(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil{
		self.tablesID.Range(func(key, value interface{}) bool {
			table := self.room.GetTable(value.(string)) //
			if table != nil{
				myTable := (table.(interface{})).(*MyTable)
				if myTable.PlayerIsTable(userID){
					tableID = value.(string)
					return false
				}
			}
			return true
		})
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	if myTable.MiniGameData.State != 1{
		return errCode.ServerError.GetI18nMap(), nil
	}

	res := make(map[string]interface{})

	myTable.GameConf = slotSexStorage.GetRoomConf()
	res["XiaZhu"] = myTable.MiniGameData.TotalSymbolScore
	res["Get"] =  myTable.MiniGameData.TotalSymbolScore * (2000 - int64(myTable.GameConf.ProfitPerThousand) * 2) / 1000
	myTable.CountDown = myTable.GameConf.BonusTimeOut
	myTable.BonusGameData.State = 1
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) onDisconnect(userID string) (map[string]interface{}, error){
	var tableID string
	self.tablesID.Range(func(key, value interface{}) bool {
		table := self.room.GetTable(value.(string)) //
		if table != nil{
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID){
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
	if !myTable.IsInFreeGame(){
		myTable.PutQueue(protocol.ClearTable)
	}
	log.Info("----------------slot disconnect----------")
	return errCode.Success(nil).GetMap(), nil
}
func (self *Room) CheckPlayerInGame(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	userID := session.GetUserID()
	res := make(map[string]interface{},2)
	res["InGame"] = false
	self.tablesID.Range(func(key, value interface{}) bool {
		table := self.room.GetTable(value.(string)) //
		if table != nil{
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(userID){
				res["InGame"] = true
				return false
			}
		}
		return true
	})
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) SwitchMode(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	if msg["modeType"] == nil{
		return errCode.ServerError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	var tableID string
	if msg["tableID"] == nil{
		self.tablesID.Range(func(key, value interface{}) bool {
			table := self.room.GetTable(value.(string)) //
			if table != nil{
				myTable := (table.(interface{})).(*MyTable)
				if myTable.PlayerIsTable(userID){
					tableID = value.(string)
					return false
				}
			}
			return true
		})
	}
	table := self.room.GetTable(tableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := table.(*MyTable)
	modeType := ModeType(msg["modeType"].(string))
	myTable.ModeType = modeType
	if modeType == TRIAL{
		initSymbol := make([]int64, len(myTable.ReelsListTrial))
		myTable.JieSuanData = JieSuanData{}
		for k,v := range myTable.ReelsListTrial{
			rand := myTable.RandInt64(1,int64(len(v) + 1))
			rand = rand -1
			initSymbol[k] = rand
		}
		myTable.TrialModeConf = TrialModeConf{
			VndBalance:    200000000,
		}
		myTable.JieSuanData.TrialData = TrialData{
			VndBalance: myTable.TrialModeConf.VndBalance,
		}
		res := make(map[string]interface{},3)
		res["TrialConf"] = myTable.TrialModeConf
		res["ReelsListTrial"] = myTable.ReelsListTrial
		res["InitSymbol"] = initSymbol
		return errCode.Success(res).GetMap(), nil
	}else if modeType == NORMAL{
		myTable.ModeType = NORMAL
		myTable.JieSuanData = JieSuanData{}
		initSymbol := make([]int64, len(myTable.ReelsList))
		for k,v := range myTable.ReelsList{
			rand := myTable.RandInt64(1,int64(len(v) + 1))
			rand = rand -1
			initSymbol[k] = rand
		}
		res := make(map[string]interface{},1)
		res["InitSymbol"] = initSymbol
		return errCode.Success(res).GetMap(), nil
	}

	return errCode.Success(nil).GetMap(), nil
}
func (self *Room) GetJackpotRecord(session gate.Session,params map[string]interface{}) (map[string]interface{},error) {
	_, err := utils.CheckParams2(params, []string{"Offset","PageSize"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	Offset,_ := utils.ConvertInt(params["Offset"])
	PageSize,_ := utils.ConvertInt(params["PageSize"])
	res := make(map[string]interface{},2)
	record:= slotStorage.QueryJackpotRecord(int(Offset),int(PageSize),game.SlotSex)
	res["JackpotRecord"] = record
	res["TotalNum"] = slotStorage.QueryJackpotRecordTotal(game.SlotSex)
	return  errCode.Success(res).GetMap(),nil
}