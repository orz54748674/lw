package guessBigSmall

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"
	"vn/common/errCode"
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
	"vn/storage/gbsStorage"
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
	writelock        sync.Mutex
	players          map[string]room.BasePlayer
	botBalance       int64
	GameState        gbsStorage.GameState
	bCheckout        bool
	eventID          string
	selectActionList []string
	IsInGame         bool

	//配置信息
	gameChip    []int64
	allCards    []int
	poolVal     map[int64]int64
	mingPercent map[int64]int64
	anPercent   map[int64]int64
}

func (s *Table) GetModule() module.RPCModule {
	return s.module
}

func (this *Table) GetSeats() map[string]room.BasePlayer {
	return this.players
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
	s.Run()
	s.initTable()

	return s
}

func (s *Table) initTable() {
	s.poolVal = make(map[int64]int64)
	s.mingPercent = make(map[int64]int64)
	s.anPercent = make(map[int64]int64)
	gameConf := gbsStorage.GetGameConf()
	for _, v := range gameConf {
		s.GameState.ChipList = append(s.GameState.ChipList, v.Chip)
		s.poolVal[v.Chip] = v.PoolVal
		s.gameChip = append(s.gameChip, v.Chip)
		s.mingPercent[v.Chip] = v.MingPercent
		s.anPercent[v.Chip] = v.AnPercent
	}
}

func (s *Table) initGame() bool {
	reboot := gameStorage.QueryGameReboot(game.GuessBigSmall)
	if reboot == "true" {
		//游戏当局结束后停止发牌
		return true
	}
	s.GameState.ShowCards = []int{}
	s.bCheckout = false
	s.GameState.Round = 0
	s.GameState.AList = []int{}
	s.GameState.CurCard = 0
	s.selectActionList = []string{}

	s.allCards = []int{}
	for i := 1; i <= 52; i++ {
		s.allCards = append(s.allCards, i)
	}
	return false
}

func (s *Table) getOneCard() int {
	tmpIdx := rand.Intn(len(s.allCards))
	tmpCards := s.allCards[tmpIdx]
	s.allCards = append(s.allCards[:tmpIdx], s.allCards[tmpIdx+1:]...)
	return tmpCards
}

func (s *Table) getCardValue(card int) int {
	value := card % 13
	if value == 1 {
		value = 14
	}
	if value == 0 {
		value = 13
	}
	return value
}

func (s *Table) getReward() (biggerReward, smallerReward int64) {
	curValue := s.getCardValue(s.GameState.CurCard)
	biggerCount := 0
	smallerCount := 0
	for _, v := range s.allCards {
		if s.getCardValue(v) > curValue {
			biggerCount = biggerCount + 1
		} else if s.getCardValue(v) < curValue {
			smallerCount = smallerCount + 1
		}
	}

	if curValue == 2 || curValue == 14 {
		return s.GameState.CurGolds, s.GameState.CurGolds
	}

	biggerReward = s.GameState.CurGolds * int64(52-len(s.GameState.ShowCards)) / int64(biggerCount)
	smallerReward = s.GameState.CurGolds * int64(52-len(s.GameState.ShowCards)) / int64(smallerCount)
	return
}

