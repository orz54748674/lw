package cardPhom

import (
	"encoding/json"
	"sort"
	"time"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/cardStorage/cardPhomStorage"
	"vn/storage/gameStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) GetEnterData(session gate.Session, msg map[string]interface{}) (err error) {
	player := &room.BasePlayerImp{}
	player.Bind(session)
	player.OnRequest(session)
	userID := session.GetUserID()
	this.Players[userID] = player
	if userID == "" {
		log.Info("your userid is empty")
		return nil
	}
	idx := this.GetPlayerIdx(userID)
	if idx < 0 {
		return nil
	}

	tableInfo := this.GetTableInfo(userID)
	_ = this.sendPack(session.GetSessionID(), game.Push, tableInfo, protocol.GetEnterData, nil)
	return nil
}
func (this *MyTable) Enter(session gate.Session) (err error) {
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
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	if wallet.VndBalance < this.MinEnterTable {
		error := errCode.BalanceNotEnough
		ret := this.DealProtocolFormat("", protocol.Enter, error)
		this.onlinePush.SendCallBackMsgNR([]string{session.GetSessionID()}, game.Push, ret)
		this.onlinePush.ExecuteCallBackMsg(this.Trace())
		return nil
	}
	this.Players[userID] = player
	idx := this.GetPlayerIdx(userID)
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	if idx >= 0 {
		if user.Type != userStorage.TypeNormal {
			this.PlayerList[idx].Role = Agent
		}
		playerNum := this.GetTablePlayerNum()
		if playerNum == 1 {
			this.Master = userID
		}
		this.PlayerList[idx].Session = session
		ret := make(map[string]interface{}, 2)
		ret["ServerID"] = this.module.GetServerID()
		this.sendPack(session.GetSessionID(), game.Push, ret, protocol.Enter, nil)
		return nil
	}

	if this.GetTablePlayerNum() >= this.TotalPlayerNum || this.RoomState == ROOM_END {
		myRoom := (this.module).(*Room)
		tableList := make([]room.BaseTable, 0)
		myRoom.tablesID.Range(func(key, value interface{}) bool {
			table := myRoom.room.GetTable(value.(string)) //
			if table != nil {
				myTable := (table.(interface{})).(*MyTable)
				if myTable.BaseScore == this.BaseScore && myTable.GetTablePlayerNum() < myTable.TotalPlayerNum {
					tableList = append(tableList, myTable)
				}
			}
			return true
		})
		if len(tableList) > 0 {
			tableIdx := myRoom.RandInt64(1, int64(len(tableList)+1)) - 1
			table := tableList[tableIdx]
			table.PutQueue(protocol.Enter, session)
		} else {
			log.Info("----cardSss---room end")
			error := errCode.ServerError
			this.sendPack(session.GetSessionID(), game.Push, "", protocol.Enter, error)
			return nil
		}
		return nil
	}
	pl := PlayerList{
		Yxb:     wallet.VndBalance,
		UserID:  userID,
		Name:    user.NickName,
		Head:    user.Avatar,
		Role:    USER,
		Session: session,
		Account: user.Account,
	}
	if user.Type != userStorage.TypeNormal {
		pl.Role = Agent
	}
	tableInfo := cardPhomStorage.GetTableInfo(this.tableID)
	for k, v := range this.PlayerList {
		if !v.IsHavePeople {
			this.PlayerList[k] = pl
			this.PlayerList[k].IsHavePeople = true
			idx = k
			break
		}
	}
	playerNum := this.GetTablePlayerNum()
	if playerNum == 1 {
		this.Master = userID
		this.PlayerList[idx].Ready = true
		tableInfo.Master = userID
	}
	cardPhomStorage.UpsertTableInfo(tableInfo, this.tableID)
	if playerNum >= this.TotalPlayerNum && this.AutoCreate {
		this.AutoCreate = false
		myRoom := (this.module).(*Room)
		myRoom.CreateTable(this.tableIDTail)
	}
	ret := make(map[string]interface{}, 2)
	ret["ServerID"] = this.module.GetServerID()
	this.sendPack(session.GetSessionID(), game.Push, ret, protocol.Enter, nil)
	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)
	realPlayerNum := this.GetTableRealPlayerNum()
	for k, v := range this.PlayerList {
		if v.IsHavePeople && k != idx {
			playerInfo := this.GetPlayerInfo(v.UserID)
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				s, _ := basegate.NewSession(this.app, sb.Session)
				_ = this.sendPack(s.GetSessionID(), game.Push, playerInfo, protocol.UpdatePlayerInfo, nil)
			}
		}
		if v.IsHavePeople && realPlayerNum >= 2 && v.Role == ROBOT && this.RoomState == ROOM_WAITING_READY {
			this.PutQueue(protocol.RobotQuitTable, v.UserID)
		}
	}
	gameStorage.UpsertGameReconnect(userID, this.module.GetServerID())
	gameStorage.UpsertInRoomNeedVnd(userID, this.MinEnterTable)
	return nil
}
func (this *MyTable) InviteEnter(session gate.Session) (err error) {
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
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	if wallet.VndBalance < this.MinEnterTable {
		error := errCode.BalanceNotEnough
		ret := this.DealProtocolFormatToLobby("", protocol.InviteEnter, error)
		this.onlinePush.SendCallBackMsgNR([]string{session.GetSessionID()}, game.Push, ret)
		this.onlinePush.ExecuteCallBackMsg(this.Trace())
		return nil
	}
	this.Players[userID] = player
	idx := this.GetPlayerIdx(userID)
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	if idx >= 0 {
		if user.Type != userStorage.TypeNormal {
			this.PlayerList[idx].Role = Agent
		}
		playerNum := this.GetTablePlayerNum()
		if playerNum == 1 {
			this.Master = userID
		}
		this.PlayerList[idx].Session = session
		ret := make(map[string]interface{}, 2)
		ret["ServerID"] = this.module.GetServerID()
		this.sendPackToLobby(session.GetSessionID(), game.Push, ret, protocol.InviteEnter, nil)
		return nil
	}

	if this.GetTablePlayerNum() >= this.TotalPlayerNum || this.RoomState == ROOM_END {
		myRoom := (this.module).(*Room)
		tableList := make([]room.BaseTable, 0)
		myRoom.tablesID.Range(func(key, value interface{}) bool {
			table := myRoom.room.GetTable(value.(string)) //
			if table != nil {
				myTable := (table.(interface{})).(*MyTable)
				if myTable.BaseScore == this.BaseScore && myTable.GetTablePlayerNum() < myTable.TotalPlayerNum {
					tableList = append(tableList, myTable)
				}
			}
			return true
		})
		if len(tableList) > 0 {
			tableIdx := myRoom.RandInt64(1, int64(len(tableList)+1)) - 1
			table := tableList[tableIdx]
			table.PutQueue(protocol.InviteEnter, session)
		} else {
			log.Info("----cardSss---room end")
			error := errCode.ServerError
			this.sendPackToLobby(session.GetSessionID(), game.Push, "", protocol.InviteEnter, error)
			return nil
		}
		return nil
	}
	pl := PlayerList{
		Yxb:     wallet.VndBalance,
		UserID:  userID,
		Name:    user.NickName,
		Head:    user.Avatar,
		Role:    USER,
		Session: session,
		Account: user.Account,
	}
	if user.Type != userStorage.TypeNormal {
		pl.Role = Agent
	}
	tableInfo := cardPhomStorage.GetTableInfo(this.tableID)
	for k, v := range this.PlayerList {
		if !v.IsHavePeople {
			this.PlayerList[k] = pl
			this.PlayerList[k].IsHavePeople = true
			idx = k
			break
		}
	}
	playerNum := this.GetTablePlayerNum()
	if playerNum == 1 {
		this.Master = userID
		this.PlayerList[idx].Ready = true
		tableInfo.Master = userID
	}
	cardPhomStorage.UpsertTableInfo(tableInfo, this.tableID)
	if playerNum >= this.TotalPlayerNum && this.AutoCreate {
		this.AutoCreate = false
		myRoom := (this.module).(*Room)
		myRoom.CreateTable(this.tableIDTail)
	}
	ret := make(map[string]interface{}, 2)
	ret["ServerID"] = this.module.GetServerID()
	this.sendPackToLobby(session.GetSessionID(), game.Push, ret, protocol.InviteEnter, nil)
	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)

	realPlayerNum := this.GetTableRealPlayerNum()
	for k, v := range this.PlayerList {
		if v.IsHavePeople && k != idx {
			playerInfo := this.GetPlayerInfo(v.UserID)
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				s, _ := basegate.NewSession(this.app, sb.Session)
				_ = this.sendPack(s.GetSessionID(), game.Push, playerInfo, protocol.UpdatePlayerInfo, nil)
			}
		}
		if v.IsHavePeople && realPlayerNum >= 2 && v.Role == ROBOT && this.RoomState == ROOM_WAITING_READY {
			this.PutQueue(protocol.RobotQuitTable, v.UserID)
		}
	}
	gameStorage.UpsertGameReconnect(userID, this.module.GetServerID())
	gameStorage.UpsertInRoomNeedVnd(userID, this.MinEnterTable)
	return nil
}
func (this *MyTable) QuitTable(userID string) (res interface{}, err map[string]interface{}) {
	idx := this.GetPlayerIdx(userID)
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if idx == -1 {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.QuitTable, errCode.ServerError)
		}
		return nil, nil
	}
	if this.RoomState != ROOM_WAITING_READY && this.PlayerList[idx].Ready { //下注状态不能退出房间
		if sb != nil {
			if this.PlayerList[idx].QuitRoom {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.QuitTable, errCode.QuitRoomCancel)
			} else {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.QuitTable, errCode.QuitRoomAfterOver)
			}

		}
		this.PlayerList[idx].QuitRoom = !this.PlayerList[idx].QuitRoom
		return nil, nil
	}

	this.PlayerList[idx] = PlayerList{}
	ret := this.DealProtocolFormat("", protocol.QuitTable, nil)
	if sb != nil {
		this.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
		this.onlinePush.ExecuteCallBackMsg(this.Trace())
	}
	this.sendPackToAll(game.Push, this.PlayerList, protocol.UpdatePlayerList, nil)
	delete(this.Players, userID)

	if userID == this.Master {
		nextIdx := this.GetNextIdx(idx)
		if nextIdx != idx {
			this.Master = this.PlayerList[nextIdx].UserID
			this.PlayerList[nextIdx].Ready = true
		}
	}

	for k, v := range this.PlayerList {
		if v.IsHavePeople && k != idx {
			playerInfo := this.GetPlayerInfo(v.UserID)
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				s, _ := basegate.NewSession(this.app, sb.Session)
				_ = this.sendPack(s.GetSessionID(), game.Push, playerInfo, protocol.UpdatePlayerInfo, nil)
			}
		}
	}
	if this.GetReadyPlayerNum() <= 1 { //
		this.CountDown = 0
	}

	gameStorage.RemoveReconnectByUid(userID)
	gameStorage.UpsertInRoomNeedVnd(userID, 0)
	return nil, nil
}
func (this *MyTable) Ready(userID string) (err error) {
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if this.RoomState == ROOM_END {
		error := errCode.ServerError
		this.sendPack(s.GetSessionID(), game.Push, "", protocol.Ready, error)
		return nil
	}
	if this.RoomState != ROOM_WAITING_READY {
		log.Info("---- cant ready---,roomstate = %s", this.RoomState)
		error := errCode.ServerError
		this.sendPack(s.GetSessionID(), game.Push, "", protocol.Ready, error)
		return nil
	}

	if userID == "" {
		log.Info("your userid is empty")
		return nil
	}
	idx := this.GetPlayerIdx(userID)
	if idx == -1 {
		this.sendPack(s.GetSessionID(), game.Push, "", protocol.Ready, errCode.ServerError)
		return nil
	}
	res := make(map[string]int)
	if this.GetReadyPlayerNum()+1 == 2 {
		this.CountDown = this.GameConf.ReadyTime
		res["CountDown"] = this.CountDown
	}
	this.PlayerList[idx].Ready = true
	res["Idx"] = idx
	this.sendPackToAll(game.Push, res, protocol.Ready, nil)

	return nil
}
func (this *MyTable) AutoReady(session gate.Session, msg map[string]interface{}) (err error) {
	if this.RoomState == ROOM_END {
		log.Info("----cardSss---room end")
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.AutoReady, error)
		return nil
	}
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)
	userID := session.GetUserID()

	if userID == "" {
		log.Info("your userid is empty")
		return nil
	}
	autoReady := msg["AutoReady"].(bool)
	idx := this.GetPlayerIdx(userID)
	this.PlayerList[idx].AutoReady = autoReady

	if autoReady && this.RoomState == ROOM_WAITING_READY && this.Master != userID { //自动准备
		this.PutQueue(protocol.Ready, userID)
	}

	return nil
}
func (this *MyTable) MasterStartGame(session gate.Session) (err error) {
	this.SeqExecFlag = true
	if this.RoomState == ROOM_END {
		log.Info("----cardSss---room end")
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.MasterStartGame, error)
		return nil
	}
	if this.RoomState != ROOM_WAITING_READY {
		log.Info("----cardSss cant start---,roomstate = %s", this.RoomState)
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.MasterStartGame, error)
		return nil
	}
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)
	userID := session.GetUserID()

	if userID == "" {
		log.Info("your userid is empty")
		return nil
	}
	if userID != this.Master {
		log.Info("----cardSss cant start---,roomstate = %s", this.RoomState)
		error := errCode.ServerError
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.MasterStartGame, error)
		return nil
	}
	this.CountDown = 0
	this.SeqExecFlag = true
	return nil
}
func (this *MyTable) PutPoker(userID string, msg map[string]interface{}) (err error) {
	this.SeqExecFlag = true
	recPoker := msg["Poker"].([]interface{})
	poker := make([]int, 0)
	for _, v := range recPoker {
		pk, _ := utils.ConvertInt(v)
		poker = append(poker, int(pk))
	}
	idx := this.GetPlayerIdx(userID)
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if idx < 0 || len(poker) != 1 {
		if idx == -1 {
			if sb != nil {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.PutPoker, errCode.ServerError)
			}
			return nil
		}
	}
	if !this.IsContainArray(poker, this.PlayerList[idx].HandPoker) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PutPoker, errCode.ServerError)
		}
		return nil
	}
	if this.IsContainArray(poker, this.PlayerList[idx].ForbidPutPoker) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PutPoker, errCode.InvalidPutPoker)
		}
		return nil
	}
	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have || (waitingList.State != PUTPOKER && waitingList.State != GivePoker && waitingList.State != PHOM) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PutPoker, errCode.ServerError)
		}
		return nil
	}

	this.PlayerList[idx].HandPoker = this.RemoveTblInList(this.PlayerList[idx].HandPoker, poker)
	data := make(map[string]interface{})

	//if this.PlayerList[idx].CalcPhomData.State != ""{
	//	this.RemoveCurRoundList(idx)
	//}

	if len(this.Bottom) == 0 || len(this.PlayerList[idx].HandPoker) == 0 {
		data["idx"] = idx
		data["nextIdx"] = -1
		data["poker"] = poker
		if len(this.PlayerList[idx].HandPoker) == 0 {
			this.WinIdx = idx
			this.PlayerList[idx].CalcPhomData.State = UThuong
		} else {
			this.CalcRankList()
			if len(this.RankList) == 0 {
				this.WinIdx = this.FirstPut
			} else {
				this.WinIdx = this.RankList[0].Idx
			}
		}
		data["PhomState"] = this.PlayerList[idx].CalcPhomData.State
		this.sendPackToAll(game.Push, data, protocol.PutPoker, nil)
		this.RoomState = ROOM_WAITING_JIESUAN
		this.PutQueue(protocol.JieSuan)
		return nil
	}

	nextIdx := this.GetNextPutIdx(idx)

	this.PlayerList[idx].PutPoker = append(this.PlayerList[idx].PutPoker, poker[0])
	eatPk := this.CheckEatPoker(this.PlayerList[nextIdx].HandPoker, this.PlayerList[nextIdx].ForbidPutPoker, poker[0])
	canEat := false
	if len(eatPk) > 0 {
		canEat = true
	}
	data["idx"] = idx
	data["nextIdx"] = nextIdx
	data["poker"] = poker
	data["PhomState"] = this.PlayerList[idx].CalcPhomData.State
	this.sendPackToAll(game.Push, data, protocol.PutPoker, nil)
	this.InitWaitingList()
	if canEat {
		this.WaitingList.Store(nextIdx, WaitingList{
			Time:     time.Now(),
			PreIdx:   idx,
			Have:     true,
			State:    EATPOKER,
			EatPoker: poker[0],
		})
	} else {
		this.WaitingList.Store(nextIdx, WaitingList{
			Time:   time.Now(),
			PreIdx: idx,
			Have:   true,
			State:  DRAWPOKER,
		})
	}

	this.SendState(this.WaitingList)
	this.CountDown = this.GameConf.PutPokerTime
	//logData,_ := json.Marshal(data)
	//log.Info("---put poker --data--%s",logData)

	//	this.HintPoker(this.PlayerList[nextIdx].UserID)
	this.SeqExecFlag = true
	return nil
}
func (this *MyTable) DrawPoker(userID string) (err error) {
	this.SeqExecFlag = true
	idx := this.GetPlayerIdx(userID)
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if idx < 0 {
		if idx == -1 {
			if sb != nil {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.DrawPoker, errCode.ServerError)
			}
			return nil
		}
	}
	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have || (waitingList.State != EATPOKER && waitingList.State != DRAWPOKER) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.DrawPoker, errCode.ServerError)
		}
		return nil
	}

	mid := this.Bottom[len(this.Bottom)-1]
	//	log.Info("-----Draw Poker %d",mid)
	this.PlayerList[idx].HandPoker = append(this.PlayerList[idx].HandPoker, mid)
	this.Bottom = append(this.Bottom[:len(this.Bottom)-1])
	data := make(map[string]interface{})
	data["idx"] = idx
	for k, v := range this.PlayerList {
		if v.IsHavePeople {
			if k == idx {
				data["poker"] = mid
			} else {
				data["poker"] = -1
			}
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				s, _ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(s.GetSessionID(), game.Push, data, protocol.DrawPoker, nil)
			}
		}
	}
	this.InitWaitingList()
	this.CalcPhomData = CalcPhomData{}
	this.CalcPhomPoker(idx, 0, this.PlayerList[idx].HandPoker, [][]int{})
	poker := make([]int, 0)
	for _, v := range this.CalcPhomData.Phom {
		for _, v1 := range v {
			poker = append(poker, v1)
		}
	}
	if len(this.CalcPhomData.Phom)+len(this.PlayerList[idx].CalcPhomData.Phom) >= 3 {
		data := make(map[string]interface{})
		data["Phom"] = this.CalcPhomData.Phom
		data["Idx"] = idx

		for _, v := range this.CalcPhomData.Phom {
			this.PlayerList[idx].HandPoker = this.RemoveTblInList(this.PlayerList[idx].HandPoker, v)
			this.PlayerList[idx].CalcPhomData.Phom = append(this.PlayerList[idx].CalcPhomData.Phom, v)
		}
		if len(this.PlayerList[idx].HandPoker) == 0 {
			this.PlayerList[idx].CalcPhomData.State = UTron
		} else {
			this.PlayerList[idx].CalcPhomData.State = UThuong
		}
		data["State"] = this.PlayerList[idx].CalcPhomData.State
		givePokerData := this.CheckGivePoker(idx, this.PlayerList[idx].HandPoker)
		if len(givePokerData) > 0 {
			this.sendPackToAll(game.Push, data, protocol.PhomPoker, nil)
			givePk := make([]int, 0)
			for _, v := range givePokerData {
				for _, v1 := range v.Poker {
					givePk = append(givePk, v1)
				}
			}
			this.WaitingList.Store(idx, WaitingList{
				Time:      time.Now(),
				Have:      true,
				GivePoker: givePk,
				State:     GivePoker,
			})
			this.SendState(this.WaitingList)
		} else {
			time.Sleep(time.Millisecond * 500)
			this.sendPackToAll(game.Push, data, protocol.PhomPoker, nil)
			//this.WaitingList[idx] = WaitingList{
			//	Time: time.Now(),
			//	Have: true,
			//	State: JieSuan,
			//}
			this.SendState(this.WaitingList)
			this.WinIdx = idx
			this.RoomState = ROOM_WAITING_JIESUAN
			this.PutQueue(protocol.JieSuan)
		}
		return
	} else if len(this.PlayerList[idx].PutPoker) >= 3 {
		if len(poker) == 0 {
			if len(this.PlayerList[idx].CalcPhomData.Phom) > 0 {
				givePokerData := this.CheckGivePoker(idx, this.PlayerList[idx].HandPoker)
				if len(givePokerData) > 0 {
					givePk := make([]int, 0)
					for _, v := range givePokerData {
						for _, v1 := range v.Poker {
							givePk = append(givePk, v1)
						}
					}
					this.WaitingList.Store(idx, WaitingList{
						Time:      time.Now(),
						Have:      true,
						GivePoker: givePk,
						State:     GivePoker,
					})
				} else {
					this.WaitingList.Store(idx, WaitingList{
						Time:     time.Now(),
						Have:     true,
						State:    PUTPOKER,
						PhomData: PhomData{State: Normal},
					})
				}
			} else {
				this.WaitingList.Store(idx, WaitingList{
					Time:     time.Now(),
					Have:     true,
					State:    PUTPOKER,
					PhomData: PhomData{State: MOM},
				})
			}

			if len(this.PlayerList[idx].CalcPhomData.Phom) == 0 && this.PlayerList[idx].CalcPhomData.State != MOM {
				data := make(map[string]interface{})
				data["Phom"] = ""
				data["Idx"] = idx
				this.PlayerList[idx].CalcPhomData.State = MOM
				data["State"] = this.PlayerList[idx].CalcPhomData.State
				this.sendPackToAll(game.Push, data, protocol.PhomPoker, nil)
			}
		} else {
			this.WaitingList.Store(idx, WaitingList{
				Time:     time.Now(),
				Have:     true,
				State:    PHOM,
				PhomData: PhomData{Poker: poker, State: Normal},
			})
		}

	} else {
		this.WaitingList.Store(idx, WaitingList{
			Time:  time.Now(),
			Have:  true,
			State: PUTPOKER,
		})
	}
	this.SendState(this.WaitingList)
	if this.PlayerList[idx].Role == ROBOT {
		this.SortPoker(userID)
	}
	this.SeqExecFlag = true
	return nil
}
func (this *MyTable) EatPoker(userID string) (err error) {
	this.SeqExecFlag = true
	idx := this.GetPlayerIdx(userID)
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if idx < 0 {
		if idx == -1 {
			if sb != nil {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.EatPoker, errCode.ServerError)
			}
			return nil
		}
	}
	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have || waitingList.State != EATPOKER {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.EatPoker, errCode.ServerError)
		}
		return nil
	}
	mid := waitingList.EatPoker
	eatScore := int64(0)
	lastRoundEat := false
	if len(this.PlayerList[idx].PutPoker) >= 3 {
		eatScore = 4 * this.BaseScore
		this.LastRoundEatIdx = idx
		lastRoundEat = true
	} else {
		eatScore = this.BaseScore * int64(1<<uint(len(this.PlayerList[idx].EatData)))
	}

	eatData := EatData{
		Poker:        mid,
		Score:        eatScore,
		PreIdx:       waitingList.PreIdx,
		LastRoundEat: lastRoundEat,
	}
	this.PlayerList[waitingList.PreIdx].PutPoker = append(this.PlayerList[waitingList.PreIdx].PutPoker[:len(this.PlayerList[waitingList.PreIdx].PutPoker)-1])
	this.PlayerList[idx].EatData = append(this.PlayerList[idx].EatData, eatData)
	this.PlayerList[idx].HandPoker = append(this.PlayerList[idx].HandPoker, mid)

	data := make(map[string]interface{})
	data["idx"] = idx
	data["preIdx"] = waitingList.PreIdx
	data["eatPoker"] = mid
	data["eatScore"] = eatScore
	data["LastRoundEat"] = lastRoundEat
	this.DealForbidPutPoker(idx)
	this.sendPackToAll(game.Push, data, protocol.EatPoker, nil)

	this.DealPhomPoker(idx)

	this.CalcPhomData = CalcPhomData{}
	this.CalcPhomPoker(idx, 0, this.PlayerList[idx].HandPoker, [][]int{})
	poker := make([]int, 0)
	for _, v := range this.CalcPhomData.Phom {
		for _, v1 := range v {
			poker = append(poker, v1)
		}
	}
	this.InitWaitingList()
	if len(this.CalcPhomData.Phom)+len(this.PlayerList[idx].CalcPhomData.Phom) >= 3 {
		data := make(map[string]interface{})
		data["Idx"] = idx
		for _, v := range this.CalcPhomData.Phom {
			this.PlayerList[idx].HandPoker = this.RemoveTblInList(this.PlayerList[idx].HandPoker, v)
			this.PlayerList[idx].CalcPhomData.Phom = append(this.PlayerList[idx].CalcPhomData.Phom, v)
		}
		data["Phom"] = this.PlayerList[idx].CalcPhomData.Phom
		if len(this.PlayerList[idx].HandPoker) == 0 {
			this.PlayerList[idx].CalcPhomData.State = UTron
		} else {
			this.PlayerList[idx].CalcPhomData.State = UThuong
		}
		data["State"] = this.PlayerList[idx].CalcPhomData.State
		givePokerData := this.CheckGivePoker(idx, this.PlayerList[idx].HandPoker)
		if len(givePokerData) > 0 {
			this.sendPackToAll(game.Push, data, protocol.PhomPoker, nil)
			givePk := make([]int, 0)
			for _, v := range givePokerData {
				for _, v1 := range v.Poker {
					givePk = append(givePk, v1)
				}
			}
			this.WaitingList.Store(idx, WaitingList{
				Time:      time.Now(),
				Have:      true,
				GivePoker: givePk,
				State:     GivePoker,
			})
			this.SendState(this.WaitingList)
		} else {
			time.Sleep(time.Millisecond * 500)
			this.sendPackToAll(game.Push, data, protocol.PhomPoker, nil)
			//this.WaitingList[idx] = WaitingList{
			//	Time: time.Now(),
			//	Have: true,
			//	State: JieSuan,
			//}
			this.SendState(this.WaitingList)
			this.WinIdx = idx
			this.RoomState = ROOM_WAITING_JIESUAN
			this.PutQueue(protocol.JieSuan)
		}
		return
	} else if len(this.PlayerList[idx].PutPoker) >= 3 {
		if len(poker) == 0 {
			this.WaitingList.Store(idx, WaitingList{
				Time:     time.Now(),
				Have:     true,
				State:    PUTPOKER,
				PhomData: PhomData{State: MOM},
			})
			if len(this.PlayerList[idx].CalcPhomData.Phom) == 0 && this.PlayerList[idx].CalcPhomData.State != MOM {
				data := make(map[string]interface{})
				data["Phom"] = ""
				data["Idx"] = idx
				this.PlayerList[idx].CalcPhomData.State = MOM
				data["State"] = this.PlayerList[idx].CalcPhomData.State
				this.sendPackToAll(game.Push, data, protocol.PhomPoker, nil)
			}
		} else {
			this.WaitingList.Store(idx, WaitingList{
				Time:     time.Now(),
				Have:     true,
				State:    PHOM,
				PhomData: PhomData{Poker: poker, State: Normal},
			})
		}
	} else {
		this.WaitingList.Store(idx, WaitingList{
			Time:  time.Now(),
			Have:  true,
			State: PUTPOKER,
		})
	}
	this.SendState(this.WaitingList)
	if this.PlayerList[idx].Role == ROBOT {
		this.SortPoker(userID)
	}
	//logData,_ := json.Marshal(data)
	//logWait,_ := json.Marshal(this.WaitingList)
	//log.Info("---put poker --data--%s",logData)
	//log.Info("---put poker --wait--%s",logWait)

	return nil
}

