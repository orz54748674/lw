package cardQzsg

import (
	"sort"
	"vn/common/protocol"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/game"
	common2 "vn/game/common"
	vGate "vn/gate"
	"vn/storage/cardStorage/cardQzsgStorage"
	"vn/storage/gameStorage"
)
func  (this *MyTable)GenerateVndBalance(min int64,max int64) int64{
	return this.RandInt64(min,max)
}
func (this *MyTable) RobotGenerate(num int){ //生成机器人
	for{
		if num >0{
			robot := common2.RandBotN(num,this.Rand)
			for _,v := range robot{
				find := false
				for _,v1 := range this.PlayerList{
					if v1.Role == ROBOT && v.Oid.String() == v1.UserID{
						find = true
						break
					}
				}
				if !find{
					this.PutQueue(protocol.RobotEnter,v)
					num -= 1
				}
			}
		}
		if num == 0 {
			break
		}
	}
}
func (this *MyTable) RobotEnter(robot common2.Bot) bool {
	//player := &room.BasePlayerImp{}

	idx := this.GetPlayerIdx(robot.Oid.Hex())
	if idx >= 0 {
		//	log.Info("you already in room")
		return false
	}
	if this.GetRobotNum() >= this.TotalPlayerNum - 1 || this.GetTablePlayerNum() >= this.TotalPlayerNum{
		return false
	}
	tableInfo := cardQzsgStorage.GetTableInfo(this.tableID)
	pl := PlayerList{
		//session: session,
		Yxb: this.GenerateVndBalance(this.BaseScore * 50,this.BaseScore * 400),
		UserID: robot.Oid.Hex(),
		Name: robot.NickName,
		Account: robot.NickName,
		Head: robot.Avatar,
		Role: ROBOT,
		Ready:false,
		IsHavePeople: true,
	}
	for k,v :=range this.PlayerList{
		if !v.IsHavePeople{
			this.PlayerList[k] = pl
			this.PlayerList[k].IsHavePeople = true
			idx = k
			break
		}
	}
	playerNum := this.GetTablePlayerNum()
	if playerNum == 1{
		this.Master = pl.UserID
		tableInfo.Master = pl.UserID
		this.PlayerList[idx].Ready = true
	}
	cardQzsgStorage.UpsertTableInfo (tableInfo,this.tableID)

	if playerNum >= this.TotalPlayerNum && this.AutoCreate{
		this.AutoCreate = false
		myRoom := (this.module).(*Room)
		myRoom.CreateTable(this.tableIDTail)
	}
	for k,v := range this.PlayerList {
		if v.IsHavePeople && k != idx{
			playerInfo := this.GetPlayerInfo(v.UserID)
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil{
				s,_ := basegate.NewSession(this.app, sb.Session)
				_ = this.sendPack(s.GetSessionID(),game.Push,playerInfo,protocol.UpdatePlayerInfo,nil)
			}
		}
	}
	return true
}
func (this *MyTable) RobotQuitTable(userID string) (bool) {
	//player := &room.BasePlayerImp{}

	idx := this.GetPlayerIdx(userID)
	if idx == -1 {
		//log.Info("you not in room  userid = %s",userID)
		return false
	}
	if userID == this.Master{
		nextIdx := this.GetNextIdx(idx)
		if nextIdx != idx{
			this.Master = this.PlayerList[nextIdx].UserID
			this.PlayerList[nextIdx].Ready = true
		}
	}
	this.PlayerList[idx] = PlayerList{}
	for k,v := range this.PlayerList {
		if v.IsHavePeople && k != idx{
			playerInfo := this.GetPlayerInfo(v.UserID)
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil{
				s,_ := basegate.NewSession(this.app, sb.Session)
				_ = this.sendPack(s.GetSessionID(),game.Push,playerInfo,protocol.UpdatePlayerInfo,nil)
			}
		}
	}
	return true
}
func (this *MyTable) RobotReady(userID string)  (err error) {
	if this.RoomState == ROOM_END{
		return nil
	}
	if this.RoomState != ROOM_WAITING_READY{
		return nil
	}
	if userID == ""{
		log.Info("your userid is empty")
		return nil
	}
	idx := this.GetPlayerIdx(userID)
	if idx == -1{
		return nil
	}
	res := make(map[string]int)
	this.PlayerList[idx].Ready = true
	if this.GetReadyPlayerNum() == 2 {
		this.CountDown = this.GameConf.ReadyTime
		res["CountDown"] = this.CountDown
	}
	res["Idx"] = idx
	this.sendPackToAll(game.Push,res,protocol.Ready,nil)

	return nil
}
type ControlData struct {
	PokerVal int
	PokerType PokerType
	FinalScore int64 //最终得分
	HandPoker []int //手牌
	BetVal	int64
	Role      Role
	Idx 	int
}
func (this *MyTable) ControlResults(){
	controlData := make([]ControlData,0)
	realPlayerIdx := make([]int,0)
	robotPlayerIdx := make([]int,0)
	for k,v := range this.PlayerList{
		if v.IsHavePeople && v.Ready{
			ctlData := ControlData{
				HandPoker: v.HandPoker,
				PokerVal: v.PokerVal,
				Role: v.Role,
				Idx: k,
				BetVal: v.BetVal,
				PokerType: v.PokerType,
			}
			controlData = append(controlData,ctlData)
			if v.Role != USER{
				robotPlayerIdx = append(robotPlayerIdx,k)
			}else{
				realPlayerIdx = append(realPlayerIdx,k)
			}
		}
	}
	dealIdx := -1
	for k1,v1 := range controlData{
		if v1.Idx == this.DealerIdx{
			dealIdx = k1
		}
	}
	for k,v := range controlData{ //先收取下注值
		if k != dealIdx{
			controlData[k].FinalScore -= v.BetVal
			controlData[dealIdx].FinalScore += v.BetVal
		}
	}
	for k,v := range controlData{
		if k != dealIdx{
			if controlData[dealIdx].PokerVal < v.PokerVal{
				controlData[k].FinalScore += v.BetVal * 2
				controlData[dealIdx].FinalScore -= v.BetVal * 2
			}
		}
	}

	sort.Slice(controlData, func(i, j int) bool { //升序排序
		return controlData[i].FinalScore < controlData[j].FinalScore
	})

	realFinalScore := int64(0)
	for _,v := range controlData{
		if v.Role == USER{
			realFinalScore += v.FinalScore
		}
	}

	gameProfit := gameStorage.QueryProfit(game.CardQzsg)
	if realFinalScore <= 0 || realFinalScore < gameProfit.BotBalance{
		return
	}

	realPlayerIdx = this.SliceOutOfOrder(realPlayerIdx)
	robotPlayerIdx = this.SliceOutOfOrder(robotPlayerIdx)

	length := 0
	if len(realPlayerIdx) + len(robotPlayerIdx) != len(controlData){
		log.Info("cardQzsg control error")
		return
	}
	for _,v := range realPlayerIdx{
		this.PlayerList[v].HandPoker = this.SliceOutOfOrder(controlData[length].HandPoker)
		this.PlayerList[v].PokerVal = controlData[length].PokerVal
		this.PlayerList[v].PokerType = controlData[length].PokerType
		length++
	}

	for _,v := range robotPlayerIdx{
		this.PlayerList[v].HandPoker = this.SliceOutOfOrder(controlData[length].HandPoker)
		this.PlayerList[v].PokerVal = controlData[length].PokerVal
		this.PlayerList[v].PokerType = controlData[length].PokerType
		length++
	}
}