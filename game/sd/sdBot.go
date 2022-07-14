package sd

import (
	"strconv"
	"time"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/game"
	common2 "vn/game/common"
	"vn/storage/sdStorage"
	"vn/storage/yxxStorage"
)

func (this *MyTable) RobotBetCalc() { //计算下注筹码
	this.RobotXiaZhuList = map[string]RobotXiaZhuList{}
	for k, v := range this.PlayerList {
		if v.Role == ROBOT {
			if k < this.SeatNum {
				xiaZhuTotal := this.RandInt64(5, 25) * v.Yxb / 100                      //获取总共下注筹码
				xiaZhuSplice := xiaZhuTotal * 2 / int64(this.GameConf.XiaZhuTime)       //下注筹码分片
				for i := len(this.GameConf.PlayerChipsList[this.tableID]); i > 0; i-- { //获取筹码
					if xiaZhuSplice >= this.GameConf.PlayerChipsList[this.tableID][i-1] {
						if i > 6 {
							rand := int(this.RandInt64(1, 6))
							xiaZhuSplice = this.GameConf.PlayerChipsList[this.tableID][i-rand]
						} else {
							xiaZhuSplice = this.GameConf.PlayerChipsList[this.tableID][i-1]
						}
						break
					}
				}
				if xiaZhuSplice >= 100 {
					this.RobotXiaZhuList[v.UserID] = RobotXiaZhuList{
						XiaZhu: map[string][]int64{},
					}
					timePos1 := strconv.FormatInt(this.RandInt64(1, int64(this.GameConf.XiaZhuTime)), 10)
					timePos2 := strconv.FormatInt(this.RandInt64(1, int64(this.GameConf.XiaZhuTime)), 10)
					for i := 1; i <= this.GameConf.XiaZhuTime; i++ {
						randNum := this.RandInt64(1, 3)
						if randNum == 2 {
							splitRand := this.RandInt64(1, 3)
							if this.RoomState != ROOM_WAITING_START && this.RoomState != ROOM_WAITING_JIESUAN && this.RoomState != ROOM_WAITING_READY {
								return
							}
							if splitRand == 2 {
								this.RobotXiaZhuList[v.UserID].XiaZhu[timePos2] = append(this.RobotXiaZhuList[v.UserID].XiaZhu[timePos2], xiaZhuSplice)
							} else {
								this.RobotXiaZhuList[v.UserID].XiaZhu[timePos1] = append(this.RobotXiaZhuList[v.UserID].XiaZhu[timePos1], xiaZhuSplice)
							}
						}
					}
				}
			} else {
				xiaZhuTotal := this.RandInt64(20, 40) * v.Yxb / 100                     //获取总共下注筹码
				xiaZhuSplice := xiaZhuTotal * 2 / int64(this.GameConf.XiaZhuTime)       //下注筹码分片
				for i := len(this.GameConf.PlayerChipsList[this.tableID]); i > 0; i-- { //获取筹码
					if xiaZhuSplice >= this.GameConf.PlayerChipsList[this.tableID][i-1] {
						xiaZhuSplice = this.GameConf.PlayerChipsList[this.tableID][i-1]
						break
					}
				}
				if xiaZhuSplice >= 100 {
					this.RobotXiaZhuList[v.UserID] = RobotXiaZhuList{
						XiaZhu: map[string][]int64{},
					}
					for i := 1; i <= this.GameConf.XiaZhuTime; i++ {
						randNum := this.RandInt64(1, 6)

						if randNum == 2 {
							xiaZhu := this.RobotXiaZhuList[v.UserID].XiaZhu[strconv.Itoa(i)]
							xiaZhu = append(xiaZhu, xiaZhuSplice)
							if this.RoomState != ROOM_WAITING_START && this.RoomState != ROOM_WAITING_JIESUAN && this.RoomState != ROOM_WAITING_READY {
								return
							}
							this.RobotXiaZhuList[v.UserID].XiaZhu[strconv.Itoa(i)] = xiaZhu
						}
					}
				}
			}
		}
	}
}
func (this *MyTable) GenerateVndBalance(min int, max int) int64 {
	return this.RandInt64(int64(min), int64(max))
}

