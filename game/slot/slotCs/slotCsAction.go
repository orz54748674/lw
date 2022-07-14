package slotCs

import (
	"encoding/json"
	"strconv"
	"time"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/activity"
	vGate "vn/gate"
	"vn/storage/gameStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) Enter(session gate.Session, msg map[string]interface{}) (err error) {
	player := &room.BasePlayerImp{}
	player.Bind(session)

	player.OnRequest(session)
	userID := session.GetUserID()
	if !this.BroadCast {
		this.BroadCast = true
	}
	if userID == "" {
		log.Info("your userid is empty")
		return nil
	}
	//modeType := ModeType(msg["modeType"].(string))
	this.Players[userID] = player
	this.UserID = userID
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	if user.Type == userStorage.TypeNormal {
		this.Role = USER
	}
	this.Name = user.NickName
	if this.ModeType == NORMAL && this.JieSuanData.BonusGame {
		if this.BonusGameData.TotalSymbolScore > 0 {
			this.EventID = string(game.SlotCs) + "_" + "Bonus" + "_" + strconv.FormatInt(time.Now().Unix(), 10)
			bill := walletStorage.NewBill(this.UserID, walletStorage.TypeIncome, walletStorage.EventGameSlotCs, this.EventID, this.BonusGameData.TotalSymbolScore)
			walletStorage.OperateVndBalance(bill)

			this.notifyWallet(this.UserID)
			gameStorage.IncProfitByUser(this.UserID, 0, -this.BonusGameData.TotalSymbolScore, 0, this.BonusGameData.TotalSymbolScore)
			wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
			betRecordParam := gameStorage.BetRecordParam{
				Uid:        this.UserID,
				GameType:   game.SlotCs,
				Income:     this.BonusGameData.TotalSymbolScore,
				BetAmount:  0,
				CurBalance: wallet.VndBalance + wallet.SafeBalance,
				SysProfit:  0,
				BotProfit:  0,
				BetDetails: "",
				GameId:     this.EventID,
				GameNo:     strconv.FormatInt(time.Now().Unix(), 10),
				GameResult: "",
				IsSettled:  true,
			}
			gameStorage.InsertBetRecord(betRecordParam)
		}
		this.JieSuanData.BonusGame = false
	}

	tableInfoRet := this.GetTableInfo()
	_ = this.sendPack(session.GetSessionID(), game.Push, tableInfoRet, protocol.Enter, nil)

	this.ModeType = NORMAL
	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)
	return nil
}
func (this *MyTable) QuitTable(session gate.Session) (err error) {
	userID := session.GetUserID()
	sb := vGate.QuerySessionBean(userID)
	//if this.IsInFreeGame(){
	//	if sb != nil {
	//		this.sendPack(session.GetSessionID(), game.Push, "", protocol.QuitTable, errCode.FreeGameCantQuit)
	//	}
	//	return nil
	//}
	ret := this.DealProtocolFormat("", protocol.QuitTable, nil)
	this.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
	this.onlinePush.ExecuteCallBackMsg(this.Trace())
	if !this.IsInCheckout && (!this.IsInFreeGame() || this.ModeType == TRIAL) {
		this.PutQueue(protocol.ClearTable)
	}
	return nil
}
func (this *MyTable) GetResults(session gate.Session, msg map[string]interface{}) (err error) {
	res := make(map[string]interface{})
	res["JieSuanData"] = this.JieSuanData
	res["BonusGameData"] = this.BonusGameData
	res["MiniGameData"] = this.MiniGameData
	this.sendPack(session.GetSessionID(), game.Push, res, protocol.GetResults, nil)
	return nil
}
func (this *MyTable) SelectBonusSymbol(session gate.Session) (err error) {
	this.BonusGameData.ClickNum++
	if this.ModeType == NORMAL {
		gameProfit := gameStorage.QueryProfitByUser(this.UserID)
		rand := this.RandInt64(1, 4)
		if this.BonusGameData.ClickNum == 1 || (rand != 1 && gameProfit.BotBalance >= BonusScoreList[0]*this.CoinValue/10+this.BonusGameData.TotalSymbolScore) {
		Again:
			rand = this.RandInt64(1, 11)
			if rand >= 1 && rand < 5 {
				this.BonusGameData.CurSymbolScore = BonusScoreList[0] * this.CoinValue / 10
			} else if rand >= 5 && rand < 8 {
				if gameProfit.BotBalance < BonusScoreList[1]*this.CoinValue/10 {
					goto Again
				}
				this.BonusGameData.CurSymbolScore = BonusScoreList[1] * this.CoinValue / 10
			} else if rand >= 8 && rand < 10 {
				if gameProfit.BotBalance < BonusScoreList[2]*this.CoinValue/10 {
					goto Again
				}
				this.BonusGameData.CurSymbolScore = BonusScoreList[2] * this.CoinValue / 10
			} else if rand == 10 {
				if gameProfit.BotBalance < BonusScoreList[3]*this.CoinValue/10 {
					goto Again
				}
				this.BonusGameData.CurSymbolScore = BonusScoreList[3] * this.CoinValue / 10
			}
			this.BonusGameData.TotalSymbolScore += this.BonusGameData.CurSymbolScore
		} else {
			this.BonusGameData.State = 2
		}
	} else {
		rand := this.RandInt64(1, 4)
		if rand != 1 || this.BonusGameData.ClickNum == 1 {
			rand = this.RandInt64(1, 5)
			if rand == 1 {
				this.BonusGameData.CurSymbolScore = BonusScoreList[0] * this.CoinValue / 10
			} else if rand == 2 {
				this.BonusGameData.CurSymbolScore = BonusScoreList[1] * this.CoinValue / 10
			} else if rand == 3 {
				this.BonusGameData.CurSymbolScore = BonusScoreList[2] * this.CoinValue / 10
			} else if rand == 4 {
				this.BonusGameData.CurSymbolScore = BonusScoreList[3] * this.CoinValue / 10
			}
			this.BonusGameData.TotalSymbolScore += this.BonusGameData.CurSymbolScore
		} else {
			this.BonusGameData.State = 2
		}
	}
	this.CountDown = this.GameConf.BonusTimeOut
	this.sendPack(session.GetSessionID(), game.Push, this.BonusGameData, protocol.SelectBonusSymbol, nil)
	return nil
}

