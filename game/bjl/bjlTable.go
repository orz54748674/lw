package bjl

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	"vn/game/activity"
	"vn/storage"
	"vn/storage/activityStorage"
	"vn/storage/bjlStorage"
	"vn/storage/gameStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Table struct {
	room.QTable
	module module.RPCModule
	app    module.App

	tableID          string
	serverID         string
	Players          sync.Map
	playerLock       sync.Mutex
	players          map[string]room.BasePlayer
	botBalance       int64
	records          []bjlStorage.Record
	curStateLastTime int
	curState         bjlStorage.Process
	GameState        bjlStorage.GameState
	initState        bool
	playerTotalBet   []int64
	systemScore      int64
	pCards           *bjlStorage.CardInfo
	stateID          int
	betsLock         sync.Mutex
	bets             []bjlStorage.BetDetail
	eventID          string

	//配置信息
	processConf          []bjlStorage.Process
	betCoins             []int64
	oddList              []int64
	posBetLimit          []int64
	cardPool             []int
	BotProfitPerThousand int
}

var (
	PLAYER      = 0
	BANKER      = 1
	HE          = 2
	PLAYER_PAIR = 3
	BANKER_PAIR = 4
)

func (s *Table) GetModule() module.RPCModule {
	return s.module
}

func (s *Table) GetSeats() map[string]room.BasePlayer {
	s.playerLock.Lock()
	defer s.playerLock.Unlock()
	return s.players
}

func (s *Table) GetApp() module.App {
	return s.app
}

//每帧都会调用
func (s *Table) Update(ds time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			buff := make([]byte, 1024)
			runtime.Stack(buff, false)
			log.Error("Update panic(%v)\n info:%s", r, string(buff))
			s.Finish()
		}
	}()
}

func NewTable(module module.RPCModule, app module.App, tableID string, opts ...room.Option) *Table {
	s := &Table{
		module:  module,
		app:     app,
		tableID: tableID,
	}
	opts = append(opts, room.TimeOut(0))
	opts = append(opts, room.Update(s.Update))
	opts = append(opts, room.NoFound(func(msg *room.QueueMsg) (value reflect.Value, e error) {
		return reflect.Zero(reflect.ValueOf("").Type()), errors.New("no found handler")
	}))
	opts = append(opts, room.SetRecoverHandle(func(msg *room.QueueMsg, err error) {
		log.Error("Recover %v Error: %v", msg.Func, err.Error())
	}))
	opts = append(opts, room.SetErrorHandle(func(msg *room.QueueMsg, err error) {
		log.Error("Error %v Error: %v", msg.Func, err.Error())
	}))
	s.players = make(map[string]room.BasePlayer)
	s.OnInit(s, opts...)

	s.initTable(module, app, tableID)

	s.Register("Leave", s.PlayerLeave)

	return s
}

func (s *Table) updateGameConf() {
	gameConf := bjlStorage.GetGameConf()
	s.betCoins = gameConf.ChipList
	s.posBetLimit = gameConf.PosBetLimit
	s.processConf = []bjlStorage.Process{
		{ID: 0, ProcessName: "onBet", ProcessLastTime: gameConf.BetTime},
		{ID: 1, ProcessName: "onSendCard", ProcessLastTime: gameConf.SendCardTime},
	}
	s.oddList = gameConf.OddList
	s.GameState.MinBet = gameConf.MinBet
	s.GameState.MaxBet = gameConf.MaxBet
	s.BotProfitPerThousand = gameConf.BotProfitPerThousand
}

