package cardLhd

import (
	"github.com/robfig/cron"
	"math/rand"
	"strconv"
	"time"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant-modules/room"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/cardStorage/cardLhdStorage"
)

func (this *MyTable) ClearTable() { //
	cardLhdStorage.RemoveTableInfo(this.tableID)

	myRoom := (this.module).(*Room)
	myRoom.DestroyTable(this.tableID)
}
func (this *MyTable) OnTimer() { //定时任务
	if !this.SeqExecFlag {
		return
	}
	pl := this.DeepCopyPlayerList(this.PlayerList)
	//log.Info("---------------- count down = %d",this.CountDown)
	if this.RoomState == ROOM_END && this.SeqExecFlag {
		this.SeqExecFlag = false
		for _, v := range pl {
			if v.Role == USER || v.Role == Agent {
				sb := vGate.QuerySessionBean(v.UserID)
				if sb != nil {
					session, _ := basegate.NewSession(this.app, sb.Session)
					erro := this.PutQueue(protocol.QuitTable, session, v.UserID)
					if erro != nil {
						log.Info("--------------- table.PutQueue error---tableID ---error = %s", erro)
					}
				}
			}
		}
		erro := this.PutQueue(protocol.ClearTable)
		if erro != nil {
			log.Info("--------------- table.PutQueue error---tableID ---error = %s", erro)
		}
		return
	}
	if this.RoomState == ROOM_WAITING_START && this.SeqExecFlag {
		this.SeqExecFlag = false
		this.RobotAdd(true)
		this.PutQueue(protocol.UpdatePlayerList)
		info := make(map[string]interface{})
		info["PlayerNum"] = this.PlayerNum
		this.sendPackToAll(game.Push, info, protocol.UpdatePlayerNum, nil)
		this.RoomState = ROOM_WAITING_READY
		this.PutQueue(protocol.ReadyGame)
		this.OnlyExecOne = true
	}

	if this.RoomState == ROOM_WAITING_XIAZHU && this.SeqExecFlag {
		this.SeqExecFlag = false
		if this.CountDown >= 1 {
			for k, v := range pl {
				if v.Role == ROBOT {
					//_,pos := this.GetXiaZhuResultMaxMin(this.XiaZhuTotal) //下注最少的位置下注
					pos := this.RandInt64(1, 33)
					if pos >= 1 && pos < 16 {
						pos = 1
					} else if pos >= 16 && pos < 31 {
						pos = 2
					} else {
						pos = 3
					}
					for _, v1 := range this.RobotXiaZhuList[v.UserID].XiaZhu[strconv.Itoa(this.CountDown)] {
						msg := make(map[string]interface{})
						msg["pos"] = strconv.FormatInt(pos, 10)
						msg["xiaZhuV"] = v1
						erro := this.PutQueue(protocol.RobotXiaZhu, v, msg)
						if erro != nil {
							log.Info("--------------- table.PutQueue error---tableID ---error = %s", erro)
						}
					}
				}
				if k == len(this.PlayerList)/2 {
					randTime := this.RandInt64(400, 600)
					time.Sleep(time.Millisecond * time.Duration(randTime))
				}
			}
		}
		if this.CountDown <= -1 {
			this.RoomState = ROOM_WAITING_JIESUAN
			erro := this.PutQueue(protocol.JieSuan)
			if erro != nil {
				log.Info("--------------- table.PutQueue error---tableID ---error = %s", erro)
			}
			this.OnlyExecOne = true
		} else {
			this.SeqExecFlag = true
		}
	}
	if this.RoomState == ROOM_WAITING_JIESUAN && this.SeqExecFlag {
		this.SeqExecFlag = false
		if this.OnlyExecOne {
			this.OnlyExecOne = false
			go this.RobotAdd(false)
		}

		if this.CountDown <= 0 {
			erro := this.PutQueue(protocol.UpdatePlayerList)
			if erro != nil {
				log.Info("--------------- table.PutQueue error---tableID ---error = %s", erro)
			}
			info := make(map[string]interface{})
			info["PlayerNum"] = this.PlayerNum
			this.sendPackToAll(game.Push, info, protocol.UpdatePlayerNum, nil)

			this.RoomState = ROOM_WAITING_READY
			this.PutQueue(protocol.ReadyGame)

			this.OnlyExecOne = true
		} else {
			this.SeqExecFlag = true
		}
	}
	if this.RoomState == ROOM_WAITING_READY && this.SeqExecFlag {
		this.SeqExecFlag = false
		if this.OnlyExecOne {
			this.OnlyExecOne = false
			//this.PutQueue(protocol.RobotBetCalc)
			this.RobotBetCalc()
			var uids []primitive.ObjectID
			uids = []primitive.ObjectID{}
			for _, v := range pl {
				if v.Role == USER || v.Role == Agent {
					uids = append(uids, utils.ConvertOID(v.UserID))
				}
			}
			userIDs := vGate.GetSessionUids(uids)
			for _, v := range pl {
				if !utils.IsContainStr(userIDs, v.UserID) && (v.Role == USER || v.Role == Agent) {
					erro := this.PutQueue(protocol.QuitTable, v.Session, v.UserID)
					if erro != nil {
						log.Info("--------------- table.PutQueue error---tableID ---error = %s", erro)
					}
				} else if v.NotXiaZhuCnt > this.GameConf.KickRoomCnt && (v.Role == USER || v.Role == Agent) {
					erro := this.PutQueue(protocol.QuitTable, v.Session, v.UserID)
					if erro != nil {
						log.Info("--------------- table.PutQueue error---tableID ---error = %s", erro)
					}
				}
			}
		}

		if this.CountDown <= 0 {
			this.RoomState = ROOM_WAITING_XIAZHU
			erro := this.PutQueue(protocol.StartGame)
			if erro != nil {
				log.Info("--------------- table.PutQueue error---tableID ---error = %s", erro)
			}
		} else {
			this.SeqExecFlag = true
		}

	}

	this.CountDown--
}
func (this *MyTable) TableInit(module module.RPCModule, app module.App, tableID string) {
	this.Players = map[string]room.BasePlayer{}
	this.Results = map[string]int{}
	this.BroadCast = false
	this.RobotXiaZhuList = map[string]RobotXiaZhuList{}
	this.PositionNum = 7
	this.PositionList = []PlayerList{}
	this.PlayerList = []PlayerList{}
	this.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	this.GameConf = cardLhdStorage.GetRoomConf()
	roomRecord := cardLhdStorage.GetRoomRecord()
	if roomRecord == nil {
		roomRecord := cardLhdStorage.RoomRecord{
			ResultsRecord: map[string]cardLhdStorage.ResultsRecord{},
		}
		cardLhdStorage.InsertRoomRecord(&roomRecord)
	}
	resultsRecord := cardLhdStorage.GetResultsRecord(this.tableID)
	if resultsRecord.ResultsRecordNum == 0 {
		resultsRecord.ResultsRecordNum = ResultsRecordNum
		resultsRecord.ResultsWinRate = map[cardLhdStorage.XiaZhuResult]int{
			cardLhdStorage.LONG: 0,
			cardLhdStorage.HU:   0,
			cardLhdStorage.HE:   0,
		}
		resultsRecord.Results = []cardLhdStorage.XiaZhuResult{}
		cardLhdStorage.UpsertResultsRecord(resultsRecord, this.tableID)
	}

	robotRange := map[cardLhdStorage.RobotType]cardLhdStorage.RobotRange{
		cardLhdStorage.Robot_0_1_K:     {Min: 1, Max: 1000},
		cardLhdStorage.Robot_1_20_K:    {Min: 1000, Max: 20000},
		cardLhdStorage.Robot_20_50_K:   {Min: 20000, Max: 50000},
		cardLhdStorage.Robot_50_100_K:  {Min: 50000, Max: 100000},
		cardLhdStorage.Robot_100_500_K: {Min: 100000, Max: 500000},
		cardLhdStorage.Robot_500_1_M:   {Min: 500000, Max: 1000000},
		cardLhdStorage.Robot_1_10_M:    {Min: 1000000, Max: 10000000},
		cardLhdStorage.Robot_10_30_M:   {Min: 10000000, Max: 30000000},
		cardLhdStorage.Robot_30_50_M:   {Min: 30000000, Max: 50000000},
	}
	for i := 0; i < 9; i++ {
		robot := cardLhdStorage.Robot{
			RobotType:  i,
			MaxBalance: robotRange[cardLhdStorage.RobotType(strconv.Itoa(i))].Max,
			MinBalance: robotRange[cardLhdStorage.RobotType(strconv.Itoa(i))].Min,
			TableID:    tableID,
		}
		this.RobotYxbConf = append(this.RobotYxbConf, robot)
	}
	robotConf := cardLhdStorage.GetTableRobotConf(this.tableID)
	if robotConf == nil { //
		for i := 0; i < 4; i++ {
			conf := cardLhdStorage.RobotConf{
				TableID:   tableID,
				StartHour: i * 6,
				MaxOffset: MaxOffset,
				StepNum:   StepNum,
				BaseNum:   40,
			}
			cardLhdStorage.UpsertRobotConf(conf)
		}
	}
	tableInfo := cardLhdStorage.GetTableInfo(tableID)
	tableInfo.TableID = tableID
	tableInfo.ServerID = module.GetServerID()
	cardLhdStorage.UpsertTableInfo(tableInfo, tableID)

	this.PlayerList = []PlayerList{}
	this.RoomState = ROOM_WAITING_START
	this.PlayerNum = 0
	this.ResultsChipList = map[cardLhdStorage.XiaZhuResult][]int64{}
	this.XiaZhuTotal = map[cardLhdStorage.XiaZhuResult]int64{}
	this.SeatNum = 7

	this.onlinePush = &vGate.OnlinePush{
		App:       app,
		TraceSpan: this.Trace(),
	}
	this.onlinePush.OnlinePushInit(this, 512)
	this.ShortCut = map[string][]game.ShortCutMode{}

	this.SeqExecFlag = true
	go func() {
		c := cron.New()
		c.AddFunc("*/1 * * * * ?", this.OnTimer)
		c.Start()
	}()

	//go this.BoottimeTimingSettlement() //凌晨刷新数据
}

