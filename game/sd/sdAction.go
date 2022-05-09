package sd

import (
	"github.com/pkg/errors"
	"sort"
	"strconv"
	"time"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/activityStorage"
	"vn/storage/gameStorage"
	"vn/storage/sdStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)
func (this *MyTable) Empty(){
	log.Info("22222222")
}
func (this *MyTable) GetEnterData(session gate.Session, msg map[string]interface{})  (err error){
	player := &room.BasePlayerImp{}
	player.Bind(session)
	player.OnRequest(session)
	userID := session.GetUserID()
	this.Players[userID] = player
	if userID == ""{
		log.Info("your userid is empty")
		return nil
	}
	idx := this.GetPlayerIdx(userID)
	if idx < 0 {
		return nil
	}
	tableInfoRet := this.GetTableInfo(true)
	playerInfoRet := this.GetPlayerInfo(userID,true)
	info := make(map[string]interface{})
	info["PlayerNum"] = this.PlayerNum
	this.sendPackToAll(game.Push, info,protocol.UpdatePlayerNum,nil)
	_ = this.sendPack(session.GetSessionID(),game.Push,tableInfoRet,protocol.UpdateTableInfo,nil)
	_ = this.sendPack(session.GetSessionID(),game.Push,playerInfoRet,protocol.UpdatePlayerInfo,nil)
	_ = this.sendPack(session.GetSessionID(),game.Push,this.PositionList,protocol.UpdatePlayerList,nil)

	delete(this.DisConnectList,userID)
	return nil
}
func (this *MyTable) Enter(session gate.Session, msg map[string]interface{})  (err error) {
	//start := time.Now().UnixNano()
	if this.RoomState == ROOM_END{
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.Enter,error)
		return nil
	}
	player := &room.BasePlayerImp{}
	player.Bind(session)
	player.OnRequest(session)
	userID := session.GetUserID()
	if !this.BroadCast{
		this.BroadCast = true
	}
	if userID == ""{
		log.Info("your userid is empty")
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.Enter,error)
		return nil
	}
	this.Players[userID] = player
	tableInfo := sdStorage.GetTableInfo(this.tableID)
	idx := this.GetPlayerIdx(userID)
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	if idx >= 0 {
		if user.Type != userStorage.TypeNormal{
			this.PlayerList[idx].Role = Agent
		}
	//	log.Info("you already in room")
		ret := make(map[string]interface{},2)
		ret["ServerID"] = tableInfo.ServerID
		ret["TableID"] = this.tableID
		this.sendPack(session.GetSessionID(),game.Push,ret,protocol.Enter,nil)
		delete(this.DisConnectList,userID)
		return nil
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))

	if msg["BaseScore"] != nil {//说明是创建人进来的
		tableInfo.BaseScore = msg["BaseScore"].(int64)
		tableInfo.MinEnterTable = msg["MinEnterTable"].(int64)
		tableInfo.TotalPlayerNum = this.GameConf.SelfTablePlayerLimit
		this.ChipsList = this.GameConf.PlayerChipsList[strconv.FormatInt(tableInfo.BaseScore,10)]
		if msg["GenerateRecord"] != nil{ //随机生成路单
			resultsRecord := sdStorage.GetResultsRecord(this.tableID)
			total := this.RandInt64(1,int64(resultsRecord.ResultsRecordNum + 1))
			for i := 0;i < int(total);i++{
				result := sdStorage.XiaZhuResult(strconv.FormatInt(this.RandInt64(1,3),10))
				resultsRecord.Results = append(resultsRecord.Results,result)
			}
			idx := len(resultsRecord.Results) - resultsRecord.ResultsRecordNum
			if idx > 0{
				resultsRecord.Results = append(resultsRecord.Results[:0],resultsRecord.Results[idx:]...)
			}
			resultsRecord.SingleNum = 0
			resultsRecord.DoubleNum = 0
			for _,v := range resultsRecord.Results{
				if v == sdStorage.SINGLE{
					resultsRecord.SingleNum++
				}else{
					resultsRecord.DoubleNum++
				}
			}
			sdStorage.UpsertResultsRecord(resultsRecord,this.tableID)
		}
	}

	if wallet.VndBalance < tableInfo.MinEnterTable{
		error := errCode.BalanceNotEnough
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.Enter,error)
		return nil
	}

	pl := PlayerList{
		//session: session,
		Yxb: wallet.VndBalance,
		UserID: userID,
		XiaZhuResult: map[sdStorage.XiaZhuResult][]int64{},
		LastXiaZhuResult:map[sdStorage.XiaZhuResult][]int64{},
		XiaZhuResultTotal: map[sdStorage.XiaZhuResult]int64{
			sdStorage.SINGLE: 0,
			sdStorage.DOUBLE: 0,
			sdStorage.Red4White0: 0,
			sdStorage.Red0White4: 0,
			sdStorage.Red3White1: 0,
			sdStorage.Red1White3: 0,
		},
		Name: user.NickName,
		Head: user.Avatar,
		Role: USER,
	}

	if user.Type != userStorage.TypeNormal{
		pl.Role = Agent
	}
	this.PlayerList = append(this.PlayerList,pl)

	this.PlayerNum = len(this.PlayerList)
	tableInfo.PlayerNum = this.PlayerNum

	sdStorage.UpsertTableInfo (tableInfo,this.tableID)

	sort.Slice(this.PlayerList, func(i, j int) bool { //排序
		return this.PlayerList[i].Yxb > this.PlayerList[j].Yxb
	})
	if this.PlayerNum < this.SeatNum && pl.Yxb >= this.PlayerList[this.PlayerNum - 1].Yxb{ //如果金币足够到相应位置，就刷新位置
		this.UpdatePlayerList()
	}

	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)
	ret := make(map[string]interface{},2)
	ret["ServerID"] = tableInfo.ServerID
	ret["TableID"] = this.tableID
	this.sendPack(session.GetSessionID(),game.Push,ret,protocol.Enter,nil)

	delete(this.DisConnectList,userID)
	gameStorage.UpsertGameReconnect(userID,this.module.GetServerID())
	return nil
}

