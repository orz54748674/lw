package yxx

import (
	"encoding/json"
	"sort"
	"strconv"
	"time"
	"vn/common/protocol"
	"vn/common/utils"
	basegate "vn/framework/mqant/gate/base"
	"vn/game"
	"vn/game/activity"
	"vn/game/pay"
	vGate "vn/gate"
	"vn/storage/activityStorage"
	"vn/storage/gameStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/walletStorage"
	"vn/storage/yxxStorage"
)

func (this *MyTable) ControlResults(xiaZhuTotal int64) map[string]yxxStorage.XiaZhuResult { //控制开盘结果
	gameProfit := gameStorage.QueryProfit(game.YuXiaXie)
	//遍历所有结果
	var resultsList []map[string]yxxStorage.XiaZhuResult
	var prizeList []map[string]yxxStorage.XiaZhuResult

	resultsList = []map[string]yxxStorage.XiaZhuResult{}
	prizeList = []map[string]yxxStorage.XiaZhuResult{}

	for i := 1; i < 7; i++ { //计算所有不亏的组合
		for j := 1; j < 7; j++ {
			for k := 1; k < 7; k++ {
				results := map[string]yxxStorage.XiaZhuResult{
					"1": yxxStorage.XiaZhuResult(strconv.Itoa(i)),
					"2": yxxStorage.XiaZhuResult(strconv.Itoa(j)),
					"3": yxxStorage.XiaZhuResult(strconv.Itoa(k)),
				}

				//计算真实正常赔付
				var playerPrize int64 = 0 //玩家中奖金额

				for _, player := range this.PlayerList { //遍历玩家
					if player.Role == USER {
						for seat, v := range player.XiaZhuResultTotal { //遍历该玩家下注
							isWinning := false
							for _, res := range results { //遍历开奖结果
								if seat == res { //单注中奖
									playerPrize += v //累计玩家中奖金额
									isWinning = true
								}
							}
							if isWinning { //中奖返回本金
								playerPrize += v
							}
						}
					}
				}

				if gameProfit.BotBalance >= playerPrize || playerPrize < xiaZhuTotal { //当前剩余够赔
					if results["1"] == results["2"] && results["2"] == results["3"] {
						if this.PrizeSwitch { //开大奖
							//中大奖
							pattern := results["1"]
							if this.XiaZhuTotal[pattern] > 0 { //有人中奖
								var totalPrize = playerPrize
								totalPrize += this.PrizePool * this.RealXiaZhuTotal[pattern] / this.XiaZhuTotal[pattern]
								if gameProfit.BotBalance >= totalPrize { //开大奖
									prizeList = append(prizeList, results)
								}
							}
						}
					} else {
						resultsList = append(resultsList, results)
					}

				}

			}
		}
	}

	if len(prizeList) > 0 { //开大奖
		idx := this.RandInt64(1, int64(len(prizeList))+1) - 1
		//log.Info("----------------------------1111111111111111111111-----------------------",prizeList,"--",idx)
		return prizeList[idx]
	}
	if len(resultsList) > 0 { //随机图案
		idx := this.RandInt64(1, int64(len(resultsList))+1) - 1
		//	log.Info("----------------------------2222222222222222222222-----------------------",resultsList,"--",idx)
		return resultsList[idx]
	}
	//	log.Info("----------------------------3333333333-----------------------")
	return this.Results
}
func (this *MyTable) JieSuan() {
	//log.Info("-------------------------start jie suan tableid = %s",this.tableID)
	//start := time.Now().UnixNano()

	this.CountDown = this.GameConf.JieSuanTime
	this.SwitchRoomState()
	//先抽流水
	var xiaZhuTotal int64 = 0
	for _, v := range this.RealXiaZhuTotal {
		xiaZhuTotal += v
	}
	value := xiaZhuTotal * int64(this.GameConf.BotProfitPerThousand) / 1000
	gameStorage.IncProfit("", game.YuXiaXie, 0, xiaZhuTotal-value, value)

	if this.Results["1"] == this.Results["2"] && this.Results["2"] == this.Results["3"] { //记录开大奖
		this.PrizeSwitch = true
	}
	this.Results = this.ControlResults(xiaZhuTotal) //控制开奖

	resultsRecord := yxxStorage.GetResultsRecord(this.tableID)
	resultsRecord.Results = append(resultsRecord.Results, this.Results)
	idx := len(resultsRecord.Results) - resultsRecord.ResultsRecordNum
	if idx > 0 {
		resultsRecord.Results = append(resultsRecord.Results[:0], resultsRecord.Results[idx:]...)
	}
	results := map[yxxStorage.XiaZhuResult]int{}
	total := 0
	for _, v := range resultsRecord.Results {
		for _, v1 := range v {
			results[v1] += 1
			total += 1
		}
	}
	resultsRecord.ResultsWinRate[yxxStorage.YU] = results[yxxStorage.YU] * 100 / total
	resultsRecord.ResultsWinRate[yxxStorage.XIA] = results[yxxStorage.XIA] * 100 / total
	resultsRecord.ResultsWinRate[yxxStorage.XIE] = results[yxxStorage.XIE] * 100 / total
	resultsRecord.ResultsWinRate[yxxStorage.LU] = results[yxxStorage.LU] * 100 / total
	resultsRecord.ResultsWinRate[yxxStorage.JI] = results[yxxStorage.JI] * 100 / total
	resultsRecord.ResultsWinRate[yxxStorage.HULU] = 100 - resultsRecord.ResultsWinRate[yxxStorage.YU] - resultsRecord.ResultsWinRate[yxxStorage.XIA] - resultsRecord.ResultsWinRate[yxxStorage.XIE] - resultsRecord.ResultsWinRate[yxxStorage.JI] - resultsRecord.ResultsWinRate[yxxStorage.LU]

	yxxStorage.UpsertResultsRecord(resultsRecord, this.tableID)

	isPrizePool := false              //是否中大奖
	var realTotalCommission int64 = 0 //真实总抽水
	curPoolTotal := this.PrizePool
	//中大奖
	if this.Results["1"] == this.Results["2"] && this.Results["2"] == this.Results["3"] {
		pattern := this.Results["1"]
		if this.XiaZhuTotal[pattern] > 0 { //有人中奖
			prizeRecord := yxxStorage.GetPrizeRecord(this.tableID)
			prizeRecord.CurCnt += 1
			prizeRecordList := yxxStorage.PrizeRecordList{
				Cnt:         prizeRecord.CurCnt,
				CreateTime:  time.Now(),
				Result:      this.Results["1"],
				ResultsPool: this.PrizePool,
				PrizeList:   []yxxStorage.PrizeList{},
			}
			for k1, v1 := range this.PlayerList { //找该玩家
				if v1.XiaZhuResultTotal[pattern] > 0 {
					this.PlayerList[k1].ResultsPool = this.PrizePool * v1.XiaZhuResultTotal[pattern] / this.XiaZhuTotal[pattern]
					//算抽水
					commission := this.PlayerList[k1].ResultsPool * int64(this.GameConf.ProfitPerThousand) / 1000
					this.PlayerList[k1].ResultsPool -= commission
					this.PlayerList[k1].TotalBackYxb += this.PlayerList[k1].ResultsPool
					this.PlayerList[k1].IsJackpot = true
					pl := yxxStorage.PrizeList{
						Name:    v1.Name,
						Results: this.PlayerList[k1].ResultsPool,
					}
					prizeRecordList.PrizeList = append(prizeRecordList.PrizeList, pl) //插入中奖玩家

					if v1.Role == USER {
						realTotalCommission += commission
						this.PlayerList[idx].SysProfit += commission
						lobbyStorage.Win(utils.ConvertOID(v1.UserID), v1.Name, this.PlayerList[k1].ResultsPool, game.YuXiaXie, true)
					}

				}

			}

			sort.Slice(prizeRecordList.PrizeList, func(i, j int) bool { //排序
				return prizeRecordList.PrizeList[i].Results > prizeRecordList.PrizeList[j].Results
			})

			prizeRecord.PrizeRecordList = append(prizeRecord.PrizeRecordList, prizeRecordList) //插入中奖期数

			idx := len(prizeRecord.PrizeRecordList) - ResultsPoolNum
			if idx > 0 {
				prizeRecord.PrizeRecordList = append(prizeRecord.PrizeRecordList[:0], prizeRecord.PrizeRecordList[idx:]...)
			}

			results = map[yxxStorage.XiaZhuResult]int{}
			total = 0
			for _, v := range prizeRecord.PrizeRecordList {
				results[v.Result] += 1
				total += 1
			}
			prizeRecord.PrizeWinRate[yxxStorage.YU] = results[yxxStorage.YU] * 100 / total
			prizeRecord.PrizeWinRate[yxxStorage.XIA] = results[yxxStorage.XIA] * 100 / total
			prizeRecord.PrizeWinRate[yxxStorage.XIE] = results[yxxStorage.XIE] * 100 / total
			prizeRecord.PrizeWinRate[yxxStorage.LU] = results[yxxStorage.LU] * 100 / total
			prizeRecord.PrizeWinRate[yxxStorage.JI] = results[yxxStorage.JI] * 100 / total
			prizeRecord.PrizeWinRate[yxxStorage.HULU] = 100 - prizeRecord.PrizeWinRate[yxxStorage.YU] - prizeRecord.PrizeWinRate[yxxStorage.XIA] - prizeRecord.PrizeWinRate[yxxStorage.XIE] - prizeRecord.PrizeWinRate[yxxStorage.LU] - prizeRecord.PrizeWinRate[yxxStorage.JI]

			yxxStorage.UpsertPrizeRecord(prizeRecord, this.tableID)

			this.PrizePool = int64(this.GameConf.InitPrizePool)
			isPrizePool = true
			this.PrizeSwitch = false

			info := struct {
				PrizePool int64
				CurInPool int64
			}{
				PrizePool: this.PrizePool,
				CurInPool: 0,
			}
			ret := this.DealProtocolFormat(info, protocol.RefreshPrizePool, nil)
			this.onlinePush.NotifyAllPlayersNR(game.Push, ret)
			this.onlinePush.ExecuteCallBackMsg(this.Trace())
		}
	}
	//计算正常赔付
	var playerPrize int64 = 0 //所有玩家中奖金额

	for idx, player := range this.PlayerList { //遍历玩家
		for seat, v := range player.XiaZhuResultTotal { //遍历该玩家下注
			isWinning := false
			for _, res := range this.Results { //遍历开奖结果
				if seat == res { //中奖
					//算抽水
					commission := v * int64(this.GameConf.ProfitPerThousand) / 1000
					this.PlayerList[idx].TotalBackYxb += v - commission
					playerPrize += v //所有玩家中奖金额
					if player.Role == USER {
						realTotalCommission += commission
						this.PlayerList[idx].SysProfit += commission
					}
					isWinning = true
				}
			}
			if isWinning { //返回本金
				//	this.PlayerList[idx].ResultsChipList = append(this.PlayerList[idx].ResultsChipList, xiaZhuV) //记录中奖筹码
				this.PlayerList[idx].TotalBackYxb += v
				playerPrize += v //所有玩家中奖金额
			}
		}
	}

	if !isPrizePool {
		var totalXiaZhu int64 = 0
		for _, v := range this.XiaZhuTotal {
			totalXiaZhu += v
		}
		var curInPool = (totalXiaZhu - playerPrize) * int64(this.GameConf.PoolScaleThousand) / 1000
		if curInPool > 0 {
			this.CurInPool += curInPool
			this.PrizePool += curInPool

			info := struct {
				PrizePool int64
				CurInPool int64
			}{
				PrizePool: this.PrizePool,
				CurInPool: curInPool,
			}

			ret := this.DealProtocolFormat(info, protocol.RefreshPrizePool, nil)
			this.onlinePush.NotifyAllPlayersNR(game.Push, ret)
			this.onlinePush.ExecuteCallBackMsg(this.Trace())
		}

	}
	//存储玩家钱包
	var realTotalPay = -realTotalCommission //赔付给真实玩家的总数 抽水需要去掉
	tmpRes := make(map[string]interface{})
	tmpRes["1"] = this.Results["1"]
	tmpRes["2"] = this.Results["2"]
	tmpRes["3"] = this.Results["3"]
	if isPrizePool {
		tmpRes["4"] = this.XiaZhuTotal[this.Results["1"]]
		tmpRes["5"] = curPoolTotal
	}
	resultStr, _ := json.Marshal(tmpRes)
	for k, v := range this.PlayerList {
		if v.Role != ROBOT {
			if v.TotalBackYxb > 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeIncome, walletStorage.EventGameYxx, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				sb := vGate.QuerySessionBean(v.UserID)
				if sb != nil {
					session, _ := basegate.NewSession(this.app, sb.Session)
					playerInfoRet := this.GetPlayerInfo(v.UserID, false)
					_ = this.sendPack(session.GetSessionID(), game.Push, playerInfoRet, protocol.UpdatePlayerInfo, nil)
				}
				if v.Role == USER {
					realTotalPay -= v.TotalBackYxb
				}
			}
			var totalXiaZhu int64 = 0
			for _, v1 := range v.XiaZhuResultTotal {
				totalXiaZhu += v1
			}
			if totalXiaZhu > 0 {
				this.PlayerList[k].BotProfit = totalXiaZhu * int64(this.GameConf.BotProfitPerThousand) / 1000
				betDetails := map[yxxStorage.XiaZhuResult]interface{}{
					yxxStorage.YU:   v.XiaZhuResultTotal[yxxStorage.YU],
					yxxStorage.XIA:  v.XiaZhuResultTotal[yxxStorage.XIA],
					yxxStorage.XIE:  v.XiaZhuResultTotal[yxxStorage.XIE],
					yxxStorage.HULU: v.XiaZhuResultTotal[yxxStorage.HULU],
					yxxStorage.LU:   v.XiaZhuResultTotal[yxxStorage.LU],
					yxxStorage.JI:   v.XiaZhuResultTotal[yxxStorage.JI],
				}
				betDetailsStr, _ := json.Marshal(betDetails)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				betRecordParam := gameStorage.BetRecordParam{
					Uid:        v.UserID,
					GameType:   game.YuXiaXie,
					Income:     v.TotalBackYxb - totalXiaZhu,
					BetAmount:  totalXiaZhu,
					CurBalance: this.PlayerList[k].Yxb + wallet.SafeBalance,
					SysProfit:  this.PlayerList[k].SysProfit,
					BotProfit:  this.PlayerList[k].BotProfit,
					BetDetails: string(betDetailsStr),
					GameId:     strconv.FormatInt(time.Now().Unix(), 10),
					GameNo:     strconv.FormatInt(time.Now().Unix(), 10),
					GameResult: string(resultStr),
					IsSettled:  true,
					IsJackpot:  v.IsJackpot,
				}
				if v.Role != USER {
					betRecordParam.SysProfit = 0
					betRecordParam.BotProfit = 0
				}
				gameStorage.InsertBetRecord(betRecordParam)
				pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), totalXiaZhu, this.EventID, game.YuXiaXie)
			}
		} else {
			this.PlayerList[k].Yxb += v.TotalBackYxb
		}
	}
	pl := this.DeepCopyPlayerList(this.PlayerList)
	go func() {
		time.Sleep(time.Second * 5)
		for _, v := range pl {
			if v.Role == USER || v.Role == Agent {
				this.notifyWallet(v.UserID)
			}
		}
	}()
	gameStorage.IncProfit("", game.YuXiaXie, realTotalCommission, realTotalPay, 0)
	//发送前端的数据结构
	var positionInfo []PositionInfo
	for k, v := range this.PlayerList {
		if k < this.SeatNum {
			info := PositionInfo{
				TotalBackYxb: v.TotalBackYxb,
				UserID:       v.UserID,
				Yxb:          v.Yxb,
			}
			positionInfo = append(positionInfo, info)
		} else {
			break
		}
	}
	//记录玩家的一些状态值(没下注的次数)
	realPlayerNum := 0
	for k, v := range this.PlayerList {
		var xiaZhu int64 = 0
		for _, v1 := range v.XiaZhuResultTotal {
			if v1 > 0 {
				xiaZhu += v1
				break
			}
		}
		if xiaZhu > 0 {
			this.PlayerList[k].NotXiaZhuCnt = 0
			gameStorage.IncGameWinLoseScore(game.YuXiaXie, v.Name, v.TotalBackYxb-xiaZhu)
		} else {
			this.PlayerList[k].NotXiaZhuCnt += 1
		}
		if v.Role == USER || v.Role == Agent {
			this.JieSuanData = JiesuanData{
				RoomState:     this.RoomState,
				CountDown:     this.CountDown,
				PrizePool:     this.PrizePool,
				Results:       this.Results,
				XiaZhuTime:    this.GameConf.XiaZhuTime,
				JieSuanTime:   this.GameConf.JieSuanTime,
				ReadyGameTime: this.GameConf.ReadyGameTime,
				CurInPool:     this.CurInPool,
				PositionInfo:  positionInfo,
				TotalBackYxb:  v.TotalBackYxb,
				IsPrizePool:   isPrizePool,
				PrizeResult:   v.ResultsPool,
			}
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				session, _ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(), game.Push, this.JieSuanData, protocol.JieSuan, nil)
			}
		}

		if v.TotalBackYxb > 0 && v.Role == USER {
			lobbyStorage.Win(utils.ConvertOID(v.UserID), v.Name, v.TotalBackYxb-v.ResultsPool, game.YuXiaXie, false)
		}
		if v.Role == USER {
			realPlayerNum += 1
		}
	}

	tableInfo := yxxStorage.GetTableInfo(this.tableID)
	tableInfo.PrizePool = this.PrizePool
	tableInfo.PrizeSwitch = this.PrizeSwitch
	yxxStorage.UpsertTableInfo(tableInfo, this.tableID)

	for _, v := range this.PlayerList {
		if v.Role != ROBOT { //玩家
			var xiaZhu int64 = 0
			for _, v1 := range v.XiaZhuResultTotal {
				if v1 > 0 {
					xiaZhu += v1
					break
				}
			}
			if xiaZhu > 0 { //有下注
				activityStorage.UpsertGameDataInBet(v.UserID, game.YuXiaXie, 0)
				activity.CalcEncouragementFunc(v.UserID)
			}
		}
	}

	reboot := gameStorage.QueryGameReboot(game.YuXiaXie)
	if reboot == "true" {
		this.RoomState = ROOM_END
	}
	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)

	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)
	//log.Info("tableID：%s real bet: %v  total back: %v plat win: %v playerNum: %d", this.tableID,xiaZhuTotal, -realTotalPay, xiaZhuTotal + realTotalPay, realPlayerNum)
	this.SeqExecFlag = true
}
