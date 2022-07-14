package yxx

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
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
	"vn/storage/yxxStorage"
)

func (this *MyTable) Empty() {
	//log.Info("22222222")
}
func (this *MyTable) Enter(session gate.Session, msg map[string]interface{}) (err error) {
	//start := time.Now().UnixNano()
	if this.RoomState == ROOM_END {
		log.Info("----yxx---room end")
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.Enter, error)
		return nil
	}
	player := &room.BasePlayerImp{}
	player.Bind(session)

	player.OnRequest(session)
	userID := session.GetUserID()
	if !this.BroadCast {
		this.BroadCast = true
	}
	if userID == "" {
		log.Info("your userid is empty")
		return nil
	}
	this.Players[userID] = player
	idx := this.GetPlayerIdx(userID)
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	if idx >= 0 {
		if user.Type != userStorage.TypeNormal {
			this.PlayerList[idx].Role = Agent
		}
		this.PlayerList[idx].Session = session
		tableInfoRet := this.GetTableInfo(true)
		playerInfoRet := this.GetPlayerInfo(userID, true)
		info := make(map[string]interface{})
		info["PlayerNum"] = this.PlayerNum
		this.sendPackToAll(game.Push, info, protocol.UpdatePlayerNum, nil)
		_ = this.sendPack(session.GetSessionID(), game.Push, tableInfoRet, protocol.UpdateTableInfo, nil)
		_ = this.sendPack(session.GetSessionID(), game.Push, playerInfoRet, protocol.UpdatePlayerInfo, nil)
		_ = this.sendPack(session.GetSessionID(), game.Push, this.PositionList, protocol.UpdatePlayerList, nil)
		//log.Info("you already in room")
		return nil
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	pl := PlayerList{
		//session: session,
		Yxb:              wallet.VndBalance,
		UserID:           userID,
		XiaZhuResult:     map[yxxStorage.XiaZhuResult][]int64{},
		LastXiaZhuResult: map[yxxStorage.XiaZhuResult][]int64{},
		XiaZhuResultTotal: map[yxxStorage.XiaZhuResult]int64{
			yxxStorage.YU:   0,
			yxxStorage.XIA:  0,
			yxxStorage.XIE:  0,
			yxxStorage.JI:   0,
			yxxStorage.LU:   0,
			yxxStorage.HULU: 0,
		},
		Name:    user.NickName,
		Head:    user.Avatar,
		Role:    USER,
		Session: session,
	}
	if user.Type != userStorage.TypeNormal {
		pl.Role = Agent
	}

	this.PlayerList = append(this.PlayerList, pl)

	this.PlayerNum = len(this.PlayerList)

	sort.Slice(this.PlayerList, func(i, j int) bool { //排序
		return this.PlayerList[i].Yxb > this.PlayerList[j].Yxb
	})
	if this.PlayerNum < this.SeatNum && pl.Yxb >= this.PlayerList[this.PlayerNum-1].Yxb { //如果金币足够到相应位置，就刷新位置
		this.UpdatePlayerList()
	}
	tableInfoRet := this.GetTableInfo(true)
	playerInfoRet := this.GetPlayerInfo(userID, true)
	info := make(map[string]interface{})
	info["PlayerNum"] = this.PlayerNum
	this.sendPackToAll(game.Push, info, protocol.UpdatePlayerNum, nil)
	_ = this.sendPack(session.GetSessionID(), game.Push, tableInfoRet, protocol.UpdateTableInfo, nil)
	_ = this.sendPack(session.GetSessionID(), game.Push, playerInfoRet, protocol.UpdatePlayerInfo, nil)
	_ = this.sendPack(session.GetSessionID(), game.Push, this.PositionList, protocol.UpdatePlayerList, nil)

	gameStorage.UpsertGameReconnect(userID, this.module.GetServerID())
	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)
	return nil
}

