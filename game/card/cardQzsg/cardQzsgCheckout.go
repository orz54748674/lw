package cardQzsg

import (
	"encoding/json"
	"sort"
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
	this.DealPoker(this.Shuffle())
	this.CountDown = this.GameConf.JieSuanTime
	this.SwitchRoomState()

	if this.BaseScore > 100 && this.GetRealReadyPlayerNum() > 0 {
		this.ControlResults()
	}

	for k, v := range this.PlayerList { //先收取下注值
		if v.IsHavePeople && v.Ready && k != this.DealerIdx {
			this.PlayerList[k].FinalScore -= v.BetVal
			this.PlayerList[this.DealerIdx].FinalScore += v.BetVal
		}
	}
	for k, v := range this.PlayerList { //先把贏家本金退了
		if v.IsHavePeople && v.Ready && k != this.DealerIdx {
			if this.PlayerList[this.DealerIdx].PokerVal < v.PokerVal {
				this.PlayerList[k].FinalScore += v.BetVal
				this.PlayerList[this.DealerIdx].FinalScore -= v.BetVal
			}
		}
	}
	pl := make([]PlayerList, len(this.PlayerList))
	copy(pl, this.PlayerList)
	sort.Slice(pl, func(i, j int) bool { //降序排序
		return pl[i].PokerVal > pl[j].PokerVal
	})
	dealerVnd := this.PlayerList[this.DealerIdx].FinalScore
	if this.PlayerList[this.DealerIdx].Role == ROBOT {
		dealerVnd += this.PlayerList[this.DealerIdx].Yxb
	} else {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[this.DealerIdx].UserID))
		dealerVnd += wallet.VndBalance
	}
	for _, v := range pl {
		idx := this.GetPlayerIdx(v.UserID)
		if v.IsHavePeople && v.Ready && idx != this.DealerIdx {
			if this.PlayerList[this.DealerIdx].PokerVal < v.PokerVal {
				if dealerVnd <= v.BetVal {
					commission := dealerVnd * int64(this.GameConf.ProfitPerThousand) / 1000
					this.PlayerList[idx].FinalScore += dealerVnd - commission
					this.PlayerList[this.DealerIdx].FinalScore -= dealerVnd
					dealerVnd = 0
					break
				} else {
					commission := v.BetVal * int64(this.GameConf.ProfitPerThousand) / 1000
					this.PlayerList[idx].FinalScore += v.BetVal - commission
					this.PlayerList[this.DealerIdx].FinalScore -= v.BetVal
					dealerVnd -= v.BetVal
				}
			}
		}
	}
	realTotalCommission := int64(0)
	//系统抽成
	for k, v := range this.PlayerList {
		if v.Ready {
			if this.PlayerList[k].FinalScore > 0 {
				this.PlayerList[k].SysProfit = this.PlayerList[k].FinalScore * int64(this.GameConf.ProfitPerThousand) / 1000
				this.PlayerList[k].FinalScore -= this.PlayerList[k].SysProfit
				if v.Role == USER {
					realTotalCommission += this.PlayerList[k].SysProfit
				}
			}
		}
	}
	this.JieSuanData.RoomState = this.RoomState
	this.JieSuanData.CountDown = this.CountDown
	for k, v := range this.PlayerList {
		if v.Ready {
			this.PlayerList[k].TotalBackYxb = this.PlayerList[k].FinalScore
			this.JieSuanData.PlayerInfo[k] = PlayerInfo{
				HandPoker:    v.HandPoker,
				PokerType:    v.PokerType,
				TotalBackYxb: this.PlayerList[k].TotalBackYxb,
			}
			this.PlayerList[k].Yxb += v.TotalBackYxb
		}
	}

	type ResultData struct {
		Account string
		IP      string
		Idx     int
		Poker   []int
		Self    bool
		Income  int64
		Dealer  bool
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
			handPoker := make([]int, len(v.HandPoker))
			for k1, v1 := range v.HandPoker {
				handPoker[k1] = switchBackendCard[v1]
			}
			dealer := false
			if k == this.DealerIdx {
				dealer = true
			}
			backendResults = append(backendResults, ResultData{
				Account: v.Account,
				IP:      ip,
				Idx:     k,
				Poker:   handPoker,
				Self:    false,
				Income:  v.TotalBackYxb,
				Dealer:  dealer,
			})
			gameStorage.IncGameWinLoseScore(game.CardQzsg, v.Name, v.TotalBackYxb)
		}
	}
	realTotalPay := -realTotalCommission //赔付给真实玩家的总数 抽水需要去掉
	botProfit := int64(0)

	for k, v := range this.PlayerList {
		if v.Ready && v.Role != ROBOT {
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				session, _ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(), game.Push, this.JieSuanData, protocol.JieSuan, nil)
			}

			if v.TotalBackYxb > 0 {
				lobbyStorage.Win(utils.ConvertOID(v.UserID), v.Name, v.TotalBackYxb, game.CardQzsg, false)
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeIncome, walletStorage.EventGameCardQzsg, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER {
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), v.TotalBackYxb, this.EventID, game.CardQzsg)
				}
			} else if v.TotalBackYxb < 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeIncome, walletStorage.EventGameCardQzsg, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER {
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), -v.TotalBackYxb, this.EventID, game.CardQzsg)
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
				GameType:   game.CardQzsg,
				Income:     v.TotalBackYxb,
				BetAmount:  v.BetVal,
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
				realTotalPay -= v.TotalBackYxb
			} else {
				betRecordParam.SysProfit = 0
			}
			gameStorage.InsertBetRecord(betRecordParam)
		}

	}
	if this.PlayerList[this.DealerIdx].Role == USER {
		for k, v := range this.PlayerList {
			if v.Ready && v.IsHavePeople {
				this.PlayerList[k].BotProfit = v.BetVal * int64(this.GameConf.BotProfitPerThousand) / 1000
				botProfit += this.PlayerList[k].BotProfit
			}
		}
	} else {
		for k, v := range this.PlayerList {
			if v.Ready && v.IsHavePeople && v.Role == USER {
				this.PlayerList[k].BotProfit = v.BetVal * int64(this.GameConf.BotProfitPerThousand) / 1000
				botProfit += this.PlayerList[k].BotProfit
			}
		}
	}
	if this.GetRealReadyPlayerNum() > 0 {
		gameStorage.IncProfit("", game.CardQzsg, realTotalCommission, realTotalPay-botProfit, botProfit)
	}
	//go func() {
	//	time.Sleep(time.Second * 3)
	//	for _,v := range this.PlayerList{
	//		if v.Role == USER || v.Role == Agent{
	//			this.notifyWallet(v.UserID)
	//		}
	//	}
	//}()

	reboot := gameStorage.QueryGameReboot(game.CardQzsg)
	if reboot == "true" {
		this.RoomState = ROOM_END
	}
	if this.GetTableRealPlayerNum() <= 0 && !this.AutoCreate {
		this.RoomState = ROOM_END
	}

	for _, v := range this.PlayerList {
		if v.Ready && v.Role != ROBOT {
			activityStorage.UpsertGameDataInBet(v.UserID, game.CardQzsg, 0)
			activity.CalcEncouragementFunc(v.UserID)
		}
	}
	//res,_ := json.Marshal(this.JieSuanData)
	//log.Info("---------------jie suan data----------------------%s",res)
	this.SeqExecFlag = true

}
