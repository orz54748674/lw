package cardCddN

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
	"vn/storage/cardStorage/cardCddNStorage"
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
	tableInfo := cardCddNStorage.GetTableInfo(this.tableID)
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
	cardCddNStorage.UpsertTableInfo(tableInfo, this.tableID)
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
	tableInfo := cardCddNStorage.GetTableInfo(this.tableID)
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
	cardCddNStorage.UpsertTableInfo(tableInfo, this.tableID)
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
	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PutPoker, errCode.ServerError)
		}
		return nil
	}
	pkType, _, _, _ := this.GetCardType(poker)
	if pkType == PokerType(0) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PutPoker, errCode.InvalidPutPoker)
		}
		return nil
	}

	if (!waitingList.FirstRound && !this.CompBigger(this.LastData.LastPutCard, poker)) || !this.CurRoundList[idx] {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.PutPoker, errCode.InvalidPutPoker)
		}
		return nil
	}

	pressScore := int64(0)
	if !waitingList.FirstRound && !this.IsPutOver(this.LastData.LastPutIdx) { //统计被压牌型
		pressScore = this.GetPokerPressList(pkType)
		if pressScore > 0 {
			for _, v := range this.CurRoundPressList.Single2 {
				if v/0x10 == 4 || v/0x10 == 3 {
					this.CurRoundPressList.Score += this.BaseScore * 6
				} else {
					this.CurRoundPressList.Score += this.BaseScore * 3
				}
			}
			this.CurRoundPressList.Single2 = make([]int, 0)
			this.CurRoundPressList.PressIdx = this.LastData.LastPutIdx
			this.CurRoundPressList.WinIdx = idx
			this.CurRoundPressList.Score += pressScore
		}
	}

	this.PlayerList[idx].HandPoker = this.RemoveTblInList(this.PlayerList[idx].HandPoker, poker)
	data := make(map[string]interface{})
	data["putRank"] = -1
	data["isSpring"] = false
	this.FirstPutIdx = idx
	this.LastData.LastPutCard = poker
	this.LastData.LastPutIdx = idx

	if len(this.PlayerList[idx].HandPoker) == 0 {
		if this.CurRoundPressList.Score > 0 {
			this.PokerPressList = append(this.PokerPressList, this.CurRoundPressList)
			this.CurRoundPressList = PokerPressList{
				Single2: make([]int, 0),
			}
		}
		isSpring := false
		for _, v := range this.PlayerList {
			if v.Ready {
				if len(v.HandPoker) == 13 {
					isSpring = true
					break
				}
			}
		}
		this.PutOverRecord = append(this.PutOverRecord, PutOverRecord{Idx: idx, Over: true, LastPutCard: poker})
		data["putRank"] = 0 //len(this.PutOverRecord) - 1
		data["isSpring"] = isSpring
		this.RemoveCurRoundList(idx)
		nextIdx := this.GetNextPutIdx(idx)
		data["idx"] = idx
		data["nextIdx"] = nextIdx
		data["poker"] = poker
		data["pkType"] = pkType

		if pressScore > 0 {
			data["PressList"] = this.CurRoundPressList
		}
		this.sendPackToAll(game.Push, data, protocol.PutPoker, nil)

		this.RoomState = ROOM_WAITING_JIESUAN
		this.PutQueue(protocol.JieSuan)
		return nil
	}
	for k, v := range this.PlayerList {
		if v.Ready {
			compList := this.CompStraightPair(v.HandPoker)
			for _, v1 := range compList {
				if v1.num == 4 {
					if (pkType == Single && poker[0]%0x10 == 2) || (pkType == Pair && poker[0]%0x10 == 2) || pkType == StraightPair3 || pkType == StraightPair4 {
						bigPk := this.CheckBiggerPoker(poker, v.HandPoker)
						if len(bigPk) > 0 {
							this.AddCurRoundList(k)
						}
					}
					break
				}
			}
		}

	}
	nextIdx := this.GetNextPutIdx(idx)

	canPut := false
	firstRound := false
	var bigPk [][]int
	if nextIdx == idx {
		firstRound = true
		canPut = true
		this.InitCurRoundList()
	} else {
		bigPk = this.CheckBiggerPoker(this.LastData.LastPutCard, this.PlayerList[nextIdx].HandPoker)
		if len(bigPk) > 0 {
			canPut = true
		}
	}

	data["idx"] = idx
	data["nextIdx"] = nextIdx
	data["poker"] = poker
	data["pkType"] = pkType
	data["FirstRound"] = firstRound
	if pressScore > 0 {
		data["PressList"] = this.CurRoundPressList
	}

	this.sendPackToAll(game.Push, data, protocol.PutPoker, nil)

	this.InitWaitingList()

	this.WaitingList.Store(nextIdx, WaitingList{
		Time:       time.Now(),
		Have:       true,
		CanPut:     canPut,
		BigPk:      bigPk,
		FirstRound: firstRound,
	})

	this.SendState(this.WaitingList)

	this.CountDown = this.GameConf.PutPokerTime
	//logData,_ := json.Marshal(data)
	//logWait,_ := json.Marshal(this.WaitingList)
	//log.Info("---put poker --data--%s",logData)
	//log.Info("---put poker --wait--%s",logWait)

	//	this.HintPoker(this.PlayerList[nextIdx].UserID)
	return nil
}