func (this *MyTable) DealXiaZhu(session gate.Session,xiaZhuV int64,xiaZhuPos sdStorage.XiaZhuResult,idx int,userID string){
	val,ok := this.XiaZhuTotal.Load(xiaZhuPos)
	if ok{
		total,_ := utils.ConvertInt(val)
		total += xiaZhuV
		this.XiaZhuTotal.Store(xiaZhuPos,total)
	}
	if this.PlayerList[idx].Role == USER{
		this.RealXiaZhuTotal[xiaZhuPos] += xiaZhuV //真实玩家下注
	}

	this.PlayerList[idx].XiaZhuResult[xiaZhuPos] = append(this.PlayerList[idx].XiaZhuResult[xiaZhuPos],xiaZhuV)
	this.PlayerList[idx].XiaZhuResultTotal[xiaZhuPos] += xiaZhuV
	this.PlayerList[idx].DoubleState = true  //开启加倍下注按钮
	activityStorage.UpsertGameDataInBet(session.GetUserID(),game.SeDie,1)
}
func (this *MyTable) XiaZhu(session gate.Session, msg map[string]interface{}) (err error) {
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)
	if this.RoomState != ROOM_WAITING_XIAZHU {
		log.Info("----------------room state not xia zhu roomstate = %d",this.RoomState)
		error := errCode.CurCanXiaZhuError
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return nil
	}
	userID := session.GetUserID()
	idx := this.GetPlayerIdx(userID)
	if idx == -1{
		error := errCode.NotInRoomError
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return nil
	}
	xiaZhuV,_ := strconv.ParseInt(msg["num"].(string), 10, 64)
	xiaZhuPos := sdStorage.XiaZhuResult(msg["pos"].(string)) //鱼虾蟹下的哪一种图案
	if (xiaZhuPos == sdStorage.SINGLE && this.PlayerList[idx].XiaZhuResultTotal[sdStorage.DOUBLE] != 0) || (xiaZhuPos == sdStorage.DOUBLE && this.PlayerList[idx].XiaZhuResultTotal[sdStorage.SINGLE] != 0){
		error := errCode.DxGameBetErr
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return nil
	}
	if  xiaZhuPos != sdStorage.SINGLE &&
		xiaZhuPos != sdStorage.DOUBLE &&
		xiaZhuPos != sdStorage.Red4White0 &&
		xiaZhuPos != sdStorage.Red0White4 &&
		xiaZhuPos != sdStorage.Red3White1 &&
		xiaZhuPos != sdStorage.Red1White3{
		log.Info("Xia zhu pos not correct")
		return nil
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	if wallet.VndBalance < xiaZhuV{
		log.Info("------------------------- player yxb not enough yxb = %d,num = %d",wallet.VndBalance,xiaZhuV)
		error := errCode.BalanceNotEnough
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return nil
	}
	bill := walletStorage.NewBill(userID,walletStorage.TypeExpenses,walletStorage.EventGameSd,this.EventID,-xiaZhuV)
	walletStorage.OperateVndBalance(bill)
	this.notifyWallet(userID)
	this.PlayerList[idx].Yxb = wallet.VndBalance - xiaZhuV
	this.DealXiaZhu(session,xiaZhuV,xiaZhuPos,idx,userID)

	info := struct {
		UserID string
		XiaZhuPos sdStorage.XiaZhuResult
		XiaZhuV int64
	}{
		UserID: userID,
		XiaZhuPos: xiaZhuPos,
		XiaZhuV: xiaZhuV,
	}
	_ = this.sendPackToAll(game.Push,info,protocol.XiaZhu,nil)

	tableInfoRet := this.GetTableInfo(false)
	playerInfoRet := this.GetPlayerInfo(userID,false)

	_ = this.sendPackToAll(game.Push,tableInfoRet,protocol.UpdateTableInfo,nil)
	_ = this.sendPack(session.GetSessionID(),game.Push,playerInfoRet,protocol.UpdatePlayerInfo,nil)
	return nil
}

