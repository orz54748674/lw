package fish

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	"vn/game/fish/fishConf"
	"vn/storage"
	"vn/storage/fishStorage"
	"vn/storage/gameStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandInt64(min, max int64) int64 {
	if min >= max {
		return max
	}
	return rand.Int63n(max-min) + min
}

func RandInt(min, max int) int {
	if min >= max {
		return max
	}
	return rand.Intn(max-min) + min
}

type Table struct {
	room.QTable
	module module.RPCModule
	app    module.App

	tableID          string
	serverID         string
	seatMax          int //房间最大座位数
	stopped          bool
	writelock        sync.Mutex
	cannonTypes      []int64 //炮台种类
	cannonGolds      []int64
	baseGold         int
	Players          map[string]*Player
	AllFish          []fishStorage.Fish
	playerFireAmount map[string]int64
	TideInfo         fishStorage.TideInfo
	GroupInfo        fishStorage.GroupInfo
	GroupLastTime    []int64
	maxFishCount     int
	curFishCount     int
	curFishID        int
	botBalance       int64
	tableType        int
	fishConf         fishStorage.FishConf
	playerNum        int8
	seatArr          []int8
	bInFishGroup     bool
	fishLock         sync.Mutex
	fishIDLock       sync.Mutex
	longPathArr      []int
	longWangPathArr  []int
	sceneID          int
	dayStr           string
	blockRate        float64
	lastLongWangCreateTime int64
	effectBet int64
}

var (
	LongID     = 125
	LeiSheID   = 201
	DianZuanID = 202
	ZhaDanID   = 203
	LunZhouID  = 204
	ShanDianID = 205
	LeiTingID  = 206
	LongWangID = 207
	FuDaiID    = 208
	FishCountBase = 30
)

func (s *Table) GetModule() module.RPCModule {
	return s.module
}

func (s *Table) GetSeats() map[string]room.BasePlayer {
	m := make(map[string]room.BasePlayer)
	for k, v := range s.Players {
		m[k] = v
	}
	return m
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
	s.OnInit(s, opts...)

	s.TableInit(module, app, tableID)

	s.Register("SitDown", s.SitDown)
	s.Register("PlayerFire", s.PlayerFire)
	s.Register("KillFish", s.KillFish)
	s.Register("ChangeCannon", s.ChangeCannon)
	s.Register("PlayerLeave", s.PlayerLeave)
	s.Register("SpecialKillFish", s.SpecialKillFish)
	s.Register("ChangeSeat", s.ChangeSeat)

	return s
}

func (s *Table) initConf() {
	s.fishConf = fishStorage.GetFishConf()
	s.longPathArr = []int{16004, 16014, 16105, 16115, 16204, 16205, 16215}
	s.longWangPathArr = []int{2902, 3003, 3101, 5101, 5201, 14001, 14006}
	s.dayStr = time.Now().Format("2006-01-02")
	s.blockRate = 0.3
	s.botBalance, s.effectBet = fishStorage.GetFishSysBalance(s.tableType)
}

func (s *Table) TableInit(module module.RPCModule, app module.App, tableID string) {
	s.GroupLastTime = []int64{200000, 300000, 400000}
	s.maxFishCount = 30
	s.curFishID = 0
	s.Players = make(map[string]*Player)
	s.playerFireAmount = make(map[string]int64)
	s.seatArr = []int8{0, 1, 2, 3}
	s.sceneID = 1
	s.initConf()
	s.Run()

	go func() {
		s.bInFishGroup = true
		endTime := utils.GetMillisecond() + s.GroupLastTime[rand.Intn(len(s.GroupLastTime))]
		for {
			s.createFish()
			time.Sleep(1 * time.Second)
			if utils.GetMillisecond() >= endTime {
				if s.bInFishGroup {
					s.sceneID += 1
					if s.sceneID > 3 {
						s.sceneID = 1
					}
					s.initFishTide()
					endTime = s.TideInfo.EndTime
				} else {
					s.initFishGroup()
					endTime = s.GroupInfo.EndTime
				}
				s.bInFishGroup = !s.bInFishGroup
			}
			if time.Now().Format("2006-01-02") != s.dayStr {
				s.InitPlayerFireTimes()
				s.dayStr = time.Now().Format("2006-01-02")
			}
		}
	}()
	go func() {
		for {
			s.deleteExpireFish()
			time.Sleep(20 * time.Second)
		}
	}()
	go s.checkOutAndRecord()

}

