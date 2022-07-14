package cardSss

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
	this.CountDown = this.GameConf.JieSuanTime
	this.SwitchRoomState()

	var calcIdx []int
	calcIdx = []int{}

	winTimes := map[int]int{}
	for k, v := range this.PlayerList {
		if v.Ready && v.StraightType <= 0 && !v.Oolong {
			calcIdx = append(calcIdx, k)
		}

		if v.Oolong && v.Ready {
			for k1, v1 := range this.PlayerList {
				if v1.Ready && !v1.Oolong && v1.StraightType <= 0 { //用来计算全垒打，有人乌龙也算全垒打
					winTimes[k1]++
				}
			}
		}
	}

	if len(calcIdx) > 1 {
		for i := 0; i < len(calcIdx); i++ {
			for j := i + 1; j < len(calcIdx); j++ {
				a := calcIdx[i]
				b := calcIdx[j]
				var win []int
				win = []int{}
				for k := 0; k < 3; k++ {
					max := 0
					min := 0

					if this.PlayerList[a].PokerVal[k] > this.PlayerList[b].PokerVal[k] {
						max = a
						min = b
					} else {
						max = b
						min = a
					}
					win = append(win, max)
					winTimes[max] += 1

					score := int64(1)
					if k == 0 {
						if this.PlayerList[max].PokerType[k] == ThreeOfAKind {
							score = 6
						}
					} else if k == 1 {
						if this.PlayerList[max].PokerType[k] == FullHouse {
							score = 4
						} else if this.PlayerList[max].PokerType[k] == FourOfAKind {
							score = 16
						} else if this.PlayerList[max].PokerType[k] == StraightFlush || this.PlayerList[max].PokerType[k] == BigStraightFlush {
							score = 20
						}
					} else if k == 2 {
						if this.PlayerList[max].PokerType[k] == FourOfAKind {
							score = 8
						} else if this.PlayerList[max].PokerType[k] == StraightFlush || this.PlayerList[max].PokerType[k] == BigStraightFlush {
							score = 10
						}
					}
					this.PlayerList[max].ResultScore[k] += score
					this.PlayerList[min].ResultScore[k] -= score

					this.PlayerList[max].FinalScore += score
					this.PlayerList[min].FinalScore -= score
				}
				if win[0] == win[1] && win[0] == win[2] { //打枪
					if win[0] == a {
						this.ShooterList = append(this.ShooterList, a)
						this.ShotList = append(this.ShotList, b)

						this.PlayerList[a].ShotScore += 6
						this.PlayerList[b].ShotScore -= 6

						this.PlayerList[a].FinalScore += 6
						this.PlayerList[b].FinalScore -= 6
					} else {
						this.ShooterList = append(this.ShooterList, b)
						this.ShotList = append(this.ShotList, a)

						this.PlayerList[b].ShotScore += 6
						this.PlayerList[a].ShotScore -= 6

						this.PlayerList[b].FinalScore += 6
						this.PlayerList[a].FinalScore -= 6
					}
				}

			}
		}
		for _, v := range calcIdx { //计算全垒打
			if this.GetReadyPlayerNum() == 4 {
				if winTimes[v] >= 3*3 {
					this.HomeRun = v
				}
			} else if this.GetReadyPlayerNum() == 3 {
				if winTimes[v] >= 2*3 {
					this.HomeRun = v
				}
			}

			if this.HomeRun >= 0 {
				for _, v1 := range calcIdx {
					if v != v1 {
						this.PlayerList[v].HomeRunScore += 6
						this.PlayerList[v1].HomeRunScore -= 6

						this.PlayerList[v].FinalScore += 6
						this.PlayerList[v1].FinalScore -= 6
					}
				}
				break
			}
		}
		//	this.JieSuanData.NotComp = false
	} else {
		//	this.JieSuanData.NotComp = true
		this.CountDown = 8
	}

	//乌龙得分
	for k, v := range this.PlayerList {
		if v.Oolong && v.Ready { //
			for k1, v1 := range this.PlayerList {
				if v1.Ready && v1.StraightType <= 0 && !v1.Oolong { //
					this.PlayerList[k1].FinalScore += 6
					this.PlayerList[k].FinalScore -= 6
				}
			}
		}
	}
	//直接得分
	for k, v := range this.PlayerList {
		if v.StraightType > 0 && v.Ready { //直接得分
			for k1, v1 := range this.PlayerList {
				if v1.Ready && k != k1 { //
					this.PlayerList[k].FinalScore += this.StraightScore[v.StraightType]
					this.PlayerList[k1].FinalScore -= this.StraightScore[v.StraightType]
				}
			}
		}
	}

	this.ShooterList = this.SliceRemoveDuplicates(this.ShooterList)
	this.ShotList = this.SliceRemoveDuplicates(this.ShotList)

	if len(this.ShotList) > 0 {
		this.CountDown += 3
	}
	if this.HomeRun >= 0 {
		this.CountDown += 3
	}
	this.JieSuanData.RoomState = this.RoomState
	this.JieSuanData.CountDown = this.CountDown
	this.JieSuanData.HomeRun = this.HomeRun
	this.JieSuanData.ShooterList = this.ShooterList
	this.JieSuanData.ShotList = this.ShotList
	totalLoseYxb := int64(0)
	for k, v := range this.PlayerList {
		if v.Ready {
			this.JieSuanData.PlayerInfo[k] = PlayerInfo{
				StraightType: v.StraightType,
				Poker:        v.HandPoker,
				PokerType:    v.PokerType,
				Oolong:       v.Oolong,
				ResultScore:  v.ResultScore,
				FinalScore:   v.FinalScore,
				ShotScore:    v.ShotScore,
				HomeRunScore: v.HomeRunScore,
			}
			this.PlayerList[k].TotalBackYxb = v.FinalScore * this.BaseScore
			if this.PlayerList[k].TotalBackYxb < 0 {
				if v.Role == ROBOT {
					if v.Yxb < -this.PlayerList[k].TotalBackYxb {
						this.PlayerList[k].TotalBackYxb = -v.Yxb
					}
				} else {
					wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
					if wallet.VndBalance < -this.PlayerList[k].TotalBackYxb {
						this.PlayerList[k].TotalBackYxb = -wallet.VndBalance
					}
				}
				totalLoseYxb -= this.PlayerList[k].TotalBackYxb
			}
		}
	}

	pl := make([]PlayerList, 0)
	copy(pl, this.PlayerList)
	sort.Slice(pl, func(i, j int) bool { //降序排序
		return pl[i].TotalBackYxb > pl[j].TotalBackYxb
	})

	for _, v := range pl {
		if v.TotalBackYxb > 0 {
			if v.Ready && totalLoseYxb <= v.TotalBackYxb {
				idx := this.GetPlayerIdx(v.UserID)
				if idx >= 0 {
					this.PlayerList[idx].TotalBackYxb = totalLoseYxb
					break
				}
			} else {
				totalLoseYxb -= v.TotalBackYxb
			}
		}
	}

	realTotalCommission := int64(0)
	//系统抽成
	for k, v := range this.PlayerList {
		if v.Ready {
			if this.PlayerList[k].TotalBackYxb > 0 {
				this.PlayerList[k].SysProfit = this.PlayerList[k].TotalBackYxb * int64(this.GameConf.ProfitPerThousand) / 1000
				this.PlayerList[k].TotalBackYxb -= this.PlayerList[k].SysProfit
				if v.Role == USER {
					realTotalCommission += this.PlayerList[k].SysProfit
				}
			}
			this.PlayerList[k].Yxb += this.PlayerList[k].TotalBackYxb

			tmp := this.JieSuanData.PlayerInfo[k]
			tmp.TotalBackYxb = this.PlayerList[k].TotalBackYxb
			this.JieSuanData.PlayerInfo[k] = tmp
		}
	}
	realTotalPay := -realTotalCommission //赔付给真实玩家的总数 抽水需要去掉
	botProfit := int64(0)
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
			handPoker := make([]int, len(v.HandPoker))
			for k1, v1 := range v.HandPoker {
				handPoker[k1] = switchBackendCard[v1]
			}
			backendResults = append(backendResults, ResultData{
				Account: v.Account,
				IP:      ip,
				Idx:     k,
				Poker:   handPoker,
				Self:    false,
				Income:  v.TotalBackYxb,
			})

			gameStorage.IncGameWinLoseScore(game.CardSss, v.Name, v.TotalBackYxb)
		}
	}
	for k, v := range this.PlayerList {
		if v.Ready && v.Role != ROBOT {
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				session, _ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(), game.Push, this.JieSuanData, protocol.JieSuan, nil)
			}

			if v.TotalBackYxb > 0 && v.Role == USER {
				lobbyStorage.Win(utils.ConvertOID(v.UserID), v.Name, v.TotalBackYxb, game.CardSss, false)
			}

			if v.TotalBackYxb > 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeIncome, walletStorage.EventGameCardSss, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER {
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), v.TotalBackYxb, this.EventID, game.CardSss)
				}
			} else if v.TotalBackYxb < 0 {
				bill := walletStorage.NewBill(v.UserID, walletStorage.TypeExpenses, walletStorage.EventGameCardSss, this.EventID, v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER {
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID), -v.TotalBackYxb, this.EventID, game.CardSss)
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
				GameType:   game.CardSss,
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
				realTotalPay -= v.TotalBackYxb
				this.PlayerList[k].BotProfit = this.BaseScore * int64(this.GameConf.BotProfitPerThousand) / 1000
				botProfit += this.PlayerList[k].BotProfit
				betRecordParam.BotProfit = this.PlayerList[k].BotProfit
			} else {
				betRecordParam.BotProfit = 0
				betRecordParam.SysProfit = 0
			}
			gameStorage.InsertBetRecord(betRecordParam)
		}
	}
	if this.GetRealReadyPlayerNum() > 0 {
		gameStorage.IncProfit("", game.CardSss, realTotalCommission, realTotalPay-botProfit, botProfit)
	}
	//go func() {
	//	time.Sleep(time.Second * 5)
	//	for _,v := range this.PlayerList{
	//		if v.Role == USER || v.Role == Agent{
	//			this.notifyWallet(v.UserID)
	//		}
	//	}
	//}()

	reboot := gameStorage.QueryGameReboot(game.CardSss)
	if reboot == "true" {
		this.RoomState = ROOM_END
	}
	if this.GetTableRealPlayerNum() <= 0 && !this.AutoCreate {
		this.RoomState = ROOM_END
	}

	for _, v := range this.PlayerList {
		if v.Ready && v.Role != ROBOT {
			activityStorage.UpsertGameDataInBet(v.UserID, game.CardSss, 0)
			activity.CalcEncouragementFunc(v.UserID)
		}
	}
	//	res,_ := json.Marshal(this.JieSuanData)
	//	log.Info("---------------jie suan data----------------------%s",res)
	this.SeqExecFlag = true

}