func (s *Table) Start(session gate.Session, msg map[string]interface{}) error {

	uid := session.GetUserID()
	param1, ok := msg["chip"].(float64)
	if !ok {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", actionStart, error)
		return nil
	}

	rebootFlg := s.initGame()
	if rebootFlg {
		error := errCode.ServerBusy
		s.sendPack(session.GetSessionID(), game.Push, "", actionStart, error)
		return nil
	}

	chip := int64(param1)
	if !utils.IsContainInt64(s.GameState.ChipList, chip) {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", actionStart, error)
		return nil
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	gData := gameStorage.QueryGameCommonData(uid)
	if (wallet.VndBalance - gData.InRoomNeedVnd) < s.GameState.SelectChip {
		error := errCode.BalanceNotEnough
		s.sendPack(session.GetSessionID(), game.Push, "", actionStart, error)
		return nil
	}
	gameProfit := gameStorage.QueryProfit(game.GuessBigSmall)
	s.botBalance = gameProfit.BotBalance

	s.selectActionList = append(s.selectActionList, "start")
	s.eventID = strconv.Itoa(int(storage.NewGlobalId("guessGameNo") + 100000))

	player := &room.BasePlayerImp{}
	player.Bind(session)
	s.players[uid] = player

	s.GameState.SelectChip = chip
	s.GameState.CurGolds = s.GameState.SelectChip
	s.GameState.CurCard = s.getOneCard()
	s.GameState.ShowCards = append(s.GameState.ShowCards, s.GameState.CurCard)
	s.GameState.BiggerReward, s.GameState.SmallerReward = s.getReward()
	s.GameState.RemainTime = 120
	s.GameState.StartTime = time.Now().Unix()
	s.GameState.EndTime = time.Now().Unix() + 120
	s.GameState.CurTime = time.Now().Unix()
	s.GameState.BWin = true
	s.IsInGame = true
	if s.getCardValue(s.GameState.CurCard) == 14 {
		s.GameState.AList = append(s.GameState.AList, s.GameState.CurCard)
	}

	go s.AutoCheckout(uid)

	bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventGameGuessBs, s.eventID, -s.GameState.SelectChip)
	walletStorage.OperateVndBalanceV1(bill)
	activityStorage.UpsertGameDataInBet(uid, game.GuessBigSmall, 1)

	err := s.sendPack(session.GetSessionID(), game.Push, s.GameState, actionStart, nil)
	if err != nil {
		log.Info("start err:", err.Error())
	}
	return err
}

func (s *Table) AutoCheckout(uid string) {
	for {
		time.Sleep(1 * time.Second)
		s.GameState.RemainTime -= 1
		if s.GameState.RemainTime == 0 && !s.bCheckout {
			info := struct {
				Score int64 `json:"score"`
			}{
				Score: 0,
			}
			if len(s.GameState.ShowCards) == 1 {
				s.GameState.BWin = false
			}
			if s.GameState.BWin {
				info.Score = s.GameState.CurGolds
			}
			s.selectActionList = append(s.selectActionList, "timeout")
			s.IsInGame = false
			s.checkout(uid)
			s.sendPackToAll(game.Push, info, actionStop, nil)
			break
		}
	}
}

func (s *Table) handleGetCard(strSelect string) int {
	cardsLen := len(s.allCards)
	for i := 1; i <= 10000; i++ {
		tmpIdx := rand.Intn(cardsLen)
		newCard := s.allCards[tmpIdx]
		score := int64(0)
		if s.getCardValue(newCard) == 14 && len(s.GameState.AList)+1 == 3 {
			score = score + s.poolVal[s.GameState.SelectChip]
		}
		if strSelect == "big" && s.getCardValue(newCard) > s.getCardValue(s.GameState.CurCard) {
			score = (s.GameState.BiggerReward - s.GameState.SelectChip) + score
		} else if strSelect == "small" && s.getCardValue(newCard) < s.getCardValue(s.GameState.CurCard) {
			score = (s.GameState.SmallerReward - s.GameState.SelectChip) + score
		} else if s.getCardValue(newCard) == s.getCardValue(s.GameState.CurCard) {
			score = s.GameState.CurGolds/10*9 - s.GameState.SelectChip + score
		} else {
			score = -s.GameState.SelectChip + score
		}
		if s.botBalance-score > 0 || i == 10000 {
			return tmpIdx
		}
	}

	return rand.Intn(cardsLen)
}