//桌子初始化
func (s *Table) initTable(module module.RPCModule, app module.App, tableID string) {
	s.updateGameConf()

	s.initState = false
	s.pCards = &s.GameState.Cards
	s.stateID = 0

	gameProfit := gameStorage.QueryProfit(game.Bjl)
	s.botBalance = gameProfit.BotBalance

	s.robotEnterAndLeave()

	s.Run()

	go func() {
		for {
			reboot := s.update()
			if reboot {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

//初始化每局游戏
func (s *Table) initGame() {
	s.updateGameConf()
	s.pCards.PlayerCards = []int{}
	s.pCards.BankerCards = []int{}
	s.pCards.PlayerDian = 0
	s.pCards.BankerDian = 0
	s.systemScore = 0
	s.playerTotalBet = []int64{0, 0, 0, 0, 0}
	s.GameState.BetInfos = []int64{0, 0, 0, 0, 0}
	s.GameState.PosRes = []bool{false, false, false, false, false}
	s.GameState.GameNo = strconv.Itoa(int(storage.NewGlobalId("bjlGameNo") + 100000))
	s.eventID = s.GameState.GameNo

	s.betsLock.Lock()
	s.bets = []bjlStorage.BetDetail{}
	s.betsLock.Unlock()

	s.Players.Range(func(k, val interface{}) bool {
		v := val.(*Player)
		v.Score = 0
		v.TotalBet = 0
		v.BetInfo = []int64{0, 0, 0, 0, 0}
		if !v.IsOnline {
			s.Players.Delete(k.(string))
			for a, b := range s.GameState.PlayerInfos {
				if b.Uid == k {
					s.GameState.PlayerInfos = append(s.GameState.PlayerInfos[:a], s.GameState.PlayerInfos[a+1:]...)
				}
			}
		}
		return true
	})

	for k, _ := range s.GameState.PlayerInfos {
		s.GameState.PlayerInfos[k].BetInfos = []int64{0, 0, 0, 0, 0}
	}
}

func (s *Table) OnCreate() {
	//可以加载数据
	s.QTable.OnCreate()
}

func (s *Table) OnDestroy() {
	s.BaseTableImp.OnDestroy()
}

func (self *Table) onGameOver() {
	self.Finish()
}

func (s *Table) GetState() bjlStorage.GameState {
	var tmpPl []bjlStorage.PlayerInfo

	s.Players.Range(func(k, val interface{}) bool {
		v := val.(*Player)
		var tmpBetInfos []int64
		for _, val := range v.BetInfo {
			tmpBetInfos = append(tmpBetInfos, val)
		}
		var tmp = bjlStorage.PlayerInfo{
			Uid:      v.UserID,
			Golds:    v.Golds,
			Head:     v.Head,
			NickName: v.Nickname,
			BetInfos: tmpBetInfos,
			Score:    v.Score,
		}
		tmpPl = append(tmpPl, tmp)
		return true
	})

	s.GameState.RemainTime = int64(s.curState.ProcessLastTime) - (utils.GetMillisecond() - s.GameState.St) + 10
	s.GameState.ServerTime = utils.GetMillisecond()
	s.GameState.PlayerInfos = tmpPl

	return s.GameState
}

func (s *Table) GetHistory() []bjlStorage.Record {
	return s.records
}

func (s *Table) checkOut() {
	s.Players.Range(func(k, val interface{}) bool {
		v := val.(*Player)
		for i := 0; i <= 4; i++ {
			//开和，闲和庄的下注要返还给玩家
			if s.GameState.PosRes[HE] {
				if i <= 1 {
					v.Score = v.Score + v.BetInfo[i]
				} else if s.GameState.PosRes[i] {
					v.Score = (v.BetInfo[i]*s.oddList[i])/100 + v.Score
				}
			} else if s.GameState.PosRes[i] {
				v.Score = (v.BetInfo[i]*s.oddList[i])/100 + v.Score
			}
		}
		if v.Score != 0 && !v.robotMsg.IsRobot {
			billType := walletStorage.TypeIncome
			bill := walletStorage.NewBill(v.UserID, billType, walletStorage.EventGameBjl, s.eventID, v.Score)
			walletStorage.OperateVndBalance(bill)
		}
		if !v.robotMsg.IsRobot {
			wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
			v.Golds = wallet.VndBalance
			if v.TotalBet > 0 {
				activityStorage.UpsertGameDataInBet(v.UserID, game.Bjl, 0)
				activity.CalcEncouragementFunc(v.UserID)
			}
		} else {
			v.Golds = v.Golds + v.Score
		}
		return true
	})
}

func (s *Table) gameRecord() {
	var records []bjlStorage.UserGameRecord

	resultMap := map[string][]int{
		"PLAYER_CARD": s.GameState.Cards.PlayerCards,
		"BANKER_CARD": s.GameState.Cards.BankerCards,
	}
	resultStr, _ := json.Marshal(resultMap)
	if s.systemScore != 0 {
		botProfit := int64(math.Abs(float64(s.systemScore * int64(s.BotProfitPerThousand) / 1000)))
		gameStorage.IncProfit("", game.Bjl, 0, s.systemScore-botProfit, botProfit)
		gameProfit := gameStorage.QueryProfit(game.Bjl)
		s.botBalance = gameProfit.BotBalance
	}

	s.Players.Range(func(k, val interface{}) bool {
		v := val.(*Player)
		if v.TotalBet > 0 && !v.robotMsg.IsRobot {
			betDetails := map[string]int64{
				"PLAYER":      v.BetInfo[0],
				"BANKER":      v.BetInfo[1],
				"HE":          v.BetInfo[2],
				"PLAYER_PAIR": v.BetInfo[3],
				"BANKER_PAIR": v.BetInfo[4],
			}
			betDetailsStr, _ := json.Marshal(betDetails)
			wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
			var recordParams gameStorage.BetRecordParam
			recordParams.Uid = v.UserID
			recordParams.GameNo = s.eventID
			recordParams.BetAmount = v.TotalBet
			recordParams.BotProfit = 0
			recordParams.SysProfit = 0
			recordParams.BetDetails = string(betDetailsStr)
			recordParams.GameResult = string(resultStr)
			recordParams.CurBalance = v.Golds + wallet.SafeBalance
			recordParams.GameType = game.Bjl
			recordParams.Income = v.Score - v.TotalBet
			recordParams.IsSettled = true
			gameStorage.InsertBetRecord(recordParams)

			createTime := time.Now().Unix()
			for pos, betValue := range v.BetInfo {
				if betValue > 0 {
					tmpScore := -betValue
					if s.GameState.PosRes[pos] {
						tmpScore = (betValue * s.oddList[pos]) / 100
					}
					tmpRecord := bjlStorage.UserGameRecord{
						Uid:        v.UserID,
						GameNo:     s.GameState.GameNo,
						Pos:        pos,
						BetCount:   betValue,
						Score:      tmpScore,
						PlayerDian: s.pCards.PlayerDian,
						BankerDian: s.pCards.BankerDian,
						CreateTime: createTime,
					}
					records = append(records, tmpRecord)
				}
			}
		}

		v.Score = 0
		v.TotalBet = 0
		return true
	})

	if len(records) > 0 {
		bjlStorage.InsertRecord(records)
	}
}

//玩家加入桌子
func (s *Table) SitDown(session gate.Session) error {
	userID := session.GetUserID()
	if userID == "" {
		log.Info("your userid is empty")
		return nil
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))

	bExist := false
	tmp, _ := s.Players.Load(userID)
	if tmp != nil {
		bExist = true
	}
	if bExist {
		tmpPlayer := tmp.(*Player)
		tmpPlayer.Golds = wallet.VndBalance
		tmpPlayer.IsOnline = true
	} else {
		user := userStorage.QueryUserId(utils.ConvertOID(userID))

		tmpInfo := map[string]interface{}{
			"userID": userID,
			"golds":  wallet.VndBalance,
			"head":   user.Avatar,
			"name":   user.NickName,
		}

		player := NewPlayer(tmpInfo)
		s.Players.Store(userID, player)

		ret := make(map[string]interface{})
		ret["uid"] = player.UserID
		ret["nickName"] = player.Nickname
		ret["head"] = player.Head
		ret["golds"] = player.Golds
		s.sendPackToAll(game.Push, ret, protocol.Enter, nil)
	}

	player := &room.BasePlayerImp{}
	player.Bind(session)
	player.OnRequest(session)
	s.playerLock.Lock()
	s.players[userID] = player
	s.playerLock.Unlock()
	return nil
}

func (s *Table) GetRecord(session gate.Session, msg map[string]interface{}) error {
	uid := session.GetUserID()
	if uid == "" {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", actionGetRecord, error)
		return nil
	}
	param1, ok1 := msg["offset"].(float64)
	offset := int(param1)
	if !ok1 {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", actionGetRecord, error)
		return nil
	}
	param2, ok2 := msg["limit"].(float64)
	if !ok2 {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", actionGetRecord, error)
		return nil

	}
	limit := int(param2)

	records := bjlStorage.GetUserRecord(uid, offset, limit)
	s.sendPack(session.GetSessionID(), game.Push, records, actionGetRecord, nil)

	return nil
}

func (s *Table) betHandle(userID string, pos int, coin int64) *common.Err {
	tmpPl, _ := s.Players.Load(userID)
	pl := tmpPl.(*Player)
	if tmpPl == nil {
		error := errCode.NotInRoomError
		return error
	}
	if !utils.IsContainInt64(s.betCoins, coin) {
		error := errCode.ErrParams
		return error
	}
	if pos < 0 || pos > 4 {
		error := errCode.ErrParams
		return error
	}

	if coin+pl.BetInfo[pos] > s.posBetLimit[pos] || pl.TotalBet+coin > s.GameState.MaxBet {
		error := errCode.BetLimit
		return error
	}
	if (pos == 0 && pl.BetInfo[1] > 0) || (pos == 1 && pl.BetInfo[0] > 0) {
		error := errCode.BetCodeErr
		return error
	}

	if !pl.robotMsg.IsRobot {
		wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
		if wallet.VndBalance < coin {
			error := errCode.BalanceNotEnough
			return error
		}
		bill := walletStorage.NewBill(userID, walletStorage.TypeExpenses, walletStorage.EventGameBjl, s.eventID, -coin)
		walletStorage.OperateVndBalance(bill)
		pl.Golds = wallet.VndBalance - coin
		activityStorage.UpsertGameDataInBet(userID, game.Bjl, 1)
	} else {
		if pl.Golds-coin < 0 {
			error := errCode.BalanceNotEnough
			return error
		}
		pl.Golds = pl.Golds - coin
	}

	pl.BetInfo[pos] = pl.BetInfo[pos] + coin
	pl.TotalBet = pl.TotalBet + coin
	s.GameState.BetInfos[pos] = s.GameState.BetInfos[pos] + coin

	info := []bjlStorage.BetDetail{
		{
			UserID:      userID,
			Pos:         pos,
			Coin:        coin,
			PlayerGolds: pl.Golds,
		},
	}

	s.betsLock.Lock()
	s.bets = append(s.bets, info[0])
	s.betsLock.Unlock()

	if !pl.robotMsg.IsRobot && pl.UserType == userStorage.TypeNormal {
		s.playerTotalBet[pos] = s.playerTotalBet[pos] + coin
	}

	return nil
}

func (s *Table) Bet(session gate.Session, msg map[string]interface{}) error {
	if s.GameState.State != "onBet" {
		return nil
	}
	userID := session.GetUserID()
	param1, ok1 := msg["pos"].(float64)
	pos := int(param1)
	if !ok1 {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", actionBet, error)
		return nil
	}
	param2, ok2 := msg["coin"].(float64)
	if !ok2 {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", actionBet, error)
		return nil
	}
	coin := int64(param2)

	error := s.betHandle(userID, pos, coin)
	if error != nil {
		s.sendPack(session.GetSessionID(), game.Push, "", actionBet, error)
		return nil
	}

	tmpPl, _ := s.Players.Load(userID)
	pl := tmpPl.(*Player)
	info := []bjlStorage.BetDetail{
		{
			UserID:      userID,
			Pos:         pos,
			Coin:        coin,
			PlayerGolds: pl.Golds,
		},
	}
	newInfo := struct {
		BetInfos []int64                `json:"betInfos"`
		Bets     []bjlStorage.BetDetail `json:"bets"`
	}{
		BetInfos: s.GameState.BetInfos,
		Bets:     info,
	}
	s.sendPack(session.GetSessionID(), game.Push, newInfo, actionBet, nil)

	return nil
}

func (s *Table) broadcastGameState() {
	s.GetState()
	s.sendPackToAll(game.Push, s.GameState, "HD_changeState", nil)
}

func (s *Table) broadcastBets() {
	for {
		time.Sleep(2 * time.Second)
		if s.GameState.State != "onBet" {
			break
		}
		s.Players.Range(func(k, val interface{}) bool {
			v := val.(*Player)
			var tmpBets []bjlStorage.BetDetail

			s.betsLock.Lock()
			for _, b := range s.bets {
				if b.UserID != k {
					tmpBets = append(tmpBets, b)
				}
			}
			s.betsLock.Unlock()

			newInfo := struct {
				BetInfos []int64                `json:"betInfos"`
				Bets     []bjlStorage.BetDetail `json:"bets"`
			}{
				BetInfos: s.GameState.BetInfos,
				Bets:     tmpBets,
			}
			if !v.robotMsg.IsRobot {
				s.playerLock.Lock()
				sessionStr := ""
				if s.players[v.UserID] != nil {
					sessionStr = s.players[v.UserID].Session().GetSessionID()
				}
				s.playerLock.Unlock()

				if sessionStr != "" {
					s.sendPack(sessionStr, game.Push, newInfo, protocol.XiaZhu, nil)
				}
			}
			return true
		})

		s.betsLock.Lock()
		s.bets = []bjlStorage.BetDetail{}
		s.betsLock.Unlock()
	}
}

func (s *Table) onBet() {
	s.initGame()
	s.broadcastGameState()
	go s.broadcastBets()
	s.robotBet()
}

func (s *Table) getHandCardDian(cards []int) int {
	dian := 0
	for _, card := range cards {
		val := card % 13
		if val >= 10 {
			val = 0
		}
		dian = dian + val
		if dian >= 10 {
			dian = dian - 10
		}
	}

	return dian
}

func (s *Table) getOneCard(cards *[]int) {
	idx := rand.Intn(len(s.cardPool))
	card := s.cardPool[idx]
	*cards = append(*cards, card)
	s.cardPool = append(s.cardPool[:idx], s.cardPool[idx+1:]...)
}

func (s *Table) checkPosRes() {
	for i := 0; i <= 4; i++ {
		s.GameState.PosRes[i] = false
	}
	if (s.pCards.PlayerCards[0])%13 == (s.pCards.PlayerCards[1] % 13) {
		s.GameState.PosRes[3] = true
	}

	if (s.pCards.BankerCards[0])%13 == (s.pCards.BankerCards[1] % 13) {
		s.GameState.PosRes[4] = true
	}

	if s.pCards.PlayerDian > s.pCards.BankerDian {
		s.GameState.PosRes[0] = true
	} else if s.pCards.PlayerDian < s.pCards.BankerDian {
		s.GameState.PosRes[1] = true
	} else {
		s.GameState.PosRes[2] = true
	}
}

func (s *Table) handleSendCard() {
	s.pCards.PlayerCards = []int{}
	s.pCards.BankerCards = []int{}

	for i := 1; i <= 2; i++ {
		s.getOneCard(&s.pCards.PlayerCards)
	}
	for i := 1; i <= 2; i++ {
		s.getOneCard(&s.pCards.BankerCards)
	}

	s.pCards.PlayerDian = s.getHandCardDian(s.pCards.PlayerCards)
	s.pCards.BankerDian = s.getHandCardDian(s.pCards.BankerCards)

	if s.pCards.PlayerDian >= 8 || s.pCards.BankerDian >= 8 {
		return
	}

	playerThirdVal := -1
	if utils.IsContainInt([]int{0, 1, 2, 3, 4, 5}, s.pCards.PlayerDian) {
		s.getOneCard(&s.pCards.PlayerCards)
		playerThirdVal = s.pCards.PlayerCards[2] % 13
		s.pCards.PlayerDian = s.getHandCardDian(s.pCards.PlayerCards)
	}

	if ((s.pCards.PlayerDian == 6 || s.pCards.PlayerDian == 7) && s.pCards.BankerDian <= 5) || s.pCards.BankerDian <= 2 {
		s.getOneCard(&s.pCards.BankerCards)
		s.pCards.BankerDian = s.getHandCardDian(s.pCards.BankerCards)
		return
	}

	if s.pCards.BankerDian >= 7 {
		return
	}

	if playerThirdVal != -1 {
		if playerThirdVal != 8 && s.pCards.BankerDian == 3 {
			s.getOneCard(&s.pCards.BankerCards)
			s.pCards.BankerDian = s.getHandCardDian(s.pCards.BankerCards)
			return
		}
		if s.pCards.BankerDian == 4 && utils.IsContainInt([]int{2, 3, 4, 5, 6, 7}, playerThirdVal) {
			s.getOneCard(&s.pCards.BankerCards)
			s.pCards.BankerDian = s.getHandCardDian(s.pCards.BankerCards)
			return
		}
		if s.pCards.BankerDian == 5 && utils.IsContainInt([]int{4, 5, 6, 7}, playerThirdVal) {
			s.getOneCard(&s.pCards.BankerCards)
			s.pCards.BankerDian = s.getHandCardDian(s.pCards.BankerCards)
			return
		}
		if s.pCards.BankerDian == 6 && utils.IsContainInt([]int{6, 7}, playerThirdVal) {
			s.getOneCard(&s.pCards.BankerCards)
			s.pCards.BankerDian = s.getHandCardDian(s.pCards.BankerCards)
			return
		}
	}
}

func (s *Table) getSystemScore() int64 {
	s.checkPosRes()

	s.systemScore = 0
	for i := PLAYER; i <= BANKER_PAIR; i++ {
		if !s.GameState.PosRes[HE] || (s.GameState.PosRes[HE] && i > BANKER) {
			if s.GameState.PosRes[i] {
				s.systemScore = s.systemScore - ((s.oddList[i]-100)*s.playerTotalBet[i])/100
			} else {
				s.systemScore = s.systemScore + s.playerTotalBet[i]
			}
		}
	}

	return s.systemScore
}

func (s *Table) onSendCard() {
	for {
		s.cardPool = []int{}
		for i := 1; i <= 52; i++ {
			for n := 1; n <= 8; n++ {
				s.cardPool = append(s.cardPool, i)
			}
		}
		s.handleSendCard()
		var tmpScore int64
		tmpScore = s.getSystemScore()
		if tmpScore >= 0 || s.botBalance+tmpScore > 0 {
			//s.botBalance = s.botBalance + tmpScore
			break
		}
	}

	cardCount := len(s.pCards.PlayerCards) + len(s.pCards.BankerCards)
	if cardCount == 4 {
		s.curState.ProcessLastTime = 22000
	} else if cardCount == 5 {
		s.curState.ProcessLastTime = 25000
	} else {
		s.curState.ProcessLastTime = 28000
	}

	s.checkOut()
	s.robotCheckout()
	s.broadcastGameState()
	s.addRecord()
	s.gameRecord()
}

func (s *Table) update() bool {
	if !s.initState {
		s.initState = true
		s.curState = s.processConf[0]
		s.GameState.State = s.curState.ProcessName
		s.GameState.St = utils.GetMillisecond()
		s.GameState.Et = s.GameState.St + int64(s.curState.ProcessLastTime)
		s.curStateLastTime = 0
		s.onBet()
		return false
	}

	s.curStateLastTime = s.curStateLastTime + 100
	if s.curStateLastTime < s.curState.ProcessLastTime {
		return false
	}

	s.stateID = s.curState.ID + 1
	if s.stateID >= 2 {
		s.stateID = 0
	}
	s.curStateLastTime = 0

	s.curState = s.processConf[s.stateID]
	s.GameState.State = s.curState.ProcessName
	s.GameState.St = utils.GetMillisecond()
	s.GameState.Et = s.GameState.St + int64(s.curState.ProcessLastTime)

	if s.stateID == 0 {
		reboot := gameStorage.QueryGameReboot(game.Bjl)
		if reboot == "true" {
			//游戏当局结束后停止发牌
			return true
		}
		s.onBet()
	} else if s.stateID == 1 {
		s.onSendCard()
	}

	return false
}

//处理玩家离开
func (s *Table) leaveHandle(userID string) error {
	tmpPl, _ := s.Players.Load(userID)
	if tmpPl == nil {
		return nil
	}
	pl := tmpPl.(*Player)
	if pl.TotalBet <= 0 {
		s.Players.Delete(userID)
		for a, b := range s.GameState.PlayerInfos {
			if b.Uid == userID {
				s.GameState.PlayerInfos = append(s.GameState.PlayerInfos[:a], s.GameState.PlayerInfos[a+1:]...)
			}
		}
		s.sendPackToAll(game.Push, s.GameState, protocol.PlayerLeave, nil)
		return nil
	}

	if pl.TotalBet > 0 {
		pl.IsOnline = false
	}

	return nil
}

func (s *Table) PlayerLeave(session gate.Session, msg map[string]interface{}) error {
	userID := session.GetUserID()

	return s.leaveHandle(userID)
}

func (s *Table) addRecord() {
	res := 0
	if s.GameState.PosRes[1] {
		res = 1
	} else if s.GameState.PosRes[2] {
		res = 2
	}
	playerPair := false
	if s.GameState.PosRes[3] {
		playerPair = true
	}
	bankerPair := false
	if s.GameState.PosRes[4] {
		bankerPair = true
	}
	record := bjlStorage.Record{
		Result:     res,
		PlayerPair: playerPair,
		BankerPair: bankerPair,
		BankerDian: s.pCards.BankerDian,
		PlayerDian: s.pCards.PlayerDian,
	}
	if len(s.records) > 100 {
		s.records = append(s.records[:0], s.records[1:]...)
	}

	s.records = append(s.records, record)
}

func (s *Table) SendShortCut(session gate.Session, msg map[string]interface{}) (err error) {
	userID := session.GetUserID()
	tmpPl, _ := s.Players.Load(userID)
	pl := tmpPl.(*Player)
	if tmpPl == nil {
		return errors.New("no this player")
	}
	Interval := time.Now().Unix() - pl.LastChatTime
	if Interval < 3 { //间隔太短
		error := errCode.TimeIntervalError
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.SendShortCut, error)
		return nil
	}
	pl.LastChatTime = time.Now().Unix()

	msg["UserId"] = userID
	_ = s.sendPackToAll(game.Push, msg, protocol.SendShortCut, nil)
	return nil
}
