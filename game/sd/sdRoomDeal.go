package sd

import (
	"fmt"
	"math/rand"
	"sort"
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
	vGate "vn/gate"
	"vn/storage/gameStorage"
	"vn/storage/sdStorage"
	"vn/storage/walletStorage"
)

func (self *Room) RoomInit(){
	gameStorage.UpsertGameReboot(game.SeDie,"false")
		roomData := sdStorage.RoomData{
			//Room: self.room,
			TablesInfo: map[string]sdStorage.TableInfo{
				//"100":{
				//	TableID: "100",
				//	ServerID: self.GetServerID(),
				//	BaseScore: 100,
				//	MinEnterTable: 1000,
				//	TotalPlayerNum: 9,
				//	Hundred: false,
				//	PlayerNum: 0,
				//},
				//"500":{
				//	TableID: "500",
				//	ServerID: self.GetServerID(),
				//	BaseScore: 500,
				//	MinEnterTable: 5000,
				//	TotalPlayerNum: 9,
				//	Hundred: false,
				//	PlayerNum: 0,
				//},
				//"1000":{
				//	TableID: "1000",
				//	ServerID: self.GetServerID(),
				//	BaseScore: 1000,
				//	MinEnterTable: 10000,
				//	TotalPlayerNum: 9,
				//	Hundred: false,
				//	PlayerNum: 0,
				//},
				//"2000":{
				//	TableID: "2000",
				//	ServerID: self.GetServerID(),
				//	BaseScore: 2000,
				//	MinEnterTable: 20000,
				//	TotalPlayerNum: 9,
				//	Hundred: false,
				//	PlayerNum: 0,
				//},
				//"5000":{
				//	TableID: "5000",
				//	ServerID: self.GetServerID(),
				//	BaseScore: 5000,
				//	MinEnterTable: 50000,
				//	TotalPlayerNum: 9,
				//	Hundred: false,
				//	PlayerNum: 0,
				//},
				//"10000":{
				//	TableID: "10000",
				//	ServerID: self.GetServerID(),
				//	BaseScore: 10000,
				//	MinEnterTable: 100000,
				//	TotalPlayerNum: 9,
				//	Hundred: false,
				//	PlayerNum: 0,
				//},
				//"20000":{
				//	TableID: "20000",
				//	ServerID: self.GetServerID(),
				//	BaseScore: 20000,
				//	MinEnterTable: 200000,
				//	TotalPlayerNum: 9,
				//	Hundred: false,
				//	PlayerNum: 0,
				//},
				//"50000":{
				//	TableID: "50000",
				//	ServerID: self.GetServerID(),
				//	BaseScore: 50000,
				//	MinEnterTable: 500000,
				//	TotalPlayerNum: 9,
				//	Hundred: false,
				//	PlayerNum: 0,
				//},
			}, //
		}
		sdStorage.UpsertTablesInfo(roomData.TablesInfo)
	gameConf := sdStorage.GetRoomConf()
	if gameConf == nil{
		gameConf = &sdStorage.Conf{
			ProfitPerThousand:20, //系统抽水 2%
			BotProfitPerThousand:80, //机器人抽水 8%
			XiaZhuTime : 15,//下注时间
			JieSuanTime : 6,		//结算时间
			ReadyGameTime :3,		 //摇盆时间
			KickRoomCnt	: 5,	//连续三轮不下注，踢出房间
			TableBaseList:[]int{100,500,1000,2000,5000,10000,20000,50000},//房间底分列表
			HundredRoomNum:3, //百人房的数量
			SelfTablePlayerLimit:9,
			ShortCutPrivate: 3,
			ShortCutInterval: 3,
			ShortYxbLimit: 20000,
			PlayerChipsList: map[string][]int64{
				"0":{100,500,1000,5000,10000,50000,100000,500000,1000000},//玩家筹码列表
				"1":{100,500,1000,5000,10000,50000,100000,500000},//玩家筹码列表
				"2":{10000,50000,100000,500000,1000000,5000000,10000000},//玩家筹码列表
				"100":{100,200,500,1000},//玩家筹码列表
				"500":{500,1000,2000,5000},//玩家筹码列表
				"1000":{1000,2000,5000,10000},//玩家筹码列表
				"2000":{2000,5000,10000,20000},//玩家筹码列表
				"5000":{5000,10000,20000,50000},//玩家筹码列表
				"10000":{10000,20000,50000,100000},//玩家筹码列表
				"20000":{20000,50000,100000,200000},//玩家筹码列表
				"50000":{50000,100000,200000,500000},//玩家筹码列表
			},
			OddsList: map[sdStorage.XiaZhuResult]int64{
				sdStorage.SINGLE: 1,
				sdStorage.DOUBLE: 1,
				sdStorage.Red4White0:15,
				sdStorage.Red3White1: 3,
				sdStorage.Red1White3: 3,
				sdStorage.Red0White4: 15,
			},
		}
		sdStorage.InsertRoomConf(gameConf)
		time.Sleep(time.Second)
	}
	for i := 0;i < gameConf.HundredRoomNum;i++{
		self.CreateTable(strconv.Itoa(i))
		time.Sleep(time.Millisecond * 200)
	}

	self.tablesID.Range(func(key, value interface{}) bool { //启动table队列
		self.room.GetTable(value.(string))
		return true
	})
}
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
		tableID = int(RandInt64(1,1000000))
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
func (self *Room) TableQueue(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	action := msg["action"].(string)
	var tableID string
	userID := session.GetUserID()
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
	table := self.room.GetTable(tableID) //(self.curTableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	erro := table.PutQueue(action, session, msg)
	if erro != nil {
		log.Info("--------------- table.PutQueue error---tableID = %s",tableID,"---error = %s",erro)
		return errCode.ServerError.GetI18nMap(), nil
	}
	return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}
