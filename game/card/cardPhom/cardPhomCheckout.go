package cardPhom

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

func (this *MyTable) JieSuan() {
	this.CountDown = this.GameConf.JieSuanTime
	this.SwitchRoomState()

	//计算吃的
	for k, v := range this.PlayerList {
		if v.Ready {
			for _, v1 := range v.EatData {
				this.PlayerList[k].EatScore += v1.Score
				this.PlayerList[v1.PreIdx].EatScore -= v1.Score
			}
		}
	}

	//计算直赢
	if this.StraightScoreData.Have {
		for k, v := range this.PlayerList {
			if v.Ready && k != this.StraightScoreData.Idx {

				this.PlayerList[k].FinalScore = -this.StraightScoreData.Score

				vndBalance := this.PlayerList[k].EatScore
				if this.PlayerList[k].Role == ROBOT {
					vndBalance += this.PlayerList[k].Yxb
				} else {
					vndBalance += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[k].UserID)).VndBalance
				}
				if vndBalance < -this.PlayerList[k].FinalScore {
					this.PlayerList[k].FinalScore = -vndBalance
				}

				this.PlayerList[this.StraightScoreData.Idx].FinalScore -= this.PlayerList[k].FinalScore
			}
		}
	} else {
		winPhomData := this.PlayerList[this.WinIdx].CalcPhomData
		remainPk := len(this.PlayerList[this.WinIdx].HandPoker)
		if len(winPhomData.Phom) == 3 {
			score := int64(0)
			if remainPk == 0 {
				score = 10 * this.BaseScore
			} else {
				score = 5 * this.BaseScore
			}
			for k, v := range this.PlayerList {
				if v.Ready && k != this.WinIdx {
					this.PlayerList[k].FinalScore = -score
				}
			}
		} else if remainPk == 0 {
			for k, v := range this.PlayerList {
				if v.Ready && k != this.WinIdx {
					this.PlayerList[k].FinalScore = -5 * this.BaseScore
				}
			}
		} else {
			noMomNum := 0
			this.PlayerList[this.WinIdx].CalcPhomData.State = First
			for k, v := range this.RankList {
				if v.Idx != this.WinIdx {
					if v.State != MOM {
						noMomNum += 1
						this.PlayerList[v.Idx].FinalScore = -NormalTimes[this.PlayingNum][k-1] * this.BaseScore
						if this.PlayingNum == 4 {
							if k == 1 {
								this.PlayerList[v.Idx].CalcPhomData.State = Second
							} else if k == 2 {
								this.PlayerList[v.Idx].CalcPhomData.State = Third
							} else if k == 3 {
								this.PlayerList[v.Idx].CalcPhomData.State = Four
							}
						} else if this.PlayingNum == 3 {
							if k == 1 {
								this.PlayerList[v.Idx].CalcPhomData.State = Third
							} else if k == 2 {
								this.PlayerList[v.Idx].CalcPhomData.State = Four
							}
						} else if this.PlayingNum == 2 {
							if k == 1 {
								this.PlayerList[v.Idx].CalcPhomData.State = Four
							}
						}

					} else {
						this.PlayerList[v.Idx].FinalScore = -4 * this.BaseScore
					}
				}
			}
			if noMomNum == 0 {
				this.PlayerList[this.WinIdx].CalcPhomData.State = XaoKhan
			}
		}

		//计算包赔
		allLoseIdx := -1
		for _, v := range this.PlayerList {
			if v.Ready && len(v.EatData) == 3 {
				allLoseIdx = v.EatData[0].PreIdx
				break
			}
		}
		if allLoseIdx < 0 {
			winPhomData := this.PlayerList[this.WinIdx].CalcPhomData
			if this.LastRoundEatIdx >= 0 && this.LastRoundEatIdx != this.WinIdx && len(winPhomData.Phom) == 3 {
				allLoseIdx = this.LastRoundEatIdx
			}
		}

		if allLoseIdx >= 0 {
			this.PlayerList[this.WinIdx].CalcPhomData.State = UDen
			this.PlayerList[allLoseIdx].CalcPhomData.State = VoNo
			for k, v := range this.PlayerList {
				if v.Ready && k != this.WinIdx && k != allLoseIdx {
					this.PlayerList[allLoseIdx].FinalScore += this.PlayerList[k].FinalScore
					this.PlayerList[k].FinalScore = 0
					this.PlayerList[k].CalcPhomData.State = Normal
				}
			}
		}

		for k, v := range this.PlayerList {
			if v.Ready && k != this.WinIdx {
				vndBalance := this.PlayerList[k].EatScore
				if this.PlayerList[k].Role == ROBOT {
					vndBalance += this.PlayerList[k].Yxb
				} else {
					vndBalance += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[k].UserID)).VndBalance
				}
				if vndBalance < -this.PlayerList[k].FinalScore {
					this.PlayerList[k].FinalScore = -vndBalance
				}
				this.PlayerList[this.WinIdx].FinalScore -= this.PlayerList[k].FinalScore
			}
		}

	}

	this.JieSuanData.RoomState = this.RoomState

	realTotalCommission := int64(0)
	for k, v := range this.PlayerList {
		if v.Ready {
			this.JieSuanData.PlayerInfo[k] = PlayerInfo{
				StraightType: v.StraightType,
				HandPoker:    v.HandPoker,
				PhomState:    v.CalcPhomData.State,
			}
			this.PlayerList[k].FinalScore += this.PlayerList[k].EatScore
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
		Account      string
		IP           string
		Idx          int
		Poker        []int
		PhomPoker    [][]int
		EatPoker     []int
		LastEatPoker []int
		GivePoker    []int
		Self         bool
		Income       int64
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
			handPk := make([]int, 0)
			for _, v1 := range v.HandPoker {
				handPk = append(handPk, switchBackendCard[v1])
			}
			phomPk := make([][]int, 0)
			for _, v1 := range v.CalcPhomData.Phom {
				phom := make([]int, 0)
				for _, v2 := range v1 {
					phom = append(phom, switchBackendCard[v2])
				}
				phomPk = append(phomPk, phom)
			}
			eatPk := make([]int, 0)
			lastEatPoker := make([]int, 0)
			for _, v1 := range v.EatData {
				if v1.LastRoundEat {
					lastEatPoker = append(lastEatPoker, switchBackendCard[v1.Poker])
				} else {
					eatPk = append(eatPk, switchBackendCard[v1.Poker])
				}
			}
			givePk := make([]int, 0)
			for _, v1 := range v.GivePoker {
				givePk = append(givePk, switchBackendCard[v1])
			}
			backendResults = append(backendResults, ResultData{
				Account:      v.Account,
				IP:           ip,
				Idx:          k,
				Self:         false,
				Poker:        handPk,
				PhomPoker:    phomPk,
				EatPoker:     eatPk,
				LastEatPoker: lastEatPoker,
				GivePoker:    givePk,
				Income:       v.TotalBackYxb,
			})
			gameStorage.IncGameWinLoseScore(game.CardPhom, v.Name, v.TotalBackYxb)
		}
	}
	resultStr, _ := json.Marshal(backendResults)
	botProfit := int64(0)
	for k, v := range this.PlayerList {
		if v.Ready && v.Role != ROBOT {
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				session, _ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(), game.Push, this.JieSuanData, protocol.JieSuan, nil)
			}

			if v.TotalBackYxb > 0 && v.Role == USER {
				lobbyStorage.Win(utils.ConvertOID(v.UserID), v.Name, v.TotalBackYxb, game.CardPhom, false)
			}
			if v.TotalBackYxb > 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeIncome, walletStorage.EventGameCardPhom, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER {
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), v.TotalBackYxb, this.EventID, game.CardPhom)
				}
			} else if v.TotalBackYxb < 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeExpenses, walletStorage.EventGameCardPhom, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER {
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), -v.TotalBackYxb, this.EventID, game.CardPhom)
				}
			}
			wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
			betRecordParam := gameStorage.BetRecordParam{
				Uid:        v.UserID,
				GameType:   game.CardPhom,
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
		if v.Ready {
			this.PlayerList[k].Yxb += v.TotalBackYxb

			tmp := this.JieSuanData.PlayerInfo[k]
			tmp.TotalBackYxb = v.TotalBackYxb
			this.JieSuanData.PlayerInfo[k] = tmp
		}

	}
	if this.GetRealReadyPlayerNum() > 0 {
		gameStorage.IncProfit("", game.CardPhom, realTotalCommission, realTotalPay-botProfit, botProfit)
	}
	//go func() {
	//	time.Sleep(time.Second * 5)
	//	for _,v := range this.PlayerList{
	//		if v.Role == USER || v.Role == Agent{
	//			this.notifyWallet(v.UserID)
	//		}
	//	}
	//}()

	reboot := gameStorage.QueryGameReboot(game.CardPhom)
	if reboot == "true" {
		this.RoomState = ROOM_END
	}
	if this.GetTableRealPlayerNum() <= 0 && !this.AutoCreate {
		this.RoomState = ROOM_END
	}

	for _, v := range this.PlayerList {
		if v.Ready && v.Role != ROBOT {
			activityStorage.UpsertGameDataInBet(v.UserID, game.CardPhom, 0)
			activity.CalcEncouragementFunc(v.UserID)
		}
	}
	//res,_ := json.Marshal(this.JieSuanData)
	//log.Info("---------------jie suan data----------------------%s",res)
	this.SeqExecFlag = true

}
