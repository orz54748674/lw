package slotLs

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
	"vn/storage/slotStorage/slotLsStorage"
	"vn/storage/walletStorage"
)
func (this *MyTable) InitCheckoutData(reelList [][]slotLsStorage.Symbol){
	var jieSuanData JieSuanData
	jieSuanData.WildTimes = 1
	jieSuanData.TotalBackScore = 0
	jieSuanData.WildPositions = []map[int64]int64{}
	jieSuanData.Result = []Result{}
	jieSuanData.JackpotPositions = []map[int64]int64{}
	jieSuanData.ResultPositions = []int64{}
	jieSuanData.MusicType = ""
	if reflect.DeepEqual(reelList,this.ReelsList){
		jieSuanData.FreeGameTimes = 0
		this.JieSuanData = jieSuanData
	}else if reflect.DeepEqual(reelList,this.ReelsListFree){
		jieSuanData.FreeGameTimes = this.JieSuanDataFree.FreeGameTimes
		jieSuanData.FreeRemainTimes = this.JieSuanDataFree.FreeRemainTimes
		jieSuanData.TotalBackScore = this.JieSuanDataFree.TotalBackScore
		this.JieSuanDataFree = jieSuanData
	}else if reflect.DeepEqual(reelList,this.ReelsListTrial){
		jieSuanData.FreeGameTimes = 0
		this.JieSuanDataTrial = jieSuanData
	}else if reflect.DeepEqual(reelList,this.ReelsListTrialFree){
		jieSuanData.FreeGameTimes = this.JieSuanDataTrialFree.FreeGameTimes
		jieSuanData.FreeRemainTimes = this.JieSuanDataTrialFree.FreeRemainTimes
		jieSuanData.TotalBackScore = this.JieSuanDataTrialFree.TotalBackScore
		this.JieSuanDataTrialFree = jieSuanData
	}
}
func (this *MyTable) CalcCheckout(reelList [][]slotLsStorage.Symbol,coinNum int64,coinValue int64){
	totalXiaZhu :=  coinNum * coinValue
	resultSymbol := make([]int64, len(reelList)) //每列的起始位置

	Again :
	for k,v := range reelList{
		rand := this.RandInt64(1,int64(len(v) + 1))
		rand = rand -1
		resultSymbol[k] = rand
	}
	var jieSuanData JieSuanData

	//if reflect.DeepEqual(reelList,this.ReelsList) {
	//	resultSymbol[0] = 3
	//	resultSymbol[1] = 8
	//	resultSymbol[2] = 7
	//	resultSymbol[3] = 6
	//	resultSymbol[4] = 15
	//}
	jieSuanData.ResultPositions = resultSymbol

	haveWildLine := false                      //中奖连线中是否有wild
	for row := int64(0);row < TotalRows;row++{ //计算每行
		hitGroupTimes := int64(1) //组合倍数
		baseIdx := resultSymbol[0] + row //第一列的页面显示的图案索引
		lineType := 1 //连线类型 三连 四连 五连之类的
		if baseIdx >= int64(len(reelList[0])){
			baseIdx = baseIdx - int64(len(reelList[0]))
		}
		baseSymbol := reelList[0][baseIdx] //第一列的页面显示的图案
		var symbolPositions []map[int64]int64
		position := map[int64]int64{
			row:0,
		}
		find := -1
		for k,v := range jieSuanData.Result{
			if v.Symbol == baseSymbol{
				jieSuanData.Result[k].SymbolPositions = append(jieSuanData.Result[k].SymbolPositions,position)
				find = k
				break
			}
		}
		if find > -1{
			jieSuanData.Result[find].GroupNum += 1
			continue
		}

		symbolPositions = append(symbolPositions,position)
		var wildPositions []map[int64]int64
		haveWild := false
		for col := int64(1);col < int64(len(resultSymbol));col++{ //第二列之后的
			colSymbolNum := int64(0)
			for rowCalc := int64(0);rowCalc < TotalRows;rowCalc ++{
				symbolIdx := resultSymbol[col] + rowCalc
				if symbolIdx >= int64(len(reelList[col])){
					symbolIdx = symbolIdx - int64(len(reelList[col]))
				}
				if (reelList[col][symbolIdx] == slotLsStorage.WILD && baseSymbol != slotLsStorage.SCATTER)|| reelList[col][symbolIdx] == baseSymbol{ //
					colSymbolNum++
					position = map[int64]int64{
						rowCalc:col,
					}
					if reelList[col][symbolIdx] == slotLsStorage.WILD { //中wild的位置
						find := false
						for _,v := range jieSuanData.WildPositions{
							if reflect.DeepEqual(v,position){
								find = true
								break
							}
						}
						if !find{
							wildPositions = append(wildPositions,position)
						}
						haveWild = true
					}else{
						symbolPositions = append(symbolPositions,position)
					}

				}
			}

			if colSymbolNum > 0{
				hitGroupTimes *= colSymbolNum
				lineType++
			}else{ //说明已经断了
				break
			}
		}

		if lineType >= MinWinLine { //3连以上才有奖励
			if haveWild{
				haveWildLine = true
				for _,v := range wildPositions{ //中wild的位置
					jieSuanData.WildPositions = append(jieSuanData.WildPositions,v)
				}
			}
			symbolScore := int64(0)
			if baseSymbol == slotLsStorage.SCATTER {
				symbolScore = OddsList[baseSymbol][lineType] * totalXiaZhu * hitGroupTimes
			}else{
				symbolScore = OddsList[baseSymbol][lineType] * coinNum / BaseCoinNum * hitGroupTimes
			}
			result := Result{
				SymbolPositions: symbolPositions,
				LineType: lineType,
				Symbol: baseSymbol,
				SymbolScore: symbolScore,
				CoinValue: coinValue,
				HaveWild: haveWild,
			}
			jieSuanData.Result = append(jieSuanData.Result,result)
		}


	}
	jieSuanData.WildTimes = 1
	if haveWildLine{
		if reflect.DeepEqual(reelList,this.ReelsList) || reflect.DeepEqual(reelList,this.ReelsListTrial) {
			wildRand := this.RandInt64(1, 1000)
			find := false
			for k, v := range WildRandList {
				for k1, v1 := range v {
					if wildRand >= k1 && wildRand < v1 {
						jieSuanData.WildTimes = k
						find = true
						break
					}
				}
				if find {
					break
				}
			}
		}else{
			wildRand := this.RandInt64(1, 1000)
			wild := 0
			if wildRand < 600{
				wild = 0
			}else if wildRand >= 600 && wildRand < 900{
				wild = 1
			}else if wildRand >= 900 && wildRand < 1000{
				wild = 2
			}
			jieSuanData.WildTimes = int64(this.FreeGameConf.Times[wild])
		}
	}
	for k,v := range jieSuanData.Result{
		jieSuanData.Result[k].SymbolScore *= v.GroupNum + 1
	}
	freeGame := false
	for _,v := range jieSuanData.Result{
		if v.Symbol == slotLsStorage.SCATTER{
			freeGame = true
			jieSuanData.TotalBackScore += v.SymbolScore
		}else{
			if v.HaveWild{
				jieSuanData.TotalBackScore += v.SymbolScore * jieSuanData.WildTimes * coinValue
			}else{
				jieSuanData.TotalBackScore += v.SymbolScore * coinValue
			}
		}
	}

	if reflect.DeepEqual(reelList,this.ReelsList) || reflect.DeepEqual(reelList,this.ReelsListFree){
		gameProfit := gameStorage.QueryProfitByUser(this.UserID)
		if jieSuanData.TotalBackScore > 0 &&(jieSuanData.TotalBackScore > gameProfit.BotBalance || (freeGame && gameProfit.BotBalance < totalXiaZhu * int64(this.GameConf.FreeGameMinTimes))){
			goto Again
		}else{
			if reflect.DeepEqual(reelList,this.ReelsList){
				gameStorage.IncProfitByUser(this.UserID,0,-jieSuanData.TotalBackScore,0,jieSuanData.TotalBackScore - totalXiaZhu)
			}else{
				gameStorage.IncProfitByUser(this.UserID,0,-jieSuanData.TotalBackScore,0,jieSuanData.TotalBackScore)
			}
		}
	}
	if jieSuanData.TotalBackScore > 0{
		musicScore := jieSuanData.TotalBackScore * 10 / totalXiaZhu
		if musicScore < 2{
			jieSuanData.MusicType = WIN1
		}else if musicScore < 4{
			jieSuanData.MusicType = WIN2
		}else if musicScore < 6{
			jieSuanData.MusicType = WIN3
		}else if musicScore < 8{
			jieSuanData.MusicType = WIN4
		}else if musicScore < 10{
			jieSuanData.MusicType = WIN5
		}else if musicScore < 30{
			jieSuanData.MusicType = BET
		}else if musicScore < 40{
			jieSuanData.MusicType = BET3
		}else if musicScore < 50{
			jieSuanData.MusicType = BET4
		}else if musicScore < 60{
			jieSuanData.MusicType = BET5
		}else if musicScore < 80{
			jieSuanData.MusicType = BET6
		}else if musicScore < 400{
			jieSuanData.MusicType = BET10
		}else if musicScore >= 400{
			jieSuanData.MusicType = BET40
		}
	}

	if reflect.DeepEqual(reelList,this.ReelsList) {
		for k, v := range CoinNum { //刷新奖池
			if coinNum == v {
				goldJackpot := totalXiaZhu * int64(this.GameConf.PoolScaleThousand)
				slotLsStorage.IncJackpot(k, goldJackpot, goldJackpot/2)
			}
		}
	}else if reflect.DeepEqual(reelList,this.ReelsListTrial){
		for k, v := range CoinNum { //刷新奖池
			if coinNum == v {
				goldJackpot := totalXiaZhu * int64(this.GameConf.PoolScaleThousand)
				this.TrialData.GoldJackpot[k] += goldJackpot
				this.TrialData.SilverJackpot[k] += goldJackpot / 2
			}
		}
	}

	jackpotRand := this.RandInt64(1,1000)
	jackpotNum := int64(0)
	find := false
	for k,v := range JackpotRandList {
		for k1,v1 := range v{
			if jackpotRand >= k1 && jackpotRand < v1{
				jackpotNum = k
				find = true
				break
			}
		}
		if find{
			break
		}
	}
	if jackpotNum > 0{
		for i := int64(0);i < jackpotNum;i++{
			position := map[int64]int64{}
			for true{
				row := this.RandInt64(1, TotalRows+ 1)
				row = row - 1
				col := this.RandInt64(1,int64(len(this.ReelsList) + 1))
				col = col -1

				find := false
				for _,v := range this.JieSuanData.JackpotPositions{
					for k1,v1 := range v{
						if k1 == row || v1 == col{
							find = true
							break
						}
					}
				}

				if !find{
					position = map[int64]int64{
						row:col,
					}
					jieSuanData.JackpotPositions = append(this.JieSuanData.JackpotPositions,position)
					break
				}
			}
		}
	}

	//计算jackpot 暂时不需要开奖
	if jackpotNum > 3{
		goto Again
	}

	jieSuanData.CoinNum = coinNum
	jieSuanData.CoinValue = coinValue
	if reflect.DeepEqual(reelList,this.ReelsList){
		//计算是否进free game
		jieSuanData.FreeGameTimes = this.JieSuanData.FreeGameTimes
		jieSuanData.FreeRemainTimes = this.JieSuanData.FreeRemainTimes
		for _,v := range jieSuanData.Result{
			if v.Symbol == slotLsStorage.SCATTER {
				this.JieSuanDataFree.FreeGameTimes += 1
				jieSuanData.FreeGameTimes += 1
				this.ModeType = Free
				break
			}
		}
		 this.JieSuanData = jieSuanData
	}else if reflect.DeepEqual(reelList,this.ReelsListFree){
		//计算是否进free game
		jieSuanData.FreeGameTimes = this.JieSuanDataFree.FreeGameTimes
		jieSuanData.FreeRemainTimes = this.JieSuanDataFree.FreeRemainTimes
		jieSuanData.TotalBackScore += this.JieSuanDataFree.TotalBackScore
		for _,v := range jieSuanData.Result{
			if v.Symbol == slotLsStorage.SCATTER {
				jieSuanData.FreeGameTimes += 1
				break
			}
		}
		 this.JieSuanDataFree = jieSuanData
	}else if reflect.DeepEqual(reelList,this.ReelsListTrial){
		//计算是否进free game
		jieSuanData.FreeGameTimes = this.JieSuanDataTrial.FreeGameTimes
		jieSuanData.FreeRemainTimes = this.JieSuanDataTrial.FreeRemainTimes
		for _,v := range jieSuanData.Result{
			if v.Symbol == slotLsStorage.SCATTER {
				this.JieSuanDataTrialFree.FreeGameTimes += 1
				jieSuanData.FreeGameTimes += 1
				this.ModeType = TRIALFREE
				break
			}
		}
		this.JieSuanDataTrial = jieSuanData
	}else if reflect.DeepEqual(reelList,this.ReelsListTrialFree){
		//计算是否进free game
		jieSuanData.FreeGameTimes = this.JieSuanDataTrialFree.FreeGameTimes
		jieSuanData.FreeRemainTimes = this.JieSuanDataTrialFree.FreeRemainTimes
		jieSuanData.TotalBackScore += this.JieSuanDataTrialFree.TotalBackScore
		for _,v := range jieSuanData.Result{
			if v.Symbol == slotLsStorage.SCATTER {
				jieSuanData.FreeGameTimes += 1
				break
			}
		}
		this.JieSuanDataTrialFree = jieSuanData
	}
}