func (s *Table) SelectBigOrSmall(session gate.Session, msg map[string]interface{}) error {
	uid := session.GetUserID()
	strSelect, ok := msg["select"].(string)
	if !ok {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", actionSelectBigOrSmall, error)
		return nil
	}
	if strSelect != "big" && strSelect != "small" {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", actionSelectBigOrSmall, error)
		return nil
	}

	if !s.GameState.BWin {
		error := errCode.Illegal
		s.sendPack(session.GetSessionID(), game.Push, "", actionSelectBigOrSmall, error)
		return nil
	}

	s.selectActionList = append(s.selectActionList, strSelect)

	tmpCardIdx := s.handleGetCard(strSelect)
	newCard := s.allCards[tmpCardIdx]
	s.allCards = append(s.allCards[:tmpCardIdx], s.allCards[tmpCardIdx+1:]...)

	if s.getCardValue(newCard) == 14 {
		s.GameState.AList = append(s.GameState.AList, newCard)
		if len(s.GameState.AList) == 3 {
			s.PoolReward(uid)
		}
	}
	if strSelect == "big" && s.getCardValue(newCard) > s.getCardValue(s.GameState.CurCard) {
		s.GameState.CurCard = newCard
		s.GameState.CurGolds = s.GameState.BiggerReward
		s.GameState.BWin = true
	} else if strSelect == "small" && s.getCardValue(newCard) < s.getCardValue(s.GameState.CurCard) {
		s.GameState.CurCard = newCard
		s.GameState.CurGolds = s.GameState.SmallerReward
		s.GameState.BWin = true
	} else if s.getCardValue(newCard) == s.getCardValue(s.GameState.CurCard) {
		s.GameState.CurCard = newCard
		s.GameState.CurGolds = s.GameState.CurGolds / 10 * 9
		s.GameState.BWin = true
	} else {
		s.GameState.CurCard = newCard
		s.GameState.BWin = false
		s.IsInGame = false
	}
	s.GameState.ShowCards = append(s.GameState.ShowCards, s.GameState.CurCard)

	if s.GameState.BWin {
		s.GameState.BiggerReward, s.GameState.SmallerReward = s.getReward()
		s.GameState.Round = s.GameState.Round + 1
		s.GameState.RemainTime = 120
		s.GameState.StartTime = time.Now().Unix()
		s.GameState.EndTime = time.Now().Unix() + 120
		s.GameState.CurTime = time.Now().Unix()
	} else {
		s.checkout(uid)
	}

	s.sendPack(session.GetSessionID(), game.Push, s.GameState, actionSelectBigOrSmall, nil)
	return nil
}

func (s *Table) PoolReward(uid string) {
	reward := s.poolVal[s.GameState.SelectChip]
	bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventGameGuessBs, s.eventID, reward)
	walletStorage.OperateVndBalanceV1(bill)

	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	betDetail, _ := json.Marshal(s.selectActionList)
	betDetailStr := string(betDetail)
	gameResByte, _ := json.Marshal(s.GameState.ShowCards)
	gameResStr := string(gameResByte)
	var recordParams gameStorage.BetRecordParam
	recordParams.Uid = uid
	recordParams.GameNo = s.eventID
	recordParams.BetAmount = s.GameState.SelectChip
	recordParams.BotProfit = 0
	recordParams.SysProfit = 0
	recordParams.BetDetails = betDetailStr
	recordParams.GameResult = gameResStr
	recordParams.CurBalance = wallet.VndBalance + wallet.SafeBalance
	recordParams.GameType = game.GuessBigSmall
	recordParams.Income = reward
	recordParams.IsSettled = true
	gameStorage.InsertBetRecord(recordParams)

	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	var record gbsStorage.GbsPoolRewardRecord
	record.CreateTime = time.Now().Unix()
	record.SelectChip = s.GameState.SelectChip
	record.Nickname = user.NickName
	record.Reward = reward
	gbsStorage.InsertPoolRewardRecord(record)
	updatePoolVal := -reward + 20*record.SelectChip
	gbsStorage.UpsertPoolVal(s.GameState.SelectChip, updatePoolVal)
	s.module.InvokeNR(string(game.GuessBigSmall), "UpdatePoolVal", s.GameState.SelectChip, updatePoolVal)
}

func (s *Table) Stop(session gate.Session, msg map[string]interface{}) error {
	if !s.GameState.BWin {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", actionStop, error)
		return nil
	} else {
		info := struct {
			Score int64 `json:"score"`
		}{
			Score: s.GameState.CurGolds,
		}
		s.sendPack(session.GetSessionID(), game.Push, info, actionStop, nil)
	}

	s.selectActionList = append(s.selectActionList, "stop")
	s.IsInGame = false
	s.checkout(session.GetUserID())

	return nil
}

