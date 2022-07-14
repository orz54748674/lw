package cardCatte

import (
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
	"vn/storage/cardStorage/cardCatteStorage"
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
	tableInfo := cardCatteStorage.GetTableInfo(this.tableID)
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
	cardCatteStorage.UpsertTableInfo(tableInfo, this.tableID)
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
	tableInfo := cardCatteStorage.GetTableInfo(this.tableID)
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
	cardCatteStorage.UpsertTableInfo(tableInfo, this.tableID)
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
	if idx < 0 {
		if idx == -1 || len(poker) != 1 || this.RoundNum > 4 {
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
	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PutPoker, errCode.ServerError)
		}
		return nil
	}
	putData := PutData{}
	if !waitingList.FirstRound {
		check := this.CheckBiggerPoker(this.MaxPoker, poker)
		if len(check) == 0 {
			if sb != nil {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.PutPoker, errCode.ServerError)
			}
			return nil
		}
		this.PlayerList[this.MaxIdx].PutPoker[len(this.PlayerList[this.MaxIdx].PutPoker)-1].State = 0
	}

	putData.State = 1
	putData.Poker = poker[0]

	this.PlayerList[idx].PutPoker = append(this.PlayerList[idx].PutPoker, putData)
	this.PlayerList[idx].HandPoker = this.RemoveTblInList(this.PlayerList[idx].HandPoker, poker)
	data := make(map[string]interface{})
	this.RemoveCurRoundList(idx)
	nextIdx := -1
	canPut := false
	bigPk := make([]int, 0)
	if waitingList.FirstRound {
		data["preMaxIdx"] = -1
		data["preMaxPoker"] = -1
	} else {
		data["preMaxIdx"] = this.MaxIdx
		data["preMaxPoker"] = this.MaxPoker
	}

	data["idx"] = idx
	data["poker"] = poker
	data["fire"] = -1

	firstPut := false
	if this.RoundNum == 4 {
		find := false
		if !waitingList.FirstRound {
			for _, v1 := range this.PlayerList[this.MaxIdx].PutPoker {
				if v1.State > 0 {
					find = true
					break
				}
			}
			if !find {
				data["fire"] = this.MaxIdx
				this.PutOverRecord = append(this.PutOverRecord, PutOverRecord{
					Idx:  this.MaxIdx,
					Over: true,
				})
			}
		}
		if len(this.PutOverRecord) >= this.PlayingNum-1 {
			data["RoundNum"] = this.RoundNum
			this.sendPackToAll(game.Push, data, protocol.PutPoker, nil)

			this.WinIdx = idx
			this.RoomState = ROOM_WAITING_JIESUAN
			this.PutQueue(protocol.JieSuan)
			return
		}
	}
	if this.GetCurRoundPlayerNum() > 0 {
		this.MaxIdx = idx
		this.MaxPoker = poker[0]
		nextIdx = this.GetNextPutIdx(idx)
		bigPk = this.CheckBiggerPoker(this.MaxPoker, this.PlayerList[nextIdx].HandPoker)
		if len(bigPk) > 0 {
			canPut = true
		}
	} else {
		nextIdx = idx
		canPut = true
	}
	if this.GetCurRoundPlayerNum() == 0 {
		this.InitCurRoundList()
		firstPut = true
		this.RoundNum++
	}
	data["nextIdx"] = nextIdx
	data["RoundNum"] = this.RoundNum

	this.sendPackToAll(game.Push, data, protocol.PutPoker, nil)

	this.InitWaitingList()
	this.WaitingList.Store(nextIdx, WaitingList{
		Time:       time.Now(),
		Have:       true,
		CanPut:     canPut,
		BigPk:      bigPk,
		FirstRound: firstPut,
	})

	this.SendState(this.WaitingList)

	this.MaxIdx = idx
	this.MaxPoker = poker[0]

	this.CountDown = this.GameConf.PutPokerTime
	//logData,_ := json.Marshal(data)
	//logWait,_ := json.Marshal(this.WaitingList)
	//log.Info("---put poker --data--%s",logData)
	//log.Info("---put poker --wait--%s",logWait)

	//	this.HintPoker(this.PlayerList[nextIdx].UserID)

	return nil
}
func (this *MyTable) CheckPoker(userID string, msg map[string]interface{}) (err error) {
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
		if idx == -1 || len(poker) != 1 || this.RoundNum > 4 {
			if sb != nil {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.CheckPoker, errCode.ServerError)
			}
			return nil
		}
	}

	if !this.IsContainArray(poker, this.PlayerList[idx].HandPoker) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.CheckPoker, errCode.ServerError)
		}
		return nil
	}
	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have || waitingList.FirstRound {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.CheckPoker, errCode.ServerError)
		}
		return nil
	}

	putData := PutData{}
	putData.State = -1
	putData.Poker = poker[0]

	this.PlayerList[idx].PutPoker = append(this.PlayerList[idx].PutPoker, putData)
	this.PlayerList[idx].HandPoker = this.RemoveTblInList(this.PlayerList[idx].HandPoker, poker)
	this.RemoveCurRoundList(idx)

	nextIdx := -1
	canPut := false
	bigPk := make([]int, 0)
	firstPut := false
	if this.GetCurRoundPlayerNum() > 0 {
		nextIdx = this.GetNextPutIdx(idx)
		bigPk = this.CheckBiggerPoker(this.MaxPoker, this.PlayerList[nextIdx].HandPoker)
		if len(bigPk) > 0 {
			canPut = true
		}
	} else {
		nextIdx = this.MaxIdx
		canPut = true
	}

	data := make(map[string]interface{})
	data["idx"] = idx
	data["nextIdx"] = nextIdx
	data["poker"] = -1
	data["fire"] = -1
	if this.RoundNum == 4 {
		find := false
		if !waitingList.FirstRound {
			for _, v1 := range this.PlayerList[idx].PutPoker {
				if v1.State > 0 {
					find = true
					break
				}
			}
			if !find {
				data["fire"] = idx
				this.PutOverRecord = append(this.PutOverRecord, PutOverRecord{
					Idx:  idx,
					Over: true,
				})
			}
		}
		if len(this.PutOverRecord) >= this.PlayingNum-1 {
			data["RoundNum"] = this.RoundNum
			this.sendPackToAll(game.Push, data, protocol.CheckPoker, nil)

			this.WinIdx = this.MaxIdx
			this.RoomState = ROOM_WAITING_JIESUAN
			this.PutQueue(protocol.JieSuan)
			return
		}
	}
	if this.GetCurRoundPlayerNum() == 0 {
		this.RoundNum++
		this.InitCurRoundList()
		firstPut = true
	}
	data["RoundNum"] = this.RoundNum

	for k, v := range this.PlayerList {
		if k == idx {
			data["poker"] = poker
		} else {
			data["poker"] = -1
		}
		if v.IsHavePeople && v.Role != ROBOT {
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				s, _ := basegate.NewSession(this.app, sb.Session)
				_ = this.sendPack(s.GetSessionID(), game.Push, data, protocol.CheckPoker, nil)
			}
		}
	}

	this.InitWaitingList()
	this.WaitingList.Store(nextIdx, WaitingList{
		Time:       time.Now(),
		Have:       true,
		CanPut:     canPut,
		BigPk:      bigPk,
		FirstRound: firstPut,
	})
	this.SendState(this.WaitingList)

	this.CountDown = this.GameConf.PutPokerTime

	//logData,_ := json.Marshal(data)
	//logWait,_ := json.Marshal(this.WaitingList)
	//log.Info("---check poker --data--%s",logData)
	//log.Info("---check poker --wait--%s",logWait)

	return nil
}
func (this *MyTable) ShowPoker(userID string, msg map[string]interface{}) (err error) {
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
	if idx == -1 || len(poker) != 2 || this.RoundNum < 5 {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.ShowPoker, errCode.ServerError)
		}
		return nil
	}
	if !this.IsContainArray(poker, this.PlayerList[idx].HandPoker) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.ShowPoker, errCode.ServerError)
		}
		return nil
	}
	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.ShowPoker, errCode.ServerError)
		}
		return nil
	}

	this.PlayerList[idx].ShowPoker = append(this.PlayerList[idx].ShowPoker, PutData{
		State: 1,
		Poker: poker[0],
	})
	this.PlayerList[idx].ShowPoker = append(this.PlayerList[idx].ShowPoker, PutData{
		State: 1,
		Poker: poker[1],
	})
	this.PlayerList[idx].HandPoker = this.RemoveTblInList(this.PlayerList[idx].HandPoker, poker)
	data := make(map[string]interface{})
	this.RemoveCurRoundList(idx)
	data["Idx"] = idx
	data["Poker"] = poker[0]
	data["preMaxIdx"] = -1
	data["preMaxPoker"] = -1
	data["FirstRound"] = false
	if !waitingList.FirstRound {
		maxPk := this.PlayerList[this.MaxIdx].ShowPoker[0].Poker
		if poker[0]/0x10 == maxPk/0x10 && poker[0] > maxPk {
			data["preMaxIdx"] = this.MaxIdx
			data["preMaxPoker"] = maxPk
			this.MaxPoker = poker[0]
			this.PlayerList[this.MaxIdx].ShowPoker[0].State = 0
			this.MaxIdx = idx
		} else {
			this.PlayerList[idx].ShowPoker[0].State = 0
		}
	} else {
		this.MaxIdx = idx
		this.MaxPoker = poker[0]
		data["FirstRound"] = true
	}
	nextIdx := -1
	bigPk := make([]int, 0)
	if this.GetCurRoundPlayerNum() > 0 {
		nextIdx = this.GetNextPutIdx(idx)
		bigPk = this.CheckBiggerPoker(this.MaxPoker, this.PlayerList[nextIdx].HandPoker)
	}
	data["NextIdx"] = nextIdx
	data["RoundNum"] = this.RoundNum
	this.sendPackToAll(game.Push, data, protocol.ShowPoker, nil)

	if this.GetCurRoundPlayerNum() == 0 {
		this.RoomState = ROOM_WAITING_JIESUAN
		this.PutQueue(protocol.JieSuan)
		return
	}
	this.InitWaitingList()
	this.WaitingList.Store(nextIdx, WaitingList{
		Time:   time.Now(),
		Have:   true,
		CanPut: true,
		BigPk:  bigPk,
	})
	this.SendState(this.WaitingList)
	this.CountDown = this.GameConf.PutPokerTime
	//logData,_ := json.Marshal(data)
	//logWait,_ := json.Marshal(this.WaitingList)
	//log.Info("---show poker --data--%s",logData)
	//log.Info("---show poker --wait--%s",logWait)

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
	sort.Slice(this.PlayerList[idx].HandPoker, func(i, j int) bool { //升序排序
		if this.PlayerList[idx].HandPoker[i]%0x10 == this.PlayerList[idx].HandPoker[j]%0x10 {
			return this.PlayerList[idx].HandPoker[i] < this.PlayerList[idx].HandPoker[j]
		}
		return this.PlayerList[idx].HandPoker[i]%0x10 < this.PlayerList[idx].HandPoker[j]%0x10
	})
	res := make(map[string]interface{}, 2)
	res["Poker"] = this.PlayerList[idx].HandPoker
	if sb != nil {
		this.sendPack(s.GetSessionID(), game.Push, res, protocol.SortPoker, nil)
	}
	return nil
}