func (this *MyTable) CalcChangeRobot(curBotNum int, robotConf sdStorage.RobotConf, start bool) map[int]int { //计算变换机器人
	needRobotType := map[int]int{}
	randAddDec := this.RandInt64(1, 3)
	robotConf.StepNum = int(this.RandInt64(1, int64(robotConf.StepNum+1)))
	if start {
		robotConf.StepNum = robotConf.BaseNum
	}
	addV := 1
	if randAddDec == 1 {
		addV = -1
		if curBotNum-robotConf.StepNum < robotConf.BaseNum-robotConf.MaxOffset {
			addV = 1
		}
	} else {
		addV = 1
		if curBotNum+robotConf.StepNum > robotConf.BaseNum+robotConf.MaxOffset {
			addV = -1
		}
	}
	for i := 0; i < robotConf.StepNum; i++ {
		rand := this.RandInt64(1, 100)
		if rand >= 1 && rand < 2 {
			robotType, _ := strconv.Atoi(string(sdStorage.Robot_30_50_M))
			needRobotType[robotType] += addV
		} else if rand >= 2 && rand < 3 {
			robotType, _ := strconv.Atoi(string(sdStorage.Robot_10_30_M))
			needRobotType[robotType] += addV
		} else if rand >= 3 && rand < 22 {
			robotType, _ := strconv.Atoi(string(sdStorage.Robot_1_10_M))
			needRobotType[robotType] += addV
		} else if rand >= 22 && rand < 35 {
			robotType, _ := strconv.Atoi(string(sdStorage.Robot_500_1_M))
			needRobotType[robotType] += addV
		} else if rand >= 35 && rand < 55 {
			robotType, _ := strconv.Atoi(string(sdStorage.Robot_100_500_K))
			needRobotType[robotType] += addV
		} else if rand >= 55 && rand < 58 {
			robotType, _ := strconv.Atoi(string(sdStorage.Robot_50_100_K))
			needRobotType[robotType] += addV
		} else if rand >= 58 && rand < 84 {
			robotType, _ := strconv.Atoi(string(sdStorage.Robot_20_50_K))
			needRobotType[robotType] += addV
		} else if rand >= 84 && rand < 92 {
			robotType, _ := strconv.Atoi(string(sdStorage.Robot_1_20_K))
			needRobotType[robotType] += addV
		} else {
			robotType, _ := strconv.Atoi(string(yxxStorage.Robot_0_1_K))
			needRobotType[robotType] += addV
		}
	}
	return needRobotType
}
func (this *MyTable) RobotGenerate(num int, tp int) { //生成机器人
	for {
		if this.RoomState != ROOM_WAITING_START && this.RoomState != ROOM_WAITING_JIESUAN {
			return
		}
		if num > 0 {
			robot := common2.RandBotN(num, this.Rand)
			for _, v := range robot {
				find := false
				for _, v1 := range this.PlayerList {
					if v1.Role == ROBOT && v.Oid.String() == v1.UserID {
						find = true
						break
					}
				}
				if !find {
					ret := this.PutQueue(protocol.RobotEnter, v, tp)
					if ret == nil {
						num -= 1
					}
				}
			}
		} else if num < 0 {
			for _, v := range this.PlayerList {
				if v.Role == ROBOT {
					if v.Yxb >= int64(this.RobotYxbConf[tp].MinBalance) && v.Yxb < int64(this.RobotYxbConf[tp].MaxBalance) {
						ret := this.PutQueue(protocol.RobotQuitTable, v.UserID)
						if ret == nil {
							num += 1
						}
						if num == 0 {
							break
						}
					}
				}
			}

		}

		if num == 0 {
			break
		}
	}
}
func (this *MyTable) RobotAdd(start bool) { //添加机器人
	hour := time.Now().Hour()
	startHour := 0
	if hour >= 0 && hour < 6 {
		startHour = 0
	} else if hour >= 6 && hour < 12 {
		startHour = 6
	} else if hour >= 12 && hour < 18 {
		startHour = 12
	} else if hour >= 18 && hour <= 23 {
		startHour = 18
	}
	robotConf := sdStorage.GetTableRobotConfByHour(this.tableID, startHour)
	curRobotType := map[int]int{} //统计当前各type机器人的数量
	for _, v := range this.PlayerList {
		if v.Role == ROBOT && v.Yxb > 50000000 {
			this.PutQueue(protocol.RobotQuitTable, v.UserID)
		}
	}
	curBotNum := 0
	for _, v := range this.PlayerList {
		if v.Role == ROBOT {
			for _, v1 := range this.RobotYxbConf {
				if v.Yxb >= int64(v1.MinBalance) && v.Yxb < int64(v1.MaxBalance) {
					curRobotType[v1.RobotType] += 1
					break
				}
			}
			curBotNum += 1
		}
	}
	for i := 0; i < 9; i++ {
		needRobot := 0
		if i == 8 {
			needRobot = curBotNum/100 - curRobotType[i]
		} else if i == 7 {
			needRobot = curBotNum/100 - curRobotType[i]
		} else if i == 6 {
			needRobot = curBotNum*19/100 - curRobotType[i] + 1
		} else if i == 5 {
			needRobot = curBotNum*13/100 - curRobotType[i]
		} else if i == 4 {
			needRobot = curBotNum*20/100 - curRobotType[i] + 1
		} else if i == 3 {
			needRobot = curBotNum*3/100 - curRobotType[i]
		} else if i == 2 {
			needRobot = curBotNum*26/100 - curRobotType[i] + 1
		} else if i == 1 {
			needRobot = curBotNum*8/100 - curRobotType[i]
		} else if i == 0 {
			needRobot = curBotNum*8/100 - curRobotType[i] + 1
		}
		if needRobot != 0 {
			curBotNum += needRobot
			this.RobotGenerate(needRobot, i)
		}
	}
	needRobotType := this.CalcChangeRobot(curBotNum, robotConf, start) //统计需要变换各type机器人的数量

	for _, v := range this.RobotYxbConf {
		for k1, v1 := range needRobotType {
			if v.RobotType == k1 {
				this.RobotGenerate(v1, v.RobotType)
			}
		}
	}

}
func (this *MyTable) RobotEnter(robot common2.Bot, tp int) bool {
	//player := &room.BasePlayerImp{}

	idx := this.GetPlayerIdx(robot.Oid.Hex())
	if idx >= 0 {
		//	log.Info("you already in room")
		return false
	}
	tableInfo := sdStorage.GetTableInfo(this.tableID)

	pl := PlayerList{
		//session: session,
		Yxb:              this.GenerateVndBalance(this.RobotYxbConf[tp].MinBalance, this.RobotYxbConf[tp].MaxBalance),
		UserID:           robot.Oid.Hex(),
		XiaZhuResult:     map[sdStorage.XiaZhuResult][]int64{},
		LastXiaZhuResult: map[sdStorage.XiaZhuResult][]int64{},
		XiaZhuResultTotal: map[sdStorage.XiaZhuResult]int64{
			sdStorage.SINGLE:     0,
			sdStorage.DOUBLE:     0,
			sdStorage.Red4White0: 0,
			sdStorage.Red0White4: 0,
			sdStorage.Red3White1: 0,
			sdStorage.Red1White3: 0,
		},

		Name: robot.NickName,
		Head: robot.Avatar,
		Role: ROBOT,
	}

	this.PlayerList = append(this.PlayerList, pl)

	this.PlayerNum = len(this.PlayerList)
	tableInfo.PlayerNum = this.PlayerNum

	sdStorage.UpsertTableInfo(tableInfo, this.tableID)

	return true
}
func (this *MyTable) RobotQuitTable(userID string) bool {
	//player := &room.BasePlayerImp{}

	idx := this.GetPlayerIdx(userID)
	if idx == -1 {
		//log.Info("you not in room  userid = %s",userID)
		return false
	}
	tableInfo := sdStorage.GetTableInfo(this.tableID)
	if this.RoomState == ROOM_WAITING_XIAZHU { //下注状态不能退出房间
		for _, v := range this.PlayerList[idx].XiaZhuResult {
			for _, v1 := range v {
				if v1 > 0 {
					log.Info("cant leave room")
					return false
				}
			}
		}
	}
	this.PlayerList = append(this.PlayerList[:idx], this.PlayerList[idx+1:]...)
	this.PlayerNum = len(this.PlayerList)
	tableInfo.PlayerNum = this.PlayerNum
	sdStorage.UpsertTableInfo(tableInfo, this.tableID)

	return true
}
func (this *MyTable) RobotDealXiaZhu(xiaZhuV int64, xiaZhuPos sdStorage.XiaZhuResult, robot PlayerList) {
	idx := this.GetPlayerIdx(robot.UserID)
	if idx == -1 {
		return
	}

	val, ok := this.XiaZhuTotal.Load(xiaZhuPos)
	if ok {
		total, _ := utils.ConvertInt(val)
		total += xiaZhuV
		this.XiaZhuTotal.Store(xiaZhuPos, total)
	}

	this.PlayerList[idx].XiaZhuResult[xiaZhuPos] = append(this.PlayerList[idx].XiaZhuResult[xiaZhuPos], xiaZhuV)
	this.PlayerList[idx].XiaZhuResultTotal[xiaZhuPos] += xiaZhuV

	this.PlayerList[idx].Yxb -= xiaZhuV
	info := struct {
		UserID    string
		XiaZhuPos sdStorage.XiaZhuResult
		XiaZhuV   int64
	}{
		UserID:    robot.UserID,
		XiaZhuPos: xiaZhuPos,
		XiaZhuV:   xiaZhuV,
	}
	_ = this.sendPackToAll(game.Push, info, protocol.XiaZhu, nil)

	tableInfoRet := this.GetTableInfo(false)
	_ = this.sendPackToAll(game.Push, tableInfoRet, protocol.UpdateTableInfo, nil)

}
func (this *MyTable) RobotXiaZhu(robot PlayerList, msg map[string]interface{}) (err error) {
	if this.RoomState != ROOM_WAITING_XIAZHU {
		log.Info("----------------room state not xia zhu roomstate = %d", this.RoomState)
		return
	}
	xiaZhuV := msg["xiaZhuV"].(int64)
	xiaZhuPos := sdStorage.XiaZhuResult(msg["pos"].(string)) //鱼虾蟹下的哪一种图案
	if xiaZhuPos != sdStorage.SINGLE &&
		xiaZhuPos != sdStorage.DOUBLE &&
		xiaZhuPos != sdStorage.Red4White0 &&
		xiaZhuPos != sdStorage.Red0White4 &&
		xiaZhuPos != sdStorage.Red3White1 &&
		xiaZhuPos != sdStorage.Red1White3 {
		log.Info("Xia zhu pos not correct pod = %s", xiaZhuPos)
		return
	}

	if robot.Yxb < xiaZhuV {
		log.Info("------------------------- player yxb not enough yxb = %d,num = %d", robot.Yxb, xiaZhuV)
		return
	}

	this.RobotDealXiaZhu(xiaZhuV, xiaZhuPos, robot)
	return nil
}