func (s *Table) GetServerTime(session gate.Session, msg map[string]interface{}) error {
	info := struct {
		ServerTime int64 `json:"serverTime"`
	}{
		ServerTime: time.Now().Unix(),
	}
	s.sendPack(session.GetSessionID(), game.Push, info, actionGetServerTime, nil)
	return nil
}

func (s *Table) insertRecord(uid string) {
	var record gbsStorage.GbsRecord
	record.Uid = uid
	record.GameNo = s.eventID
	record.BWin = s.GameState.BWin
	record.SelectChip = s.GameState.SelectChip
	record.Score = -s.GameState.SelectChip
	if s.GameState.BWin {
		record.Score = s.GameState.CurGolds - s.GameState.SelectChip
	}
	record.CreateTime = time.Now().Unix()
	gbsStorage.InsertRecord(record)
}

func (s *Table) checkout(uid string) {
	s.bCheckout = true
	score := int64(0)
	billType := walletStorage.TypeIncome
	var mingProfit, anProfit int64
	if s.GameState.BWin {
		score = s.GameState.CurGolds * 98 / 100
		bill := walletStorage.NewBill(uid, billType, walletStorage.EventGameGuessBs, s.eventID, score)
		walletStorage.OperateVndBalanceV1(bill)
		mingProfit = s.GameState.CurGolds * s.mingPercent[s.GameState.SelectChip] / 1000
		anProfit = s.GameState.CurGolds * s.anPercent[s.GameState.SelectChip] / 1000
	} else {
		mingProfit = 0
		anProfit = s.GameState.SelectChip * s.anPercent[s.GameState.SelectChip] / 1000
		gbsStorage.UpsertPoolVal(s.GameState.SelectChip, s.GameState.SelectChip/1000*3)
		s.module.InvokeNR(string(game.GuessBigSmall), "UpdatePoolVal", s.GameState.SelectChip, s.GameState.SelectChip/1000*3)
	}
	s.insertRecord(uid)
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	recordScore := -s.GameState.SelectChip
	if s.GameState.BWin {
		recordScore = score - s.GameState.SelectChip
	}

	gameStorage.IncProfit(uid, game.GuessBigSmall, mingProfit, -(recordScore + anProfit), anProfit)

	betDetail, _ := json.Marshal(s.selectActionList)
	betDetailStr := string(betDetail)
	gameResByte, _ := json.Marshal(s.GameState.ShowCards)
	gameResStr := string(gameResByte)
	var recordParams gameStorage.BetRecordParam
	recordParams.Uid = uid
	recordParams.GameNo = s.eventID
	recordParams.BetAmount = s.GameState.SelectChip
	recordParams.BotProfit = anProfit
	recordParams.SysProfit = mingProfit
	recordParams.BetDetails = betDetailStr
	recordParams.GameResult = gameResStr
	recordParams.CurBalance = wallet.VndBalance + wallet.SafeBalance
	recordParams.GameType = game.GuessBigSmall
	recordParams.Income = recordScore
	recordParams.IsSettled = true
	gameStorage.InsertBetRecord(recordParams)
	s.GameState.BWin = false

	activityStorage.UpsertGameDataInBet(uid, game.GuessBigSmall, 0)
	activity.CalcEncouragementFunc(uid)
}

func (s *Table) UpdatePoolVal(chip, val int64) {
	fmt.Println("chip.............val", chip, val)
	s.poolVal[chip] = s.poolVal[chip] + val
	var poolConf []gbsStorage.GameConf
	for _, v := range s.gameChip {
		tmp := gbsStorage.GameConf{}
		tmp.Chip = v
		tmp.PoolVal = s.poolVal[v]
		poolConf = append(poolConf, tmp)
	}
	s.sendPackToAll(game.Push, poolConf, actionUpdatePoolVal, nil)
}
