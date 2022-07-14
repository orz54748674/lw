package sd

import (
	"encoding/json"
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
	"vn/storage/sdStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) ControlResults(xiaZhuTotal int64) map[string]sdStorage.Result { //控制开盘结果
	//遍历所有结果
	var resultsList []map[string]sdStorage.Result
	gameProfit := gameStorage.QueryProfit(game.SeDie)
	resultsList = []map[string]sdStorage.Result{}

	for i := 1; i < 3; i++ { //计算所有不亏的组合
		for j := 1; j < 3; j++ {
			for k := 1; k < 3; k++ {
				for h := 1; h < 3; h++ {
					results := map[string]sdStorage.Result{
						"1": sdStorage.Result(strconv.Itoa(i)),
						"2": sdStorage.Result(strconv.Itoa(j)),
						"3": sdStorage.Result(strconv.Itoa(k)),
						"4": sdStorage.Result(strconv.Itoa(h)),
					}
					//计算真实正常赔付
					redNum := 0
					for _, v := range results {
						if v == sdStorage.RED {
							redNum++
						}
					}
					singleFLag := false
					if redNum%2 == 1 {
						singleFLag = true
					}
					var prizeResults []sdStorage.XiaZhuResult
					prizeResults = []sdStorage.XiaZhuResult{}

					if singleFLag {
						prizeResults = append(prizeResults, sdStorage.SINGLE)
					} else {
						prizeResults = append(prizeResults, sdStorage.DOUBLE)
					}
					switch redNum {
					case 0:
						prizeResults = append(prizeResults, sdStorage.Red0White4)
					case 1:
						prizeResults = append(prizeResults, sdStorage.Red1White3)
					case 3:
						prizeResults = append(prizeResults, sdStorage.Red3White1)
					case 4:
						prizeResults = append(prizeResults, sdStorage.Red4White0)
					}

					var playerPrize int64 = 0 //玩家中奖金额

					for _, player := range this.PlayerList { //遍历玩家
						if player.Role == USER {
							for seat, v := range player.XiaZhuResultTotal { //遍历该玩家下注
								isWinning := false
								for _, res := range prizeResults { //遍历开奖结果
									if seat == res { //单注中奖
										playerPrize += v * this.GameConf.OddsList[seat] //累计玩家中奖金额
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
						resultsList = append(resultsList, results)
					}
				}
			}
		}
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
	this.CountDown = this.GameConf.JieSuanTime
	this.SwitchRoomState()
	//先抽流水
	var xiaZhuTotal int64 = 0
	for _, v := range this.RealXiaZhuTotal {
		xiaZhuTotal += v
	}
	value := xiaZhuTotal * int64(this.GameConf.BotProfitPerThousand) / 1000
	gameStorage.IncProfit("", game.SeDie, 0, xiaZhuTotal-value, value)

	this.Results = this.ControlResults(xiaZhuTotal) //控制开奖

	resultsRecord := sdStorage.GetResultsRecord(this.tableID)
	redNum := 0
	for _, v := range this.Results {
		if v == sdStorage.RED {
			redNum++
		}
	}
	singleFLag := false
	if redNum%2 == 1 {
		singleFLag = true
	}
	var result sdStorage.XiaZhuResult
	if singleFLag {
		result = sdStorage.SINGLE
	} else {
		result = sdStorage.DOUBLE
	}
	resultsRecord.Results = append(resultsRecord.Results, result)
	idx := len(resultsRecord.Results) - resultsRecord.ResultsRecordNum
	if idx > 0 {
		resultsRecord.Results = append(resultsRecord.Results[:0], resultsRecord.Results[idx:]...)
	}
	resultsRecord.SingleNum = 0
	resultsRecord.DoubleNum = 0
	for _, v := range resultsRecord.Results {
		if v == sdStorage.SINGLE {
			resultsRecord.SingleNum++
		} else {
			resultsRecord.DoubleNum++
		}
	}
	sdStorage.UpsertResultsRecord(resultsRecord, this.tableID)

	//计算正常赔付
	var realTotalCommission int64 = 0 //真实总抽水

	if singleFLag {
		this.PrizeResults = append(this.PrizeResults, sdStorage.SINGLE)
	} else {
		this.PrizeResults = append(this.PrizeResults, sdStorage.DOUBLE)
	}
	switch redNum {
	case 0:
		this.PrizeResults = append(this.PrizeResults, sdStorage.Red0White4)
	case 1:
		this.PrizeResults = append(this.PrizeResults, sdStorage.Red1White3)
	case 3:
		this.PrizeResults = append(this.PrizeResults, sdStorage.Red3White1)
	case 4:
		this.PrizeResults = append(this.PrizeResults, sdStorage.Red4White0)
	}
	for idx, player := range this.PlayerList { //遍历玩家
		for seat, v := range player.XiaZhuResultTotal { //遍历该玩家下注
			isWinning := false
			for _, res := range this.PrizeResults { //遍历开奖结果
				if seat == res { //中奖
					//算抽水
					total := v * this.GameConf.OddsList[seat]
					commission := int64(this.GameConf.ProfitPerThousand) * total / 1000
					this.PlayerList[idx].TotalBackYxb += total - commission
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
			}
		}
	}

	//存储玩家钱包
	var realTotalPay int64 = -realTotalCommission //赔付给真实玩家的总数 抽水需要去掉
	resultStr, _ := json.Marshal(this.Results)
	for k, v := range this.PlayerList {
		if v.Role != ROBOT {
			if v.TotalBackYxb > 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeIncome, walletStorage.EventGameSd, this.EventID, v.TotalBackYxb)
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
				betDetails := map[sdStorage.XiaZhuResult]interface{}{
					sdStorage.DOUBLE:     v.XiaZhuResultTotal[sdStorage.DOUBLE],
					sdStorage.SINGLE:     v.XiaZhuResultTotal[sdStorage.SINGLE],
					sdStorage.Red1White3: v.XiaZhuResultTotal[sdStorage.Red1White3],
					sdStorage.Red3White1: v.XiaZhuResultTotal[sdStorage.Red3White1],
					sdStorage.Red0White4: v.XiaZhuResultTotal[sdStorage.Red0White4],
					sdStorage.Red4White0: v.XiaZhuResultTotal[sdStorage.Red4White0],
				}
				betDetailsStr, _ := json.Marshal(betDetails)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				betRecordParam := gameStorage.BetRecordParam{
					Uid:        v.UserID,
					GameType:   game.SeDie,
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
				}
				gameStorage.InsertBetRecord(betRecordParam)
				pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), totalXiaZhu, this.EventID, game.SeDie)
			}

		} else {
			this.PlayerList[k].Yxb += v.TotalBackYxb
		}
	}
	gameStorage.IncProfit("", game.SeDie, realTotalCommission, realTotalPay, 0)
	pl := this.DeepCopyPlayerList(this.PlayerList)
	go func() {
		time.Sleep(time.Second * 5)
		for _, v := range pl {
			if v.Role == USER || v.Role == Agent {
				this.notifyWallet(v.UserID)
			}
		}
	}()
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
	realPlayerNum := 0
	//记录玩家的一些状态值(没下注的次数)
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
			gameStorage.IncGameWinLoseScore(game.SeDie, v.Name, v.TotalBackYxb-xiaZhu)
		} else {
			this.PlayerList[k].NotXiaZhuCnt += 1
		}
		if v.Role == USER || v.Role == Agent {
			//发送前端的数据结构
			this.JieSuanData = JiesuanData{
				RoomState:     this.RoomState,
				CountDown:     this.CountDown,
				Results:       this.Results,
				PrizeResults:  this.PrizeResults,
				XiaZhuTime:    this.GameConf.XiaZhuTime,
				JieSuanTime:   this.GameConf.JieSuanTime,
				ReadyGameTime: this.GameConf.ReadyGameTime,
				ResultsRecord: resultsRecord,

				PositionInfo: positionInfo,
				TotalBackYxb: v.TotalBackYxb,
			}
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				session, _ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(), game.Push, this.JieSuanData, protocol.JieSuan, nil)
			}
		}
		if v.TotalBackYxb > 0 && v.Role == USER {
			lobbyStorage.Win(utils.ConvertOID(v.UserID), v.Name, v.TotalBackYxb, game.SeDie, false)
		}
		if v.Role == USER {
			realPlayerNum += 1
		}
	}

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
				activityStorage.UpsertGameDataInBet(v.UserID, game.SeDie, 0)
				activity.CalcEncouragementFunc(v.UserID)
			}
		}
	}
	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)
	reboot := gameStorage.QueryGameReboot(game.SeDie)
	if reboot == "true" {
		this.RoomState = ROOM_END
	}
	this.SeqExecFlag = true
	//log.Info("tableID：%s real bet: %v  total back: %v plat win: %v playerNum: %d", this.tableID,xiaZhuTotal, -realTotalPay, xiaZhuTotal + realTotalPay, realPlayerNum)
}
