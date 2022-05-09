package suoha

import (
	"errors"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"
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
	"vn/storage/gameStorage"
	"vn/storage/suohaStorage"
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
	records          []suohaStorage.Record
	curStateLastTime int
	curState         suohaStorage.Process
	GameState        suohaStorage.GameState
	initState        bool
	systemScore      int64
	pCards           *suohaStorage.CardInfo
	stateID          int
	betsLock         sync.Mutex
	bets             []suohaStorage.BetDetail
	eventID          string

	//配置信息
	processConf          []suohaStorage.Process
	betCoins             []int64
	oddList              []int64
	posBetLimit          []int64
	cardPool             []int
	BotProfitPerThousand int
	FreePlayers          []string
	PlayingPlayers       []string
	Base                 int64
	uid2HandCards        map[string][]int
	FirstSeat            int
	CurSeat              int
	seat2Uid             map[int]string
	actionBySeatArr      []int
	uid2RoundBet         map[string][]int64
	gameConf             suohaStorage.Conf
	gameStartTimer       *time.Timer
	secondRoundTimer     *time.Timer
	autoActionTimer      *time.Timer
	tableConf            suohaStorage.BaseInfo
	curStartTime int64
}

var (
	MaxPlayer = 5
)

const (
	onFirstRoundSendCard = "HD_onFirstRoundSendCard"
	onUserSelectShowCard = "HD_onUserSelectShowCard"
	onUserAddGolds       = "HD_onUserAddGolds"
	onUserReady          = "HD_onUserReady"
)

const (
	FirstRound  = "FirstRound"
	SecondRound = "SecondRound"
	ThirdRound  = "ThirdRound"
	FourthRound = "FourthRound"
	FifthRound  = "FifthRound"
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

	s.initTable()

	s.Register("Leave", s.PlayerLeave)

	return s
}

func (s *Table) updateGameConf() {
	s.gameConf = suohaStorage.GetGameConf()
	//s.betCoins = gameConf.ChipList
	//s.posBetLimit = gameConf.PosBetLimit
	//s.processConf = []suohaStorage.Process{
	//	{ID: 0, ProcessName: "onBet", ProcessLastTime: gameConf.BetTime},
	//	{ID: 1, ProcessName: "onSendCard", ProcessLastTime: gameConf.SendCardTime},
	//}
	//s.oddList = gameConf.OddList
	//s.GameState.MinBet = gameConf.MinBet
	//s.GameState.MaxBet = gameConf.MaxBet
	//s.BotProfitPerThousand = gameConf.BotProfitPerThousand

}

//桌子初始化
func (s *Table) initTable() {
	s.updateGameConf()

	s.initState = false
	s.stateID = 0

	gameProfit := gameStorage.QueryProfit(game.SuoHa)
	s.botBalance = gameProfit.BotBalance

	s.Run()
}

func (s *Table) initCards() {
	s.cardPool = []int{}
	for i := 1; i <= 52; i++ {
		s.cardPool = append(s.cardPool, i)
	}
}

func (s *Table) PlayerReady(session gate.Session, msg map[string]interface{}) error {
	if s.GameState.State != "ready" {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.Illegal)
		return nil
	}
	uid := session.GetUserID()
	if uid == "" {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.ErrParams)
		return nil
	}

	if tmp, ok := s.Players.Load(uid); !ok {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.ErrParams)
		return nil
	} else {
		tmpPlayer := tmp.(*Player)
		tmpPlayer.IsReady = true
		info := struct {
			Uid string
		}{
			Uid: uid,
		}
		s.sendPackToAll(game.Push, info, onUserReady, nil)
		readyCount := 0
		allReady := true
		s.Players.Range(func(key, value interface{}) bool {
			v := value.(*Player)
			if v.IsReady {
				readyCount++
			} else {
				allReady = false
			}
			return true
		})

		if allReady {
			s.GameStart()
		} else if readyCount >= 2 {
			s.gameStartTimer = time.AfterFunc(10*time.Second, s.GameStart)
		}

		return nil
	}
}