func (this *MyTable) LastXiaZhu(session gate.Session, msg map[string]interface{}) error{
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)
	if this.RoomState != ROOM_WAITING_XIAZHU {
		log.Info("----------------room state not xia zhu roomstate = %d",this.RoomState)
		error := errCode.CurCanXiaZhuError
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return nil
	}
	userID := session.GetUserID()
	idx := this.GetPlayerIdx(userID)
	if idx == -1{
		error := errCode.NotInRoomError
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return nil
	}

	if !this.PlayerList[idx].LastState{
		log.Info("----------------LastState not true-----")
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return nil
	}
	var needYxb int64 = 0
	for _,v := range this.PlayerList[idx].LastXiaZhuResult{
		for _,v1 := range v{
			needYxb += v1
		}
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	if wallet.VndBalance < needYxb{
		log.Info("------------------------- player yxb not enough yxb = %d,num = %d",wallet.VndBalance,needYxb)
		error := errCode.BalanceNotEnough
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return nil
	}
	var xiaZhuV int64 = 0
	for k,v := range this.PlayerList[idx].LastXiaZhuResult{
		for _,v1 := range v{
			if v1 > 0{
				this.DealXiaZhu(session,v1,k,idx,userID)
				xiaZhuV += v1
			}
		}
	}
	info := struct {
		UserID string
		XiaZhuResult map[sdStorage.XiaZhuResult][]int64
	}{
		UserID: userID,
		XiaZhuResult:this.PlayerList[idx].LastXiaZhuResult,
	}
	_ = this.sendPackToAll(game.Push,info,protocol.LastXiaZhu,nil)

	bill := walletStorage.NewBill(userID,walletStorage.TypeExpenses,walletStorage.EventGameSd,this.EventID,-xiaZhuV)
	walletStorage.OperateVndBalance(bill)
	this.notifyWallet(userID)
	this.PlayerList[idx].Yxb = wallet.VndBalance - xiaZhuV

	tableInfoRet := this.GetTableInfo(false)
	playerInfoRet := this.GetPlayerInfo(userID,false)

	_ = this.sendPackToAll(game.Push,tableInfoRet,protocol.UpdateTableInfo,nil)
	_ = this.sendPack(session.GetSessionID(),game.Push,playerInfoRet,protocol.UpdatePlayerInfo,nil)
	return nil
}
func (this *MyTable) DoubleXiaZhu(session gate.Session, msg map[string]interface{}) (err error) {
	player := this.FindPlayer(session)
	if player == nil {
		return errors.New("no join")
	}
	player.OnRequest(session)
	if this.RoomState != ROOM_WAITING_XIAZHU {
		log.Info("----------------room state not xia zhu roomstate = %d",this.RoomState)
		error := errCode.CurCanXiaZhuError
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return nil
	}

	userID := session.GetUserID()
	idx := this.GetPlayerIdx(userID)
	if idx == -1{
		return errors.New("no idx")
	}
	if !this.PlayerList[idx].DoubleState{
		log.Info("----------------DoubleState not true-----")
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return
	}

	var needYxb int64 = 0
	for _,v := range this.PlayerList[idx].XiaZhuResult{
		for _,v1 := range v{
			if v1 > 0{
				needYxb += v1
			}
		}
	}
	//needYxb *= 2
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	if wallet.VndBalance < needYxb{
		log.Info("------------------------- player yxb not enough yxb = %d,num = %d",wallet.VndBalance,needYxb)
		error := errCode.BalanceNotEnough
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.XiaZhu,error)
		return
	}
	var XiaZhuResult map[sdStorage.XiaZhuResult][]int64
	XiaZhuResult = map[sdStorage.XiaZhuResult][]int64{}
	if needYxb > 0{ //当前局有下注，就是当前局的下注
		for k,v := range this.PlayerList[idx].XiaZhuResult{ //当前局有下注，就是当前局的下注
			for _,v1 := range v{
				if v1 > 0{
					this.DealXiaZhu(session,v1,k,idx,userID) //双倍下注
					//this.DealXiaZhu(session,v1,k,idx,userID)
					//XiaZhuResult[k] = append(XiaZhuResult[k],v1)
					XiaZhuResult[k] = append(XiaZhuResult[k],v1)
				}
			}
		}
	}else{ //否则就是上一轮的下注
		for _,v := range this.PlayerList[idx].LastXiaZhuResult{
			for _,v1 := range v{
				if v1 > 0{
					needYxb += v1
				}
			}
		}
		//needYxb *= 2
		if wallet.VndBalance < needYxb{
			log.Info("------------------------- player yxb not enough yxb = %d,num = %d",wallet.VndBalance,needYxb)
			return
		}
		for k,v := range this.PlayerList[idx].LastXiaZhuResult{
			for _,v1 := range v{
				if v1 > 0{
					this.DealXiaZhu(session,v1,k,idx,userID) //双倍下注
					//this.DealXiaZhu(session,v1,k,idx,userID)
					//XiaZhuResult[k] = append(XiaZhuResult[k],v1)
					XiaZhuResult[k] = append(XiaZhuResult[k],v1)
				}
			}
		}
	}
	info := struct {
		UserID string
		XiaZhuResult map[sdStorage.XiaZhuResult][]int64
	}{
		UserID: userID,
		XiaZhuResult:XiaZhuResult,
	}
	_ = this.sendPackToAll(game.Push,info,protocol.DoubleXiaZhu,nil)
	bill := walletStorage.NewBill(userID,walletStorage.TypeExpenses,walletStorage.EventGameSd,this.EventID,-needYxb)
	walletStorage.OperateVndBalance(bill)
	this.notifyWallet(userID)
	this.PlayerList[idx].Yxb = wallet.VndBalance - needYxb

	tableInfoRet := this.GetTableInfo(false)
	playerInfoRet := this.GetPlayerInfo(userID,false)

	_ = this.sendPackToAll(game.Push,tableInfoRet,protocol.UpdateTableInfo,nil)
	_ = this.sendPack(session.GetSessionID(),game.Push,playerInfoRet,protocol.UpdatePlayerInfo,nil)
	return nil
}