func (s *Table) InitPlayerFireTimes() {
	for _, player := range s.Players {
		player.fireTimes = 0
		player.UpsertFireTimes(s.tableType)
	}
}

func (s *Table) AllowJoin() bool {
	s.writelock.Lock()
	defer s.writelock.Unlock()
	if s.playerNum >= 4 {
		return false
	}
	s.playerNum++
	s.maxFishCount = FishCountBase * int(s.playerNum)
	return true
}

func (s *Table) OnCreate() {
	//可以加载数据
	s.QTable.OnCreate()
}

func (s *Table) OnDestroy() {
	s.BaseTableImp.OnDestroy()
	s.stopped = true
}

func (self *Table) onGameOver() {
	self.Finish()
}

func (s *Table) GetState() fishStorage.RoomState {
	var tmpPl []fishStorage.PlayerInfo
	for _, v := range s.Players {
		var tmp = fishStorage.PlayerInfo{
			Uid:         v.UserID,
			Golds:       v.Golds,
			Head:        v.Head,
			Name:        v.Name,
			Seat:        v.Seat,
			CannonType:  int8(v.CannonType),
			CannonGolds: v.CannonGolds,
		}
		tmpPl = append(tmpPl, tmp)
	}
	tmpState := fishStorage.RoomState{
		ServerTime: utils.GetMillisecond(),
		Players:    tmpPl,
		Fishs:      s.AllFish,
		Scene:      s.sceneID,
	}

	return tmpState
}

func (s *Table) addFish(fish []fishStorage.Fish) {
	s.fishLock.Lock()
	for _, v := range fish {
		s.AllFish = append(s.AllFish, v)
	}
	s.fishLock.Unlock()
}

func (s *Table) deleteFish(fishID int) {
	s.fishLock.Lock()
	for k, v := range s.AllFish {
		if v.FishID == fishID {
			s.AllFish = append(s.AllFish[:k], s.AllFish[k+1:]...)
			break
		}
	}
	s.fishLock.Unlock()
}

func (s *Table) findFish(fishId int) (fishStorage.Fish, bool) {
	s.fishLock.Lock()
	defer s.fishLock.Unlock()
	for _, v := range s.AllFish {
		if v.FishID == fishId {
			return v, true
		}
	}
	fmt.Println("not found fish id:", fishId)
	return fishStorage.Fish{}, false
}

//初始化鱼群
func (s *Table) initFishGroup() {
	s.GroupInfo.StartTime = utils.GetMillisecond()
	s.GroupInfo.EndTime = s.GroupInfo.StartTime + s.GroupLastTime[rand.Intn(len(s.GroupLastTime))]
}

//广播鱼潮信息
func (s *Table) onFishTide() {
	var fishs []fishStorage.Fish
	interv := utils.GetMillisecond() - s.TideInfo.StartTime
	fishType := 0
	var delID []int

	for _, v := range s.TideInfo.Fishs {
		if v.Delay < int(interv) {
			fishType = v.FishType
			break
		}
	}

	if fishType == 0 {
		return
	}

	for k, v := range s.TideInfo.Fishs {
		if fishType == v.FishType {
			fish := fishStorage.Fish{
				FishID:    s.getFishID(),
				FishType:  v.FishType,
				Path:      v.Path,
				LiveTime:  v.LiveTime,
				StartTime: utils.GetMillisecond() + int64(v.Delay) - interv,
			}
			fishs = append(fishs, fish)
			delID = append(delID, k)
		}
	}

	if len(fishs) > 0 {
		s.addFish(fishs)

		fishsInfo := fishStorage.OnFish{
			Fishs:      fishs,
			ServerTime: utils.GetMillisecond(),
		}
		s.sendPackToAll(game.Push, fishsInfo, protocol.FishTide, nil)

		//	删除元素
		var tmpFishs []fishConf.FishTideKindConf
		for k, v := range s.TideInfo.Fishs {
			if !utils.IsContainInt(delID, k) {
				tmpFishs = append(tmpFishs, v)
			}
		}
		s.TideInfo.Fishs = tmpFishs
	}
}