func (s *Table) GameStart() {
	s.gameStartTimer.Stop()
	s.initCards()
	s.Players.Range(func(key, value interface{}) bool {
		v := value.(*Player)
		v.RoundBet = []int64{0, 0, 0, 0, 0}
		v.TotalBet = 0
		v.Score = 0
		v.HandCards = []int{}
		return true
	})
	s.FirstRound()
}

func (s *Table) FirstRound() {
	s.GameState.State = "FirstRound"
	s.broadcastGameState()
	s.Players.Range(func(key, value interface{}) bool {
		v := value.(*Player)
		s.uid2HandCards[v.UserID] = append(s.uid2HandCards[v.UserID], s.getOneCard())
		s.uid2HandCards[v.UserID] = append(s.uid2HandCards[v.UserID], s.getOneCard())
		s.uid2RoundBet[v.UserID] = append(s.uid2RoundBet[v.UserID], s.Base)
		v.HandCards = []int{0, 0}
		info := struct {
			Cards []int `json:"Cards"`
		}{
			Cards: s.uid2HandCards[v.UserID],
		}
		s.sendPack(s.players[v.UserID].Session().GetSessionID(), game.Push, info, onFirstRoundSendCard, nil)
		return true
	})

	s.secondRoundTimer = time.AfterFunc(10*time.Second, s.SecondRound)
}

func (s *Table) SelectFirstShowCard(session gate.Session, msg map[string]interface{}) error {
	uid := session.GetUserID()
	if uid == "" {
		return nil
	}

	param1, ok := msg["cardVal"].(float64)
	if !ok {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.ErrParams)
		return nil
	}
	cardVal := int(param1)
	if !utils.IsContainInt(s.uid2HandCards[uid], cardVal) {
		return nil
	}

	tmp, _ := s.Players.Load(uid)
	tmpPlayer := tmp.(*Player)
	tmpPlayer.SelectShowCard = cardVal
	if len(tmpPlayer.HandCards) >= 2 {
		tmpPlayer.HandCards[1] = cardVal
	}

	info := struct {
		Uid           string `json:"Uid"`
		SelectCardVal int    `json:"SelectCardVal"`
	}{
		Uid:           uid,
		SelectCardVal: cardVal,
	}

	s.sendPackToAll(game.Push, info, onUserSelectShowCard, nil)

	allSelect := true
	s.Players.Range(func(key, value interface{}) bool {
		v := value.(*Player)
		if v.IsReady && v.SelectShowCard == 0 {
			allSelect = false
			return false
		}
		return true
	})
	if allSelect {
		s.SecondRound()
	}

	return nil
}

func (s *Table) getActionBySeatArr(flg bool) {
	tmpMaxCard := 0
	s.Players.Range(func(key, value interface{}) bool {
		v := value.(*Player)
		if v.IsReady && !v.IsUp {
			compareCard := 0
			if flg {
				compareCard = v.SelectShowCard
			} else {
				compareCard = s.uid2HandCards[v.UserID][len(s.uid2HandCards[v.UserID])]
			}
			if tmpMaxCard == 0 {
				tmpMaxCard = compareCard
				s.FirstSeat = v.Seat
			} else {
				if s.Card2Bigger(tmpMaxCard, compareCard) {
					tmpMaxCard = compareCard
					s.FirstSeat = v.Seat
				}
			}
		}
		return true
	})

	s.actionBySeatArr = append(s.actionBySeatArr, s.FirstSeat)
	for i := 1; i <= 4; i++ {
		nextSeat := s.FirstSeat + i
		if nextSeat > 5 {
			nextSeat = nextSeat - 5
		}
		tmp, _ := s.Players.Load(s.seat2Uid[nextSeat])
		tmpPlayer := tmp.(*Player)
		if tmpPlayer.IsReady && !tmpPlayer.IsUp {
			s.actionBySeatArr = append(s.actionBySeatArr, nextSeat)
		}
	}
}