func (this *MyTable) DealXiaZhu(session gate.Session, xiaZhuV int64, xiaZhuPos yxxStorage.XiaZhuResult, idx int, userID string) {
	this.XiaZhuTotal[xiaZhuPos] += xiaZhuV //桌子对应位置加下注
	if this.PlayerList[idx].Role == USER {
		this.RealXiaZhuTotal[xiaZhuPos] += xiaZhuV //真实玩家下注
	}

	this.ResultsChipList[xiaZhuPos] = append(this.ResultsChipList[xiaZhuPos], xiaZhuV) //记录下注筹码

	this.PlayerList[idx].XiaZhuResult[xiaZhuPos] = append(this.PlayerList[idx].XiaZhuResult[xiaZhuPos], xiaZhuV)
	this.PlayerList[idx].XiaZhuResultTotal[xiaZhuPos] += xiaZhuV
	this.PlayerList[idx].DoubleState = true //开启加倍下注按钮

	activityStorage.UpsertGameDataInBet(session.GetUserID(), game.YuXiaXie, 1)
}
func (this *MyTable) XiaZhu(session gate.Session, msg map[string]interface{}) (err error) {
	//start := time.Now().UnixNano()
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)
	if this.RoomState != ROOM_WAITING_XIAZHU {
		error := errCode.CurCanXiaZhuError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return nil
	}
	userID := session.GetUserID()
	idx := this.GetPlayerIdx(userID)
	if idx == -1 {
		error := errCode.NotInRoomError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return nil
	}
	xiaZhuV, _ := strconv.ParseInt(msg["num"].(string), 10, 64)
	xiaZhuPos := yxxStorage.XiaZhuResult(msg["pos"].(string)) //鱼虾蟹下的哪一种图案
	if xiaZhuPos != yxxStorage.HULU &&
		xiaZhuPos != yxxStorage.JI &&
		xiaZhuPos != yxxStorage.LU &&
		xiaZhuPos != yxxStorage.XIE &&
		xiaZhuPos != yxxStorage.XIA &&
		xiaZhuPos != yxxStorage.YU {
		log.Info("Xia zhu pos not correct pos = %s", xiaZhuPos)
		return nil
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	if wallet.VndBalance < xiaZhuV {
		error := errCode.BalanceNotEnough
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return nil
	}

	bill := walletStorage.NewBill(userID, walletStorage.TypeExpenses, walletStorage.EventGameYxx, this.EventID, -xiaZhuV)
	//start := time.Now().UnixNano()
	walletStorage.OperateVndBalance(bill)
	//end := time.Now().UnixNano()
	//log.Info("cost time 000 = %d",time.Duration(end -start) / time.Millisecond)
	this.notifyWallet(userID)
	this.PlayerList[idx].Yxb = wallet.VndBalance - xiaZhuV
	this.DealXiaZhu(session, xiaZhuV, xiaZhuPos, idx, userID)

	info := struct {
		UserID    string
		XiaZhuPos yxxStorage.XiaZhuResult
		XiaZhuV   int64
	}{
		UserID:    userID,
		XiaZhuPos: xiaZhuPos,
		XiaZhuV:   xiaZhuV,
	}
	_ = this.sendPackToAll(game.Push, info, protocol.XiaZhu, nil)
	tableInfoRet := this.GetTableInfo(false)
	playerInfoRet := this.GetPlayerInfo(userID, false)

	_ = this.sendPackToAll(game.Push, tableInfoRet, protocol.UpdateTableInfo, nil)
	_ = this.sendPack(session.GetSessionID(), game.Push, playerInfoRet, protocol.UpdatePlayerInfo, nil)
	//end = time.Now().UnixNano()
	//log.Info("cost time 111 = %d",time.Duration(end -start) / time.Millisecond)
	return nil
}
func (this *MyTable) LastXiaZhu(session gate.Session, msg map[string]interface{}) error {
	//start := time.Now().UnixNano()
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)
	if this.RoomState != ROOM_WAITING_XIAZHU {
		error := errCode.CurCanXiaZhuError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return nil
	}
	userID := session.GetUserID()
	idx := this.GetPlayerIdx(userID)
	if idx == -1 {
		error := errCode.NotInRoomError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return nil
	}

	if !this.PlayerList[idx].LastState {
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return nil
	}
	var needYxb int64 = 0
	for _, v := range this.PlayerList[idx].LastXiaZhuResult {
		for _, v1 := range v {
			needYxb += v1
		}
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	if wallet.VndBalance < needYxb {
		error := errCode.BalanceNotEnough
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return nil
	}
	var xiaZhuV int64 = 0
	for k, v := range this.PlayerList[idx].LastXiaZhuResult {
		for _, v1 := range v {
			if v1 > 0 {
				this.DealXiaZhu(session, v1, k, idx, userID)
				xiaZhuV += v1
			}
		}
	}
	info := struct {
		UserID       string
		XiaZhuResult map[yxxStorage.XiaZhuResult][]int64
	}{
		UserID:       userID,
		XiaZhuResult: this.PlayerList[idx].LastXiaZhuResult,
	}
	_ = this.sendPackToAll(game.Push, info, protocol.LastXiaZhu, nil)
	bill := walletStorage.NewBill(userID, walletStorage.TypeExpenses, walletStorage.EventGameYxx, this.EventID, -xiaZhuV)
	walletStorage.OperateVndBalance(bill)
	this.notifyWallet(userID)
	this.PlayerList[idx].Yxb = wallet.VndBalance - xiaZhuV

	tableInfoRet := this.GetTableInfo(false)
	playerInfoRet := this.GetPlayerInfo(userID, false)

	_ = this.sendPackToAll(game.Push, tableInfoRet, protocol.UpdateTableInfo, nil)
	_ = this.sendPack(session.GetSessionID(), game.Push, playerInfoRet, protocol.UpdatePlayerInfo, nil)
	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)
	return nil
}
func (this *MyTable) DoubleXiaZhu(session gate.Session, msg map[string]interface{}) (err error) {
	//start := time.Now().UnixNano()
	player := this.FindPlayer(session)
	if player == nil {
		return errors.New("no join")
	}
	player.OnRequest(session)
	if this.RoomState != ROOM_WAITING_XIAZHU {
		log.Info("----------------room state not xia zhu roomstate = %d", this.RoomState)
		error := errCode.CurCanXiaZhuError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return nil
	}

	userID := session.GetUserID()
	idx := this.GetPlayerIdx(userID)
	if idx == -1 {
		return errors.New("no idx")
	}
	if !this.PlayerList[idx].DoubleState {
		log.Info("----------------DoubleState not true-----")
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return
	}

	var needYxb int64 = 0
	for _, v := range this.PlayerList[idx].XiaZhuResult {
		for _, v1 := range v {
			if v1 > 0 {
				needYxb += v1
			}
		}
	}
	//needYxb *= 2
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	if wallet.VndBalance < needYxb {
		log.Info("------------------------- player yxb not enough yxb = %d,num = %d", wallet.VndBalance, needYxb)
		error := errCode.BalanceNotEnough
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return
	}
	var XiaZhuResult map[yxxStorage.XiaZhuResult][]int64
	XiaZhuResult = map[yxxStorage.XiaZhuResult][]int64{}
	if needYxb > 0 { //当前局有下注，就是当前局的下注
		for k, v := range this.PlayerList[idx].XiaZhuResult { //当前局有下注，就是当前局的下注
			for _, v1 := range v {
				if v1 > 0 {
					this.DealXiaZhu(session, v1, k, idx, userID) //双倍下注
					//this.DealXiaZhu(session,v1,k,idx,userID)
					//XiaZhuResult[k] = append(XiaZhuResult[k],v1)
					XiaZhuResult[k] = append(XiaZhuResult[k], v1)
				}
			}
		}
	} else { //否则就是上一轮的下注
		for _, v := range this.PlayerList[idx].LastXiaZhuResult {
			for _, v1 := range v {
				if v1 > 0 {
					needYxb += v1
				}
			}
		}
		//needYxb *= 2
		if wallet.VndBalance < needYxb {
			log.Info("------------------------- player yxb not enough yxb = %d,num = %d", wallet.VndBalance, needYxb)
			return
		}
		for k, v := range this.PlayerList[idx].LastXiaZhuResult {
			for _, v1 := range v {
				if v1 > 0 {
					this.DealXiaZhu(session, v1, k, idx, userID) //双倍下注
					//this.DealXiaZhu(session,v1,k,idx,userID)
					//XiaZhuResult[k] = append(XiaZhuResult[k],v1)
					XiaZhuResult[k] = append(XiaZhuResult[k], v1)
				}
			}
		}
	}
	info := struct {
		UserID       string
		XiaZhuResult map[yxxStorage.XiaZhuResult][]int64
	}{
		UserID:       userID,
		XiaZhuResult: XiaZhuResult,
	}
	_ = this.sendPackToAll(game.Push, info, protocol.DoubleXiaZhu, nil)
	bill := walletStorage.NewBill(userID, walletStorage.TypeExpenses, walletStorage.EventGameYxx, this.EventID, -needYxb)
	walletStorage.OperateVndBalance(bill)
	this.notifyWallet(userID)
	this.PlayerList[idx].Yxb = wallet.VndBalance - needYxb

	tableInfoRet := this.GetTableInfo(false)
	playerInfoRet := this.GetPlayerInfo(userID, false)

	_ = this.sendPackToAll(game.Push, tableInfoRet, protocol.UpdateTableInfo, nil)
	_ = this.sendPack(session.GetSessionID(), game.Push, playerInfoRet, protocol.UpdatePlayerInfo, nil)

	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)
	return nil
}

