package cardLhd

import (
	"encoding/json"
	"strconv"
	"time"
	"vn/common/protocol"
	"vn/common/utils"
	basegate "vn/framework/mqant/gate/base"
	"vn/game"
	"vn/game/activity"
	vGate "vn/gate"
	"vn/storage/activityStorage"
	"vn/storage/cardStorage/cardLhdStorage"
	"vn/storage/gameStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) GenerateRandResults(xiaZhuTotal int64) bool { //
	tCard := make([]int, len(card))
	copy(tCard, card)
	for i := 1; i < 3; i++ {
		ret := this.RandInt64(1, int64(len(tCard)+1))
		this.Results[strconv.Itoa(i)] = tCard[ret-1]
		tCard = append(tCard[:ret-1], tCard[ret:]...)
	}
	gameProfit := gameStorage.QueryProfit(game.CardLhd)
	//遍历所有结果
	result := cardLhdStorage.XiaZhuResult("")
	if this.Results["1"]%0x10 > this.Results["2"]%0x10 {
		result = cardLhdStorage.LONG
	} else if this.Results["1"]%0x10 < this.Results["2"]%0x10 {
		result = cardLhdStorage.HU
	} else if this.Results["1"]%0x10 == this.Results["2"]%0x10 {
		result = cardLhdStorage.HE
	}

	//计算真实正常赔付
	var playerPrize int64 = 0 //玩家中奖金额

	for _, player := range this.PlayerList { //遍历玩家
		if player.Role == USER {
			for seat, v := range player.XiaZhuResultTotal { //遍历该玩家下注
				isWinning := false
				if seat == result { //单注中奖
					playerPrize += v * this.GameConf.OddsList[result] //累计玩家中奖金额
					isWinning = true
				} else if result == cardLhdStorage.HE {
					playerPrize += v / 2 //退一半
				}
				if isWinning { //中奖返回本金
					playerPrize += v
				}
			}
		}
	}

	if gameProfit.BotBalance >= playerPrize || playerPrize < xiaZhuTotal { //当前剩余够赔
		return true
	}
	playerPrize = 0

	tmp := this.Results["1"]
	this.Results["1"] = this.Results["2"]
	this.Results["2"] = tmp

	for _, player := range this.PlayerList { //遍历玩家
		if player.Role == USER {
			for seat, v := range player.XiaZhuResultTotal { //遍历该玩家下注
				isWinning := false
				if seat == result { //单注中奖
					playerPrize += v * this.GameConf.OddsList[result] //累计玩家中奖金额
					isWinning = true
				} else if result == cardLhdStorage.HE {
					playerPrize += v / 2 //退一半
				}
				if isWinning { //中奖返回本金
					playerPrize += v
				}
			}
		}
	}

	if gameProfit.BotBalance >= playerPrize || playerPrize < xiaZhuTotal { //当前剩余够赔
		return true
	}

	return false
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
	gameStorage.IncProfit("", game.CardLhd, 0, xiaZhuTotal-value, value)

	cnt := 0
	for true {
		res := this.GenerateRandResults(xiaZhuTotal) //控制开奖
		if res {
			break
		}
		if cnt > 10 {
			break
		}
		cnt++
	}

	result := cardLhdStorage.XiaZhuResult("")
	if this.Results["1"]%0x10 > this.Results["2"]%0x10 {
		result = cardLhdStorage.LONG
	} else if this.Results["1"]%0x10 < this.Results["2"]%0x10 {
		result = cardLhdStorage.HU
	} else if this.Results["1"]%0x10 == this.Results["2"]%0x10 {
		result = cardLhdStorage.HE
	}

	resultsRecord := cardLhdStorage.GetResultsRecord(this.tableID)
	resultsRecord.Results = append(resultsRecord.Results, result)
	idx := len(resultsRecord.Results) - resultsRecord.ResultsRecordNum
	if idx > 0 {
		resultsRecord.Results = append(resultsRecord.Results[:0], resultsRecord.Results[idx:]...)
	}
	results := map[cardLhdStorage.XiaZhuResult]int{}
	total := 0
	for _, v := range resultsRecord.Results {
		results[v] += 1
		total += 1
	}
	for i := len(resultsRecord.Results) - 1; i >= 0; i-- {
		results[resultsRecord.Results[i]] += 1
		total += 1
		if total >= 20 {
			break
		}
	}
	resultsRecord.ResultsWinRate[cardLhdStorage.LONG] = results[cardLhdStorage.LONG] * 100 / total
	resultsRecord.ResultsWinRate[cardLhdStorage.HU] = results[cardLhdStorage.HU] * 100 / total
	resultsRecord.ResultsWinRate[cardLhdStorage.HE] = 100 - resultsRecord.ResultsWinRate[cardLhdStorage.LONG] - resultsRecord.ResultsWinRate[cardLhdStorage.HU]

	cardLhdStorage.UpsertResultsRecord(resultsRecord, this.tableID)

	//计算正常赔付
	var playerPrize int64 = 0         //所有玩家中奖金额
	var realTotalCommission int64 = 0 //真实总抽水

	for idx, player := range this.PlayerList { //遍历玩家
		for seat, v := range player.XiaZhuResultTotal { //遍历该玩家下注
			isWinning := false
			if seat == result { //中奖
				//算抽水
				prize := v * this.GameConf.OddsList[result]
				commission := prize * int64(this.GameConf.ProfitPerThousand) / 1000
				this.PlayerList[idx].TotalBackYxb += prize - commission
				playerPrize += prize //所有玩家中奖金额
				if player.Role == USER {
					realTotalCommission += commission
					this.PlayerList[idx].SysProfit += commission
				}
				isWinning = true
			} else if result == cardLhdStorage.HE {
				this.PlayerList[idx].TotalBackYxb += v / 2 //退一半
				playerPrize += v / 2                       //退一半
			}
			if isWinning { //返回本金
				this.PlayerList[idx].TotalBackYxb += v
				playerPrize += v //所有玩家中奖金额
			}
		}
	}

	//存储玩家钱包
	var realTotalPay = -realTotalCommission //赔付给真实玩家的总数 抽水需要去掉
	backendResults := make(map[string]interface{})
	for k, v := range this.Results {
		backendResults[k] = switchBackendCard[v]
	}
	resultStr, _ := json.Marshal(backendResults)
	for k, v := range this.PlayerList {
		if v.Role != ROBOT {
			if v.TotalBackYxb > 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeIncome, walletStorage.EventGameCardLhd, this.EventID, v.TotalBackYxb)
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
				betDetails := map[cardLhdStorage.XiaZhuResult]interface{}{
					cardLhdStorage.LONG: v.XiaZhuResultTotal[cardLhdStorage.LONG],
					cardLhdStorage.HU:   v.XiaZhuResultTotal[cardLhdStorage.HU],
					cardLhdStorage.HE:   v.XiaZhuResultTotal[cardLhdStorage.HE],
				}
				betDetailsStr, _ := json.Marshal(betDetails)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				betRecordParam := gameStorage.BetRecordParam{
					Uid:        v.UserID,
					GameType:   game.CardLhd,
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
				if v.Role != USER {
					betRecordParam.SysProfit = 0
					betRecordParam.BotProfit = 0
				}
				gameStorage.InsertBetRecord(betRecordParam)
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
	gameStorage.IncProfit("", game.CardLhd, realTotalCommission, realTotalPay, 0)
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

	clientResult := make(map[string]interface{})
	clientResult["1"] = result
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
			gameStorage.IncGameWinLoseScore(game.CardLhd, v.Name, v.TotalBackYxb-xiaZhu)
		} else {
			this.PlayerList[k].NotXiaZhuCnt += 1
		}
		if v.Role == USER || v.Role == Agent {
			this.JieSuanData = JiesuanData{
				RoomState:     this.RoomState,
				CountDown:     this.CountDown,
				Poker:         this.Results,
				XiaZhuTime:    this.GameConf.XiaZhuTime,
				JieSuanTime:   this.GameConf.JieSuanTime,
				ReadyGameTime: this.GameConf.ReadyGameTime,
				PositionInfo:  positionInfo,
				TotalBackYxb:  v.TotalBackYxb,
				Results:       clientResult,
				ResultsRecord: resultsRecord,
			}
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				session, _ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(), game.Push, this.JieSuanData, protocol.JieSuan, nil)
			}
		}

		if v.TotalBackYxb > 0 && v.Role == USER {
			lobbyStorage.Win(utils.ConvertOID(v.UserID), v.Name, v.TotalBackYxb, game.CardLhd, false)
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
				activityStorage.UpsertGameDataInBet(v.UserID, game.CardLhd, 0)
				activity.CalcEncouragementFunc(v.UserID)
			}
		}
	}
	reboot := gameStorage.QueryGameReboot(game.CardLhd)
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
