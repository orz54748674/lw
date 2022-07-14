package cardCatte

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
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) CalcRemainPokerScore(pk []int, putPk []PutData) int64 {
	score := int64(0)
	for _, v := range pk {
		if v%0x10 == 0x0e {
			score = 2 * this.BaseScore
		}
	}
	for _, v := range putPk { //盖的A
		if v.State < 1 && v.Poker%0x10 == 0x0e {
			score = 2 * this.BaseScore
		}
	}
	return score
}
func (this *MyTable) JieSuan() {
	this.CountDown = this.GameConf.JieSuanTime
	this.SwitchRoomState()
	this.RoundNum++
	fiveMaxIdx := -1
	sixMaxIdx := -1
	if this.StraightScoreData.Type > 0 { //直赢
		for k, v := range this.PlayerList {
			if v.Ready && k != this.WinIdx {
				this.PlayerList[k].FinalScore -= 12 * this.BaseScore
				this.PlayerList[this.WinIdx].FinalScore -= this.PlayerList[k].FinalScore
			}
		}
	} else if this.WinIdx >= 0 {
		for k, v := range this.PlayerList {
			if v.Ready && k != this.WinIdx {
				remainPokerScore := this.CalcRemainPokerScore(v.HandPoker, v.PutPoker)
				this.PlayerList[k].FinalScore -= 8*this.BaseScore + remainPokerScore

				this.PlayerList[this.WinIdx].FinalScore -= this.PlayerList[k].FinalScore
			}
		}
	} else {
		fiveMaxIdx = this.MaxIdx
		//fiveMaxPk := this.PlayerList[fiveMaxIdx].ShowPoker[0].Poker

		//for k,_ := range this.WaitingList{
		//	if k != fiveMaxIdx{
		//		pk := this.PlayerList[k].ShowPoker[0].Poker
		//		if  pk / 0x10 == fiveMaxPk / 0x10 && pk > fiveMaxPk {
		//			this.PlayerList[fiveMaxIdx].ShowPoker[0].State = 0
		//			fiveMaxIdx = k
		//			fiveMaxPk = pk
		//		}else{
		//			this.PlayerList[k].ShowPoker[0].State = 0
		//		}
		//	}
		//}

		sixMaxIdx = fiveMaxIdx
		sixMaxPk := this.PlayerList[sixMaxIdx].ShowPoker[1].Poker
		for k, v := range this.PlayerList {
			if v.Ready && v.IsHavePeople && k != sixMaxIdx && !this.PlayerIsOver(k) {
				pk := this.PlayerList[k].ShowPoker[1].Poker
				if pk/0x10 == sixMaxPk/0x10 && pk > sixMaxPk {
					this.PlayerList[sixMaxIdx].ShowPoker[1].State = 0
					sixMaxIdx = k
					sixMaxPk = pk
				} else {
					this.PlayerList[k].ShowPoker[1].State = 0
				}
			}
		}

		this.WinIdx = sixMaxIdx

		for k, v := range this.PlayerList {
			if v.Ready && k != this.WinIdx {
				remainPokerScore := this.CalcRemainPokerScore(v.HandPoker, v.PutPoker)
				showNum := 0
				for _, v1 := range v.PutPoker {
					if v1.State > 0 {
						showNum++
					}
				}

				if showNum == 1 {
					this.PlayerList[k].FinalScore -= 5 * this.BaseScore
				} else if showNum == 2 {
					this.PlayerList[k].FinalScore -= 4 * this.BaseScore
				} else if showNum == 3 {
					this.PlayerList[k].FinalScore -= 3 * this.BaseScore
				} else {
					this.PlayerList[k].FinalScore -= 8 * this.BaseScore
				}

				this.PlayerList[k].FinalScore -= remainPokerScore
				this.PlayerList[this.WinIdx].FinalScore -= this.PlayerList[k].FinalScore
			}
		}
	}

	this.JieSuanData.RoomState = this.RoomState
	this.JieSuanData.CountDown = this.CountDown
	this.JieSuanData.PutOverRecord = this.PutOverRecord
	this.JieSuanData.FiveMaxIdx = fiveMaxIdx
	this.JieSuanData.SixMaxIdx = sixMaxIdx
	animationType := AnimationType(0)
	realTotalCommission := int64(0)
	for k, v := range this.PlayerList {
		if v.Ready {
			if k == this.WinIdx {
				if this.StraightScoreData.Type > 0 {
					animationType = AnimationType(this.StraightScoreData.Type)
				} else {
					animationType = WinNormal
				}
			} else {
				showNum := 0
				for _, v1 := range v.PutPoker {
					if v1.State > 0 {
						showNum++
					}
				}
				if showNum == 0 {
					animationType = LoserAllCheck
				} else if this.CalcRemainPokerScore(v.HandPoker, v.PutPoker) > 0 {
					animationType = LoserHaveA
				} else {
					animationType = LoserNormal
				}

			}
			this.JieSuanData.PlayerInfo[k] = PlayerInfo{
				StraightType:  v.StraightType,
				HandPoker:     v.HandPoker,
				ShowPoker:     v.ShowPoker,
				AnimationType: animationType,
			}
			this.PlayerList[k].TotalBackYxb = this.PlayerList[k].FinalScore
			if this.PlayerList[k].TotalBackYxb > 0 {
				this.PlayerList[k].SysProfit = this.PlayerList[k].TotalBackYxb * int64(this.GameConf.ProfitPerThousand) / 1000
				this.PlayerList[k].TotalBackYxb -= this.PlayerList[k].SysProfit
				if v.Role == USER {
					realTotalCommission += this.PlayerList[k].SysProfit
				}
			}
		}
	}

	realTotalPay := -realTotalCommission //赔付给真实玩家的总数 抽水需要去掉
	for k, v := range this.PlayerList {
		if v.Ready {
			this.PlayerList[k].Yxb += v.TotalBackYxb

			tmp := this.JieSuanData.PlayerInfo[k]
			tmp.TotalBackYxb = v.TotalBackYxb
			this.JieSuanData.PlayerInfo[k] = tmp
			if v.Role == USER {
				realTotalPay -= v.TotalBackYxb
			}
		}
	}
	type ResultData struct {
		Account string
		IP      string
		Idx     int
		Poker   []PutData
		Self    bool
		Income  int64
	}
	backendResults := make([]ResultData, 0)
	for k, v := range this.PlayerList {
		if v.Ready && v.IsHavePeople {
			ip := ""
			if v.Role != ROBOT {
				login := userStorage.QueryLogin(utils.ConvertOID(v.UserID))
				if login != nil {
					ip = login.LastIp
				} else {
					ip = "UnKnow"
				}
			} else {
				ip = "0.0.0.0"
			}
			poker := make([]PutData, 0)
			for _, v1 := range v.PutPoker {
				pkData := PutData{
					State: v1.State,
					Poker: switchBackendCard[v1.Poker],
				}
				poker = append(poker, pkData)
			}
			for _, v1 := range v.HandPoker {
				pkData := PutData{
					State: -1,
					Poker: switchBackendCard[v1],
				}
				if k == this.WinIdx {
					pkData.State = 1
				}
				poker = append(poker, pkData)
			}
			for _, v1 := range v.ShowPoker {
				pkData := PutData{
					State: v1.State,
					Poker: switchBackendCard[v1.Poker],
				}
				poker = append(poker, pkData)
			}

			backendResults = append(backendResults, ResultData{
				Account: v.Account,
				IP:      ip,
				Idx:     k,
				Self:    false,
				Poker:   poker,
				Income:  v.TotalBackYxb,
			})
			gameStorage.IncGameWinLoseScore(game.CardCatte, v.Name, v.TotalBackYxb)
		}
	}
	botProfit := int64(0)
	for k, v := range this.PlayerList {
		if v.Ready && v.Role != ROBOT {
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				session, _ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(), game.Push, this.JieSuanData, protocol.JieSuan, nil)
			}

			if v.TotalBackYxb > 0 && v.Role == USER {
				lobbyStorage.Win(utils.ConvertOID(v.UserID), v.Name, v.TotalBackYxb, game.CardCatte, false)
			}

			if v.TotalBackYxb > 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeIncome, walletStorage.EventGameCardCatte, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER {
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), v.TotalBackYxb, this.EventID, game.CardCatte)
				}
			} else if v.TotalBackYxb < 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeExpenses, walletStorage.EventGameCardCatte, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER {
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), -v.TotalBackYxb, this.EventID, game.CardCatte)
				}
			}
			backendResultsCopy := make([]ResultData, len(backendResults))
			copy(backendResultsCopy, backendResults)
			for k1, v1 := range backendResultsCopy {
				if v1.Idx == k {
					backendResultsCopy[k1].Self = true
					break
				}
			}

			resultStr, _ := json.Marshal(backendResultsCopy)
			wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
			betRecordParam := gameStorage.BetRecordParam{
				Uid:        v.UserID,
				GameType:   game.CardCatte,
				Income:     v.TotalBackYxb,
				BetAmount:  this.BaseScore,
				CurBalance: this.PlayerList[k].Yxb + wallet.SafeBalance,
				SysProfit:  this.PlayerList[k].SysProfit,
				BotProfit:  0,
				BetDetails: "",
				GameId:     strconv.FormatInt(time.Now().Unix(), 10),
				GameNo:     strconv.FormatInt(time.Now().Unix(), 10),
				GameResult: string(resultStr),
				IsSettled:  true,
			}
			if v.Role == USER {
				this.PlayerList[k].BotProfit = this.BaseScore * int64(this.GameConf.BotProfitPerThousand) / 1000
				botProfit += this.PlayerList[k].BotProfit
				betRecordParam.BotProfit = this.PlayerList[k].BotProfit
			} else {
				betRecordParam.SysProfit = 0
			}
			gameStorage.InsertBetRecord(betRecordParam)

		}

	}
	if this.GetRealReadyPlayerNum() > 0 {
		gameStorage.IncProfit("", game.CardCatte, realTotalCommission, realTotalPay-botProfit, botProfit)
	}
	//go func() {
	//	time.Sleep(time.Second * 5)
	//	for _,v := range this.PlayerList{
	//		if v.Role == USER || v.Role == Agent{
	//			this.notifyWallet(v.UserID)
	//		}
	//	}
	//}()

	reboot := gameStorage.QueryGameReboot(game.CardCatte)
	if reboot == "true" {
		this.RoomState = ROOM_END
	}
	if this.GetTableRealPlayerNum() <= 0 && !this.AutoCreate {
		this.RoomState = ROOM_END
	}

	for _, v := range this.PlayerList {
		if v.Ready && v.Role != ROBOT {
			activityStorage.UpsertGameDataInBet(v.UserID, game.CardCatte, 0)
			activity.CalcEncouragementFunc(v.UserID)
		}
	}
	//res,_ := json.Marshal(this.JieSuanData)
	//log.Info("---------------jie suan data----------------------%s",res)
	this.SeqExecFlag = true
}