func (this *MyTable) CheckPoker(userID string) (err error) {
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
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.CheckPoker, errCode.ServerError)
			}
			return nil
		}
	}
	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have || (waitingList.FirstRound && this.GetCurRoundPlayerNum() != 0) {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.CheckPoker, errCode.ServerError)
		}
		return nil
	}

	nextIdx := 0
	this.RemoveCurRoundList(idx)
	if this.GetCurRoundPlayerNum() == 0 {
		this.InitCurRoundList()
		nextIdx = this.GetNextPutIdx(idx)
		this.FirstPutIdx = nextIdx
	} else {
		nextIdx = this.GetNextPutIdx(idx)
	}
	firstRound := false
	canPut := false
	bigPk := this.CheckBiggerPoker(this.LastData.LastPutCard, this.PlayerList[nextIdx].HandPoker)
	if len(bigPk) > 0 {
		canPut = true
	}
	if nextIdx == this.FirstPutIdx {
		firstRound = true
		canPut = true
		if this.CurRoundPressList.Score > 0 {
			this.PokerPressList = append(this.PokerPressList, this.CurRoundPressList)
			this.CurRoundPressList = PokerPressList{
				Single2: make([]int, 0),
			}
		}
		this.InitCurRoundList()
	}

	data := make(map[string]interface{})
	data["idx"] = idx
	data["nextIdx"] = nextIdx
	data["FirstRound"] = firstRound

	this.sendPackToAll(game.Push, data, protocol.CheckPoker, nil)

	this.InitWaitingList()

	this.WaitingList.Store(nextIdx, WaitingList{
		Time:       time.Now(),
		Have:       true,
		CanPut:     canPut,
		FirstRound: firstRound,
		BigPk:      bigPk,
	})
	this.SendState(this.WaitingList)

	this.CountDown = this.GameConf.PutPokerTime

	//logData,_ := json.Marshal(data)
	//logWait,_ := json.Marshal(this.WaitingList)
	//log.Info("---check poker --data--%s",logData)
	//log.Info("---check poker --wait--%s",logWait)

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
func (this *MyTable) HintPoker(userID string) (err error) {
	idx := this.GetPlayerIdx(userID)
	sb := vGate.QuerySessionBean(userID)
	var s gate.Session
	if sb != nil {
		s, _ = basegate.NewSession(this.app, sb.Session)
	}
	if idx < 0 {
		if idx == -1 {
			if sb != nil {
				this.sendPack(s.GetSessionID(), game.Push, "", protocol.HintPoker, errCode.ServerError)
			}
			return nil
		}
	}

	val, _ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if !waitingList.Have {
		if sb != nil {
			this.sendPack(s.GetSessionID(), game.Push, "", protocol.HintPoker, errCode.ServerError)
		}
	}

	bigPk := this.CheckBiggerPoker(this.LastData.LastPutCard, this.PlayerList[idx].HandPoker)
	res := make(map[string]interface{}, 2)
	res["Poker"] = bigPk
	if sb != nil {
		this.sendPack(s.GetSessionID(), game.Push, res, protocol.HintPoker, nil)
	}
	return nil
}