func (this *MyTable) GetResultsRecord(session gate.Session, msg map[string]interface{}) (res interface{},err map[string]interface{}) {
	player := this.FindPlayer(session)
	if player == nil {
		return nil,errCode.ServerError.SetKey().GetMap()
	}
	player.OnRequest(session)
	resultsRecord := sdStorage.GetResultsRecord(this.tableID)

	return resultsRecord,nil
}

func (this *MyTable) GetPlayerList(session gate.Session, msg map[string]interface{}) (res interface{},err map[string]interface{}) {
	player := this.FindPlayer(session)
	if player == nil {
		return nil,errCode.ServerError.SetKey().GetMap()
	}
	player.OnRequest(session)
	var list = this.PlayerList

	sort.Slice(list, func(i, j int) bool { //排序
		return list[i].Yxb > list[j].Yxb
	})

	type PlayerList struct {
		Name string
		Yxb int64
		Head string
	}
	var playerList []PlayerList
	for _,v := range list{
		pl := PlayerList{
			Name: v.Name,
			Yxb: v.Yxb,
			Head: v.Head,
		}
		playerList = append(playerList,pl)
	}
	info := struct {
		PlayerList []PlayerList
	}{
		PlayerList: playerList,
	}

	return info,nil
}
func (this *MyTable) QuitTable(userID string,reLogin bool) (res interface{},err map[string]interface{}) {
	idx := this.GetPlayerIdx(userID)
	var session gate.Session
	if this.PlayerList[idx].Role != ROBOT{
		session = this.Players[userID].Session()
	}
	sb := vGate.QuerySessionBean(userID)
	tableInfo := sdStorage.GetTableInfo(this.tableID)
	if idx == -1{
		if sb != nil{
			this.sendPack(session.GetSessionID(),game.Push,"",protocol.QuitTable,errCode.ServerError)
		}
		return nil,nil
	}

	if reLogin{
		this.DisConnectList[userID] = true
	}
	if this.RoomState == ROOM_WAITING_XIAZHU{ //下注状态不能退出房间
		for _,v := range this.PlayerList[idx].XiaZhuResult{
			for _,v1 := range v{
				if v1 > 0 {
					log.Info("cant leave room")
					if sb != nil {
						this.sendPack(session.GetSessionID(), game.Push, "", protocol.QuitTable, errCode.XiaZhuCantQuit)
					}
					return nil,nil
				}
			}
		}
	}
	sort.Slice(this.PlayerList, func(i, j int) bool { //排序
		return this.PlayerList[i].Yxb > this.PlayerList[j].Yxb
	})
	idx = this.GetPlayerIdx(userID)
	this.PlayerList = append(this.PlayerList[:idx], this.PlayerList[idx+1:]...)
	if idx <= this.SeatNum{ //如果金币足够到相应位置，就刷新位置
		this.UpdatePlayerList()
	}
	this.PlayerNum = len(this.PlayerList)
	tableInfo.PlayerNum = this.PlayerNum
	sdStorage.UpsertTableInfo(tableInfo,this.tableID)



	ret := this.GetTableInfo(false)

	info := make(map[string]interface{})
	info["PlayerNum"] = tableInfo.PlayerNum
	this.sendPackToAll(game.Push, info,protocol.UpdatePlayerNum,nil)
	if sb != nil {
		ret := this.DealProtocolFormat("",protocol.QuitTable,nil)
		this.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
		this.onlinePush.ExecuteCallBackMsg(this.Trace())
	}
	//delete(this.ShortCut,userID)
	delete(this.Players,userID)
	if tableInfo.PlayerNum <= 0 && !tableInfo.Hundred{  //私人房解散
		this.RoomState = ROOM_END
	}
	delete(this.DisConnectList,userID)
	gameStorage.RemoveReconnectByUid(userID)
	return ret,nil
}
func (this *MyTable) Disconnect(session gate.Session, msg map[string]interface{}) (err error) {
	player := this.FindPlayer(session)
	if player == nil {
		return errors.New("no join")
	}
	player.OnRequest(session)

	return nil
}
//func (this *MyTable) GetShortCutList(session gate.Session, msg map[string]interface{}) (err error) {
//	userID := session.GetUserID()
//	idx := this.GetPlayerIdx(userID)
//	if idx == -1{
//		return errors.New("no idx")
//	}
//	if this.ShortCut[userID] == nil{
//		this.ShortCut[userID] = game.ShortCut[game.SeDie]
//	}
//	if this.PlayerList[idx].Yxb < int64(this.GameConf.ShortYxbLimit){
//		error := errCode.ChatYxbLimitError
//		this.sendPack(session.GetSessionID(),game.Push,"",protocol.GetShortCutList,error)
//		return nil
//	}
//	this.sendPack(session.GetSessionID(),game.Push,this.ShortCut[userID],protocol.GetShortCutList,nil)
//	return nil
//}
//func (this *MyTable) SendShortCut(session gate.Session, msg map[string]interface{}) (err error) {
//	userID := session.GetUserID()
//	idx := this.GetPlayerIdx(userID)
//	if idx == -1{
//		return errors.New("no idx")
//	}
//	Interval := time.Now().Unix() - this.PlayerList[idx].LastChatTime.Unix()
//	if Interval < int64(this.GameConf.ShortCutInterval){//间隔太短
//		error := errCode.TimeIntervalError
//		this.sendPack(session.GetSessionID(),game.Push,"",protocol.SendShortCut,error)
//		return nil
//	}
//	this.PlayerList[idx].LastChatTime = time.Now()
//	if this.ShortCut[userID] == nil{
//		this.ShortCut[userID] = game.ShortCut[game.SeDie]
//	}
//	Type := msg["Type"].(string)
//	Text := msg["Text"].(string)
//	if Type == "Private"{
//		sc := game.ShortCutMode{
//			Type: Type,
//			Text: Text,
//		}
//		lenSystem := len(game.ShortCut[game.SeDie])
//		this.ShortCut[userID] = append(this.ShortCut[userID],sc)
//		PrivateLen := len(this.ShortCut[userID]) - lenSystem - this.GameConf.ShortCutPrivate
//		if PrivateLen > 0{
//			this.ShortCut[userID] = append(this.ShortCut[userID][:lenSystem],this.ShortCut[userID][lenSystem + PrivateLen:]...)
//		}
//	}
//	msg["UserID"] = userID
//	_ = this.sendPackToAll(game.Push,msg,protocol.SendShortCut,nil)
//	return nil
//}
func (this *MyTable) SendShortCut(session gate.Session, msg map[string]interface{}) (err error) {
	userID := session.GetUserID()
	idx := this.GetPlayerIdx(userID)
	if idx == -1{
		return errors.New("no idx")
	}
	Interval := time.Now().Unix() - this.PlayerList[idx].LastChatTime.Unix()
	if Interval < int64(this.GameConf.ShortCutInterval){//间隔太短
		error := errCode.TimeIntervalError
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.SendShortCut,error)
		return nil
	}
	this.PlayerList[idx].LastChatTime = time.Now()

	msg["UserId"] = userID
	_ = this.sendPackToAll(game.Push,msg,protocol.SendShortCut,nil)
	return nil
}