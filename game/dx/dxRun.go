package dx

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"runtime"
	"sort"
	"strconv"
	"time"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	"vn/gate"
	"vn/storage"
	"vn/storage/activityStorage"
	"vn/storage/chatStorage"
	"vn/storage/dxStorage"
	"vn/storage/gameStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

//var dxResponseTopic = "dx/response"
type dxRun struct {
	app         module.App
	settings    *conf.ModuleSettings
	table       *DxTable
	TimeLeft    int
	action      string
	curDx       *dxStorage.Dx
	dxConf      *dxStorage.Conf
	bot         *Bot
	testJackpot bool
	allowBot    bool
	checkout    *checkout
	curRoundBet []CurRoundBet
}
type CurRoundBet struct {
	NickName string
	Bets     int64
	Position string
}

var (
	openResultWaiteTime = 15
	timeLeftDefault     = 30
	//timeLeftDefault     = 20
	stopBetLeftTime = 4
	curRoundBetMax  = 20
	curRoundGroup   = "dxCurBet"

	actionDoingBet = "doingBet"
	actionResult   = "result"
	actionWaite    = "waiteNext"
	actionCheckout = "checkout"
	actionRefund   = "refund"
)

func (s *dxRun) initConf() {
	s.testJackpot = false
	testJackpot := storage.QueryConf(storage.KTestJackpot)
	if testJackpot == "1" {
		s.testJackpot = true
	}
	s.allowBot = true
	notAllowBot := storage.QueryConf(storage.KNotAllowBot)
	if notAllowBot == "1" {
		s.allowBot = false
	}
	s.dxConf = dxStorage.GetDxConf()
	//log.Info("initConf allowBot: %v", s.allowBot)
}
func (s *dxRun) Run() {
	s.bot = &Bot{}
	s.bot.Init()
	log.Info("dx running...")
	defer func() {
		if r := recover(); r != nil {
			buff := make([]byte, 1024)
			runtime.Stack(buff, false)
			log.Error("Update panic(%v)\n info:%s", r, string(buff))
			//s.Run()
		}
	}()
	for {
		s.initConf()
		s.curDx = dxStorage.NewDxGame()
		s.curRoundBet = make([]CurRoundBet, 0)
		//log.Info("NewDxGame: %v", s.curDx)
		//s.gameProfit = gameStorage.QueryProfit(game.BiDaXiao)
		s.bot.NewDxGame()
		for timeLeft := timeLeftDefault; timeLeft > 0; timeLeft-- {
			s.action = actionBet
			s.TimeLeft = timeLeft
			start := time.Now().UnixNano()
			if s.allowBot {
				s.bot.DoingBet(timeLeft, s.curDx, &s.curRoundBet)
			}
			s.doingBet(timeLeft)
			go s.doingCurRoundBet()
			end := time.Now().UnixNano()
			time.Sleep(1*time.Second - time.Duration(end-start))
		}
		s.TimeLeft = 0
		s.openResult()
		for i := openResultWaiteTime; i >= 0; i-- {
			s.action = actionWaite
			time.Sleep(1 * time.Second)
			msg := make(map[string]interface{})
			msg["TimeLeft"] = i
			msg["Action"] = actionWaite
			s.toResult(msg)
			if i == 3 {
				dx := *s.curDx
				go func() {
					defer func() {
						if r := recover(); r != nil {
							buff := make([]byte, 1024)
							runtime.Stack(buff, false)
							log.Error("panic(%v)\n info:%s", r, string(buff))
						}
					}()
					s.checkout.notifyCheckout(dx)
				}()
			}
		}
		time.Sleep(1 * time.Second)
		//log.Info("UpdateDx: %v", s.curDx)
		dxStorage.UpdateDx(s.curDx)
		reboot := gameStorage.QueryGameReboot(game.BiDaXiao)
		if reboot == "true" {
			log.Info("dx game is stop,cuz reboot config is true")
			break
		}
	}
}
func (s *dxRun) openResult() {
	s.parseRefund()
	s.parseResult()
	dxStorage.UpdateDx(s.curDx)
	msg := make(map[string]interface{})
	msg["dx"] = s.curDx.Notify
	msg["TimeLeft"] = 0
	msg["WaiteTime"] = openResultWaiteTime
	msg["Action"] = actionResult
	msg["curJackpot"] = s.GetNextJackpot()
	s.toResult(msg)
	go func() {
		checkout := &checkout{
			curDx:      s.curDx,
			onlinePush: s.table.onlinePush,
			dxConf:     s.dxConf,
		}
		s.checkout = checkout
		checkout.checkOut()
	}()
}
func (s *dxRun) GetNextJackpot() int64 {
	if s.curDx.ResultJackpot == 1 {
		return 0
	}
	jackpot := dxStorage.GetJackpot()
	amount := jackpot.Amount
	var intoJackpot int64
	jackpotPerThousand := int64(s.dxConf.JackpotPerThousand)
	if s.curDx.Result == dxStorage.ResultSmall {
		intoJackpot = s.curDx.BetSmall * jackpotPerThousand / 1000
	} else if s.curDx.Result == dxStorage.ResultBig {
		intoJackpot = s.curDx.BetBig * jackpotPerThousand / 1000
	}
	amount += intoJackpot
	return amount
}
func (s *dxRun) parseResult() {
	isCheat, isCheatJackpot := s.isCheat()
	var dice []uint8
	if isCheat {
		if s.curDx.RealBetBig > s.curDx.RealBetSmall {
			dice = s.getDice(-1)
		} else {
			dice = s.getDice(1)
		}
	} else {
		dice = s.getDice(0)
	}
	s.curDx.Dice1 = dice[0]
	s.curDx.Dice2 = dice[1]
	s.curDx.Dice3 = dice[2]
	sum := s.curDx.Dice1 + s.curDx.Dice2 + s.curDx.Dice3
	if sum > 10 {
		if s.isJackpot(dice) && s.curDx.BetBigCount%5 == 0 {
			if isCheatJackpot {
				s.bot.oneBet("big", s.curDx, &s.curRoundBet)
			} else {
				s.curDx.ResultJackpot = 1
			}
		} else if s.isJackpot(dice) && s.testJackpot {
			s.curDx.ResultJackpot = 1
			diff := int(s.curDx.BetBigCount % 5)
			if diff != 0 {
				for i := 0; i < (5 - diff); i++ {
					s.bot.oneBet("big", s.curDx, &s.curRoundBet)
				}
			}
		}
		s.curDx.Result = dxStorage.ResultBig
	} else {
		if s.isJackpot(dice) && s.curDx.BetSmallCount%5 == 0 {
			if isCheatJackpot {
				s.bot.oneBet("small", s.curDx, &s.curRoundBet)
			} else {
				s.curDx.ResultJackpot = 1
			}
		} else if s.isJackpot(dice) && s.testJackpot {
			s.curDx.ResultJackpot = 1
			diff := int(s.curDx.BetSmallCount % 5)
			if diff != 0 {
				for i := 0; i < (5 - diff); i++ {
					s.bot.oneBet("small", s.curDx, &s.curRoundBet)
				}
			}
		}
		s.curDx.Result = dxStorage.ResultSmall
	}
}