//func (this *MyTable) GenerateRandResults(){
//	tCard := make([]int,len(card))
//	copy(tCard,card)
//	for i := 1;i < 3;i++{
//		ret := this.RandInt64(1,int64(len(tCard) + 1))
//		this.Results[strconv.Itoa(i)] = tCard[ret - 1]
//		tCard = append(tCard[:ret - 1],tCard[ret:]...)
//	}
//	//log.Info("--------generate rand results = ",this.Results)
//}
func (this *MyTable) ReadyGame() {
	this.Results = map[string]int{}
	//this.GenerateRandResults()

	this.GameConf = cardLhdStorage.GetRoomConf()
	this.CountDown = this.GameConf.ReadyGameTime
	this.ResultsChipList = map[cardLhdStorage.XiaZhuResult][]int64{}
	this.XiaZhuTotal = map[cardLhdStorage.XiaZhuResult]int64{
		cardLhdStorage.LONG: 0,
		cardLhdStorage.HU:   0,
		cardLhdStorage.HE:   0,
	}
	this.RealXiaZhuTotal = map[cardLhdStorage.XiaZhuResult]int64{
		cardLhdStorage.LONG: 0,
		cardLhdStorage.HU:   0,
		cardLhdStorage.HE:   0,
	}
	this.EventID = string(game.CardLhd) + this.tableID + "_" + strconv.FormatInt(time.Now().Unix(), 10)

	for k, v := range this.PlayerList {
		var lastXiaZhu int64 = 0
		for _, v1 := range v.XiaZhuResult {
			for _, v2 := range v1 {
				lastXiaZhu += v2
			}
		}
		if lastXiaZhu > 0 {
			v.LastXiaZhuResult = v.XiaZhuResult
			v.LastState = true
		}

		v.XiaZhuResult = map[cardLhdStorage.XiaZhuResult][]int64{}
		v.XiaZhuResultTotal = map[cardLhdStorage.XiaZhuResult]int64{
			cardLhdStorage.LONG: 0,
			cardLhdStorage.HU:   0,
			cardLhdStorage.HE:   0,
		}
		//v.ResultsChipList = []int64{}
		v.TotalBackYxb = 0
		v.SysProfit = 0
		v.BotProfit = 0
		this.PlayerList[k] = v

		if v.Role == USER || v.Role == Agent {
			info := this.GetPlayerInfo(v.UserID, false)
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				session, _ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(), game.Push, info, protocol.UpdatePlayerInfo, nil)
			}
		}

	}

	this.SwitchRoomState()
	this.SeqExecFlag = true
}
func (this *MyTable) StartGame() {
	this.CountDown = this.GameConf.XiaZhuTime
	this.SwitchRoomState()
	this.SeqExecFlag = true
}