//生成鱼潮信息
func (s *Table) initFishTide() {
	tmpRandNum := rand.Intn(len(fishConf.FishTide))
	s.TideInfo.StartTime = utils.GetMillisecond()
	s.TideInfo.EndTime = s.TideInfo.StartTime + fishConf.FishTide[tmpRandNum].Time
	s.TideInfo.Fishs = fishConf.FishTide[tmpRandNum].Kind
	s.AllFish = []fishStorage.Fish{}

	info := struct {
		SceneID int
	}{
		SceneID: s.sceneID,
	}

	s.sendPackToAll(game.Push, info, protocol.FishTideCome, nil)
}

func (s *Table) getFishID() int {
	s.fishIDLock.Lock()
	s.fishIDLock.Unlock()
	s.curFishID = s.curFishID + 1
	if s.curFishID > 2000000 {
		s.curFishID = 1
	}
	return s.curFishID
}

//生成鱼群
func (s *Table) onFishGroup() {
	var fishs []fishStorage.Fish
	randNum := rand.Intn(len(fishConf.FishGroupKindConf))
	var fishGroupKind = fishConf.FishGroupKindConf[randNum]
	fishType := fishConf.FishTypeConf[fishGroupKind.FishType]

	fishNum := int(RandInt64(int64(fishGroupKind.MinFollowCount), int64(fishGroupKind.MaxFollowCount)))

	for i := 0; i < fishNum; i++ {
		fish := fishStorage.Fish{
			FishID:    s.getFishID(),
			FishType:  fishType.ID,
			Path:      fishConf.FishPath[rand.Intn(len(fishConf.FishPath))],
			LiveTime:  fishType.Time,
			StartTime: int64(i*fishType.Interval) + utils.GetMillisecond(),
		}
		if fishType.ID == LongWangID || fishType.ID == FuDaiID {
			fish.Path = s.longWangPathArr[rand.Intn(len(s.longWangPathArr))]
			if fishType.ID == LongWangID {
				if (time.Now().Unix() - s.lastLongWangCreateTime) < 120 {
					continue
				} else {
					s.lastLongWangCreateTime = time.Now().Unix()
				}
			}
		}
		if fishType.ID == LongID {
			fish.Path = s.longPathArr[rand.Intn(len(s.longPathArr))]
		}
		fishs = append(fishs, fish)
	}

	fishsInfo := fishStorage.OnFish{
		Fishs:      fishs,
		ServerTime: utils.GetMillisecond(),
	}

	s.addFish(fishs)

	if fishNum > 0 {
		s.sendPackToAll(game.Push, fishsInfo, protocol.FishGroup, nil)
	}
}

//删除过期的鱼
func (s *Table) deleteExpireFish() {
	s.fishLock.Lock()
	defer s.fishLock.Unlock()

	var tmpFish []fishStorage.Fish
	nowTime := utils.GetMillisecond()
	for _, v := range s.AllFish {
		expireTime := v.StartTime + int64(v.LiveTime) + 2000
		if nowTime <= expireTime {
			tmpFish = append(tmpFish, v)
		} else {
			//fmt.Println("delete fish........", v)
		}
	}

	s.AllFish = tmpFish
}

func (s *Table) createFish() {
	if s.bInFishGroup {
		s.onFishGroup()
	} else {
		s.onFishTide()
	}
}

