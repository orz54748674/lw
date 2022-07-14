package slotCs

import (
	"encoding/json"
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
	"vn/storage/slotStorage/slotCsStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) InitCheckoutData(reelList [][]slotCsStorage.Symbol) {
	this.BonusGameData = BonusGameData{}
	this.MiniGameData = MiniGameData{
		SymbolList: []int{0, 0, 1, 1},
		State:      1,
	}

	var jieSuanData JieSuanData
	jieSuanData.TotalBackScore = 0
	jieSuanData.Result = []Result{}
	jieSuanData.BonusTimes = []int64{}
	jieSuanData.ResultPositions = []int64{}
	jieSuanData.MusicType = ""
	jieSuanData.GetJackpot = false
	jieSuanData.FreeGame = false
	jieSuanData.TrialData = this.JieSuanData.TrialData
	jieSuanData.FreeRemainTimes = this.JieSuanData.FreeRemainTimes
	this.JieSuanData = jieSuanData

}
func (this *MyTable) CalcCheckout(reelList [][]slotCsStorage.Symbol, coinNum int64, coinValue int64) {
	totalXiaZhu := coinNum * coinValue
	resultSymbol := make([]int64, len(reelList)) //每列的起始位置
Again:

	for k, v := range reelList {
		rand := this.RandInt64(1, int64(len(v)+1))
		rand = rand - 1
		resultSymbol[k] = rand
	}
	var jieSuanData JieSuanData

	//resultSymbol[0]=5
	//resultSymbol[1]=6
	//resultSymbol[2]=3
	//resultSymbol[3]=25
	//resultSymbol[4]=2
	jieSuanData.ResultPositions = resultSymbol

	maxBonus := 0
	maxScatter := 0
	maxJackpot := 0
	for line := int64(1); line <= coinNum; line++ { //计算每条线
		baseIdx := resultSymbol[0] + LineCoordinates[line][0].Row //第一列的页面显示的图案索引
		if baseIdx >= int64(len(reelList[0])) {
			baseIdx = baseIdx - int64(len(reelList[0]))
		}
		baseSymbol := reelList[0][baseIdx] //第一列的页面显示的图案

		maxScore := int64(0)
		maxSymbol := baseSymbol
		maxLineType := 0 //连线类型 三连 四连 五连之类的
		wildReplaceList := []slotCsStorage.Symbol{baseSymbol}
		if baseSymbol == slotCsStorage.WILD {
			wildReplaceList = WildReplaceList
		}

		bonus := 0
		scatter := 0
		if baseSymbol == slotCsStorage.BONUS {
			bonus++
		} else if baseSymbol == slotCsStorage.SCATTER {
			scatter++
		}
		for col := int64(1); col < int64(len(resultSymbol)); col++ {
			symbolIdx := resultSymbol[col] + LineCoordinates[line][col].Row
			if symbolIdx >= int64(len(reelList[col])) {
				symbolIdx = symbolIdx - int64(len(reelList[col]))
			}
			if reelList[col][symbolIdx] == slotCsStorage.BONUS { //
				bonus++
			} else if reelList[col][symbolIdx] == slotCsStorage.SCATTER {
				scatter++
			}
		}
		if bonus > maxBonus {
			maxBonus = bonus
		}
		if scatter > maxScatter {
			maxScatter = scatter
		}

		for _, v := range wildReplaceList {
			lineNum := 1
			for col := int64(1); col < int64(len(resultSymbol)); col++ {
				symbolIdx := resultSymbol[col] + LineCoordinates[line][col].Row
				if symbolIdx >= int64(len(reelList[col])) {
					symbolIdx = symbolIdx - int64(len(reelList[col]))
				}
				if reelList[col][symbolIdx] == slotCsStorage.WILD || reelList[col][symbolIdx] == v { //
					lineNum++
				} else {
					break
				}
			}

			if v == slotCsStorage.JACKPOT {
				if lineNum > maxJackpot {
					maxJackpot = lineNum
				}

				if lineNum >= 2 {
					score := OddsList[v][lineNum]
					if score > maxScore {
						maxScore = score
						maxLineType = lineNum
						maxSymbol = v
					}
				}

			} else if lineNum >= MinWinLine {
				score := OddsList[v][lineNum]
				if score > maxScore {
					maxScore = score
					maxLineType = lineNum
					maxSymbol = v
				}
			}
		}

		if maxScore > 0 || bonus >= 3 || scatter >= 3 {
			result := Result{
				LineType:    maxLineType,
				Symbol:      maxSymbol,
				SymbolScore: maxScore,
				CoinValue:   coinValue,
				LineSerial:  line,
			}
			jieSuanData.Result = append(jieSuanData.Result, result)

			jieSuanData.TotalBackScore += maxScore * coinValue
		}

	}

	if this.ModeType == NORMAL {
		gameProfit := gameStorage.QueryProfitByUser(this.UserID)
		if maxJackpot == len(resultSymbol) || (jieSuanData.TotalBackScore > 0 && jieSuanData.TotalBackScore > gameProfit.BotBalance) { //重新生成
			goto Again
		} else if maxScatter == 3 && gameProfit.BotBalance < totalXiaZhu*int64(this.GameConf.FreeGameMinTimes[0]) {
			goto Again
		} else if maxScatter == 4 && gameProfit.BotBalance < totalXiaZhu*int64(this.GameConf.FreeGameMinTimes[1]) {
			goto Again
		} else if maxScatter == 5 && gameProfit.BotBalance < totalXiaZhu*int64(this.GameConf.FreeGameMinTimes[2]) {
			goto Again
		} else if maxBonus == 3 && gameProfit.BotBalance < totalXiaZhu*int64(this.GameConf.BonusGameMinTimes[0]) {
			goto Again
		} else if maxBonus == 4 && gameProfit.BotBalance < totalXiaZhu*int64(this.GameConf.BonusGameMinTimes[1]) {
			goto Again
		} else if maxBonus == 5 && gameProfit.BotBalance < totalXiaZhu*int64(this.GameConf.BonusGameMinTimes[2]) {
			goto Again
		} else {
			gameStorage.IncProfitByUser(this.UserID, 0, -jieSuanData.TotalBackScore, 0, jieSuanData.TotalBackScore-totalXiaZhu)
		}
	} else if this.ModeType == TRIAL {
		if maxJackpot == len(resultSymbol) {
			jieSuanData.GetJackpot = true
			jieSuanData.TotalBackScore += 5000 * coinValue
		}
	}

	if this.ModeType == NORMAL && this.JieSuanData.FreeRemainTimes <= 0 { //免费转不累加奖池
		for k, v := range CoinNum { //刷新奖池
			if coinNum == v {
				goldJackpot := totalXiaZhu * int64(this.GameConf.PoolScaleThousand)
				slotCsStorage.IncJackpot(k, goldJackpot)
			}
		}
	}
	if maxBonus >= 3 {
		jieSuanData.BonusGame = true
		jieSuanData.BonusTimes = BonusTimes[maxBonus]
	}
	if maxScatter >= 3 {
		jieSuanData.FreeGame = true
		jieSuanData.FreeRemainTimes += ScatterTimes[maxScatter]
	}

	if jieSuanData.TotalBackScore > 0 {
		musicScore := jieSuanData.TotalBackScore / coinValue
		if jieSuanData.GetJackpot {
			jieSuanData.MusicType = WinJackPot
		} else if musicScore >= 300 {
			jieSuanData.MusicType = Win500
		} else {
			jieSuanData.MusicType = WinNormal
		}
	}

	//计算jackpot 暂时不需要开奖

	jieSuanData.CoinNum = coinNum
	jieSuanData.CoinValue = coinValue
	//计算是否进free game
	jieSuanData.FreeRemainTimes += this.JieSuanData.FreeRemainTimes
	jieSuanData.TrialData = this.JieSuanData.TrialData
	this.JieSuanData = jieSuanData
}

