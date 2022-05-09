package cardCddS

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

func (this *MyTable) CalcRemainPokerScore(pk []int) int64{
	score := int64(0)
	for _,v := range pk{
		if v == 0x3f || v == 0x4f{
			score += this.BaseScore
		}else if v == 0x1f || v == 0x2f{
			score += this.BaseScore / 2
		}
	}
	compList := this.CompStraightPair(pk)
	haveStraightPair4 := false
	haveStraightPair3 := false
	for _,v := range compList{
		if v.num == 4{
			haveStraightPair4 = true
			break
		}else if v.num == 3{
			haveStraightPair3 = true
		}
	}
	if haveStraightPair4{
		score += this.BaseScore * 3
	}else if haveStraightPair3{
		score += this.BaseScore * 3 / 2
	}

	compList = this.CompFourOfAKind(pk)
	if len(compList) > 0{
		score += this.BaseScore * 3
	}
	return score
}
func (this *MyTable) JieSuan(){
	this.CountDown = this.GameConf.JieSuanTime
	this.SwitchRoomState()

	baseScore := this.BaseScore
	winAllIdx := -1
	if this.StraightScoreData.Type > 0{ //报到
		baseScore = 2 * this.BaseScore
		winAllIdx = this.StraightScoreData.Idx
		this.CountDown += 3
		this.LastFirstIdx = -1
	}else if this.PutOverRecord[0].IsSpring{
		baseScore = 2 * this.BaseScore
		winAllIdx = this.PutOverRecord[0].Idx
		this.LastFirstIdx = this.PutOverRecord[0].Idx
	}else{
		this.LastFirstIdx = this.PutOverRecord[0].Idx
	}

	haveSpring := false
	for _,v := range this.SpringList{
		if v{
			haveSpring = true
			break
		}
	}

	if winAllIdx >= 0{ //通杀
		for k,v := range this.PlayerList{
			if v.Ready && k != winAllIdx{
				remainPokerScore := this.CalcRemainPokerScore(v.HandPoker)
				this.PlayerList[k].FinalScore -= baseScore + remainPokerScore
				if v.Role == ROBOT{
					if v.Yxb < -this.PlayerList[k].FinalScore{
						this.PlayerList[k].FinalScore = -v.Yxb
					}
				}else{
					wallet := walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[k].UserID))
					if wallet.VndBalance < -this.PlayerList[k].FinalScore{
						this.PlayerList[k].FinalScore = -wallet.VndBalance
					}
				}

				this.PlayerList[winAllIdx].FinalScore -= this.PlayerList[k].FinalScore
			}
		}
	}else if haveSpring{
		//计算被压得分
		for _,v := range this.PokerPressList{
			this.PlayerList[v.PressIdx].PressScore -= v.Score
			this.PlayerList[v.WinIdx].PressScore += v.Score
		}
		//计算排名得分
		if len(this.PutOverRecord) == 3{
			for k,v := range this.SpringList{
				if v{
					this.PlayerList[k].FinalScore -= 2 * this.BaseScore

					vndBalance := this.PlayerList[k].PressScore
					if this.PlayerList[k].Role == ROBOT{
						vndBalance += this.PlayerList[k].Yxb
					}else{
						vndBalance += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[k].UserID)).VndBalance
					}
					if vndBalance < -this.PlayerList[k].FinalScore{
						this.PlayerList[k].FinalScore = -vndBalance
						this.PlayerList[this.PutOverRecord[0].Idx].FinalScore -= this.PlayerList[k].FinalScore
					}else{
						this.PlayerList[this.PutOverRecord[0].Idx].FinalScore += 2 * this.BaseScore
					}

					remainPokerScore := this.CalcRemainPokerScore(this.PlayerList[k].HandPoker)
					if vndBalance + this.PlayerList[k].FinalScore < 0{
						remainPokerScore = 0
					}else if vndBalance + this.PlayerList[k].FinalScore < remainPokerScore{
						remainPokerScore = vndBalance + this.PlayerList[k].FinalScore
					}
					this.PlayerList[this.PutOverRecord[0].Idx].FinalScore += remainPokerScore
					this.PlayerList[k].FinalScore -= remainPokerScore
				}
			}

			this.PlayerList[this.PutOverRecord[2].Idx].FinalScore -= this.BaseScore
			vndBalance2 := this.PlayerList[this.PutOverRecord[2].Idx].PressScore
			if this.PlayerList[this.PutOverRecord[2].Idx].Role == ROBOT{
				vndBalance2 += this.PlayerList[this.PutOverRecord[2].Idx].Yxb
			}else{
				vndBalance2 += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[this.PutOverRecord[2].Idx].UserID)).VndBalance
			}
			if vndBalance2 < -this.PlayerList[this.PutOverRecord[2].Idx].FinalScore{
				this.PlayerList[this.PutOverRecord[2].Idx].FinalScore = -vndBalance2
				this.PlayerList[this.PutOverRecord[1].Idx].FinalScore -= this.PlayerList[this.PutOverRecord[2].Idx].FinalScore
			}else{
				this.PlayerList[this.PutOverRecord[1].Idx].FinalScore += this.BaseScore
			}
			remainPokerScore := this.CalcRemainPokerScore(this.PlayerList[this.PutOverRecord[2].Idx].HandPoker)
			if vndBalance2 + this.PlayerList[this.PutOverRecord[2].Idx].FinalScore < 0{
				remainPokerScore = 0
			}else if vndBalance2 + this.PlayerList[this.PutOverRecord[2].Idx].FinalScore < remainPokerScore{
				remainPokerScore = vndBalance2 + this.PlayerList[this.PutOverRecord[2].Idx].FinalScore
			}
			this.PlayerList[this.PutOverRecord[1].Idx].FinalScore += remainPokerScore
			this.PlayerList[this.PutOverRecord[2].Idx].FinalScore -= remainPokerScore
		}else if len(this.PutOverRecord) == 2{
			for k,v := range this.SpringList{
				if v{
					this.PlayerList[k].FinalScore -= 2 * this.BaseScore

					vndBalance := this.PlayerList[k].PressScore
					if this.PlayerList[k].Role == ROBOT{
						vndBalance += this.PlayerList[k].Yxb
					}else{
						vndBalance += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[k].UserID)).VndBalance
					}
					if vndBalance < -this.PlayerList[k].FinalScore{
						this.PlayerList[k].FinalScore = -vndBalance
						this.PlayerList[this.PutOverRecord[0].Idx].FinalScore -= this.PlayerList[k].FinalScore
					}else{
						this.PlayerList[this.PutOverRecord[0].Idx].FinalScore += 2 * this.BaseScore
					}

					remainPokerScore := this.CalcRemainPokerScore(this.PlayerList[k].HandPoker)
					if vndBalance + this.PlayerList[k].FinalScore < 0{
						remainPokerScore = 0
					}else if vndBalance + this.PlayerList[k].FinalScore < remainPokerScore{
						remainPokerScore = vndBalance + this.PlayerList[k].FinalScore
					}
					this.PlayerList[this.PutOverRecord[0].Idx].FinalScore += remainPokerScore
					this.PlayerList[k].FinalScore -= remainPokerScore
				}
			}
			this.PlayerList[this.PutOverRecord[1].Idx].FinalScore -= this.BaseScore
			vndBalance1 := this.PlayerList[this.PutOverRecord[1].Idx].PressScore
			if this.PlayerList[this.PutOverRecord[1].Idx].Role == ROBOT{
				vndBalance1 += this.PlayerList[this.PutOverRecord[1].Idx].Yxb
			}else{
				vndBalance1 += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[this.PutOverRecord[1].Idx].UserID)).VndBalance
			}
			if vndBalance1 < -this.PlayerList[this.PutOverRecord[1].Idx].FinalScore{
				this.PlayerList[this.PutOverRecord[1].Idx].FinalScore = -vndBalance1
				this.PlayerList[this.PutOverRecord[0].Idx].FinalScore -= this.PlayerList[this.PutOverRecord[1].Idx].FinalScore
			}else{
				this.PlayerList[this.PutOverRecord[0].Idx].FinalScore += this.BaseScore
			}
		}
	} else{
		//计算被压得分
		for _,v := range this.PokerPressList{
			this.PlayerList[v.PressIdx].PressScore -= v.Score
			this.PlayerList[v.WinIdx].PressScore += v.Score
		}

		//计算排名得分
		if this.PlayingNum == 4{
			this.PlayerList[this.PutOverRecord[3].Idx].FinalScore -= this.BaseScore

			vndBalance3 := this.PlayerList[this.PutOverRecord[3].Idx].PressScore
			if this.PlayerList[this.PutOverRecord[3].Idx].Role == ROBOT{
				vndBalance3 += this.PlayerList[this.PutOverRecord[3].Idx].Yxb
			}else{
				vndBalance3 += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[this.PutOverRecord[3].Idx].UserID)).VndBalance
			}
			if vndBalance3 < -this.PlayerList[this.PutOverRecord[3].Idx].FinalScore{
				this.PlayerList[this.PutOverRecord[3].Idx].FinalScore = -vndBalance3
				this.PlayerList[this.PutOverRecord[0].Idx].FinalScore -= this.PlayerList[this.PutOverRecord[3].Idx].FinalScore
			}else{
				this.PlayerList[this.PutOverRecord[0].Idx].FinalScore += this.BaseScore
			}

			remainPokerScore := this.CalcRemainPokerScore(this.PlayerList[this.PutOverRecord[3].Idx].HandPoker)
			if vndBalance3 + this.PlayerList[this.PutOverRecord[3].Idx].FinalScore < 0{
				remainPokerScore = 0
			}else if vndBalance3 + this.PlayerList[this.PutOverRecord[3].Idx].FinalScore < remainPokerScore{
				remainPokerScore = vndBalance3 + this.PlayerList[this.PutOverRecord[3].Idx].FinalScore
			}
			this.PlayerList[this.PutOverRecord[2].Idx].FinalScore += remainPokerScore
			this.PlayerList[this.PutOverRecord[3].Idx].FinalScore -= remainPokerScore


			this.PlayerList[this.PutOverRecord[2].Idx].FinalScore -= this.BaseScore / 2
			vndBalance2 := this.PlayerList[this.PutOverRecord[2].Idx].PressScore
			if this.PlayerList[this.PutOverRecord[2].Idx].Role == ROBOT{
				vndBalance2 += this.PlayerList[this.PutOverRecord[2].Idx].Yxb
			}else{
				vndBalance2 += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[this.PutOverRecord[2].Idx].UserID)).VndBalance
			}
			if vndBalance2 < -this.PlayerList[this.PutOverRecord[2].Idx].FinalScore{
				this.PlayerList[this.PutOverRecord[2].Idx].FinalScore = -vndBalance2
				this.PlayerList[this.PutOverRecord[1].Idx].FinalScore -= this.PlayerList[this.PutOverRecord[2].Idx].FinalScore
			}else{
				this.PlayerList[this.PutOverRecord[1].Idx].FinalScore += this.BaseScore / 2
			}

		}else if this.PlayingNum == 3{
			this.PlayerList[this.PutOverRecord[2].Idx].FinalScore -= this.BaseScore
			this.PlayerList[this.PutOverRecord[1].Idx].FinalScore -= this.BaseScore / 2
			remainPokerScore := this.CalcRemainPokerScore(this.PlayerList[this.PutOverRecord[2].Idx].HandPoker)
			this.PlayerList[this.PutOverRecord[2].Idx].FinalScore -= remainPokerScore
			this.PlayerList[this.PutOverRecord[1].Idx].FinalScore += remainPokerScore
			////黑桃3得分
			//if (len(this.PutOverRecord[0].LastPutCard) == 1 && this.PutOverRecord[0].LastPutCard[0] == 0x13) ||
			//	(len(this.PlayerList[2].HandPoker) == 0 && this.PlayerList[2].HandPoker[0] == 0x13){
			//	this.PlayerList[2].FinalScore -= this.BaseScore
			//}

			vndBalance1 := this.PlayerList[this.PutOverRecord[1].Idx].PressScore
			if this.PlayerList[this.PutOverRecord[1].Idx].Role == ROBOT{
				vndBalance1 += this.PlayerList[this.PutOverRecord[1].Idx].Yxb
			}else{
				vndBalance1 += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[this.PutOverRecord[1].Idx].UserID)).VndBalance
			}
			if vndBalance1 < -this.PlayerList[this.PutOverRecord[1].Idx].FinalScore{
				this.PlayerList[this.PutOverRecord[1].Idx].FinalScore = -vndBalance1
				this.PlayerList[this.PutOverRecord[0].Idx].FinalScore -= this.PlayerList[this.PutOverRecord[1].Idx].FinalScore
			}else{
				this.PlayerList[this.PutOverRecord[0].Idx].FinalScore += this.BaseScore
			}
			vndBalance2 := this.PlayerList[this.PutOverRecord[2].Idx].PressScore
			if this.PlayerList[this.PutOverRecord[2].Idx].Role == ROBOT{
				vndBalance2 += this.PlayerList[this.PutOverRecord[2].Idx].Yxb
			}else{
				vndBalance2 += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[this.PutOverRecord[2].Idx].UserID)).VndBalance
			}
			if vndBalance2 < -this.PlayerList[this.PutOverRecord[2].Idx].FinalScore{
				this.PlayerList[this.PutOverRecord[2].Idx].FinalScore = -vndBalance2
				this.PlayerList[this.PutOverRecord[0].Idx].FinalScore -= this.PlayerList[this.PutOverRecord[2].Idx].FinalScore
			}else{
				this.PlayerList[this.PutOverRecord[0].Idx].FinalScore += this.BaseScore / 2
			}

		}else if this.PlayingNum == 2{
			this.PlayerList[this.PutOverRecord[1].Idx].FinalScore -= this.BaseScore
			remainPokerScore := this.CalcRemainPokerScore(this.PlayerList[this.PutOverRecord[1].Idx].HandPoker)
			this.PlayerList[this.PutOverRecord[1].Idx].FinalScore -= remainPokerScore

			////黑桃3得分
			//if (len(this.PutOverRecord[0].LastPutCard) == 1 && this.PutOverRecord[0].LastPutCard[0] == 0x13) ||
			//	(len(this.PlayerList[1].HandPoker) == 0 && this.PlayerList[1].HandPoker[0] == 0x13){
			//	this.PlayerList[1].FinalScore -= this.BaseScore
			//}

			vndBalance1 := this.PlayerList[this.PutOverRecord[1].Idx].PressScore
			if this.PlayerList[this.PutOverRecord[1].Idx].Role == ROBOT{
				vndBalance1 += this.PlayerList[this.PutOverRecord[1].Idx].Yxb
			}else{
				vndBalance1 += walletStorage.QueryWallet(utils.ConvertOID(this.PlayerList[this.PutOverRecord[1].Idx].UserID)).VndBalance
			}
			if vndBalance1 < -this.PlayerList[this.PutOverRecord[1].Idx].FinalScore{
				this.PlayerList[this.PutOverRecord[1].Idx].FinalScore = -vndBalance1
			}

			this.PlayerList[this.PutOverRecord[0].Idx].FinalScore -= this.PlayerList[this.PutOverRecord[1].Idx].FinalScore
		}


	}

	this.JieSuanData.RoomState = this.RoomState
	this.JieSuanData.CountDown = this.CountDown
	this.JieSuanData.LastPutCard = this.LastPutCard
	this.JieSuanData.LastPutIdx = this.LastPutIdx
	this.JieSuanData.PutOverRecord = this.PutOverRecord
	for k,v := range this.PlayerList{
		if v.Ready{
			isSpring := false
			if (winAllIdx >= 0 && k != winAllIdx) || this.SpringList[k]{
				isSpring =true
			}
			this.JieSuanData.PlayerInfo[k] = PlayerInfo{
				StraightType: v.StraightType,
				HandPoker: v.HandPoker,
				IsSpring: isSpring,
			}
			this.PlayerList[k].FinalScore += this.PlayerList[k].PressScore
			this.PlayerList[k].TotalBackYxb = this.PlayerList[k].FinalScore
			if this.PlayerList[k].TotalBackYxb > 0{
				this.PlayerList[k].SysProfit = this.PlayerList[k].TotalBackYxb * int64(this.GameConf.ProfitPerThousand) / 1000
				this.PlayerList[k].TotalBackYxb -= this.PlayerList[k].SysProfit
			}
		}
	}

	for k,v := range this.PlayerList{
		if v.Ready{
			this.PlayerList[k].Yxb += v.TotalBackYxb

			tmp := this.JieSuanData.PlayerInfo[k]
			tmp.TotalBackYxb = v.TotalBackYxb
			this.JieSuanData.PlayerInfo[k] = tmp
		}
	}
	type ResultData struct {
		Account string
		IP string
		Idx int
		Poker []int
		Self bool
		Income int64
	}
	backendResults := make([]ResultData,0)
	for k,v := range this.PlayerList{
		if v.Ready && v.IsHavePeople{
			ip := ""
			if v.Role != ROBOT{
				login := userStorage.QueryLogin(utils.ConvertOID(v.UserID))
				if login != nil{
					ip = login.LastIp
				}else{
					ip = "UnKnow"
				}
			}else{
				ip = "0.0.0.0"
			}
			handPk := make([]int,0)
			for _,v1 := range v.OriginPoker{
				handPk = append(handPk,switchBackendCard[v1])
			}
			backendResults = append(backendResults,ResultData{
				Account: v.Account,
				IP: ip,
				Idx: k,
				Self: false,
				Poker: handPk,
				Income: v.TotalBackYxb,
			})
			gameStorage.IncGameWinLoseScore(game.CardCddS,v.Name,v.TotalBackYxb)
		}
	}
	for k,v := range this.PlayerList{
		if v.Ready && v.Role != ROBOT{
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil{
				session,_ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(),game.Push,this.JieSuanData,protocol.JieSuan,nil)
			}

			if v.TotalBackYxb > 0 && v.Role == USER{
				lobbyStorage.Win(utils.ConvertOID(v.UserID),v.Name, v.TotalBackYxb,game.CardCddS,false)
			}

			if v.TotalBackYxb > 0{
				bill := walletStorage.NewBill(v.UserID,walletStorage.TypeIncome,walletStorage.EventGameCardCddS,this.EventID,v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER{
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID),v.TotalBackYxb,this.EventID,game.CardCddS)
				}
			}else if v.TotalBackYxb < 0{
				bill := walletStorage.NewBill(v.UserID,walletStorage.TypeExpenses,walletStorage.EventGameCardCddS,this.EventID,v.TotalBackYxb)
				walletStorage.OperateVndBalance(bill)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				this.PlayerList[k].Yxb = wallet.VndBalance
				if v.Role == USER{
					pay.CheckoutAgentIncome(utils.ConvertOID(v.UserID),-v.TotalBackYxb,this.EventID,game.CardCddS)
				}
			}
			backendResultsCopy := make([]ResultData,len(backendResults))
			copy(backendResultsCopy,backendResults)
			for k1,v1 := range backendResultsCopy{
				if v1.Idx == k{
					backendResultsCopy[k1].Self = true
					break
				}
			}

			resultStr,_ := json.Marshal(backendResultsCopy)
			wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
			betRecordParam := gameStorage.BetRecordParam{
				Uid: v.UserID,
				GameType: game.CardCddS,
				Income: v.TotalBackYxb ,
				BetAmount: this.BaseScore,
				CurBalance: this.PlayerList[k].Yxb + wallet.SafeBalance,
				SysProfit: this.PlayerList[k].SysProfit,
				BotProfit: 0,
				BetDetails: "",
				GameId: strconv.FormatInt(time.Now().Unix(),10),
				GameNo: strconv.FormatInt(time.Now().Unix(),10),
				GameResult: string(resultStr),
				IsSettled: true,
			}
			if v.Role != USER{
				betRecordParam.SysProfit = 0
			}
			gameStorage.InsertBetRecord(betRecordParam)

		}

	}
	//go func() {
	//	time.Sleep(time.Second * 5)
	//	for _,v := range this.PlayerList{
	//		if v.Role == USER || v.Role == Agent{
	//			this.notifyWallet(v.UserID)
	//		}
	//	}
	//}()

	reboot := gameStorage.QueryGameReboot(game.CardCddS)
	if reboot == "true"{
		this.RoomState = ROOM_END
	}
	if this.GetTableRealPlayerNum() <= 0 && !this.AutoCreate{
		this.RoomState = ROOM_END
	}

	for _,v := range this.PlayerList {
		if v.Ready && v.Role != ROBOT {
			activityStorage.UpsertGameDataInBet(v.UserID,game.CardCddS,0)
			activity.CalcEncouragementFunc(v.UserID)
		}
	}
//	res,_ := json.Marshal(this.JieSuanData)
//	log.Info("---------------jie suan data----------------------%s",res)
	this.SeqExecFlag = true

}