func (s *Table) SecondRound() {
	s.secondRoundTimer.Stop()
	s.GameState.Round = "Second"
	s.Players.Range(func(key, value interface{}) bool {
		v := value.(*Player)
		if v.IsReady && v.SelectShowCard == 0 {
			v.SelectShowCard = s.uid2HandCards[v.UserID][rand.Intn(len(s.uid2HandCards[v.UserID]))]
		}
		return true
	})
	s.getActionBySeatArr(true)
	s.GameState.CurBet = 0
	s.GameState.CurUid = s.seat2Uid[s.FirstSeat]
	s.broadcastGameState()

	s.autoActionTimer = time.AfterFunc(10*time.Second, s.AutoAction)
}

func (s *Table) AutoAction() {
	s.ActionHandle(s.GameState.CurUid, "up", 0)
}

func (s *Table) GetNextSeat(curSeat int) int {
	for i := 1; i <= 4; i++ {
		nextSeat := curSeat + 1
		if nextSeat > 5 {
			nextSeat = nextSeat - 5
		}
		if _, ok := s.seat2Uid[nextSeat]; ok {
			tmp, _ := s.Players.Load(s.seat2Uid[nextSeat])
			pp := tmp.(*Player)
			if pp.IsReady && !pp.IsUp {
				return pp.Seat
			}
		}
	}
	return curSeat
}

func (s *Table) PlayerAddBet(uid string, golds int64) {
	tmp, _ := s.Players.Load(uid)
	tmpPlayer := tmp.(*Player)
	if s.GameState.Round == "SecondRound" {
		if len(tmpPlayer.RoundBet) < 2 {
			tmpPlayer.RoundBet = append(tmpPlayer.RoundBet, golds)
		} else {
			tmpPlayer.RoundBet[1] = tmpPlayer.RoundBet[1] + golds
		}
	}
	if s.GameState.Round == "ThirdRound" {
		if len(tmpPlayer.RoundBet) < 3 {
			tmpPlayer.RoundBet = append(tmpPlayer.RoundBet, golds)
		} else {
			tmpPlayer.RoundBet[2] = tmpPlayer.RoundBet[2] + golds
		}
	}
	if s.GameState.Round == "FourthRound" {
		if len(tmpPlayer.RoundBet) < 4 {
			tmpPlayer.RoundBet = append(tmpPlayer.RoundBet, golds)
		} else {
			tmpPlayer.RoundBet[3] = tmpPlayer.RoundBet[3] + golds
		}
	}
	if s.GameState.Round == "FifthRound" {
		if len(tmpPlayer.RoundBet) < 5 {
			tmpPlayer.RoundBet = append(tmpPlayer.RoundBet, golds)
		} else {
			tmpPlayer.RoundBet[4] = tmpPlayer.RoundBet[4] + golds
		}
	}
}