func (this *MyTable) Spin(session gate.Session, msg map[string]interface{}) (err error) {
	if this.JieSuanData.BonusGame { //进入副本
		log.Info("enter bonus game")
		error := errCode.ErrParams
		this.sendPack(session.GetSessionID(), game.Push, "", protocol.Spin, error)
		return nil
	}
	this.GameConf = slotCsStorage.GetRoomConf()
	this.IsInCheckout = true
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)
	isFreeGame := false
	if this.JieSuanData.FreeRemainTimes > 0 {
		isFreeGame = true
		this.JieSuanData.FreeRemainTimes--
		this.EventID = string(game.SlotCs) + "_" + "Free" + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	} else {
		this.EventID = string(game.SlotCs) + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	if this.ModeType == NORMAL {
		this.InitCheckoutData(this.ReelsList)
	} else {
		this.InitCheckoutData(this.ReelsListTrial)
	}

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
			bill := walletStorage.NewBill(this.UserID, walletStorage.TypeExpenses, walletStorage.EventGameSlotCs, this.EventID, -totalXiaZhu)
			walletStorage.OperateVndBalance(bill)
			value := totalXiaZhu * int64(this.GameConf.BotProfitPerThousand) / 1000
			gameStorage.IncProfitByUser(this.UserID, 0, totalXiaZhu-value, value, 0)
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
		this.CalcCheckout(this.ReelsList, this.CoinNum, this.CoinValue)
		resultRecordStr = this.DealGameResultRecord(this.JieSuanData.ResultPositions, this.ReelsList)
	} else {
		this.CalcCheckout(this.ReelsListTrial, this.CoinNum, this.CoinValue)
	}
	if this.ModeType == NORMAL {
		if this.JieSuanData.TotalBackScore > 0 {
			bill := walletStorage.NewBill(this.UserID, walletStorage.TypeIncome, walletStorage.EventGameSlotCs, this.EventID, this.JieSuanData.TotalBackScore)
			walletStorage.OperateVndBalance(bill)

			lobbyStorage.Win(utils.ConvertOID(this.UserID), this.Name, this.JieSuanData.TotalBackScore-this.ResultsPool, game.SlotCs, false)
		}

		wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
		if !isFreeGame {
			totalXiaZhu := this.CoinNum * this.CoinValue
			betRecordParam := gameStorage.BetRecordParam{
				Uid:        this.UserID,
				GameType:   game.SlotCs,
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
			pay.CheckoutAgentIncome(utils.ConvertOID(this.UserID), totalXiaZhu, this.EventID, game.SlotCs)
		} else {
			betRecordParam := gameStorage.BetRecordParam{
				Uid:        this.UserID,
				GameType:   game.SlotCs,
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

	if this.ModeType == NORMAL {
		this.notifyWallet(this.UserID)
	}

	res, _ := json.Marshal(this.JieSuanData)
	log.Info("----------------------slot cs jiesuan ---%s", res)
	this.IsInCheckout = false
	if this.ModeType == NORMAL {
		activity.CalcEncouragementFunc(this.UserID)
	}
	return nil
}
