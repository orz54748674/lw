package dx

import (
	"encoding/json"
	"fmt"
	"strconv"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/activity"
	"vn/gate"
	"vn/storage/activityStorage"
	"vn/storage/agentStorage"
	"vn/storage/dxStorage"
	"vn/storage/gameStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

type checkout struct {
	curDx               *dxStorage.Dx
	onlinePush          *gate.OnlinePush
	dxConf              *dxStorage.Conf
	//jackpotPersonAmount int64

}

func (s *checkout) checkOut() {
	//log.Info("start checkOut: %v", s.curDx)
	gameId := s.curDx.ShowId
	refundType := s.getNeedRefundType()
	allRefund := dxStorage.QueryAllRefund(gameId, s.curDx.BetBig, refundType)
	for _, dxBetLog := range *allRefund {
		s.refund(&dxBetLog)
		dxBetLog.HasCheckout = 1
		if dxBetLog.UserType == dxStorage.UserTypeNormal {
			log.Info("refund user: %v", dxBetLog.String())
		}
		dxStorage.UpdateDxBetLog(&dxBetLog)
	}
	//退款通知
	realBetLogs := dxStorage.QueryDxBetLogNeedNotify(s.curDx.ShowId)
	for _,cc := range realBetLogs{
		s.notifyUserRefund(cc.Uid,cc.Refund)
		notifyWallet(s.onlinePush,cc.Uid)
		//全退
		if cc.GetRealBet() == 0 && len(cc.UserType) >0 &&
			cc.UserType[0] == dxStorage.UserTypeNormal {
			wallet := walletStorage.QueryWallet(utils.ConvertOID(cc.Uid))
			betDetails := map[string]interface{}{
				"Big": 0,
				"Small": 0,
				"Refund": cc.Refund,
			}
			betDetailsStr,_ := json.Marshal(betDetails)
			gameResult := fmt.Sprintf("%d,%d,%d",s.curDx.Dice1,s.curDx.Dice2,s.curDx.Dice3)
			betRecordParam := gameStorage.BetRecordParam{
				Uid: cc.Uid,
				GameType: game.BiDaXiao,
				Income: 0,
				BetAmount: 0,
				CurBalance: wallet.VndBalance + wallet.SafeBalance,
				SysProfit: 0,
				BotProfit: 0,
				BetDetails: string(betDetailsStr),
				GameId: strconv.Itoa(int(s.curDx.ShowId)),
				GameNo: strconv.Itoa(int(s.curDx.ShowId)),
				GameResult: gameResult,
				IsSettled: true,
			}
			gameStorage.InsertBetRecord(betRecordParam)
		}
	}

	allDxBetLog := dxStorage.QueryAllBetLog(gameId)
	//s.parseJackpotAmount()
	//realBetLogs := dxStorage.QueryDxBetLogUnCheckOut(gameId)
	for _, dxBetLog := range *allDxBetLog {
		s.checkOutPerson(&dxBetLog) //每个人结算
		dxBetLog.HasCheckout = 1
		dxStorage.UpdateDxBetLog(&dxBetLog)
	}
	//s.checkoutUserIncomeData(realBetLogs,gameId)

	//s.notifyCheckout(gameId)
	s.checkoutProfit()
	s.checkoutJackpot()
	//log.Info("UpdateDx: %v", s.curDx)
	dxStorage.UpdateDx(s.curDx)


}
//func (s *checkout) checkoutUserIncomeData(realBetLogs []dxStorage.DxUserBet,gameId int64)  {
//	for _,cc := range realBetLogs{
//
//	}
//}
func (s *checkout)notifyCheckout(dx dxStorage.Dx) {
	realBetLogs := dxStorage.QueryDxBetLogNeedNotify(dx.ShowId)
	openResult := dx.Result
	for _,cc := range realBetLogs{
		if cc.GetRealBet() >0 && len(cc.UserType) >0 &&
			cc.UserType[0] == dxStorage.UserTypeNormal {

			wallet := walletStorage.QueryWallet(utils.ConvertOID(cc.Uid))
			botProfit := int64(s.dxConf.BotProfitPerThousand) * cc.GetRealBet() / 1000
			systemProfit := s.getSystemProfit(openResult,cc.Big,cc.Small,cc.Refund)
			win := cc.Result
			if dx.ResultJackpot == 1{
				jackpotLog := dxStorage.QueryJackpotDetails(cc.Uid,dx.ShowId)
				if jackpotLog != nil{
					win += jackpotLog.Amount
				}
			}
			betDetails := map[string]interface{}{
				"Big": cc.Big,
				"Small": cc.Small,
				"Refund": cc.Refund,
			}
			betDetailsStr,_ := json.Marshal(betDetails)
			gameResult := fmt.Sprintf("%d,%d,%d",dx.Dice1,dx.Dice2,dx.Dice3)
			betRecordParam := gameStorage.BetRecordParam{
				Uid: cc.Uid,
				GameType: game.BiDaXiao,
				Income: win,
				BetAmount: cc.GetRealBet(),
				CurBalance: wallet.VndBalance + wallet.SafeBalance,
				SysProfit: systemProfit,
				BotProfit: botProfit,
				BetDetails: string(betDetailsStr),
				GameId: strconv.Itoa(int(dx.ShowId)),
				GameNo: strconv.Itoa(int(dx.ShowId)),
				GameResult: gameResult,
				IsSettled: true,
			}
			gameStorage.InsertBetRecord(betRecordParam)
			activityStorage.UpsertGameDataInBet(cc.Uid,game.BiDaXiao,0)
			activity.CalcEncouragementFunc(cc.Uid)
		}
		if cc.GetRealBet() >0 {
			notifyAmount := cc.Result
			gameResult := s.curDx.Result
			if gameResult == dxStorage.ResultBig && cc.Big > 0 {
				notifyAmount += cc.Big - cc.Refund
			} else if gameResult == dxStorage.ResultSmall && cc.Small > 0 {
				notifyAmount += cc.Small - cc.Refund
			}
			s.notifyUserCheckout(cc.Uid,notifyAmount)
			notifyWallet(s.onlinePush, cc.Uid)
		}
	}
}
//func (s *checkout) parseJackpotAmount() {
//	if s.curDx.ResultJackpot == 1 {
//		if s.curDx.Result == dxStorage.ResultBig {
//			s.jackpotPersonAmount = s.curDx.Jackpot / s.curDx.BetBigCount
//		} else {
//			s.jackpotPersonAmount = s.curDx.Jackpot / s.curDx.BetSmallCount
//		}
//	} else {
//		s.jackpotPersonAmount = 0
//	}
//}
func (s *checkout) checkoutProfit() {
	conf := s.dxConf
	if s.curDx.RealBetSmall == 0 && s.curDx.RealBetBig == 0 {
		return
	}
	realBet := s.curDx.RealBetBig + s.curDx.RealBetSmall - s.curDx.RealRefundBig - s.curDx.RealRefundSmall
	if s.curDx.Result == dxStorage.ResultBig {
		s.curDx.BotProfit = realBet * int64(conf.BotProfitPerThousand) / 1000
	} else {
		s.curDx.BotProfit = realBet * int64(conf.BotProfitPerThousand) / 1000
	}
	var botAmount int64
	if s.curDx.ResultJackpot == 1 {
		jackpot := dxStorage.GetJackpot()
		var payJackpot int64 = 0
		if s.curDx.Result == dxStorage.ResultBig{
			payJackpot = jackpot.Amount * (s.curDx.RealBetBig-s.curDx.RealRefundBig)/s.curDx.BetBig
		}else{
			payJackpot = jackpot.Amount * (s.curDx.RealBetSmall-s.curDx.RealRefundSmall)/s.curDx.BetSmall
		}
		botAmount = s.curDx.SystemWin - s.curDx.BotProfit - payJackpot
	}else{
		botAmount = s.curDx.SystemWin - s.curDx.BotProfit
	}
	gameStorage.IncProfit("",game.BiDaXiao, s.curDx.SystemProfit,
		botAmount, s.curDx.BotProfit)
}
func (s *checkout) refund(dxBetLog *dxStorage.DxBetLog) {
	dxBetLog.Refund = dxBetLog.Small + dxBetLog.Big
	s.openWinBill(dxBetLog)
}
func (s *checkout) getNeedRefundType() string {
	refundType := "small"
	if s.curDx.RefundBig > 0 {
		refundType = "big"
	}
	return refundType
}
func (s *checkout) checkoutJackpot() {
	jackpot := dxStorage.GetJackpot()
	if s.curDx.ResultJackpot == 1 {
		jackpot.Amount = 0
		jackpot.RealAmount = 0
	}else{
		var intoJackpot int64
		var intoJackpotReal int64
		jackpotPerThousand := int64(s.dxConf.JackpotPerThousand)
		if s.curDx.Result == dxStorage.ResultSmall {
			intoJackpot = s.curDx.BetSmall * jackpotPerThousand / 1000
			intoJackpotReal = s.curDx.RealBetSmall * jackpotPerThousand / 1000
		} else {
			intoJackpot = s.curDx.BetBig * jackpotPerThousand / 1000
			intoJackpotReal = s.curDx.RealBetBig * jackpotPerThousand / 1000
		}
		jackpot.Amount += intoJackpot
		jackpot.RealAmount += intoJackpotReal
	}
	dxStorage.UpdateJackpot(jackpot)
}
func (s *checkout) getSystemProfit(gameResult uint8,big int64,small int64,refund int64) int64 {
	var systemProfit int64 = 0
	profitPerThousand := int64(s.dxConf.ProfitPerThousand)
	if gameResult == dxStorage.ResultBig && big > 0 {
		systemProfit = (big-refund) * profitPerThousand / 1000
	} else if gameResult == dxStorage.ResultSmall && small > 0 {
		systemProfit = (small-refund) * profitPerThousand / 1000
	} else{}
	return systemProfit
}
func (s *checkout) checkOutPerson(dxBetLog *dxStorage.DxBetLog) {
	gameResult := s.curDx.Result
	var systemProfit int64 = 0
	profitPerThousand := int64(s.dxConf.ProfitPerThousand)
	if gameResult == dxStorage.ResultBig && dxBetLog.Big > 0 {
		systemProfit = dxBetLog.Big * profitPerThousand / 1000
		dxBetLog.Result = game.Win*dxBetLog.Big - systemProfit
		s.openWinBill(dxBetLog)
	} else if gameResult == dxStorage.ResultSmall && dxBetLog.Small > 0 {
		systemProfit = dxBetLog.Small * profitPerThousand / 1000
		dxBetLog.Result = game.Win*dxBetLog.Small - systemProfit
		s.openWinBill(dxBetLog)
	} else {
		dxBetLog.Result = game.Lost * (dxBetLog.Big + dxBetLog.Small)
	}

	if dxBetLog.UserType == dxStorage.UserTypeNormal {
		s.curDx.SystemWin += -1 * dxBetLog.Result
		//s.curDx.AgentProfit // TODO 代理抽水计算
		s.curDx.SystemProfit += systemProfit
		uid := utils.ConvertOID(dxBetLog.Uid)
		user := userStorage.QueryUserId(utils.ConvertOID(dxBetLog.Uid))
		lobbyStorage.Win(uid,user.NickName, dxBetLog.Result,game.BiDaXiao,false)
	}
	gameStorage.IncGameWinLoseScore(game.BiDaXiao,dxBetLog.NickName,dxBetLog.Result)
	// jackpot 分奖
	if s.curDx.ResultJackpot == 1 {
		var amount int64
		if gameResult == dxStorage.ResultBig && dxBetLog.Big>0{
			amount = s.curDx.Jackpot * (dxBetLog.Big-dxBetLog.Refund)/s.curDx.BetBig
		}else if gameResult == dxStorage.ResultSmall && dxBetLog.Small>0{
			amount = s.curDx.Jackpot * (dxBetLog.Small-dxBetLog.Refund)/s.curDx.BetSmall
		}
		if amount > 0 {
			details := dxStorage.NewJackpotLog(s.curDx.ShowId, dxBetLog.Uid,dxBetLog.NickName,
				amount,dxBetLog.UserType,int(s.curDx.Result))
			dxStorage.InsertJackpotLog(details)
			gameStorage.IncGameWinLoseScore(game.BiDaXiao,dxBetLog.NickName,amount)
			if dxBetLog.UserType != dxStorage.UserTypeBot{
				s.openJackpotBill(details)
				uid := utils.ConvertOID(dxBetLog.Uid)
				user := userStorage.QueryUserId(utils.ConvertOID(dxBetLog.Uid))
				lobbyStorage.Win(uid,user.NickName, amount,game.BiDaXiao,true)

				betDetails := map[string]interface{}{
					"Big": dxBetLog.Big,
					"Small": dxBetLog.Small,
					"Refund": dxBetLog.Refund,
				}
				betDetailsStr,_ := json.Marshal(betDetails)
				gameResult := fmt.Sprintf("%d,%d,%d",s.curDx.Dice1,s.curDx.Dice2,s.curDx.Dice3)
				wallet := walletStorage.QueryWallet(utils.ConvertOID(dxBetLog.Uid))
				//realBet := dxBetLog.Big + dxBetLog.Small - dxBetLog.Refund
				betRecordParam := gameStorage.BetRecordParam{
					Uid: dxBetLog.Uid,
					GameType: game.BiDaXiao,
					Income: amount,
					BetAmount: 0,
					CurBalance: wallet.VndBalance + wallet.SafeBalance,
					SysProfit: 0,
					BotProfit: 0,
					BetDetails: string(betDetailsStr),
					GameId: strconv.Itoa(int(s.curDx.ShowId)),
					GameNo: strconv.Itoa(int(s.curDx.ShowId)),
					GameResult: gameResult,
					IsSettled: true,
					IsJackpot: true,
				}
				gameStorage.InsertBetRecord(betRecordParam)

				log.Info("open jackpot %v", details)
			}
		}
	}
}
func (s *checkout) openWinBill(dxBetLog *dxStorage.DxBetLog) {
	if dxBetLog.UserType == dxStorage.UserTypeBot { //机器人不用通知
		return
	}
	eventId := strconv.Itoa(int(s.curDx.ShowId))
	money := dxBetLog.Result + dxBetLog.Small + dxBetLog.Big
	bill := walletStorage.
		NewBill(dxBetLog.Uid, walletStorage.TypeIncome, walletStorage.EventGameDx, eventId, money)
	walletStorage.OperateVndBalance(bill)
}
func (s *checkout) openJackpotBill(details *dxStorage.DxJackpotDetails)  {
	eventId := strconv.Itoa(int(details.GameId))
	money := details.Amount
	bill := walletStorage.
		NewBill(details.Uid, walletStorage.TypeIncome, walletStorage.EventGameDxJackpot,
			eventId, money)
	walletStorage.OperateVndBalance(bill)
	//notifyWallet(s.onlinePush, details.Uid)
}
func (s *checkout) notifyUserCheckout(uid string,result int64) {
	sb := gate.QuerySessionBean(uid)
	if sb == nil{
		return
	}
	msg := make(map[string]interface{})
	msg["GameType"] = game.BiDaXiao
	msg["Action"] = actionCheckout
	msg["amount"] = result
	data, _ := json.Marshal(msg)
	_ = s.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, data)
}
func (s *checkout) notifyUserRefund(uid string,result int64) {
	sb := gate.QuerySessionBean(uid)
	if sb == nil{
		return
	}
	msg := make(map[string]interface{})
	msg["GameType"] = game.BiDaXiao
	msg["Action"] = actionRefund
	msg["amount"] = result
	data, _ := json.Marshal(msg)
	_ = s.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, data)
}
func notifyWallet(onlinePush *gate.OnlinePush, uid string) {
	sb := gate.QuerySessionBean(uid)
	if sb == nil{
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	msg := make(map[string]interface{})
	msg["Wallet"] = wallet
	msg["Action"] = "wallet"
	msg["GameType"] = game.All
	b, _ := json.Marshal(msg)
	_ = onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, b)
	agentStorage.OnWalletChange(uid)
}