func (self *Room) GetShortCutList(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	var tableID string
	userID := session.GetUserID()
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
	table := self.room.GetTable(tableID) //(self.curTableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	erro := table.PutQueue(protocol.GetShortCutList, session, msg)
	if erro != nil {
		log.Info("--------------- table.PutQueue error---tableID = %s",tableID,"---error = %s",erro)
		return errCode.ServerError.GetI18nMap(), nil
	}
	return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}
func (self *Room) SendShortCut(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	var tableID string
	userID := session.GetUserID()
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
	table := self.room.GetTable(tableID) //(self.curTableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	erro := table.PutQueue(protocol.SendShortCut, session, msg)
	if erro != nil {
		log.Info("--------------- table.PutQueue error---tableID = %s",tableID,"---error = %s",erro)
		return errCode.ServerError.GetI18nMap(), nil
	}
	return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}
func (self *Room) QuitTable(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	var tableID string
	userID := session.GetUserID()
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
	table := self.room.GetTable(tableID) //(self.curTableID)
	if table != nil{
		erro := table.PutQueue(protocol.QuitTable, userID,false)
		if erro != nil {
			log.Info("--------------- table.PutQueue error---tableID = %s",tableID,"---error = %s",erro)
			return errCode.ServerError.GetI18nMap(), nil
		}
		return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
	}else{
		return  errCode.Success("").GetMap(),nil
	}

}
func (self *Room) GetEnterData(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	var tableID string
	userID := session.GetUserID()
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
	table := self.room.GetTable(tableID) //(self.curTableID)
	if table == nil {
		log.Info("--------------- table not exist---tableID = %s",tableID)
		return errCode.ServerError.GetI18nMap(), nil
	}
	erro := table.PutQueue(protocol.GetEnterData, session, msg)
	if erro != nil {
		log.Info("--------------- table.PutQueue error---tableID = %s",tableID,"---error = %s",erro)
		return errCode.ServerError.GetI18nMap(), nil
	}
	return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}
func (self *Room) Enter(session gate.Session, msg map[string]interface{}) (map[string]interface{},error) {
	//	table_id := msg["table_id"].(string)
	action := msg["action"].(string)
	tableID := msg["TableID"].(string)

	tableIDInt,_ := utils.ConvertInt(tableID)
	if tableIDInt < 100000 && tableIDInt >= 100 { //创建房间
		tableInfo := sdStorage.GetTableInfo(tableID)
		msg := make(map[string]interface{})
		msg["BaseScore"] = tableInfo.BaseScore
		msg["MinEnterTable"] = tableInfo.MinEnterTable
		msg["GenerateRecord"] = true
		return self.CreateTableReq(session,msg)
	}else{
		table := self.room.GetTable(tableID)
		if table == nil {
			log.Info("--------------- table not exist---tableID = %s",tableID)
			return errCode.ServerError.GetI18nMap(), nil
		}
		myTable := (table.(interface{})).(*MyTable)
		if !myTable.Hundred{
			if myTable.PlayerNum >= myTable.GameConf.SelfTablePlayerLimit{
				tableInfo := sdStorage.GetTableInfo(tableID)
				msg := make(map[string]interface{})
				msg["BaseScore"] = tableInfo.BaseScore
				msg["MinEnterTable"] = tableInfo.MinEnterTable
				msg["GenerateRecord"] = true
				return self.CreateTableReq(session,msg)
			}
		}
		erro := table.PutQueue(action, session, msg)
		if erro != nil {
			log.Info("--------------- table.PutQueue error---tableID = %s",tableID,"---error = %s",erro)
			return errCode.ServerError.GetI18nMap(), nil
		}
	}
	return  errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}