//func (this *MyTable) GetPhomPoker(userID string)  (err error) {
//	idx := this.GetPlayerIdx(userID)
//	robot := true
//	var session gate.Session
//	if this.PlayerList[idx].Role != ROBOT{
//		robot = false
//		session = this.Players[userID].Session()
//	}
//	if idx < 0{
//		if idx == -1{
//			if !robot{
//				this.sendPack(session.GetSessionID(),game.Push,"",protocol.GetPhomPoker,errCode.ServerError)
//			}
//			return nil
//		}
//	}
//	if !this.WaitingList[idx].Have || this.WaitingList[idx].State != PHOM{
//		if !robot {
//			this.sendPack(session.GetSessionID(), game.Push, "", protocol.GetPhomPoker, errCode.ServerError)
//		}
//		return nil
//	}
//
//	this.CalcPhomPoker(idx,0,this.PlayerList[idx].HandPoker,[][]int{})
//	poker := make([]int,0)
//	for _,v := range this.PlayerList[idx].CalcPhomData.phom{
//		for _,v1 := range v{
//			poker = append(poker,v1)
//		}
//	}
//	res := make(map[string]interface{})
//	res["State"] = ""
//	res["Idx"] = idx
//	if len(poker) == 0{
//		res["State"] = "MOM"
//	}
//	for k,v := range this.PlayerList{
//		if k == idx{
//			res["Poker"] = poker
//		}else{
//			res["Poker"] = -1
//		}
//		this.sendPack(this.Players[v.UserID].Session().GetSessionID(),game.Push,res,protocol.GetPhomPoker,nil)
//	}
//	this.PlayerList[idx].CalcPhomData = CalcPhomData{}
//	return nil
//}
func (this *MyTable) PhomPoker(userID string, msg map[string]interface{}) (err error) {
	this.SeqExecFlag = true
	recPoker := msg["Poker"].([]interface{})
	poker := make([]int, 0)
	for _, v := range recPoker {
		pk, _ := utils.ConvertInt(v)
		poker = append(poker, int(pk))
	}
	idx := this.GetPlayerIdx(userID)
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if idx < 0 || len(poker) < 3 {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PhomPoker, errCode.InvalidPhomPoker)
		}
		playerList, _ := json.Marshal(this.PlayerList[idx])
		log.Info("---phom playerList %s", playerList)
		log.Info("---phom poker---", idx, len(poker))
		log.Info("---phom recPoker---", recPoker)
		log.Info("---------------", idx, this.WaitingList)
		return nil
	}
	if !this.IsContainArray(poker, this.PlayerList[idx].HandPoker) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PhomPoker, errCode.InvalidPhomPoker)
		}
		playerList, _ := json.Marshal(this.PlayerList[idx])
		log.Info("---phom playerList %s", playerList)
		res1, _ := json.Marshal(poker)
		res2, _ := json.Marshal(this.PlayerList[idx].HandPoker)
		log.Info("---phom poker---%s--%s", res1, res2)
		log.Info("---phom recPoker---", recPoker)
		log.Info("---------------", idx, this.WaitingList)
		return nil
	}
	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have || waitingList.State != PHOM {
		if sb != nil {

			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PhomPoker, errCode.InvalidPhomPoker)
		}
		playerList, _ := json.Marshal(this.PlayerList[idx])
		log.Info("---phom playerList %s", playerList)
		res1, _ := json.Marshal(waitingList)
		log.Info("---phom poker---%s--", res1)
		log.Info("---phom recPoker---", recPoker)
		log.Info("---------------", idx, this.WaitingList)
		return nil
	}
	eatPk := make([]int, 0)
	for _, v := range this.PlayerList[idx].EatData {
		eatPk = append(eatPk, v.Poker)
	}
	phomPoker := make([]int, 0)
	for _, v := range this.PlayerList[idx].CalcPhomData.Phom {
		for _, v1 := range v {
			phomPoker = append(phomPoker, v1)
		}
	}
	eatPk = this.RemoveTblInList(eatPk, phomPoker)
	if !this.IsContainArray(eatPk, poker) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PhomPoker, errCode.InvalidPhomPoker)
		}
		playerList, _ := json.Marshal(this.PlayerList[idx])
		log.Info("---phom playerList %s", playerList)
		res1, _ := json.Marshal(eatPk)
		res2, _ := json.Marshal(poker)
		log.Info("---phom poker---%s--%s", res1, res2)
		log.Info("---phom recPoker---", recPoker)
		log.Info("---------------", idx, this.WaitingList)
		return nil
	}
	if this.FirstPut < 0 {
		this.FirstPut = idx
	}
	ok, res := this.CheckPhomPoker(poker, [][]int{})
	if ok {
		this.InitWaitingList()
		for _, v := range res {
			this.PlayerList[idx].HandPoker = this.RemoveTblInList(this.PlayerList[idx].HandPoker, v)
			this.PlayerList[idx].CalcPhomData.Phom = append(this.PlayerList[idx].CalcPhomData.Phom, v)
		}
		this.DealForbidPutPoker(idx)
		this.PlayerList[idx].CalcPhomData.State = Normal
		if len(this.PlayerList[idx].HandPoker) == 0 {
			data := make(map[string]interface{})
			data["Phom"] = this.PlayerList[idx].CalcPhomData.Phom
			data["Idx"] = idx
			this.PlayerList[idx].CalcPhomData.State = UTron
			data["State"] = this.PlayerList[idx].CalcPhomData
			this.sendPackToAll(game.Push, data, protocol.PhomPoker, nil)
			this.WinIdx = idx
			this.RoomState = ROOM_WAITING_JIESUAN
			this.PutQueue(protocol.JieSuan)
			return
		}
		givePokerData := this.CheckGivePoker(idx, this.PlayerList[idx].HandPoker)
		if len(res) > 0 && len(givePokerData) > 0 {
			givePk := make([]int, 0)
			for _, v := range givePokerData {
				for _, v1 := range v.Poker {
					givePk = append(givePk, v1)
				}
			}
			this.WaitingList.Store(idx, WaitingList{
				Time:      time.Now(),
				Have:      true,
				GivePoker: givePk,
				State:     GivePoker,
			})
		} else if len(this.PlayerList[idx].CalcPhomData.Phom) == 3 {
			data := make(map[string]interface{})
			data["Phom"] = this.PlayerList[idx].CalcPhomData.Phom
			data["Idx"] = idx
			this.PlayerList[idx].CalcPhomData.State = UThuong
			data["State"] = this.PlayerList[idx].CalcPhomData.State
			this.sendPackToAll(game.Push, data, protocol.PhomPoker, nil)

			this.WinIdx = idx
			this.RoomState = ROOM_WAITING_JIESUAN
			this.PutQueue(protocol.JieSuan)
			return
		} else {
			this.WaitingList.Store(idx, WaitingList{
				Time:  time.Now(),
				Have:  true,
				State: PUTPOKER,
			})
		}

		data := make(map[string]interface{})
		data["Phom"] = this.PlayerList[idx].CalcPhomData.Phom
		data["Idx"] = idx
		if len(res) == 0 {
			this.PlayerList[idx].CalcPhomData.State = MOM
		}
		data["State"] = this.PlayerList[idx].CalcPhomData.State
		this.sendPackToAll(game.Push, data, protocol.PhomPoker, nil)

		this.SendState(this.WaitingList)
	} else {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PhomPoker, errCode.InvalidPhomPoker)
		}
		playerList, _ := json.Marshal(this.PlayerList[idx])
		log.Info("---phom playerList %s", playerList)
		res1, _ := json.Marshal(poker)
		log.Info("---phom poker---%s--", res1)
		log.Info("---phom recPoker---", recPoker)
		log.Info("---------------", idx, this.WaitingList)
	}
	//	this.HintPoker(this.PlayerList[nextIdx].UserID)
	return nil
}
func (this *MyTable) GivePoker(userID string, msg map[string]interface{}) (err error) {
	this.SeqExecFlag = true
	recPoker := msg["Poker"].([]interface{})
	poker := make([]int, 0)
	for _, v := range recPoker {
		pk, _ := utils.ConvertInt(v)
		poker = append(poker, int(pk))
	}
	idx := this.GetPlayerIdx(userID)
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if idx < 0 {
		if idx == -1 {
			if sb != nil {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.GivePoker, errCode.InvalidGivePoker)
			}
			return nil
		}
	}
	if !this.IsContainArray(poker, this.PlayerList[idx].HandPoker) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.GivePoker, errCode.InvalidGivePoker)
		}
		return nil
	}
	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have || waitingList.State != GivePoker || !this.IsContainArray(poker, waitingList.GivePoker) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.GivePoker, errCode.InvalidGivePoker)
		}
		return nil
	}

	givePokerData := this.CheckGivePoker(idx, this.PlayerList[idx].HandPoker)
	this.PlayerList[idx].HandPoker = this.RemoveTblInList(this.PlayerList[idx].HandPoker, poker)
	resGivePoker := make([]GivePokerData, len(givePokerData))
	copy(resGivePoker, givePokerData)
	for k, _ := range resGivePoker {
		resGivePoker[k].Poker = []int{}
		resGivePoker[k].GetPhomPk = []int{}
	}
	for _, v := range poker {
		for k1, v1 := range givePokerData {
			if this.IsContainElement(v, v1.Poker) {
				this.PlayerList[idx].GivePoker = append(this.PlayerList[idx].GivePoker, v)
				this.PlayerList[v1.GetIdx].CalcPhomData.Phom[v1.GetPhomIdx] = append(this.PlayerList[v1.GetIdx].CalcPhomData.Phom[v1.GetPhomIdx], v)
				sort.Slice(this.PlayerList[v1.GetIdx].CalcPhomData.Phom[v1.GetPhomIdx], func(i, j int) bool { //升序排序
					if this.PlayerList[v1.GetIdx].CalcPhomData.Phom[v1.GetPhomIdx][i]%0x10 == this.PlayerList[v1.GetIdx].CalcPhomData.Phom[v1.GetPhomIdx][j]%0x10 {
						return this.PlayerList[v1.GetIdx].CalcPhomData.Phom[v1.GetPhomIdx][i] < this.PlayerList[v1.GetIdx].CalcPhomData.Phom[v1.GetPhomIdx][j]
					}
					return this.PlayerList[v1.GetIdx].CalcPhomData.Phom[v1.GetPhomIdx][i]%0x10 < this.PlayerList[v1.GetIdx].CalcPhomData.Phom[v1.GetPhomIdx][j]%0x10
				})
				resGivePoker[k1].GetPhomPk = this.PlayerList[v1.GetIdx].CalcPhomData.Phom[v1.GetPhomIdx]
				resGivePoker[k1].Poker = append(resGivePoker[k1].Poker, v)
			}
		}
	}

	data := make(map[string]interface{})
	data["Idx"] = idx
	data["GivePokerData"] = resGivePoker
	if len(this.PlayerList[idx].HandPoker) == 0 {
		if len(this.PlayerList[idx].CalcPhomData.Phom) >= 3 {
			this.PlayerList[idx].CalcPhomData.State = UThuong
		} else {
			this.PlayerList[idx].CalcPhomData.State = UTron
		}
	}
	data["State"] = this.PlayerList[idx].CalcPhomData.State
	this.sendPackToAll(game.Push, data, protocol.GivePoker, nil)

	if len(this.PlayerList[idx].HandPoker) == 0 {
		time.Sleep(time.Second)
		this.WinIdx = idx
		this.RoomState = ROOM_WAITING_JIESUAN
		this.PutQueue(protocol.JieSuan)
		return
	}
	this.InitWaitingList()

	this.WaitingList.Store(idx, WaitingList{
		Time:  time.Now(),
		Have:  true,
		State: PUTPOKER,
	})
	this.SendState(this.WaitingList)

	//	this.HintPoker(this.PlayerList[nextIdx].UserID)
	return nil
}
func (this *MyTable) SortPoker(userID string) (err error) {
	idx := this.GetPlayerIdx(userID)
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if idx < 0 {
		if idx == -1 {
			if sb != nil {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.SortPoker, errCode.ServerError)
			}
			return nil
		}
	}

	//if this.PlayerList[idx].CalcPhomData.State != ""{
	//	if sb != nil{
	//		this.sendPack(s.GetSessionID(),game.Push,"",protocol.SortPoker,errCode.ServerError)
	//	}
	//	return nil
	//}
	sort.Slice(this.PlayerList[idx].HandPoker, func(i, j int) bool { //升序排序
		if this.PlayerList[idx].HandPoker[i]%0x10 == this.PlayerList[idx].HandPoker[j]%0x10 {
			return this.PlayerList[idx].HandPoker[i] < this.PlayerList[idx].HandPoker[j]
		}
		return this.PlayerList[idx].HandPoker[i]%0x10 < this.PlayerList[idx].HandPoker[j]%0x10
	})

	handPk := make([]int, len(this.PlayerList[idx].HandPoker))
	copy(handPk, this.PlayerList[idx].HandPoker)
	this.CalcPhomData = CalcPhomData{}
	this.CalcPhomPoker(idx, 0, handPk, [][]int{})
	phomPk := make([]int, 0)
	for _, v := range this.CalcPhomData.Phom {
		for _, v1 := range v {
			phomPk = append(phomPk, v1)
		}
	}
	handPk = this.RemoveTblInList(handPk, phomPk)
	notStraight := make([]int, 0)
	for _, v := range handPk {
		remainPk := this.FindNotInArrayList([]int{v}, handPk)
		find := false
		for _, v1 := range remainPk {
			if v%0x10 == v1%0x10 {
				find = true
				break
			}

			if v-v1 <= 2 && v-v1 >= -2 {
				find = true
				break
			}
		}
		if !find {
			notStraight = append(notStraight, v)
		}
	}
	straight := this.RemoveTblInList(handPk, notStraight)
	lastHandPk := make([]int, 0)
	for _, v := range phomPk {
		lastHandPk = append(lastHandPk, v)
	}
	for _, v := range straight {
		lastHandPk = append(lastHandPk, v)
	}
	for _, v := range notStraight {
		lastHandPk = append(lastHandPk, v)
	}

	this.PlayerList[idx].HandPoker = lastHandPk
	this.CalcPhomData = CalcPhomData{}

	res := make(map[string]interface{}, 2)
	res["Poker"] = this.PlayerList[idx].HandPoker
	if sb != nil {
		this.sendPack(s.GetSessionID(), game.Push, res, protocol.SortPoker, nil)
	}
	return nil
}