func (this *MyTable) GetResultsRecord(session gate.Session, msg map[string]interface{}) (res interface{}, err map[string]interface{}) {
	player := this.FindPlayer(session)
	if player == nil {
		return nil, errCode.ServerError.SetKey().GetMap()
	}
	player.OnRequest(session)
	resultsRecord := yxxStorage.GetResultsRecord(this.tableID)

	return resultsRecord, nil
}
func (this *MyTable) GetPrizeRecord(session gate.Session, msg map[string]interface{}) (res interface{}, err map[string]interface{}) {
	player := this.FindPlayer(session)
	if player == nil {
		return nil, errCode.ServerError.SetKey().GetMap()
	}
	page, _ := utils.ConvertInt(msg["Page"])
	player.OnRequest(session)
	record := make(map[string]interface{})
	prizeRecord := yxxStorage.GetPrizeRecord(this.tableID)
	if page < 1 || int(page) > len(prizeRecord.PrizeRecordList) {
		record["MaxPage"] = 0
		return record, nil
	}
	curRecordList := prizeRecord.PrizeRecordList[len(prizeRecord.PrizeRecordList)-int(page)]
	record["CurCnt"] = "JP" + strconv.FormatInt(curRecordList.Cnt, 10)
	record["PrizeWinRate"] = prizeRecord.PrizeWinRate
	record["CreateAt"] = curRecordList.CreateTime.Local()
	record["Result"] = curRecordList.Result
	record["ResultsPool"] = curRecordList.ResultsPool
	record["PrizeList"] = curRecordList.PrizeList
	record["MaxPage"] = len(prizeRecord.PrizeRecordList)
	return record, nil
}
func (this *MyTable) GetPlayerList(session gate.Session, msg map[string]interface{}) (res interface{}, err map[string]interface{}) {
	player := this.FindPlayer(session)
	if player == nil {
		return nil, errCode.ServerError.SetKey().GetMap()
	}
	player.OnRequest(session)
	var list = this.PlayerList

	sort.Slice(list, func(i, j int) bool { //排序
		return list[i].Yxb > list[j].Yxb
	})

	type PlayerList struct {
		Name string
		Yxb  int64
		Head string
	}
	var playerList []PlayerList
	for _, v := range list {
		pl := PlayerList{
			Name: v.Name,
			Yxb:  v.Yxb,
			Head: v.Head,
		}
		playerList = append(playerList, pl)
	}
	info := struct {
		PlayerList []PlayerList
	}{
		PlayerList: playerList,
	}

	return info, nil
}
func (this *MyTable) QuitTable(session gate.Session, userID string) (res interface{}, err map[string]interface{}) {
	//start := time.Now().UnixNano()
	player := this.FindPlayer(session)
	sb := vGate.QuerySessionBean(userID)

	if player == nil {
		if sb != nil {
			this.sendPack(session.GetSessionID(), game.Push, "", protocol.QuitTable, errCode.ServerError)
		}
		return nil, nil
	}
	player.OnRequest(session)
	idx := this.GetPlayerIdx(userID)

	if idx == -1 {
		if sb != nil {
			this.sendPack(session.GetSessionID(), game.Push, "", protocol.QuitTable, errCode.ServerError)
		}
		return nil, nil
	}
	if this.RoomState == ROOM_WAITING_XIAZHU { //下注状态不能退出房间
		for _, v := range this.PlayerList[idx].XiaZhuResult {
			for _, v1 := range v {
				if v1 > 0 {
					log.Info("cant leave room")
					if sb != nil {
						this.sendPack(session.GetSessionID(), game.Push, "", protocol.QuitTable, errCode.XiaZhuCantQuit)
					}
					return nil, nil
				}
			}
		}
	}
	sort.Slice(this.PlayerList, func(i, j int) bool { //排序
		return this.PlayerList[i].Yxb > this.PlayerList[j].Yxb
	})
	idx = this.GetPlayerIdx(userID)
	this.PlayerList = append(this.PlayerList[:idx], this.PlayerList[idx+1:]...)
	if idx <= this.SeatNum { //如果金币足够到相应位置，就刷新位置
		this.UpdatePlayerList()
	}

	this.PlayerNum = len(this.PlayerList)

	ret := this.GetTableInfo(false)

	info := make(map[string]interface{})
	info["PlayerNum"] = this.PlayerNum
	this.sendPackToAll(game.Push, info, protocol.UpdatePlayerNum, nil)
	if sb != nil {
		ret := this.DealProtocolFormat("", protocol.QuitTable, nil)
		this.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
		this.onlinePush.ExecuteCallBackMsg(this.Trace())
	}
	delete(this.Players, userID)
	gameStorage.RemoveReconnectByUid(userID)
	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)
	return ret, nil
}

func (this *MyTable) Disconnect(session gate.Session, msg map[string]interface{}) (err error) {
	player := this.FindPlayer(session)
	if player == nil {
		return errors.New("no join")
	}
	player.OnRequest(session)

	return nil
}
func (this *MyTable) SendShortCut(session gate.Session, msg map[string]interface{}) (err error) {
	userID := session.GetUserID()
	idx := this.GetPlayerIdx(userID)
	if idx == -1 {
		return errors.New("no idx")
	}
	Interval := time.Now().Unix() - this.PlayerList[idx].LastChatTime.Unix()
	if Interval < int64(this.GameConf.ShortCutInterval) { //间隔太短
		error := errCode.TimeIntervalError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.SendShortCut, error)
		return nil
	}
	this.PlayerList[idx].LastChatTime = time.Now()

	msg["UserId"] = userID
	_ = this.sendPackToAll(game.Push, msg, protocol.SendShortCut, nil)
	return nil
}
