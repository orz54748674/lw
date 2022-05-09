package cardSss

import (
	"sort"
	"vn/common/protocol"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/game"
	common2 "vn/game/common"
	vGate "vn/gate"
	"vn/storage/cardStorage/cardSssStorage"
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
	tableInfo := cardSssStorage.GetTableInfo(this.tableID)
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
	cardSssStorage.UpsertTableInfo (tableInfo,this.tableID)

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
	StraightType StraightType
	PokerVal []int
	PokerType []PokerType
	FinalScore int64 //最终得分
	HandPoker []int //手牌
	Role      Role
}
func (this *MyTable) ControlResults(){
	controlData := make([]ControlData,0)
	realPlayerIdx := make([]int,0)
	robotPlayerIdx := make([]int,0)
	for k,v := range this.PlayerList{
		if v.IsHavePeople && v.Ready{
			straightType := this.CheckStraightScore(v.HandPoker)
			handPoker := this.SortPokerFunc("",v.HandPoker)
			ThreeShowPoker := this.GetThreeKindPoker(handPoker)
			ctlData := ControlData{
				HandPoker: handPoker,
				StraightType: straightType,
				Role: v.Role,
			}
			ctlData.PokerType = make([]PokerType,3)
			ctlData.PokerVal = make([]int,3)
			if ThreeShowPoker != nil && straightType <= 0 {
				for k1,v1 := range ThreeShowPoker{
					ctlData.PokerType[k1],ctlData.PokerVal[k1] = this.CheckPokerType(v1)
				}
			}

			controlData = append(controlData,ctlData)
			if v.Role != USER{
				robotPlayerIdx = append(robotPlayerIdx,k)
			}else{
				realPlayerIdx = append(realPlayerIdx,k)
			}
		}
	}

	var calcIdx []int
	calcIdx = []int{}

	winTimes := map[int]int{}
	for k,v:= range controlData{
		if v.StraightType <= 0{
			calcIdx = append(calcIdx,k)
		}
	}

	if len(calcIdx) > 1{
		for i := 0;i < len(calcIdx);i++{
			for j := i + 1;j < len(calcIdx);j++{
				a := calcIdx[i]
				b := calcIdx[j]
				var win []int
				win = []int{}
				for k := 0;k < 3;k++{
					max := 0
					min := 0

					if controlData[a].PokerVal[k] > controlData[b].PokerVal[k]{
						max = a
						min = b
					}else{
						max = b
						min = a
					}
					win = append(win,max)
					winTimes[max] += 1

					score := int64(1)
					if k == 0{
						if controlData[max].PokerType[k] == ThreeOfAKind{
							score = 6
						}
					}else if k == 1{
						if controlData[max].PokerType[k] == FullHouse{
							score = 4
						}else if controlData[max].PokerType[k] == FourOfAKind{
							score = 16
						}else if controlData[max].PokerType[k] == StraightFlush || controlData[max].PokerType[k] == BigStraightFlush{
							score = 20
						}
					}else if k == 2{
						if controlData[max].PokerType[k] == FourOfAKind{
							score = 8
						}else if controlData[max].PokerType[k] == StraightFlush || controlData[max].PokerType[k] == BigStraightFlush{
							score = 10
						}
					}
					controlData[max].FinalScore += score
					controlData[min].FinalScore -= score
				}
				if win[0] == win[1] && win[0] == win[2]{ //打枪
					if win[0] == a{
						controlData[a].FinalScore += 6
						controlData[b].FinalScore -= 6
					}else{
						controlData[b].FinalScore += 6
						controlData[a].FinalScore -= 6
					}
				}

			}
		}
		homeRun := -1
		for _,v := range calcIdx{ //计算全垒打
			if this.GetReadyPlayerNum() == 4{
				if winTimes[v] >= 3 * 3{
					homeRun = v
				}
			}else if this.GetReadyPlayerNum() == 3{
				if winTimes[v] >= 2 * 3{
					homeRun = v
				}
			}

			if homeRun >= 0{
				for _,v1 := range calcIdx{
					if v != v1{
						controlData[v].FinalScore += 6
						controlData[v1].FinalScore -= 6
					}
				}
				break
			}
		}
	}


	//直接得分
	for k,v := range controlData{
		if v.StraightType > 0 { //直接得分
			for k1,_ := range controlData{
				controlData[k].FinalScore += this.StraightScore[v.StraightType]
				controlData[k1].FinalScore -= this.StraightScore[v.StraightType]
			}
		}
	}
	sort.Slice(controlData, func(i, j int) bool { //升序排序
		return controlData[i].FinalScore < controlData[j].FinalScore
	})

	realFinalScore := int64(0)
	for _,v := range controlData{
		if v.Role == USER{
			realFinalScore += v.FinalScore * this.BaseScore
		}
	}

	gameProfit := gameStorage.QueryProfit(game.CardSss)
	if realFinalScore <= 0 || realFinalScore < gameProfit.BotBalance{
		return
	}

	realPlayerIdx = this.SliceOutOfOrder(realPlayerIdx)
	robotPlayerIdx = this.SliceOutOfOrder(robotPlayerIdx)

	length := 0
	if len(realPlayerIdx) + len(robotPlayerIdx) != len(controlData){
		log.Info("cardSss control error")
		return
	}
	for _,v := range realPlayerIdx{
		this.PlayerList[v].HandPoker = this.SliceOutOfOrder(controlData[length].HandPoker)
		length++
	}

	for _,v := range robotPlayerIdx{
		this.PlayerList[v].HandPoker = this.SliceOutOfOrder(controlData[length].HandPoker)
		length++
	}
}