//func (this *MyTable) BonusTimeOut(session gate.Session)  (err error)  {
//	userID := session.GetUserID()
//	sb := vGate.QuerySessionBean(userID)
//	if !this.JieSuanData.BonusGame{
//		if sb != nil {
//			this.sendPack(session.GetSessionID(), game.Push, "", protocol.SelectBonusSymbol, errCode.ServerError)
//		}
//		return nil
//	}
//	this.JieSuanData.BonusGame = false
//	if this.BonusGameData.TotalSymbolScore == 0{
//		this.BonusGameData.TotalSymbolScore = BonusScoreList[0]
//	}
//	if this.BonusGameData.Times == 0{
//		this.BonusGameData.Times = this.JieSuanData.BonusTimes[0]
//	}
//	this.BonusGameData.TotalSymbolScore *= this.BonusGameData.Times
//
//	if this.ModeType == NORMAL{
//		this.EventID = string(game.SlotCs) + "_" + "Bonus" + "_" + strconv.FormatInt(time.Now().Unix(),10)
//		bill := walletStorage.NewBill(this.UserID,walletStorage.TypeIncome,walletStorage.EventGameSlotCs,this.EventID,this.BonusGameData.TotalSymbolScore)
//		walletStorage.OperateVndBalance(bill)
//		this.notifyWallet(userID)
//	}
//	this.JieSuanData.TotalBackScore += this.BonusGameData.TotalSymbolScore
//	this.BonusGameData.IsTimeOut = true
//	this.sendPack(session.GetSessionID(), game.Push, this.BonusGameData, protocol.SelectBonusTimes, nil)
//	return nil
//}
func (this *MyTable) SelectBonusTimes(session gate.Session, msg map[string]interface{}) (err error) {
	userID := session.GetUserID()
	this.JieSuanData.BonusGame = false
	serial, _ := utils.ConvertInt(msg["Serial"])
	gameProfit := gameStorage.QueryProfitByUser(this.UserID)
Again:
	rand := this.RandInt64(1, 7)
	this.BonusGameData.CurSymbolScore = this.BonusGameData.TotalSymbolScore
	if rand >= 1 && rand < 4 {
		this.BonusGameData.Times = this.JieSuanData.BonusTimes[0]
	} else if rand >= 4 && rand < 6 {
		if this.ModeType == NORMAL && gameProfit.BotBalance < this.JieSuanData.BonusTimes[1]*this.BonusGameData.TotalSymbolScore {
			goto Again
		}
		this.BonusGameData.Times = this.JieSuanData.BonusTimes[1]
	} else if rand == 6 {
		if this.ModeType == NORMAL && gameProfit.BotBalance < this.JieSuanData.BonusTimes[2]*this.BonusGameData.TotalSymbolScore {
			goto Again
		}
		this.BonusGameData.Times = this.JieSuanData.BonusTimes[2]
	}
	this.BonusGameData.TotalSymbolScore *= this.BonusGameData.Times
	this.BonusGameData.Serial = int(serial)
	this.BonusGameData.TimesList = this.JieSuanData.BonusTimes
	for k, v := range this.BonusGameData.TimesList {
		if v == this.BonusGameData.Times && k != int(serial) {
			this.BonusGameData.TimesList[k] = this.BonusGameData.TimesList[serial]
			this.BonusGameData.TimesList[serial] = this.BonusGameData.Times
		}
	}

	if this.ModeType == NORMAL && this.BonusGameData.TotalSymbolScore > 0 {
		this.EventID = string(game.SlotCs) + "_" + "Bonus" + "_" + strconv.FormatInt(time.Now().Unix(), 10)
		bill := walletStorage.NewBill(this.UserID, walletStorage.TypeIncome, walletStorage.EventGameSlotCs, this.EventID, this.BonusGameData.TotalSymbolScore)
		walletStorage.OperateVndBalance(bill)
		this.notifyWallet(userID)
		gameStorage.IncProfitByUser(this.UserID, 0, -this.BonusGameData.TotalSymbolScore, 0, this.BonusGameData.TotalSymbolScore)
		wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
		betRecordParam := gameStorage.BetRecordParam{
			Uid:        this.UserID,
			GameType:   game.SlotCs,
			Income:     this.BonusGameData.TotalSymbolScore,
			BetAmount:  0,
			CurBalance: wallet.VndBalance + wallet.SafeBalance,
			SysProfit:  0,
			BotProfit:  0,
			BetDetails: "",
			GameId:     this.EventID,
			GameNo:     strconv.FormatInt(time.Now().Unix(), 10),
			GameResult: "",
			IsSettled:  true,
		}
		gameStorage.InsertBetRecord(betRecordParam)
	} else {
		this.JieSuanData.TrialData.VndBalance += this.BonusGameData.TotalSymbolScore
	}
	this.JieSuanData.TotalBackScore += this.BonusGameData.TotalSymbolScore

	this.sendPack(session.GetSessionID(), game.Push, this.BonusGameData, protocol.SelectBonusTimes, nil)

	return nil
}
func (this *MyTable) SelectMiniSymbol(session gate.Session, msg map[string]interface{}) (err error) {
	userID := session.GetUserID()
	sb := vGate.QuerySessionBean(userID)
	if this.JieSuanData.TotalBackScore <= 0 || this.MiniGameData.State != 1 || this.MiniGameData.ClickNum >= 3 {
		if sb != nil {
			this.sendPack(session.GetSessionID(), game.Push, "", protocol.SelectMiniSymbol, errCode.ServerError)
		}
		return nil
	}
	this.MiniGameData.ClickNum++
	if this.MiniGameData.ClickNum == 1 {
		this.MiniGameData.TotalSymbolScore = this.JieSuanData.TotalBackScore
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	if wallet.VndBalance < this.MiniGameData.TotalSymbolScore {
		if sb != nil {
			this.sendPack(session.GetSessionID(), game.Push, "", protocol.SelectMiniSymbol, errCode.BalanceNotEnough)
			return
		}
	}
	this.JieSuanData.BonusGame = false
	serial, _ := utils.ConvertInt(msg["Serial"])
	gameProfit := gameStorage.QueryProfitByUser(this.UserID)
	rand := this.RandInt64(1, 3)
	if rand == 1 || gameProfit.BotBalance < this.MiniGameData.TotalSymbolScore {
		this.MiniGameData.CurSymbol = 0
		this.MiniGameData.State = 2
	} else {
		this.MiniGameData.CurSymbol = 1
	}
	this.MiniGameData.Serial = int(serial)
	for k, v := range this.MiniGameData.SymbolList {
		if v == this.MiniGameData.CurSymbol && k != int(serial) {
			this.MiniGameData.SymbolList[k] = this.MiniGameData.SymbolList[serial]
			this.MiniGameData.SymbolList[serial] = this.MiniGameData.CurSymbol
		}
	}
	botProfit := this.MiniGameData.TotalSymbolScore * int64(this.GameConf.BotProfitPerThousand) / 1000
	betAmount := this.MiniGameData.TotalSymbolScore
	if this.MiniGameData.CurSymbol == 1 { //中了
		getScore := this.MiniGameData.TotalSymbolScore * (1000 - int64(this.GameConf.ProfitPerThousand)*2) / 1000
		sysProfit := this.MiniGameData.TotalSymbolScore - getScore
		this.MiniGameData.TotalSymbolScore += getScore
		this.EventID = string(game.SlotCs) + "_" + "Mini" + "_" + strconv.FormatInt(time.Now().Unix(), 10)
		bill := walletStorage.NewBill(this.UserID, walletStorage.TypeIncome, walletStorage.EventGameSlotCs, this.EventID, getScore)
		walletStorage.OperateVndBalance(bill)
		profit := this.MiniGameData.TotalSymbolScore * int64(this.GameConf.ProfitPerThousand) * 2 / 1000
		gameStorage.IncProfitByUser(this.UserID, profit, -getScore-botProfit, botProfit, getScore)

		wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
		betData := make(map[string]interface{})
		betData["win"] = 1
		betDetail, _ := json.Marshal(betData)
		betRecordParam := gameStorage.BetRecordParam{
			Uid:        this.UserID,
			GameType:   game.SlotCs,
			Income:     getScore,
			BetAmount:  betAmount,
			CurBalance: wallet.VndBalance + wallet.SafeBalance,
			SysProfit:  sysProfit,
			BotProfit:  botProfit,
			BetDetails: string(betDetail),
			GameId:     this.EventID,
			GameNo:     strconv.FormatInt(time.Now().Unix(), 10),
			GameResult: "",
			IsSettled:  true,
		}
		gameStorage.InsertBetRecord(betRecordParam)
	} else {
		this.EventID = string(game.SlotCs) + "_" + "Mini" + "_" + strconv.FormatInt(time.Now().Unix(), 10)
		bill := walletStorage.NewBill(this.UserID, walletStorage.TypeExpenses, walletStorage.EventGameSlotCs, this.EventID, -this.MiniGameData.TotalSymbolScore)
		walletStorage.OperateVndBalance(bill)

		gameStorage.IncProfitByUser(this.UserID, 0, this.MiniGameData.TotalSymbolScore-botProfit, botProfit, -this.MiniGameData.TotalSymbolScore)
		this.MiniGameData.TotalSymbolScore = 0
		if this.ModeType == NORMAL {
			activity.CalcEncouragementFunc(this.UserID)
		}
		wallet := walletStorage.QueryWallet(utils.ConvertOID(this.UserID))
		betData := make(map[string]interface{})
		betData["win"] = 0
		betDetail, _ := json.Marshal(betData)
		betRecordParam := gameStorage.BetRecordParam{
			Uid:        this.UserID,
			GameType:   game.SlotCs,
			Income:     -this.MiniGameData.TotalSymbolScore,
			BetAmount:  betAmount,
			CurBalance: wallet.VndBalance + wallet.SafeBalance,
			SysProfit:  0,
			BotProfit:  botProfit,
			BetDetails: string(betDetail),
			GameId:     this.EventID,
			GameNo:     strconv.FormatInt(time.Now().Unix(), 10),
			GameResult: "",
			IsSettled:  true,
		}
		gameStorage.InsertBetRecord(betRecordParam)
	}

	this.JieSuanData.TotalBackScore = this.MiniGameData.TotalSymbolScore
	this.sendPack(session.GetSessionID(), game.Push, this.MiniGameData, protocol.SelectMiniSymbol, nil)
	this.notifyWallet(userID)
	return nil
}