func (s *Table) checkOutAndRecord() {
	for {
		time.Sleep(10 * time.Second)
		s.checkOutAndRecordHandle("")
		//s.gameRecord("")
		s.initConf()
		s.botBalance, s.effectBet = fishStorage.GetFishSysBalance(s.tableType)
	}
}

func (s *Table) checkOutAndRecordHandle(userID string) {
	eventID := "Fish" + "-" + s.tableID + "-" + strconv.Itoa(int(time.Now().Unix()))
	for _, v := range s.Players {
		if userID == "" || (userID == v.UserID) {
			//if userID == v.UserID {
			//	v.Score = v.Score + v.FireAmount
			//}
			tmpScore := v.Score
			if tmpScore != 0 {
				billType := walletStorage.TypeIncome
				if tmpScore < 0 {
					billType = walletStorage.TypeExpenses
				}
				bill := walletStorage.NewBill(v.UserID, billType, walletStorage.EventGameFish, eventID, tmpScore)
				walletStorage.OperateVndBalanceV1(bill)

				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
				var recordParams gameStorage.BetRecordParam
				recordParams.Uid = v.UserID
				recordParams.GameNo = strconv.Itoa(int(storage.NewGlobalId("fishGameNo") + 100000))
				recordParams.BetAmount = v.TotalBet
				recordParams.BotProfit = 0
				recordParams.SysProfit = 0
				recordParams.BetDetails = ""
				recordParams.GameResult = ""
				recordParams.CurBalance = wallet.VndBalance
				recordParams.GameType = game.Fish
				recordParams.Income = v.Score
				recordParams.IsSettled = true
				gameStorage.InsertBetRecord(recordParams)
				if v.UserType == userStorage.TypeNormal {
					fishStorage.UpsertFishSysBalance(s.tableType, -tmpScore, v.TotalBet)
					s.botBalance -= tmpScore
					s.effectBet += v.TotalBet
				}
				v.UpdateScore(-tmpScore)
			}
			if v.TotalBet > 0 {
				fishStorage.UpsertFireTimes(v.UserID, s.tableType, v.fireTimes)
				v.TotalBet = 0
			}
		}
	}
}

//func (s *Table) gameRecord(userID string) {
//	for _, v := range s.Players {
//		if userID == "" || userID == v.UserID {
//			//if userID == v.UserID {
//			//	v.Score = v.Score + v.FireAmount
//			//}
//			if v.Score != 0 {
//				if v.Score < 0 {
//				} else if v.Score > 0 {
//				}
//				wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
//				var recordParams gameStorage.BetRecordParam
//				recordParams.Uid = v.UserID
//				recordParams.GameNo = strconv.Itoa(int(storage.NewGlobalId("fishGameNo") + 100000))
//				recordParams.BetAmount = v.TotalBet
//				recordParams.BotProfit = 0
//				recordParams.SysProfit = 0
//				recordParams.BetDetails = ""
//				recordParams.GameResult = ""
//				recordParams.CurBalance = v.Golds + wallet.SafeBalance
//				recordParams.GameType = game.Fish
//				recordParams.Income = v.Score
//				recordParams.IsSettled = true
//				gameStorage.InsertBetRecord(recordParams)
//				fishStorage.UpsertFishSysBalance(s.tableType, -v.Score)
//				s.botBalance -= v.Score
//				v.Score = 0
//				v.TotalBet = 0
//			}
//
//			fishStorage.UpsertFireTimes(v.UserID, s.tableType, v.fireTimes)
//		}
//	}
//}

func (s *Table) PlayerDisconnect(userID string) {
	var pl = s.Players[userID]
	if pl != nil {
		if pl.Score != 0 {
			s.checkOutAndRecordHandle(pl.UserID)
			//s.gameRecord(pl.UserID)
		}
		s.playerNum--
		s.maxFishCount = FishCountBase * int(s.playerNum)
		s.seatArr = append(s.seatArr, pl.Seat)
		delete(s.Players, userID)
	}
}
