package slotDance

import (
	"encoding/json"
	"reflect"
	"strconv"
	"time"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant/gate"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/activity"
	"vn/game/pay"
	vGate "vn/gate"
	"vn/storage/gameStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/slotStorage/slotDanceStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) InitCheckoutData() {
	var jieSuanData JieSuanData
	jieSuanData.WildTimes = 1
	jieSuanData.TotalBackScore = 0
	jieSuanData.WildPositions = []map[int64]int64{}
	jieSuanData.ScatterPositions = []map[int64]int64{}
	jieSuanData.Result = []Result{}
	jieSuanData.ResultPositions = []int64{}
	jieSuanData.MusicType = ""
	jieSuanData.AnimationType = ""
	jieSuanData.TrialData = this.JieSuanData.TrialData
	jieSuanData.FreeData = this.JieSuanData.FreeData
	this.JieSuanData = jieSuanData
}
func (this *MyTable) CalcCheckout(reelList [][]slotDanceStorage.Symbol, coinNum int64, coinValue int64, isFreeGame bool) {
	totalXiaZhu := coinNum * coinValue
	resultSymbol := make([]int64, len(reelList)) //每列的起始位置

Again:
	for k, v := range reelList {
		rand := this.RandInt64(1, int64(len(v)+1))
		rand = rand - 1
		resultSymbol[k] = rand
	}
	var jieSuanData JieSuanData
	//if this.JieSuanData.FreeData.FreeRemainTimes <= 0 && this.JieSuanData.FreeData.FreeTimes == 0{
	//	resultSymbol[0] = 0
	//	resultSymbol[1] = 8
	//	resultSymbol[2] = 8
	//	resultSymbol[3] = 15
	//	resultSymbol[4] = 4
	//}
	jieSuanData.ResultPositions = resultSymbol

	for row := int64(0); row < TotalRows; row++ { //计算每行
		hitGroupTimes := int64(1)        //组合倍数
		baseIdx := resultSymbol[0] + row //第一列的页面显示的图案索引
		lineType := 1                    //连线类型 三连 四连 五连之类的
		if baseIdx >= int64(len(reelList[0])) {
			baseIdx = baseIdx - int64(len(reelList[0]))
		}
		baseSymbol := reelList[0][baseIdx] //第一列的页面显示的图案
		position := map[int64]int64{
			row: 0,
		}
		find := -1
		for k, v := range jieSuanData.Result {
			if v.Symbol == baseSymbol {
				jieSuanData.Result[k].SymbolPositions = append(jieSuanData.Result[k].SymbolPositions, position)
				find = k
				break
			}
		}
		if find > -1 {
			jieSuanData.Result[find].GroupNum += 1
			continue
		}

		symbolPositions := make([]map[int64]int64, 0)
		scatterPosition := make([]map[int64]int64, 0)
		if baseSymbol == slotDanceStorage.SCATTER {
			scatterPosition = append(scatterPosition, position)
		} else {
			symbolPositions = append(symbolPositions, position)
		}

		wildPositions := make([]map[int64]int64, 0)
		haveWild := false
		for col := int64(1); col < int64(len(resultSymbol)); col++ { //第二列之后的
			colSymbolNum := int64(0)
			for rowCalc := int64(0); rowCalc < TotalRows; rowCalc++ {
				symbolIdx := resultSymbol[col] + rowCalc
				if symbolIdx >= int64(len(reelList[col])) {
					symbolIdx = symbolIdx - int64(len(reelList[col]))
				}
				if (reelList[col][symbolIdx] == slotDanceStorage.WILD && baseSymbol != slotDanceStorage.SCATTER) || reelList[col][symbolIdx] == baseSymbol { //
					colSymbolNum++
					position = map[int64]int64{
						rowCalc: col,
					}
					if reelList[col][symbolIdx] == slotDanceStorage.WILD { //中wild的位置
						find := false
						for _, v := range jieSuanData.WildPositions {
							if reflect.DeepEqual(v, position) {
								find = true
								break
							}
						}
						if !find {
							wildPositions = append(wildPositions, position)
						}
						haveWild = true
					} else if baseSymbol == slotDanceStorage.SCATTER {
						scatterPosition = append(scatterPosition, position)
					} else {
						symbolPositions = append(symbolPositions, position)
					}

				}
			}

			if colSymbolNum > 0 {
				hitGroupTimes *= colSymbolNum
				lineType++
			} else { //说明已经断了
				break
			}
		}

		if lineType >= MinWinLine && baseSymbol != slotDanceStorage.SCATTER { //3连以上才有奖励
			if haveWild {
				for _, v := range wildPositions { //中wild的位置
					jieSuanData.WildPositions = append(jieSuanData.WildPositions, v)
				}
			}
			symbolScore := int64(0)
			symbolScore = OddsList[baseSymbol][lineType] * coinNum / BaseCoinNum * hitGroupTimes
			result := Result{
				SymbolPositions: symbolPositions,
				LineType:        lineType,
				Symbol:          baseSymbol,
				SymbolScore:     symbolScore * coinValue,
				CoinValue:       coinValue,
				HaveWild:        haveWild,
			}
			jieSuanData.Result = append(jieSuanData.Result, result)
		} else if lineType > 0 && baseSymbol == slotDanceStorage.SCATTER {
			for _, v := range scatterPosition { //scatter的位置
				jieSuanData.ScatterPositions = append(jieSuanData.ScatterPositions, v)
			}
		}

		if lineType >= 5 && baseSymbol == slotDanceStorage.SCATTER {
			jieSuanData.FreeData.FreeGame = true
		}
	}
	for k, v := range jieSuanData.Result {
		jieSuanData.Result[k].SymbolScore *= v.GroupNum + 1
	}
	for _, v := range jieSuanData.Result {
		jieSuanData.TotalBackScore += v.SymbolScore
	}

	if jieSuanData.TotalBackScore > 0 {
		musicScore := jieSuanData.TotalBackScore * 10 / totalXiaZhu
		if musicScore < 10 {
			jieSuanData.MusicType = WIN1
		} else if musicScore < 30 {
			jieSuanData.MusicType = WIN2
		} else if musicScore < 40 {
			jieSuanData.MusicType = WIN3
		} else if musicScore >= 40 {
			jieSuanData.MusicType = WINBig
			if isFreeGame {
				jieSuanData.AnimationType = BigAnimation2
			} else {
				jieSuanData.AnimationType = BigAnimation1
			}
		}
	}

	if isFreeGame {
		jieSuanData.TotalBackScore *= this.JieSuanData.FreeData.FreeTimes
		jieSuanData.FreeData.FreeStepTimes = this.JieSuanData.FreeData.FreeStepTimes
		jieSuanData.FreeData.FreeTimes = this.JieSuanData.FreeData.FreeTimes
		jieSuanData.FreeData.FreeRemainTimes = this.JieSuanData.FreeData.FreeRemainTimes
		jieSuanData.FreeData.FreeUsedTimes = this.JieSuanData.FreeData.FreeUsedTimes
		jieSuanData.FreeData.FreeTotalScore = this.JieSuanData.FreeData.FreeTotalScore
	}
	if this.ModeType == NORMAL {
		gameProfit := gameStorage.QueryProfitByUser(this.UserID)
		if jieSuanData.TotalBackScore > 0 && (jieSuanData.TotalBackScore > gameProfit.BotBalance || (jieSuanData.FreeData.FreeGame && gameProfit.BotBalance < totalXiaZhu*int64(this.GameConf.FreeGameMinTimes))) {
			goto Again
		} else {
			gameStorage.IncProfitByUser(this.UserID, 0, -jieSuanData.TotalBackScore, 0, jieSuanData.TotalBackScore-totalXiaZhu)
		}
	}
	if jieSuanData.FreeData.FreeGame {
		totalScatter := 0
		for col := int64(0); col < int64(len(resultSymbol)); col++ { //计算每列
			for row := int64(0); row < TotalRows; row++ { //计算每行
				symbolIdx := resultSymbol[col] + row
				if symbolIdx >= int64(len(reelList[col])) {
					symbolIdx = symbolIdx - int64(len(reelList[col]))
				}
				if reelList[col][symbolIdx] == slotDanceStorage.SCATTER {
					totalScatter++
				}
			}
		}
		if totalScatter == 5 {
			jieSuanData.FreeData.FreeStepTimes = append(jieSuanData.FreeData.FreeStepTimes, 1)
		} else if totalScatter == 6 {
			jieSuanData.FreeData.FreeStepTimes = append(jieSuanData.FreeData.FreeStepTimes, 2)
		} else if totalScatter >= 7 {
			jieSuanData.FreeData.FreeStepTimes = append(jieSuanData.FreeData.FreeStepTimes, 3)
		}
		jieSuanData.FreeData.FreeRemainTimes += 10
		if !isFreeGame {
			jieSuanData.FreeData.FreeTimes = jieSuanData.FreeData.FreeStepTimes[0]
		}
	}

	jieSuanData.CoinNum = coinNum
	jieSuanData.CoinValue = coinValue

	//计算是否进free game
	jieSuanData.TrialData = this.JieSuanData.TrialData

	if isFreeGame {
		jieSuanData.FreeData.FreeTotalScore += jieSuanData.TotalBackScore
		if jieSuanData.FreeData.FreeUsedTimes%10 == 0 && jieSuanData.FreeData.FreeRemainTimes > 0 {
			jieSuanData.FreeData.FreeStepTimes = append(jieSuanData.FreeData.FreeStepTimes[:0], jieSuanData.FreeData.FreeStepTimes[1:]...)
		}
		if jieSuanData.FreeData.FreeRemainTimes > 0 {
			jieSuanData.FreeData.FreeTimes += jieSuanData.FreeData.FreeStepTimes[0]
		}
	}
	this.JieSuanData = jieSuanData
}

