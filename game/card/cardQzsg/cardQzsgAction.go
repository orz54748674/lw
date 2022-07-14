package cardQzsg

import (
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/cardStorage/cardQzsgStorage"
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
	tableInfo := cardQzsgStorage.GetTableInfo(this.tableID)
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
	cardQzsgStorage.UpsertTableInfo(tableInfo, this.tableID)
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
	tableInfo := cardQzsgStorage.GetTableInfo(this.tableID)
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
	cardQzsgStorage.UpsertTableInfo(tableInfo, this.tableID)
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
	return nil
}
func (this *MyTable) GrabDealer(userID string, msg map[string]interface{}) (err error) {
	idx := this.GetPlayerIdx(userID)
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if idx == -1 {
		error := errCode.NotInRoomError
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.GrabDealer, error)
		}
		return nil
	}
	waitingList, _ := this.WaitingList.Load(idx)
	if this.RoomState != ROOM_WAITING_GRABDEALER || waitingList.(bool) {
		error := errCode.CurGrabDealerError
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.GrabDealer, error)
		}
		return nil
	}
	grabDealer, _ := utils.ConvertInt(msg["GrabDealer"])
	if grabDealer > 0 {
		this.GrabDealerList = append(this.GrabDealerList, idx)
	}
	this.WaitingList.Store(idx, true)
	res := make(map[string]interface{})
	res["Idx"] = idx
	res["GrabDealer"] = grabDealer

	this.sendPackToAll(game.Push, res, protocol.GrabDealer, nil)
	for k, v := range this.PlayerList {
		if v.Ready {
			waitingList, _ := this.WaitingList.Load(k)
			if !waitingList.(bool) {
				return nil
			}
		}
	}

	if len(this.GrabDealerList) == 0 {
		readyPlayer := this.GetReadyPlayerIdx()
		rand := this.RandInt64(1, int64(len(readyPlayer)+1)) - 1
		this.DealerIdx = readyPlayer[rand]
	} else {
		rand := this.RandInt64(1, int64(len(this.GrabDealerList)+1)) - 1
		this.DealerIdx = this.GrabDealerList[rand]
	}
	info := make(map[string]interface{})
	info["DealerIdx"] = this.DealerIdx
	info["GrabDealerList"] = this.GrabDealerList
	this.sendPackToAll(game.Push, info, protocol.NotifyDealerResult, nil)
	for k, _ := range this.PlayerList {
		if k != this.DealerIdx {
			this.WaitingList.Store(k, false)
		} else {
			this.WaitingList.Store(k, true)
		}
	}

	this.RoomState = ROOM_WAITING_XIAZHU
	this.CountDown = this.GameConf.XiaZhuTime
	this.SwitchRoomState()
	this.SeqExecFlag = true
	return nil
}
func (this *MyTable) XiaZhu(userID string, msg map[string]interface{}) (err error) {
	idx := this.GetPlayerIdx(userID)
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if idx == -1 {
		error := errCode.NotInRoomError
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		}
		return nil
	}
	waitingList, _ := this.WaitingList.Load(idx)
	if this.RoomState != ROOM_WAITING_XIAZHU || waitingList.(bool) || this.DealerIdx == idx {
		error := errCode.CurCanXiaZhuError
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		}
		return nil
	}

	betV, _ := utils.ConvertInt(msg["betV"])

	if this.PlayerList[idx].Role == ROBOT {
		if this.PlayerList[idx].Yxb < betV {
			log.Info("------------------------- player yxb not enough yxb = %d,num = %d", this.PlayerList[idx].Yxb, betV)
			error := errCode.BalanceNotEnough
			if sb != nil {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
			}
			return nil
		}
		//		this.PlayerList[idx].Yxb -= betV
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
		if wallet.VndBalance < betV {
			log.Info("------------------------- player yxb not enough yxb = %d,num = %d", wallet.VndBalance, betV)
			error := errCode.BalanceNotEnough
			if sb != nil {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
			}
			return nil
		}
		//bill := walletStorage.NewBill(userID,walletStorage.TypeExpenses,walletStorage.EventGameCardQzsg,this.EventID,-betV)
		//walletStorage.OperateVndBalance(bill)
		//this.notifyWallet(userID)
		//this.PlayerList[idx].Yxb = wallet.VndBalance - betV
	}

	this.WaitingList.Store(idx, true)
	info := make(map[string]interface{})
	info["idx"] = idx
	info["betV"] = betV
	this.PlayerList[idx].BetVal = betV
	//this.PlayerList[this.DealerIdx].FinalScore += betV
	//this.PlayerList[idx].FinalScore -= betV
	_ = this.sendPackToAll(game.Push, info, protocol.XiaZhu, nil)
	for k, v := range this.PlayerList {
		if v.Ready {
			waitingList, _ := this.WaitingList.Load(k)
			if !waitingList.(bool) {
				return nil
			}
		}
	}

	this.RoomState = ROOM_WAITING_JIESUAN
	this.PutQueue(protocol.JieSuan)
	return nil
}