func (this *MyTable) Spin(session gate.Session,msg map[string]interface{})  (err error) {
	if this.JieSuanDataFree.FreeRemainTimes > 0 || this.JieSuanDataFree.FreeGameTimes > 0{ //free剩余次数
		error := errCode.ErrParams
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.Spin,error)
		return nil
	}
	this.GameConf = slotLsStorage.GetRoomConf()
	this.IsInCheckout = true
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)
	this.InitCheckoutData(this.ReelsList)
	this.EventID = string(game.SlotLs) + "_" + strconv.FormatInt(time.Now().Unix(),10)
	this.CoinNum = msg["CoinNum"].(int64)
	this.CoinValue = msg["CoinValue"].(int64)

	totalXiaZhu :=  this.CoinNum * this.CoinValue
	wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
	if wallet.VndBalance < totalXiaZhu{ //下注金额不足
		error := errCode.BalanceNotEnough
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.Spin,error)
		return nil
	}
	value := totalXiaZhu * int64(this.GameConf.BotProfitPerThousand) / 1000
	gameStorage.IncProfitByUser(this.UserID,0,totalXiaZhu - value,value,0)
	bill := walletStorage.NewBill(this.UserID,walletStorage.TypeExpenses,walletStorage.EventGameSlotLs,this.EventID,-totalXiaZhu)
	walletStorage.OperateVndBalance(bill)

	this.CalcCheckout(this.ReelsList,this.CoinNum,this.CoinValue)

	betData := make(map[string]interface{})
	betData["CoinNum"] = this.CoinNum
	betData["CoinValue"] = this.CoinValue
	betDetail,_ := json.Marshal(betData)
	var resultRecordStr string
	resultRecordStr = this.DealGameResultRecord(this.JieSuanData.ResultPositions,this.ReelsList,this.JieSuanData.JackpotPositions,this.JieSuanData.WildTimes)
	if this.JieSuanData.TotalBackScore > 0{
		bill = walletStorage.NewBill(this.UserID,walletStorage.TypeIncome,walletStorage.EventGameSlotLs,this.EventID,this.JieSuanData.TotalBackScore)
		walletStorage.OperateVndBalance(bill)
		lobbyStorage.Win(utils.ConvertOID(this.UserID),this.Name, this.JieSuanData.TotalBackScore - this.ResultsPool,game.SlotLs,false)
	}

	wallet = walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
	betRecordParam := gameStorage.BetRecordParam{
		Uid: this.UserID,
		GameType: game.SlotLs,
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
	pay.CheckoutAgentIncome(utils.ConvertOID(this.UserID),totalXiaZhu,this.EventID,game.SlotLs)

	if this.JieSuanData.FreeGameTimes > 0{
		this.JieSuanDataFree.TotalBackScore = this.JieSuanData.TotalBackScore
	}
	sb := vGate.QuerySessionBean(this.UserID)
	if sb != nil{
		session,_ := basegate.NewSession(this.app, sb.Session)
		this.sendPack(session.GetSessionID(),game.Push,this.JieSuanData,protocol.JieSuan,nil)
	}

	this.notifyWallet(this.UserID)

	res,_ := json.Marshal(this.JieSuanData)
	log.Info("----------------------slot ls jiesuan ---%s",res)
	if this.JieSuanData.FreeGameTimes <= 0{
		this.IsInCheckout = false
	}
	activity.CalcEncouragementFunc(this.UserID)
	return nil
}