func (this *MyTable) Spin(session gate.Session, msg map[string]interface{}) (err error) {
	this.IsInCheckout = true
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)

	this.GameConf = slotDanceStorage.GetRoomConf()
	isFreeGame := false
	if this.JieSuanData.FreeData.FreeRemainTimes > 0 {
		this.JieSuanData.FreeData.FreeUsedTimes++
		this.JieSuanData.FreeData.FreeRemainTimes--
		isFreeGame = true
		this.EventID = string(game.SlotDance) + "_" + "Free" + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	} else {
		this.EventID = string(game.SlotDance) + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	this.InitCheckoutData()

	if !isFreeGame {
		this.CoinNum = msg["CoinNum"].(int64)
		this.CoinValue = msg["CoinValue"].(int64)
		totalXiaZhu := this.CoinNum * this.CoinValue
		if this.ModeType == NORMAL {
			wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
			if wallet.VndBalance < totalXiaZhu { //下注金额不足
				error := errCode.BalanceNotEnough
				this.sendPack(session.GetSessionID(), game.Push, "", protocol.Spin, error)
				return nil
			}
			value := totalXiaZhu * int64(this.GameConf.BotProfitPerThousand) / 1000
			gameStorage.IncProfitByUser(this.UserID, 0, totalXiaZhu-value, value, 0)
			bill := walletStorage.NewBill(this.UserID, walletStorage.TypeExpenses, walletStorage.EventGameSlotCs, this.EventID, -totalXiaZhu)
			walletStorage.OperateVndBalance(bill)
		} else {
			if this.JieSuanData.TrialData.VndBalance < totalXiaZhu { //下注金额不足
				error := errCode.CurCanXiaZhuError
				this.sendPack(session.GetSessionID(), game.Push, "", protocol.Spin, error)
				return nil
			}
			this.JieSuanData.TrialData.VndBalance -= totalXiaZhu
		}
	}

	betData := make(map[string]interface{})
	betData["CoinNum"] = this.CoinNum
	betData["CoinValue"] = this.CoinValue
	betDetail, _ := json.Marshal(betData)
	var resultRecordStr string
	if this.ModeType == NORMAL {
		this.CalcCheckout(this.ReelsList, this.CoinNum, this.CoinValue, isFreeGame)
		resultRecordStr = this.DealGameResultRecord(this.JieSuanData.ResultPositions, this.ReelsList)
	} else {
		this.CalcCheckout(this.ReelsListTrial, this.CoinNum, this.CoinValue, isFreeGame)
	}
	if this.ModeType == NORMAL {
		if this.JieSuanData.TotalBackScore > 0 {
			bill := walletStorage.NewBill(this.UserID, walletStorage.TypeIncome, walletStorage.EventGameSlotDance, this.EventID, this.JieSuanData.TotalBackScore)
			walletStorage.OperateVndBalance(bill)

			lobbyStorage.Win(utils.ConvertOID(this.UserID), this.Name, this.JieSuanData.TotalBackScore-this.ResultsPool, game.SlotDance, false)
		}

		wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
		if !isFreeGame {
			totalXiaZhu := this.CoinNum * this.CoinValue
			betRecordParam := gameStorage.BetRecordParam{
				Uid:        this.UserID,
				GameType:   game.SlotDance,
				Income:     this.JieSuanData.TotalBackScore - totalXiaZhu,
				BetAmount:  totalXiaZhu,
				CurBalance: wallet.VndBalance + wallet.SafeBalance,
				SysProfit:  0,
				BotProfit:  0,
				BetDetails: string(betDetail),
				GameId:     this.EventID,
				GameNo:     strconv.FormatInt(time.Now().Unix(), 10),
				GameResult: resultRecordStr,
				IsSettled:  true,
			}
			gameStorage.InsertBetRecord(betRecordParam)
			pay.CheckoutAgentIncome(utils.ConvertOID(this.UserID), totalXiaZhu, this.EventID, game.SlotDance)
		} else {
			betRecordParam := gameStorage.BetRecordParam{
				Uid:        this.UserID,
				GameType:   game.SlotDance,
				Income:     this.JieSuanData.TotalBackScore,
				BetAmount:  0,
				CurBalance: wallet.VndBalance + wallet.SafeBalance,
				SysProfit:  0,
				BotProfit:  0,
				BetDetails: string(betDetail),
				GameId:     this.EventID,
				GameNo:     strconv.FormatInt(time.Now().Unix(), 10),
				GameResult: resultRecordStr,
				IsSettled:  true,
			}
			gameStorage.InsertBetRecord(betRecordParam)
		}
	} else {
		this.JieSuanData.TrialData.VndBalance += this.JieSuanData.TotalBackScore
	}

	sb := vGate.QuerySessionBean(this.UserID)
	if sb != nil {
		session, _ := basegate.NewSession(this.app, sb.Session)
		this.sendPack(session.GetSessionID(), game.Push, this.JieSuanData, protocol.JieSuan, nil)
	}

	if this.ModeType == NORMAL && this.JieSuanData.FreeData.FreeRemainTimes == 0 {
		this.notifyWallet(this.UserID)
	}

	res, _ := json.Marshal(this.JieSuanData)
	log.Info("----------------------slot dance jiesuan ---%s", res)

	if isFreeGame && this.JieSuanData.FreeData.FreeRemainTimes == 0 {
		this.JieSuanData.FreeData = FreeData{}
	}
	this.IsInCheckout = false
	if this.ModeType == NORMAL {
		activity.CalcEncouragementFunc(this.UserID)
	}
	return nil
}