func (s *Table) ActionHandle(uid, act string, golds int64) {
	info := struct {
		Uid   string
		Act   string
		Golds int64
	}{
		Uid:   uid,
		Act:   act,
		Golds: golds,
	}

	tmp, _ := s.Players.Load(uid)
	tmpPlayer := tmp.(*Player)
	if act == "up" {
		tmpPlayer.IsUp = true
		s.sendPackToAll(game.Push, info, actionPlayerAction, nil)
	} else if act == "add" {
		if tmpPlayer.Golds-golds < 0 || tmpPlayer.RoundMsg.IsAdd {
			s.sendPackToAll(game.Push, "", actionGetRecord, errCode.ErrParams)
		}
		tmpPlayer.Golds = tmpPlayer.Golds - golds
		tmpPlayer.RoundMsg.IsAdd = true
		s.sendPackToAll(game.Push, info, actionPlayerAction, nil)
	} else {
		tmpPlayer.RoundMsg.CurBet = golds
		tmpPlayer.Golds = tmpPlayer.Golds - golds
		s.sendPackToAll(game.Push, info, actionPlayerAction, nil)
	}
	s.GameState.CurBet = golds

	if act != "up" {
		s.PlayerAddBet(uid, golds)
	}

	nextRound := true
	s.Players.Range(func(key, value interface{}) bool {
		v := value.(*Player)
		if !v.IsAllIn && v.RoundMsg.CurBet != s.GameState.CurBet {
			nextRound = false
			return false
		}
		return true
	})

	if nextRound {
		sendCardMsg := struct {
			Uid2Card map[string]int
		}{}
		sendCardMsg.Uid2Card = make(map[string]int)
		for k, _ := range s.uid2HandCards {
			s.uid2HandCards[k] = append(s.uid2HandCards[k], s.getOneCard())
			sendCardMsg.Uid2Card[k] = s.uid2HandCards[k][len(s.uid2HandCards[k])]
		}
		if s.GameState.Round == "FifthRound" {
			s.checkOut()
		} else {
			if s.GameState.Round == "SecondRound" {
				s.GameState.Round = "ThirdRound"
			} else if s.GameState.Round == "ThirdRound" {
				s.GameState.Round = "FourthRound"
			} else if s.GameState.Round == "FourthRound" {
				s.GameState.Round = "FifthRound"
			}
			s.getActionBySeatArr(false)
			s.GameState.CurBet = 0
			s.GameState.CurUid = s.seat2Uid[s.FirstSeat]
			s.broadcastGameState()
			s.sendPackToAll(game.Push, sendCardMsg, "HD_sendCard", nil)
			s.autoActionTimer = time.AfterFunc(10*time.Second, s.AutoAction)
		}
	} else {
		nextSeat := s.GetNextSeat(tmpPlayer.Seat)
		s.GameState.CurUid = s.seat2Uid[nextSeat]
		s.broadcastGameState()
	}
}

func (s *Table) PlayerAction(session gate.Session, msg map[string]interface{}) error {
	uid := session.GetUserID()
	if uid == "" || uid != s.GameState.CurUid {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.ErrParams)
		return nil
	}

	act, ok := msg["act"].(string)
	if !ok {
		s.sendPack(session.GetSessionID(), game.Push, "", actionGetRecord, errCode.ErrParams)
		return nil
	}
	param1, ok := msg["golds"].(float64)
	if !ok {
		s.sendPack(session.GetSessionID(), game.Push, "", actionGetRecord, errCode.ErrParams)
		return nil
	}
	golds := int64(param1)

	s.autoActionTimer.Stop()

	s.ActionHandle(uid, act, golds)

	return nil
}