//result 1 开大， result -1 开小， 0 不作弊
func (s *dxRun) getDice(result int) []uint8 {
	dice := make([]uint8, 3)
	if s.testJackpot {
		gameId := s.curDx.ShowId
		if gameId%3 == 0 {
			if gameId%2 == 0 {
				dice[0] = 1
				dice[1] = 1
				dice[2] = 1
			} else {
				dice[0] = 6
				dice[1] = 6
				dice[2] = 6
			}
			return dice
		}
	}

	dice[0] = randomDice() + 1
	dice[1] = randomDice() + 1
	dice[2] = randomDice() + 1
	sum := dice[0] + dice[1] + dice[2]
	if result == 0 {
		return dice
	}
	if sum > 10 && result > 0 {
		return dice
	} else if sum <= 10 && result < 0 {
		return dice
	} else {
		return s.getDice(result)
	}
}
func (s dxRun) isJackpot(dice []uint8) bool {
	if (dice[0] == 1 || dice[0] == 6) && dice[0] == dice[1] && dice[1] == dice[2] {
		return true
	}
	return false
}
func (s *dxRun) isCheat() (bool, bool) {
	if s.curDx.RealBetSmall == 0 && s.curDx.RealBetBig == 0 {
		return false, false
	}
	dxConf := s.dxConf
	profit := gameStorage.QueryProfit(game.BiDaXiao)
	jackpot := dxStorage.GetJackpot()
	var pay int64 //计算赔付
	var payJackpot int64
	realOpenAmount := s.curDx.RealBetSmall
	if s.curDx.RealBetSmall < s.curDx.RealBetBig {
		realOpenAmount = s.curDx.RealBetBig
	}
	pay = realOpenAmount*(1000-int64(dxConf.ProfitPerThousand))/1000 +
		realOpenAmount*int64(dxConf.BotProfitPerThousand)/1000
	if s.curDx.BetBig > s.curDx.BetSmall { //取小的押注
		payJackpot = jackpot.Amount * s.curDx.RealBetSmall / s.curDx.BetSmall
	} else {
		payJackpot = jackpot.Amount * s.curDx.RealBetBig / s.curDx.BetBig
	}
	isCheat := profit.BotBalance < pay
	isCheatJackpot := profit.BotBalance < (pay + payJackpot)
	log.Info("cur bot Balance: %v, will pay: %v, isCheat: %v ,isCheatJackpot:%v ,jackpot will pay:%v",
		profit.BotBalance, pay, isCheat, isCheatJackpot, payJackpot)
	return isCheat, isCheatJackpot
}
func (s *dxRun) parseRefund() {
	gameId := s.curDx.ShowId
	refund := s.curDx.BetBig - s.curDx.BetSmall
	re := utils.Abs(refund)
	var realRefund int64
	if refund > 0 {
		s.curDx.BetBig -= refund
		s.curDx.RefundBig = refund
		realRefund = dxStorage.QueryRealRefundAmount(gameId, s.curDx.BetSmall, "big")
		s.curDx.RealBetBig -= realRefund
		s.curDx.RealRefundBig = realRefund
	} else {
		s.curDx.BetSmall -= re
		s.curDx.RefundSmall = re
		realRefund = dxStorage.QueryRealRefundAmount(gameId, s.curDx.BetBig, "small")
		s.curDx.RealBetSmall -= realRefund
		s.curDx.RealRefundSmall = realRefund
	}
	//log.Info("gameId: %v, RealRefundBig: %v, RealRefundSmall: %v",
	//	s.curDx.ShowId, s.curDx.RealRefundBig, s.curDx.RealRefundSmall)
}
func (s *dxRun) doingBet(timeLeft int) {
	msg := make(map[string]interface{})
	msg["dx"] = s.curDx.Notify
	msg["TimeLeft"] = timeLeft
	msg["Action"] = actionDoingBet
	s.toResult(msg)
}
func (s *dxRun) toResult(msg map[string]interface{}) {
	msg["GameType"] = game.BiDaXiao
	//log.Info("msg: %v", msg)
	byte, err := json.Marshal(msg)
	if err != nil {
		log.Error(err.Error())
	}
	if err := s.table.onlinePush.NotifyAllPlayersNR(game.Push, byte); err != nil {
		log.Error(err.Error())
	}
}
func (s *dxRun) doingCurRoundBet() {
	msg := make(map[string]interface{})
	sort.Slice(s.curRoundBet, func(i, j int) bool {
		return s.curRoundBet[i].Bets > s.curRoundBet[j].Bets
	})
	var res []CurRoundBet
	if len(s.curRoundBet) > curRoundBetMax {
		res = append(s.curRoundBet[:0], s.curRoundBet[:curRoundBetMax]...)
	} else {
		res = s.curRoundBet
	}
	msg["Data"] = res
	msg["Action"] = actionCurRoundBet
	msg["Code"] = 0
	s.toCurRoundBet(msg)
}
func (s *dxRun) toCurRoundBet(msg map[string]interface{}) {
	msg["GameType"] = game.BiDaXiao
	//log.Info("msg: %v", msg)
	byte, err := json.Marshal(msg)
	if err != nil {
		log.Error(err.Error())
	}
	uids := chatStorage.QueryGroup(curRoundGroup)
	userIds := utils.ConvertUidToOid(uids)
	sessionIds := gate.GetSessionIds(userIds)
	if err := s.table.onlinePush.SendCallBackMsgNR(sessionIds, game.Push, byte); err != nil {
		log.Error(err.Error())
	}
}
func randomDice() uint8 {
	result, _ := rand.Int(rand.Reader, big.NewInt(6))
	return uint8(result.Int64())
}
func (s *dxRun) Bet(uid string, big int64, small int64) map[string]interface{} {
	if s.TimeLeft < stopBetLeftTime {
		return errCode.GameStopBet.GetI18nMap()
	}
	if big > 0 && small > 0 {
		return errCode.DxGameBetErr.GetI18nMap()
	}
	if big < 0 || small < 0 || (big == 0 && small == 0) {
		return errCode.ErrParams.GetI18nMap()
	}
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	//if user.Type == userStorage.TypeCompanyPlay{
	//	return errCode.ConnectCustomerService.GetI18nMap()
	//}
	//log.Info("bet 1.0,uid: %v, goId: %d" ,uid, utils.Goid())
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	var needVnd int64 = 0
	commonData := gameStorage.QueryGameCommonData(uid)
	if commonData != nil {
		needVnd = commonData.InRoomNeedVnd
	}
	if wallet.VndBalance-needVnd < (big + small) {
		return errCode.BalanceNotEnough.GetI18nMap()
	}
	gameId := s.curDx.ShowId
	//log.Info("bet 1.1")
	//q, notFound := dxStorage.QueryBetLog(bson.M{"Uid": uid, "GameId": gameId})//todo
	myBig, mySmall, _ := dxStorage.QueryMyBet(gameId, uid)
	//log.Info("bet 1.2")
	if big > 0 && mySmall > 0 {
		return errCode.DxGameBetErr.GetI18nMap()
	} else if small > 0 && myBig > 0 {
		return errCode.DxGameBetErr.GetI18nMap()
	}

	if myBig == 0 && mySmall == 0 { //加人数
		if big > 0 {
			s.curDx.BetBigCount += 1
			if user.Type == userStorage.TypeNormal {
				s.curDx.RealBetBigCount += 1
			}
		} else {
			s.curDx.BetSmallCount += 1
			if user.Type == userStorage.TypeNormal {
				s.curDx.RealBetSmallCount += 1
			}
		}
	}
	s.curDx.BetSmall += small
	s.curDx.BetBig += big
	userType := dxStorage.UserTypeNormal
	if user.Type == userStorage.TypeNormal {
		s.curDx.RealBetBig += big
		s.curDx.RealBetSmall += small
	} else {
		userType = "CompanyPlay"
	}

	dxBetLog := &dxStorage.DxBetLog{
		Uid:       uid,
		NickName:  user.NickName,
		GameId:    gameId,
		Big:       big,
		Small:     small,
		CurBig:    s.curDx.BetBig,
		CurSmall:  s.curDx.BetSmall,
		UserType:  userType,
		MoneyType: "vnd",
		CreateAt:  utils.Now(),
	}
	//log.Info("bet 1.3")
	dxStorage.InsertBetLog(dxBetLog)
	//log.Info("bet 1.4")
	money := (big + small) * -1
	eventId := strconv.Itoa(int(s.curDx.ShowId))
	bill := walletStorage.
		NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventGameDx, eventId, money)
	//log.Info("bet 1.5")
	_ = walletStorage.OperateVndBalance(bill)
	activityStorage.UpsertGameDataInBet(uid, game.BiDaXiao, 1)
	notifyWallet(s.table.onlinePush, uid)
	//log.Info("bet end,uid: %v, goId: %d" ,uid, utils.Goid())
	find := false
	var position string
	var amount int64
	if big > 0 {
		position = "big"
		amount = big
	} else {
		position = "small"
		amount = small
	}
	for k, v := range s.curRoundBet {
		if user.NickName == v.NickName {
			s.curRoundBet[k].Bets += amount
		}
	}
	if !find {
		s.curRoundBet = append(s.curRoundBet, CurRoundBet{
			NickName: user.NickName,
			Bets:     amount,
			Position: position,
		})
	}
	return errCode.Success(map[string]int64{"big": myBig + big, "small": mySmall + small}).
		GetMap()
}