func (this *MyTable) SpinFree(session gate.Session,msg map[string]interface{})  (err error) {
	if this.JieSuanDataFree.FreeRemainTimes <= 0{ //free剩余次数
		log.Info("------%d",this.JieSuanDataFree.FreeRemainTimes)
		error := errCode.ErrParams
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.SpinFree,error)
		return nil
	}
	this.GameConf = slotLsStorage.GetRoomConf()
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)

	PreTotalBackScore := this.JieSuanDataFree.TotalBackScore
	this.CalcCheckout(this.ReelsListFree,this.CoinNum,this.CoinValue)

	betData := make(map[string]interface{})
	betData["CoinNum"] = this.CoinNum
	betData["CoinValue"] = this.CoinValue
	betDetail,_ := json.Marshal(betData)
	var resultRecordStr string
	resultRecordStr = this.DealGameResultRecord(this.JieSuanDataFree.ResultPositions,this.ReelsListFree,this.JieSuanDataFree.JackpotPositions,this.JieSuanDataFree.WildTimes)
	this.JieSuanDataFree.FreeRemainTimes--

	income := this.JieSuanDataFree.TotalBackScore - PreTotalBackScore
	this.EventID = string(game.SlotLs) + "_" + "Free" + "_" + strconv.FormatInt(time.Now().Unix(),10)
	bill := walletStorage.NewBill(this.UserID,walletStorage.TypeIncome,walletStorage.EventGameSlotLs,this.EventID,income)
	walletStorage.OperateVndBalance(bill)
	wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
	betRecordParam := gameStorage.BetRecordParam{
		Uid: this.UserID,
		GameType: game.SlotLs,
		Income: income,
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

	sb := vGate.QuerySessionBean(this.UserID)
	if sb != nil{
		session,_ := basegate.NewSession(this.app, sb.Session)
		this.sendPack(session.GetSessionID(),game.Push,this.JieSuanDataFree,protocol.JieSuan,nil)
	}

	if this.JieSuanDataFree.FreeRemainTimes <= 0 && this.JieSuanDataFree.FreeGameTimes <= 0{ //Free game over
		this.JieSuanData.TotalBackScore = this.JieSuanDataFree.TotalBackScore
		lobbyStorage.Win(utils.ConvertOID(this.UserID),this.Name, this.JieSuanData.TotalBackScore - this.ResultsPool,game.SlotLs,false)

		//pay.CheckoutAgentIncome(utils.ConvertOID(this.UserID),totalXiaZhu,this.EventID,game.SlotLs)

		this.IsInCheckout = false

		this.InitCheckoutData(this.ReelsListFree)
		this.ModeType = NORMAL
	}
	res,_ := json.Marshal(this.JieSuanDataFree)
	log.Info("----------------------slot ls jiesuan Free ---%s",res)
	return nil
}
func (this *MyTable) SpinTrial(session gate.Session,msg map[string]interface{})  (err error) {
	if this.ModeType != TRIAL{ //
		error := errCode.ErrParams
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.SpinTrial,error)
		return nil
	}
	if this.JieSuanDataTrialFree.FreeRemainTimes > 0 || this.JieSuanDataTrialFree.FreeGameTimes > 0{ //free剩余次数
		error := errCode.ErrParams
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.SpinTrial,error)
		return nil
	}
	this.IsInCheckout = true
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)
	this.InitCheckoutData(this.ReelsList)
	//this.EventID = string(game.SlotLs) + "_" + strconv.FormatInt(time.Now().Unix(),10)
	this.CoinNum = msg["CoinNum"].(int64)
	this.CoinValue = msg["CoinValue"].(int64)

	totalXiaZhu :=  this.CoinNum * this.CoinValue
	if this.TrialData.VndBalance < totalXiaZhu{ //下注金额不足
		error := errCode.BalanceNotEnough
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.SpinTrial,error)
		return nil
	}
	this.TrialData.VndBalance -= totalXiaZhu
	this.CalcCheckout(this.ReelsListTrial,this.CoinNum,this.CoinValue)

	if this.JieSuanDataTrial.FreeGameTimes <= 0 {
		if this.JieSuanDataTrial.TotalBackScore > 0{
			this.TrialData.VndBalance += this.JieSuanDataTrial.TotalBackScore
		}

	}else{
		this.JieSuanDataTrialFree.TotalBackScore = this.JieSuanDataTrial.TotalBackScore
		this.JieSuanDataTrialFree.TrialData = this.TrialData
		this.JieSuanDataTrialFree.CoinNum = this.JieSuanDataTrial.CoinNum
		this.JieSuanDataTrialFree.CoinValue = this.JieSuanDataTrial.CoinValue
	}
	this.JieSuanDataTrial.TrialData = this.TrialData
	sb := vGate.QuerySessionBean(this.UserID)
	if sb != nil{
		session,_ := basegate.NewSession(this.app, sb.Session)
		this.sendPack(session.GetSessionID(),game.Push,this.JieSuanDataTrial,protocol.JieSuan,nil)
	}
	res,_ := json.Marshal(this.JieSuanDataTrial)
	log.Info("----------------------slot ls jiesuan ---%s",res)
	if this.JieSuanDataTrial.FreeGameTimes <= 0{
		this.IsInCheckout = false
	}
	return nil
}
func (this *MyTable) SpinTrialFree(session gate.Session,msg map[string]interface{})  (err error) {
	if this.JieSuanDataTrialFree.FreeRemainTimes <= 0{ //free剩余次数
		log.Info("------%d",this.JieSuanDataTrialFree.FreeRemainTimes)
		error := errCode.ErrParams
		this.sendPack(session.GetSessionID(),game.Push,"",protocol.SpinTrialFree,error)
		return nil
	}
	player := this.FindPlayer(session)
	if player == nil {
		return nil
	}
	player.OnRequest(session)

	this.CalcCheckout(this.ReelsListTrialFree,this.CoinNum,this.CoinValue)

	this.JieSuanDataTrialFree.FreeRemainTimes--

	this.JieSuanDataTrialFree.TrialData = this.TrialData
	sb := vGate.QuerySessionBean(this.UserID)
	if sb != nil{
		session,_ := basegate.NewSession(this.app, sb.Session)
		this.sendPack(session.GetSessionID(),game.Push,this.JieSuanDataTrialFree,protocol.JieSuan,nil)
	}
	if this.JieSuanDataTrialFree.FreeRemainTimes <= 0 && this.JieSuanDataTrialFree.FreeGameTimes <= 0{ //Free game over
		this.JieSuanDataTrial.TotalBackScore = this.JieSuanDataTrialFree.TotalBackScore
		this.TrialData.VndBalance += this.JieSuanDataTrial.TotalBackScore
		this.IsInCheckout = false
		this.JieSuanDataTrial.TrialData = this.TrialData
		this.InitCheckoutData(this.ReelsListTrialFree)
		this.ModeType = TRIAL
	}
	res,_ := json.Marshal(this.JieSuanDataFree)
	log.Info("----------------------slot ls jiesuan Free ---%s",res)
	return nil
}