//初始化每局游戏
func (s *Table) initGame() {
	s.initCards()
	s.uid2HandCards = make(map[string][]int)
	s.Players.Range(func(key, value interface{}) bool {
		v := value.(*Player)
		s.uid2HandCards[v.UserID] = append(s.uid2HandCards[v.UserID], s.getOneCard())
		s.uid2HandCards[v.UserID] = append(s.uid2HandCards[v.UserID], s.getOneCard())

		return true
	})
	s.updateGameConf()
	s.pCards.PlayerCards = []int{}
	s.pCards.BankerCards = []int{}
	s.pCards.PlayerDian = 0
	s.pCards.BankerDian = 0
	s.systemScore = 0
	s.GameState.GameNo = strconv.Itoa(int(storage.NewGlobalId("bjlGameNo") + 100000))
	s.eventID = s.GameState.GameNo

	s.betsLock.Lock()
	s.bets = []suohaStorage.BetDetail{}
	s.betsLock.Unlock()

	s.Players.Range(func(k, val interface{}) bool {
		v := val.(*Player)
		v.Score = 0
		v.TotalBet = 0
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

func (s *Table) GetState() suohaStorage.GameState {
	var tmpPl []suohaStorage.PlayerInfo

	s.Players.Range(func(k, val interface{}) bool {
		v := val.(*Player)
		var tmp = suohaStorage.PlayerInfo{
			Uid:      v.UserID,
			Golds:    v.Golds,
			Head:     v.Head,
			NickName: v.Nickname,
			IsUp:     v.IsUp,
			IsAllIn:  v.IsAllIn,
			IsReady:  v.IsReady,
		}

		tmpPl = append(tmpPl, tmp)
		return true
	})

	s.GameState.RemainTime = 10000 - (utils.GetMillisecond() - s.curStartTime)
	s.GameState.ServerTime = utils.GetMillisecond()
	s.GameState.PlayerInfos = tmpPl

	return s.GameState
}

func (s *Table) GetHistory() []suohaStorage.Record {
	return s.records
}

func (s *Table) GetWinUid() string {
	winUid := ""
	s.Players.Range(func(key, value interface{}) bool {
		v := value.(*Player)
		if !v.IsUp {
			if winUid == "" {
				winUid = v.UserID
			} else {
				if s.CompareTwoPair(s.uid2HandCards[winUid], s.uid2HandCards[v.UserID]) {
					winUid = v.UserID
				}
			}
		}
		return true
	})
	return winUid
}

func (s *Table) checkOut() {
	winUid := s.GetWinUid()
	s.Players.Range(func(k, val interface{}) bool {
		v := val.(*Player)
		if v.IsReady {
			if winUid == v.UserID {
				addRound := 4
				if v.IsUp {
					addRound = v.AllInRound
				}
				for _, arr := range s.uid2RoundBet {
					for round, bet := range arr {
						if round <= addRound {
							v.Score = v.Score + bet
						}
					}
				}
			} else {
				addRound := 4
				if v.IsUp {
					addRound = v.AllInRound
				}
				for round, bet := range s.uid2RoundBet[v.UserID] {
					if round <= addRound {
						v.Score = v.Score - bet
					}
				}
			}
		}
		if v.Score != 0 && !v.robotMsg.IsRobot {
			billType := walletStorage.TypeIncome
			if v.Score < 0 {
				billType = walletStorage.TypeExpenses
			}
			bill := walletStorage.NewBill(v.UserID, billType, walletStorage.EventGameBjl, s.eventID, v.Score)
			walletStorage.OperateVndBalance(bill)
		}
		if !v.robotMsg.IsRobot {
			wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
			v.Golds = wallet.VndBalance
			if v.TotalBet > 0 {
				activityStorage.UpsertGameDataInBet(v.UserID, game.SuoHa, 0)
				activity.CalcEncouragementFunc(v.UserID)
			}
		}

		return true
	})
}

func (s *Table) gameRecord() {
	//var records []suohaStorage.UserGameRecord
	//
	//resultMap := map[string][]int{
	//	"PLAYER_CARD": s.GameState.Cards.PlayerCards,
	//	"BANKER_CARD": s.GameState.Cards.BankerCards,
	//}
	//resultStr, _ := json.Marshal(resultMap)
	//if s.systemScore != 0 {
	//	botProfit := int64(math.Abs(float64(s.systemScore * int64(s.BotProfitPerThousand) / 1000)))
	//	gameStorage.IncProfit("", game.SuoHa, 0, s.systemScore-botProfit, botProfit)
	//	gameProfit := gameStorage.QueryProfit(game.SuoHa)
	//	s.botBalance = gameProfit.BotBalance
	//}
	//
	//s.Players.Range(func(k, val interface{}) bool {
	//	v := val.(*Player)
	//	if v.TotalBet > 0 && !v.robotMsg.IsRobot {
	//		betDetails := map[string]int64{
	//			"PLAYER":      v.BetInfo[0],
	//			"BANKER":      v.BetInfo[1],
	//			"HE":          v.BetInfo[2],
	//			"PLAYER_PAIR": v.BetInfo[3],
	//			"BANKER_PAIR": v.BetInfo[4],
	//		}
	//		betDetailsStr, _ := json.Marshal(betDetails)
	//		wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
	//		var recordParams gameStorage.BetRecordParam
	//		recordParams.Uid = v.UserID
	//		recordParams.GameNo = s.eventID
	//		recordParams.BetAmount = v.TotalBet
	//		recordParams.BotProfit = 0
	//		recordParams.SysProfit = 0
	//		recordParams.BetDetails = string(betDetailsStr)
	//		recordParams.GameResult = string(resultStr)
	//		recordParams.CurBalance = v.Golds + wallet.SafeBalance
	//		recordParams.GameType = game.SuoHa
	//		recordParams.Income = v.Score - v.TotalBet
	//		recordParams.IsSettled = true
	//		gameStorage.InsertBetRecord(recordParams)
	//
	//		createTime := time.Now().Unix()
	//		for pos, betValue := range v.BetInfo {
	//			if betValue > 0 {
	//				tmpScore := -betValue
	//				if s.GameState.PosRes[pos] {
	//					tmpScore = (betValue * s.oddList[pos]) / 100
	//				}
	//				tmpRecord := suohaStorage.UserGameRecord{
	//					Uid:        v.UserID,
	//					GameNo:     s.GameState.GameNo,
	//					Pos:        pos,
	//					BetCount:   betValue,
	//					Score:      tmpScore,
	//					PlayerDian: s.pCards.PlayerDian,
	//					BankerDian: s.pCards.BankerDian,
	//					CreateTime: createTime,
	//				}
	//				records = append(records, tmpRecord)
	//			}
	//		}
	//	}
	//
	//	v.Score = 0
	//	v.TotalBet = 0
	//	return true
	//})
	//
	//if len(records) > 0 {
	//	suohaStorage.InsertRecord(records)
	//}
}

//玩家加入桌子
func (s *Table) SitDown(session gate.Session, base, carryGolds int64) (suohaStorage.GameState, error) {
	userID := session.GetUserID()
	if userID == "" {
		log.Info("your userid is empty")
		return s.GameState, errors.New("empty uid")
	}

	if s.Base == 0 {
		s.Base = base
		s.GameState.Base = s.Base
		s.GameState.State = "ready"
	}

	if len(s.players) >= MaxPlayer {
		return s.GameState, errors.New("There are no vacant seats")
	}

	bExist := false
	tmp, _ := s.Players.Load(userID)
	if tmp != nil {
		bExist = true
	}
	if bExist {
		tmpPlayer := tmp.(*Player)
		tmpPlayer.IsOnline = true
	} else {
		user := userStorage.QueryUserId(utils.ConvertOID(userID))

		tmpInfo := map[string]interface{}{
			"userID": userID,
			"golds":  carryGolds,
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
	s.GetState()
	return s.GameState, nil
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

	records := suohaStorage.GetUserRecord(uid, offset, limit)
	s.sendPack(session.GetSessionID(), game.Push, records, actionGetRecord, nil)

	return nil
}

func (s *Table) broadcastGameState() {
	s.GetState()
	s.sendPackToAll(game.Push, s.GameState, "HD_changeState", nil)
}

func (s *Table) onBet() {
	s.initGame()
	s.broadcastGameState()
}

func (s *Table) getOneCard() int {
	idx := rand.Intn(len(s.cardPool))
	card := s.cardPool[idx]
	s.cardPool = append(s.cardPool[:idx], s.cardPool[idx+1:]...)
	return card
}

func (s *Table) checkPosRes() {
}

func (s *Table) getSystemScore() int64 {
	s.checkPosRes()
	s.systemScore = 0
	return s.systemScore
}

func (s *Table) ChargeGolds(session gate.Session, msg map[string]interface{}) error {
	uid := session.GetUserID()
	tmp, ok := s.Players.Load(uid)
	if !ok {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.ErrParams)
		return nil
	}
	tmpPlayer := tmp.(*Player)

	param1, ok := msg["golds"].(float64)
	if !ok {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.ErrParams)
		return nil
	}
	golds := int64(param1)
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	if golds <= 0 || wallet.VndBalance-tmpPlayer.Golds < golds || tmpPlayer.Golds+golds > s.tableConf.MaxEnter {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.Illegal)
		return nil
	}
	tmpPlayer.Golds = tmpPlayer.Golds + golds

	info := struct {
		Uid      string
		AddGolds int64
	}{
		Uid:      uid,
		AddGolds: golds,
	}
	s.sendPackToAll(game.Push, info, onUserAddGolds, nil)

	return nil
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
	s.sendPackToAll(game.Push, msg, protocol.SendShortCut, nil)
	return nil
}
