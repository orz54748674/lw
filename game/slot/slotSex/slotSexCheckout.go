package slotSex

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
	"vn/storage/slotStorage/slotSexStorage"
	"vn/storage/walletStorage"
)
func (this *MyTable) InitCheckoutData(){
	this.BonusGameData = BonusGameData{}
	this.MiniGameData = MiniGameData{
		SymbolList: []int{0,0,1,1},
		State: 0,
	}

	var jieSuanData JieSuanData
	jieSuanData.TotalBackScore = 0
	jieSuanData.Result = []Result{}
	jieSuanData.BonusTimes = []int64{}
	jieSuanData.ResultPositions = []int64{}
	jieSuanData.MusicType = ""
	jieSuanData.GetJackpot = false
	jieSuanData.FreeGame = false
	jieSuanData.AnimationData = AnimationData{}
	jieSuanData.TrialData = this.JieSuanData.TrialData
	jieSuanData.FreeRemainTimes = this.JieSuanData.FreeRemainTimes
	this.JieSuanData = jieSuanData

}
func (this *MyTable) CalcCheckout(reelList [][]slotSexStorage.Symbol,coinNum int64,coinValue int64){
	totalXiaZhu :=  coinNum * coinValue
	resultSymbol := make([]int64, len(reelList)) //每列的起始位置
	Again :

	for k,v := range reelList{
		rand := this.RandInt64(1,int64(len(v) + 1))
		rand = rand -1
		resultSymbol[k] = rand
	}
	var jieSuanData JieSuanData

	//if this.JieSuanData.FreeRemainTimes > 0{
	//	resultSymbol[0] = 6
	//	resultSymbol[1] = 14
	//	resultSymbol[2] = 23
	//	resultSymbol[3] = 16
	//	resultSymbol[4] = 2
	//}else{
	//	resultSymbol[0] = 14
	//	resultSymbol[1] = 4
	//	resultSymbol[2] = 14
	//	resultSymbol[3] = 1
	//	resultSymbol[4] = 2
	//}

	jieSuanData.ResultPositions = resultSymbol

	maxBonus := 0
	maxScatter := 0
	maxJackpot := 0
	for line := int64(1);line <= coinNum;line++ { //计算每条线
		baseIdx := resultSymbol[0] + LineCoordinates[line][0].Row //第一列的页面显示的图案索引
		if baseIdx >= int64(len(reelList[0])) {
			baseIdx = baseIdx - int64(len(reelList[0]))
		}
		baseSymbol := reelList[0][baseIdx] //第一列的页面显示的图案

		maxScore := int64(0)
		maxSymbol := baseSymbol
		maxLineType := 0 //连线类型 三连 四连 五连之类的
		wildReplaceList := []slotSexStorage.Symbol{baseSymbol}
		if baseSymbol == slotSexStorage.WILD {
			wildReplaceList = WildReplaceList
		}


		bonus := 0
		scatter := 0
		symbolPositions := make([]map[int64]int64,0)
		bonusPosition := make([]map[int64]int64,0)
		scatterPosition := make([]map[int64]int64,0)
		position := map[int64]int64{
			LineCoordinates[line][0].Row:0,
		}
		if baseSymbol == slotSexStorage.BONUS{
			bonus++
			bonusPosition = append(bonusPosition,position)
		}else if baseSymbol == slotSexStorage.SCATTER{
			scatter++
			scatterPosition = append(scatterPosition,position)
		}
		for col := int64(1); col < int64(len(resultSymbol)); col++ {
			symbolIdx := resultSymbol[col] + LineCoordinates[line][col].Row
			if symbolIdx >= int64(len(reelList[col])) {
				symbolIdx = symbolIdx - int64(len(reelList[col]))
			}
			position = map[int64]int64{
				LineCoordinates[line][col].Row:col,
			}
			if reelList[col][symbolIdx] == slotSexStorage.BONUS{ //
				bonus++
				bonusPosition = append(bonusPosition,position)
			}else if reelList[col][symbolIdx] == slotSexStorage.SCATTER{
				scatter++
				scatterPosition = append(scatterPosition,position)
			}
		}
		if bonus > maxBonus{
			maxBonus = bonus
		}
		if scatter > maxScatter{
			maxScatter = scatter
		}

		for _, v := range wildReplaceList {
			Positions := make([]map[int64]int64,0)
			Positions = append(Positions,map[int64]int64{
				LineCoordinates[line][0].Row:0,
			})
			lineNum := 1
			for col := int64(1); col < int64(len(resultSymbol)); col++ {
				symbolIdx := resultSymbol[col] + LineCoordinates[line][col].Row
				if symbolIdx >= int64(len(reelList[col])) {
					symbolIdx = symbolIdx - int64(len(reelList[col]))
				}
				position = map[int64]int64{
					LineCoordinates[line][col].Row:col,
				}
				if reelList[col][symbolIdx] == slotSexStorage.WILD || reelList[col][symbolIdx] == v { //
					lineNum++
					Positions = append(Positions,position)
				} else {
					break
				}
			}

			if v == slotSexStorage.JACKPOT{
				if lineNum > maxJackpot{
					maxJackpot = lineNum
				}

				if lineNum >= 2{
					score := OddsList[v][lineNum]
					if score > maxScore {
						maxScore = score
						maxLineType = lineNum
						symbolPositions = Positions
						maxSymbol = v
					}
				}

			}else if lineNum >= MinWinLine {
				score := OddsList[v][lineNum]
				if score > maxScore {
					maxScore = score
					maxLineType = lineNum
					symbolPositions = Positions
					maxSymbol = v
				}
			}
		}
		if maxScore > 0 {
			winPositions := make([]map[int64]int64,0)
			if maxScore > 0{
				for _,v := range symbolPositions{
					winPositions = append(winPositions,v)
				}
			}
			result := Result{
				WinPositions:winPositions,
				LineType: maxLineType,
				Symbol: maxSymbol,
				SymbolScore: maxScore,
				CoinValue: coinValue,
				LineSerial:line,
			}
			jieSuanData.Result = append(jieSuanData.Result,result)

			jieSuanData.TotalBackScore += maxScore * coinValue
		}
		if bonus >= 3{
			winPositions := make([]map[int64]int64,0)
			if bonus >= 3{
				for _,v := range bonusPosition{
					winPositions = append(winPositions,v)
				}
			}
			result := Result{
				WinPositions:winPositions,
				LineType: maxLineType,
				Symbol:slotSexStorage.BONUS,
				SymbolScore: 0,
				CoinValue: coinValue,
				LineSerial:line,
			}
			jieSuanData.Result = append(jieSuanData.Result,result)
		}

		if scatter >= 3{
			winPositions := make([]map[int64]int64,0)
			if scatter >= 3{
				for _,v := range scatterPosition{
					winPositions = append(winPositions,v)
				}
			}
			result := Result{
				WinPositions:winPositions,
				LineType: maxLineType,
				Symbol: slotSexStorage.SCATTER,
				SymbolScore: 0,
				CoinValue: coinValue,
				LineSerial:line,
			}
			jieSuanData.Result = append(jieSuanData.Result,result)
		}

	}

	if this.ModeType == NORMAL{
		gameProfit := gameStorage.QueryProfitByUser(this.UserID)
		if maxJackpot == len(resultSymbol) || (jieSuanData.TotalBackScore > gameProfit.BotBalance && jieSuanData.TotalBackScore > 0){ //重新生成
			goto Again
		}else if maxScatter == 3 && gameProfit.BotBalance < totalXiaZhu * int64(this.GameConf.FreeGameMinTimes[0]){
			goto Again
		}else if maxScatter == 4 && gameProfit.BotBalance < totalXiaZhu * int64(this.GameConf.FreeGameMinTimes[1]){
			goto Again
		}else if maxScatter == 5 && gameProfit.BotBalance < totalXiaZhu * int64(this.GameConf.FreeGameMinTimes[2]){
			goto Again
		}else if maxBonus == 3 && gameProfit.BotBalance < totalXiaZhu * int64(this.GameConf.BonusGameMinTimes[0]){
			goto Again
		}else if maxBonus == 4 && gameProfit.BotBalance < totalXiaZhu * int64(this.GameConf.BonusGameMinTimes[1]){
			goto Again
		}else if maxBonus == 5 && gameProfit.BotBalance < totalXiaZhu * int64(this.GameConf.BonusGameMinTimes[2]){
			goto Again
		}else {
			gameStorage.IncProfitByUser(this.UserID,0,-jieSuanData.TotalBackScore,0,jieSuanData.TotalBackScore - totalXiaZhu)
		}
	}else if this.ModeType == TRIAL{
		if maxJackpot == len(resultSymbol){
			jieSuanData.GetJackpot = true
			jieSuanData.TotalBackScore += 5000 * coinValue
		}
	}

	if this.ModeType == NORMAL && this.JieSuanData.FreeRemainTimes <= 0{ //免费转不累加奖池
		for k, v := range CoinNum { //刷新奖池
			if coinNum == v {
				goldJackpot := totalXiaZhu * int64(this.GameConf.PoolScaleThousand)
				slotSexStorage.IncJackpot(k, goldJackpot)
			}
		}
	}
	if maxBonus >= 3{
		jieSuanData.BonusGame = true
		jieSuanData.BonusTimes = BonusTimes[maxBonus]
	}
	if maxScatter >= 3{
		jieSuanData.FreeGame = true
		jieSuanData.FreeRemainTimes += ScatterTimes[maxScatter]
	}

	if jieSuanData.TotalBackScore > 0{
		musicScore := jieSuanData.TotalBackScore / coinValue
		if jieSuanData.GetJackpot{
			jieSuanData.MusicType = WinJackPot
		//}else if musicScore >= 500{
		//	jieSuanData.MusicType = Win500
		}else{
			jieSuanData.MusicType = WinNormal
		}

		if musicScore >= 500 || maxJackpot >= 3{
			rand := this.RandInt64(1,int64(len(BigAnimationList) + 1)) - 1
			jieSuanData.AnimationData.AnimationType = BigAnimationList[rand]
		}else{
			colAnimation := make([]AnimationData,0)
			randAnimation := false
			for _,v := range jieSuanData.Result{
				if v.LineType >= 4{
					playColList := make([]int,0)
					for _,v1 := range v.WinPositions{
						for _,v2 := range v1{
							playColList = append(playColList,int(v2))
						}
					}
					rand := this.RandInt64(1,int64(len(playColList) + 1)) - 1
					col := playColList[rand]
					if v.Symbol == slotSexStorage.S5 {
						colAnimation = append(colAnimation,AnimationData{
							AnimationType: ColAnimation1,
							PlayCol: col,
						})
					}else if v.Symbol == slotSexStorage.S6 {
						colAnimation = append(colAnimation,AnimationData{
							AnimationType: ColAnimation2,
							PlayCol: col,
						})
					}else if v.Symbol == slotSexStorage.S7 {
						colAnimation = append(colAnimation,AnimationData{
							AnimationType: ColAnimation3,
							PlayCol: col,
						})
					}else if v.Symbol == slotSexStorage.S8 {
						colAnimation = append(colAnimation,AnimationData{
							AnimationType: ColAnimation4,
							PlayCol: col,
						})
					}else if v.LineType >= 5 &&
						(v.Symbol == slotSexStorage.S1 ||
							v.Symbol == slotSexStorage.S2 ||
							v.Symbol == slotSexStorage.S3 ||
							v.Symbol == slotSexStorage.S4) {
							randAnimation = true
					}

				}
			}

			if len(colAnimation) > 0{
				rand := this.RandInt64(1,int64(len(colAnimation) + 1)) - 1
				jieSuanData.AnimationData = colAnimation[rand]
			}else if randAnimation{
				rand := this.RandInt64(1,int64(len(ColAnimationList) + 1)) - 1
				col := this.RandInt64(1,6) - 1
				jieSuanData.AnimationData = AnimationData{
					AnimationType: ColAnimationList[rand],
					PlayCol: int(col),
				}
			}
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

func (this *MyTable) Spin(session gate.Session,msg map[string]interface{})  (err error) {
	if this.JieSuanData.BonusGame{ //进入副本
		log.Info("enter bonus game")
		error := errCode.ErrParams
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.Spin,error)
		return nil
	}
	this.GameConf = slotSexStorage.GetRoomConf()
	this.IsInCheckout = true
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)
	isFreeGame := false
	if this.JieSuanData.FreeRemainTimes > 0{
		isFreeGame = true
		this.JieSuanData.FreeRemainTimes--
		this.EventID = string(game.SlotSex) + "_" + "Free" + "_" + strconv.FormatInt(time.Now().Unix(),10)
	}else{
		this.EventID = string(game.SlotSex) + "_" + strconv.FormatInt(time.Now().Unix(),10)
	}
	if this.ModeType == NORMAL{
		this.InitCheckoutData()
	}else{
		this.InitCheckoutData()
	}

	if !isFreeGame {
		this.CoinNum = msg["CoinNum"].(int64)
		this.CoinValue = msg["CoinValue"].(int64)
		totalXiaZhu :=  this.CoinNum * this.CoinValue
		if this.ModeType == NORMAL{
			wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
			if wallet.VndBalance < totalXiaZhu{ //下注金额不足
				error := errCode.BalanceNotEnough
				this.sendPack(session.GetSessionID(),game.Push,"",protocol.Spin,error)
				return nil
			}
			value := totalXiaZhu * int64(this.GameConf.BotProfitPerThousand) / 1000
			gameStorage.IncProfitByUser(this.UserID,0,totalXiaZhu - value,value,0)
			bill := walletStorage.NewBill(this.UserID,walletStorage.TypeExpenses,walletStorage.EventGameSlotSex,this.EventID,-totalXiaZhu)
			walletStorage.OperateVndBalance(bill)
		}else{
			if this.JieSuanData.TrialData.VndBalance < totalXiaZhu{ //下注金额不足
				error := errCode.CurCanXiaZhuError
				this.sendPack(session.GetSessionID(),game.Push,"",protocol.Spin,error)
				return nil
			}
			this.JieSuanData.TrialData.VndBalance -= totalXiaZhu
		}
	}
	betData := make(map[string]interface{})
	betData["CoinNum"] = this.CoinNum
	betData["CoinValue"] = this.CoinValue
	betDetail,_ := json.Marshal(betData)
	var resultRecordStr string
	if this.ModeType == NORMAL {
		this.CalcCheckout(this.ReelsList, this.CoinNum, this.CoinValue)
		resultRecordStr = this.DealGameResultRecord(this.JieSuanData.ResultPositions,this.ReelsList)
	}else{
		this.CalcCheckout(this.ReelsListTrial, this.CoinNum, this.CoinValue)
	}

	if this.ModeType == NORMAL{
		if this.JieSuanData.TotalBackScore > 0{
			bill := walletStorage.NewBill(this.UserID,walletStorage.TypeIncome,walletStorage.EventGameSlotSex,this.EventID,this.JieSuanData.TotalBackScore)
			walletStorage.OperateVndBalance(bill)

			lobbyStorage.Win(utils.ConvertOID(this.UserID),this.Name, this.JieSuanData.TotalBackScore - this.ResultsPool,game.SlotSex,false)
		}

		wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
		if !isFreeGame {
			totalXiaZhu :=  this.CoinNum * this.CoinValue
			betRecordParam := gameStorage.BetRecordParam{
				Uid: this.UserID,
				GameType: game.SlotSex,
				Income: this.JieSuanData.TotalBackScore - totalXiaZhu,
				BetAmount: totalXiaZhu,
				CurBalance: wallet.VndBalance + wallet.SafeBalance,
				SysProfit: 0,
				BotProfit: 0,
				BetDetails: string(betDetail),
				GameId: this.EventID,
				GameNo: strconv.FormatInt(time.Now().Unix(),10),
				GameResult: resultRecordStr,
				IsSettled: true,
			}
			gameStorage.InsertBetRecord(betRecordParam)
			pay.CheckoutAgentIncome(utils.ConvertOID(this.UserID), totalXiaZhu, this.EventID, game.SlotSex)
		}else{
			betRecordParam := gameStorage.BetRecordParam{
				Uid: this.UserID,
				GameType: game.SlotSex,
				Income: this.JieSuanData.TotalBackScore,
				BetAmount: 0,
				CurBalance: wallet.VndBalance + wallet.SafeBalance,
				SysProfit: 0,
				BotProfit: 0,
				BetDetails: string(betDetail),
				GameId: this.EventID,
				GameNo: strconv.FormatInt(time.Now().Unix(),10),
				GameResult: resultRecordStr,
				IsSettled: true,
			}
			gameStorage.InsertBetRecord(betRecordParam)
		}
	}else{
		this.JieSuanData.TrialData.VndBalance += this.JieSuanData.TotalBackScore
	}

	sb := vGate.QuerySessionBean(this.UserID)
	if sb != nil{
		session,_ := basegate.NewSession(this.app, sb.Session)
		this.sendPack(session.GetSessionID(),game.Push,this.JieSuanData,protocol.JieSuan,nil)
	}

	if this.ModeType == NORMAL{
		this.notifyWallet(this.UserID)
	}


	res,_ := json.Marshal(this.JieSuanData)
	log.Info("----------------------slot sex jiesuan ---%s",res)
	this.IsInCheckout = false
	if this.ModeType == NORMAL{
		activity.CalcEncouragementFunc(this.UserID)
	}
	return nil
}