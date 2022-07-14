package cardCddN

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

func (this *MyTable) CalcRemainPokerScore(pk []int) int64 {
	score := int64(0)
	for _, v := range pk {
		if v == 0x3f || v == 0x4f {
			score += this.BaseScore * 6
		} else if v == 0x1f || v == 0x2f {
			score += this.BaseScore * 3
		}
	}
	compList := this.CompStraightPair(pk)
	haveStraightPair4 := false
	haveStraightPair3 := false
	for _, v := range compList {
		if v.num == 4 {
			haveStraightPair4 = true
			break
		} else if v.num == 3 {
			haveStraightPair3 = true
			break
		}
	}
	if haveStraightPair4 {
		score += this.BaseScore * 18
	} else if haveStraightPair3 {
		score += this.BaseScore * 14
	}

	compList = this.CompFourOfAKind(pk)
	if len(compList) > 0 {
		score += this.BaseScore * 14
	}
	return score
}
func (this *MyTable) JieSuan() {
	this.CountDown = this.GameConf.JieSuanTime
	this.SwitchRoomState()

	baseScore := this.BaseScore
	winAllIdx := -1
	if this.StraightScoreData.Type > 0 { //报到
		baseScore = 26 * this.BaseScore
		winAllIdx = this.StraightScoreData.Idx
		this.CountDown += 3
		this.LastData.LastFirstIdx = -1
	} else {
		this.LastData.LastFirstIdx = this.PutOverRecord[0].Idx
	}
	lastBlack3 := false
	if winAllIdx >= 0 { //通杀
		for k, v := range this.PlayerList {
			if v.Ready && k != winAllIdx {
				remainPokerScore := this.CalcRemainPokerScore(v.HandPoker)
				this.PlayerList[k].FinalScore -= baseScore + remainPokerScore
				if v.Role == ROBOT {
					if v.Yxb < -this.PlayerList[k].FinalScore {
						this.PlayerList[k].FinalScore = -v.Yxb
					}
				} else {
					wallet := walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[k].UserID))
					if wallet.VndBalance < -this.PlayerList[k].FinalScore {
						this.PlayerList[k].FinalScore = -wallet.VndBalance
					}
				}

				this.PlayerList[winAllIdx].FinalScore -= this.PlayerList[k].FinalScore
			}
		}
	} else {
		//计算被压得分
		for _, v := range this.PokerPressList {
			this.PlayerList[v.PressIdx].PressScore -= v.Score
			this.PlayerList[v.WinIdx].PressScore += v.Score
		}

		////黑桃3得分
		if len(this.PutOverRecord[0].LastPutCard) == 1 && this.PutOverRecord[0].LastPutCard[0] == 0x13 {
			lastBlack3 = true
		}
		//计算得分
		for k, v := range this.PlayerList {
			if v.Ready && k != this.PutOverRecord[0].Idx {
				remainPokerScore := this.CalcRemainPokerScore(v.HandPoker)
				if len(v.HandPoker) == 1 && v.HandPoker[0] == 0x13 { //最后黑桃3被捉
					this.PlayerList[k].FinalScore -= 13*this.BaseScore + remainPokerScore
				} else {
					this.PlayerList[k].FinalScore -= int64(len(v.HandPoker))*this.BaseScore + remainPokerScore
					if lastBlack3 {
						this.PlayerList[k].FinalScore -= int64(len(v.HandPoker)) * this.BaseScore
					}
					if len(v.HandPoker) == 13 {
						this.PlayerList[k].FinalScore -= int64(len(v.HandPoker)) * this.BaseScore
					}
				}

				if v.Role == ROBOT {
					if v.Yxb < -this.PlayerList[k].FinalScore {
						this.PlayerList[k].FinalScore = -v.Yxb
					}
				} else {
					wallet := walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[k].UserID))
					if wallet.VndBalance < -this.PlayerList[k].FinalScore {
						this.PlayerList[k].FinalScore = -wallet.VndBalance
					}
				}
				this.PlayerList[this.PutOverRecord[0].Idx].FinalScore -= this.PlayerList[k].FinalScore
			}
		}
	}

	this.JieSuanData.RoomState = this.RoomState
	this.JieSuanData.CountDown = this.CountDown
	this.JieSuanData.LastPutCard = this.LastData.LastPutCard
	this.JieSuanData.LastPutIdx = this.LastData.LastPutIdx
	this.JieSuanData.PutOverRecord = this.PutOverRecord
	this.JieSuanData.LastBlack3 = lastBlack3
	realTotalCommission := int64(0)
	for k, v := range this.PlayerList {
		if v.Ready {
			isSpring := false
			if (winAllIdx >= 0 && k != winAllIdx) || len(v.HandPoker) == 13 {
				isSpring = true
			}
			black3 := false
			if len(v.HandPoker) == 1 && v.HandPoker[0] == 0x13 {
				black3 = true
			}
			this.JieSuanData.PlayerInfo[k] = PlayerInfo{
				StraightType: v.StraightType,
				HandPoker:    v.HandPoker,
				IsSpring:     isSpring,
				LastBlack3:   black3,
			}
			this.PlayerList[k].FinalScore += this.PlayerList[k].PressScore
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
		Poker   []int
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
			handPk := make([]int, 0)
			for _, v1 := range v.OriginPoker {
				handPk = append(handPk, switchBackendCard[v1])
			}
			backendResults = append(backendResults, ResultData{
				Account: v.Account,
				IP:      ip,
				Idx:     k,
				Self:    false,
				Poker:   handPk,
				Income:  v.TotalBackYxb,
			})
			gameStorage.IncGameWinLoseScore(game.CardCddN, v.Name, v.TotalBackYxb)
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
				lobbyStorage.Win(utils.ConvertOID(v.UserID), v.Name, v.TotalBackYxb, game.CardCddN, false)
			}

			if v.TotalBackYxb > 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeIncome, walletStorage.EventGameCardCddN, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER {
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), v.TotalBackYxb, this.EventID, game.CardCddN)
				}
			} else if v.TotalBackYxb < 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeExpenses, walletStorage.EventGameCardCddN, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER {
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), -v.TotalBackYxb, this.EventID, game.CardCddN)
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
				GameType:   game.CardCddN,
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
				betRecordParam.SysProfit = this.PlayerList[k].BotProfit
			} else {
				betRecordParam.SysProfit = 0
			}
			gameStorage.InsertBetRecord(betRecordParam)

		}

	}
	if this.GetRealReadyPlayerNum() > 0 {
		gameStorage.IncProfit("", game.CardCddN, realTotalCommission, realTotalPay-botProfit, botProfit)
	}
	//go func() {
	//	time.Sleep(time.Second * 5)
	//	for _,v := range this.PlayerList{
	//		if v.Role == USER || v.Role == Agent{
	//			this.notifyWallet(v.UserID)
	//		}
	//	}
	//}()

	reboot := gameStorage.QueryGameReboot(game.CardCddN)
	if reboot == "true" {
		this.RoomState = ROOM_END
	}
	if this.GetTableRealPlayerNum() <= 0 && !this.AutoCreate {
		this.RoomState = ROOM_END
	}
	for _, v := range this.PlayerList {
		if v.Ready && v.Role != ROBOT {
			activityStorage.UpsertGameDataInBet(v.UserID, game.CardCddN, 0)
			activity.CalcEncouragementFunc(v.UserID)
		}
	}
	//	res,_ := json.Marshal(this.JieSuanData)
	//	log.Info("---------------jie suan data----------------------%s",res)
	this.SeqExecFlag = true

}