func (self *Room) onLogin(uid string) (map[string]interface{}, error){
//	log.Info("--------------------------------------login success----------------")

	self.tablesID.Range(func(key, value interface{}) bool {
		table := self.room.GetTable(value.(string)) //
		if table != nil{
			myTable := (table.(interface{})).(*MyTable)
			if myTable.PlayerIsTable(uid){
				msg := make(map[string]interface{})
				sb := vGate.QuerySessionBean(uid)
				if sb == nil{
					log.Info("session is nil")
					return false
				}
				msg["ServerID"] = self.GetServerID()
				msg["TableID"] = myTable.tableID
				ret := myTable.DealProtocolFormat(msg,protocol.Reenter,nil)
				myTable.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
				myTable.onlinePush.ExecuteCallBackMsg(myTable.Trace())
				return false
			}
		}
		return true
	})
	return nil, nil
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
				log.Info("-------------aaa-----")
				res["TableInfo"] = myTable.GetTableInfo(true)
				res["SelfInfo"] = myTable.GetPlayerInfo(userID,true)
				res["PlayerInfo"] = myTable.PositionList
				return false
			}
		}
		return true
	})
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
	myTable.PutQueue(protocol.QuitTable,userID,true)
	return nil, nil
}
//func (self *Room) Info (session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
//	self.tablesID.Range(func(key, value interface{}) bool {
//		self.room.GetTable(value.(string))
//		return true
//	})
//	return errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
//}

func (self *Room) GetPlayerList(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
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
		log.Info("--------------- table not exist---")
		return errCode.ServerError.GetI18nMap(), nil
	}
	myTable := (table.(interface{})).(*MyTable)
	res,erro := myTable.GetPlayerList(session,msg)
	if res == nil{
		return erro,nil
	}
	return errCode.Success(res).GetMap(), nil
}

func (self *Room) GetTableList(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	tablesInfo := sdStorage.GetTablesInfo()
	type tableInfo struct {
		TableID string
		BaseScore int64
		MinEnterTable int64
		PlayerNum int
		TotalPlayerNum int
		ServerID string
		Hundred bool
	}
	var res []tableInfo
	res = []tableInfo{}
	for _,v := range tablesInfo{
		ti := tableInfo{
			TableID:v.TableID,
			BaseScore: v.BaseScore,
			MinEnterTable: v.MinEnterTable,
			PlayerNum: v.PlayerNum,
			TotalPlayerNum: v.TotalPlayerNum,
			ServerID: v.ServerID,
			Hundred: v.Hundred,
		}
		res = append(res,ti)
	}
	sort.Slice(res, func(i, j int) bool { //排序
		if res[i].BaseScore < res[j].BaseScore{
			return true
		}else if res[i].BaseScore == res[j].BaseScore {
			if res[i].PlayerNum <= res[j].PlayerNum{
				return true
			}else{
				return false
			}
		}else {
			return false
		}
	})
	return errCode.Success(res).GetMap(), nil
}

func (self *Room) GetBaseScoreList(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	userID := session.GetUserID()
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	gameConf := sdStorage.GetRoomConf()
	var res []int
	res = []int{}
	for _,v := range gameConf.TableBaseList{
		if wallet.VndBalance / 10 > int64(v){
			res = append(res,v)
		}else{
			break
		}
	}
	if len(res) == 0{ //余额不足
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	return errCode.Success(res).GetMap(), nil
}
func (self *Room) CreateTableReq(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	reboot := gameStorage.QueryGameReboot(game.SeDie)
	if "true" == reboot{ //准备重启
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	userID := session.GetUserID()
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))

	var baseScore int64 = 0
	if _, ok := msg["BaseScore"]; ok{
		baseScore,_ = utils.ConvertInt(msg["BaseScore"])
	}
	if wallet.VndBalance / 10 < baseScore || baseScore == 0{
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	table,_,tableID := self.CreateTable("")
	if table == nil{
		return errCode.CantCreateTableError.GetI18nMap(), nil
	}
	table = self.room.GetTable(tableID)
	tbl := make(map[string]interface{})
	tbl["BaseScore"] = baseScore
	tbl["MinEnterTable"] = baseScore * 10
	if msg["GenerateRecord"] != nil{
		tbl["GenerateRecord"] = msg["GenerateRecord"].(bool)
	}
	table.PutQueue(protocol.Enter,session,tbl)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}
func (self *Room) GetWinLoseRank(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	res := make(map[string]interface{},2)
	res["RankList"] = gameStorage.GetGameWinLoseRank(game.SeDie,20)
	return errCode.Success(res).GetMap(),nil
}