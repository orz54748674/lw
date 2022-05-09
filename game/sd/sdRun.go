package sd

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
	"vn/storage/sdStorage"
)

func (this *MyTable) ClearTable() { //
	if !this.Hundred{
		sdStorage.RemoveResultsRecord(this.tableID)
	}
	sdStorage.RemoveTableInfo(this.tableID)

	myRoom := (this.module).(*Room)
	myRoom.DestroyTable(this.tableID)
}
func (this *MyTable) OnTimer() { //定时任务
	if !this.SeqExecFlag{
		return
	}
	//log.Info("---------------- count down = %d",this.CountDown)
	pl := this.DeepCopyPlayerList(this.PlayerList)
	if this.RoomState == ROOM_END && this.SeqExecFlag{
		this.SeqExecFlag = false
		for _,v := range pl{
			if v.Role == USER || v.Role == Agent{
				sb := vGate.QuerySessionBean(v.UserID)
				if sb != nil{
					this.PutQueue(protocol.QuitTable,v.UserID,false)
				}
			}
		}
		this.PutQueue(protocol.ClearTable)
		return
	}
	if this.RoomState == ROOM_WAITING_START && this.SeqExecFlag{
		this.SeqExecFlag = false
		if this.Hundred {
			this.RobotAdd(true)
			this.PutQueue(protocol.UpdatePlayerList)
		}

		info := make(map[string]interface{})
		info["PlayerNum"] = this.PlayerNum
		this.sendPackToAll(game.Push, info,protocol.UpdatePlayerNum,nil)
		this.RoomState = ROOM_WAITING_READY
		this.PutQueue(protocol.ReadyGame)
		this.OnlyExecOne = true
	}

	if this.RoomState == ROOM_WAITING_XIAZHU && this.SeqExecFlag{
		this.SeqExecFlag = false
		if this.CountDown >= 1 && this.Hundred{
			for k,v := range pl{
				if v.Role == ROBOT{
					//_,pos := this.GetXiaZhuResultMaxMin(this.XiaZhuTotal) //下注最少的位置下注
					pos := this.RandInt64(1,18)
					if pos > 6{ //单双下注概率高点
						if pos % 2 == 0{
							pos = 2
						}else{
							pos = 1
						}
					}
					for _,v1 := range this.RobotXiaZhuList[v.UserID].XiaZhu[strconv.Itoa(this.CountDown)]{
						msg := make(map[string]interface{})
						msg["pos"] = strconv.FormatInt(pos,10)
						msg["xiaZhuV"] = v1
						this.PutQueue(protocol.RobotXiaZhu,v,msg)
					}
				}
				if k == len(pl) / 2{
					time.Sleep(time.Millisecond * 500)
				}
			}
		}
		if this.CountDown <= -1{
			this.RoomState = ROOM_WAITING_JIESUAN
			this.PutQueue(protocol.JieSuan)
			this.OnlyExecOne = true
		}else{
			this.SeqExecFlag = true
		}

	}
	if this.RoomState == ROOM_WAITING_JIESUAN && this.SeqExecFlag{
		this.SeqExecFlag = false
		if this.OnlyExecOne && this.Hundred{
			this.OnlyExecOne = false
			go this.RobotAdd(false)
		}

		if this.CountDown <= 0 {
			this.PutQueue(protocol.UpdatePlayerList)
			info := make(map[string]interface{})
			info["PlayerNum"] = this.PlayerNum
			this.sendPackToAll(game.Push, info,protocol.UpdatePlayerNum,nil)
			this.RoomState = ROOM_WAITING_READY
			this.PutQueue(protocol.ReadyGame)

			this.OnlyExecOne = true
		}else{
			this.SeqExecFlag = true
		}

	}
	if this.RoomState == ROOM_WAITING_READY && this.SeqExecFlag{
		this.SeqExecFlag = false
		if this.OnlyExecOne{
			this.OnlyExecOne = false
			//this.PutQueue(protocol.RobotBetCalc)
			this.RobotBetCalc()
			var uids []primitive.ObjectID
			uids = []primitive.ObjectID{}
			for _,v := range pl{
				if v.Role == USER || v.Role == Agent{
					uids = append(uids,utils.ConvertOID(v.UserID))
				}
			}
			for _,v := range pl{
				if this.DisConnectList[v.UserID] && (v.Role == USER || v.Role == Agent){
					erro := this.PutQueue(protocol.QuitTable,v.UserID,false)
					if erro != nil {
						log.Info("--------------- table.PutQueue error---tableID ---error = %s", erro)
					}
				} else if v.NotXiaZhuCnt > this.GameConf.KickRoomCnt && (v.Role == USER || v.Role == Agent){
					erro := this.PutQueue(protocol.QuitTable,v.UserID,false)
					if erro != nil {
						log.Info("--------------- table.PutQueue error---tableID ---error = %s", erro)
					}
				}
			}
		}

		if this.CountDown <= 0 {
			this.RoomState = ROOM_WAITING_XIAZHU
			this.PutQueue(protocol.StartGame)
		}else{
			this.SeqExecFlag = true
		}

	}
	this.CountDown --
}
func (this *MyTable) TableInit(module module.RPCModule,app module.App,tableID string){
	this.Players = map[string]room.BasePlayer{}
	this.Results = map[string]sdStorage.Result{}
	this.PrizeResults = []sdStorage.XiaZhuResult{}
	this.BroadCast =false
	this.RobotXiaZhuList = map[string]RobotXiaZhuList{}
	this.PositionList = []PlayerList{}
	this.PlayerList = []PlayerList{}
	this.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	roomRecord := sdStorage.GetRoomRecord()
	if roomRecord == nil{
		roomRecord := sdStorage.RoomRecord{
			ResultsRecord: map[string]sdStorage.ResultsRecord{},
		}
		sdStorage.InsertRoomRecord(&roomRecord)
	}
	resultsRecord := sdStorage.GetResultsRecord(this.tableID)
	resultsRecord.ResultsRecordNum = ResultsRecordNum
	sdStorage.UpsertResultsRecord(resultsRecord,this.tableID)

	this.GameConf = sdStorage.GetRoomConf()
	tableInfo := sdStorage.GetTableInfo(tableID)

	tableInfo.TableID = tableID
	this.PlayerList = []PlayerList{}
	this.RoomState = ROOM_WAITING_START

	tableInfo.PlayerNum = 0
	//this.XiaZhuTotal = map[sdStorage.XiaZhuResult]int64{}
	tableInfo.ServerID = module.GetServerID()

	tableIDInt,_ :=  strconv.Atoi(this.tableID)
	if tableIDInt < this.GameConf.HundredRoomNum{ //百人场
		this.SeatNum = 10
		tableInfo.Hundred = true
		this.Hundred = true
		this.ChipsList = this.GameConf.PlayerChipsList[this.tableID]
		tableInfo.BaseScore = int64(tableIDInt)
	}else{
		this.SeatNum = 10
		tableInfo.Hundred = false
		this.Hundred = false
	}
	if tableInfo.Hundred{
		robotRange := map[sdStorage.RobotType]sdStorage.RobotRange{
			sdStorage.Robot_0_1_K:     {Min: 1, Max: 1000},
			sdStorage.Robot_1_20_K:    {Min: 1000,Max: 20000},
			sdStorage.Robot_20_50_K:   {Min: 20000,Max: 50000},
			sdStorage.Robot_50_100_K:  {Min: 50000,Max: 100000},
			sdStorage.Robot_100_500_K: {Min: 100000,Max: 500000},
			sdStorage.Robot_500_1_M:   {Min: 500000,Max: 1000000},
			sdStorage.Robot_1_10_M:    {Min: 1000000,Max: 10000000},
			sdStorage.Robot_10_30_M:   {Min: 10000000,Max: 30000000},
			sdStorage.Robot_30_50_M:   {Min: 30000000,Max: 50000000},
		}
		for i := 0;i < 9;i++{
			robot := sdStorage.Robot{
				RobotType: i,
				MaxBalance: robotRange[sdStorage.RobotType(strconv.Itoa(i))].Max,
				MinBalance: robotRange[sdStorage.RobotType(strconv.Itoa(i))].Min,
				TableID: tableID,
			}
			this.RobotYxbConf = append(this.RobotYxbConf,robot)
		}
		robotConf := sdStorage.GetTableRobotConf(this.tableID)
		if robotConf == nil{ //
			for i := 0;i < 4;i++{
				conf := sdStorage.RobotConf{
					TableID: tableID,
					StartHour: i * 6,
					MaxOffset: MaxOffset,
					StepNum: StepNum,
					BaseNum: 40,
				}
				sdStorage.UpsertRobotConf(conf)
			}
		}

	}
	sdStorage.UpsertTableInfo(tableInfo,tableID)


	this.onlinePush = &vGate.OnlinePush{
		App:       app,
		TraceSpan: this.Trace(),
	}
	this.onlinePush.OnlinePushInit(this, 512)
	//this.ShortCut = map[string][]game.ShortCutMode{}
	this.DisConnectList = map[string]bool{}
	this.SeqExecFlag = true
	go func() {
		c := cron.New()
		c.AddFunc("*/1 * * * * ?",this.OnTimer)
		c.Start()
	}()
}
func (this *MyTable) GenerateRandResults(){
	for i := 1;i < 5;i++{
		ret := this.RandInt64(1,3)
		this.Results[strconv.Itoa(i)] = sdStorage.Result(strconv.FormatInt(ret,10))
	}
	//log.Info("--------generate rand results = ",this.Results)
}
func (this *MyTable) ReadyGame() {
	//	log.Info("-------------------------ready game tableid = %s",this.tableID)

	this.Results = map[string]sdStorage.Result{}
	this.PrizeResults = []sdStorage.XiaZhuResult{}
	this.GenerateRandResults()

	this.GameConf = sdStorage.GetRoomConf()
	this.CountDown = this.GameConf.ReadyGameTime
	//this.XiaZhuTotal = map[sdStorage.XiaZhuResult] int64 {
	//	sdStorage.SINGLE: 0,
	//	sdStorage.DOUBLE: 0,
	//	sdStorage.Red4White0: 0,
	//	sdStorage.Red0White4: 0,
	//	sdStorage.Red3White1: 0,
	//	sdStorage.Red1White3: 0,
	//}
	this.XiaZhuTotal.Store(sdStorage.SINGLE,int64(0))
	this.XiaZhuTotal.Store(sdStorage.DOUBLE,int64(0))
	this.XiaZhuTotal.Store(sdStorage.Red4White0,int64(0))
	this.XiaZhuTotal.Store(sdStorage.Red0White4,int64(0))
	this.XiaZhuTotal.Store(sdStorage.Red3White1,int64(0))
	this.XiaZhuTotal.Store(sdStorage.Red1White3,int64(0))

	this.RealXiaZhuTotal = map[sdStorage.XiaZhuResult] int64 {
		sdStorage.SINGLE: 0,
		sdStorage.DOUBLE: 0,
		sdStorage.Red4White0: 0,
		sdStorage.Red0White4: 0,
		sdStorage.Red3White1: 0,
		sdStorage.Red1White3: 0,
	}
	this.EventID = string(game.SeDie) + this.tableID + "_" + strconv.FormatInt(time.Now().Unix(),10)

	for k,v := range this.PlayerList{
		var lastXiaZhu int64 = 0
		for _,v1 := range v.XiaZhuResult{
			for _,v2 := range  v1{
				lastXiaZhu += v2
			}
		}
		if lastXiaZhu > 0{
			v.LastXiaZhuResult = v.XiaZhuResult
			v.LastState = true
		}

		v.XiaZhuResult = map[sdStorage.XiaZhuResult][]int64{}
		v.XiaZhuResultTotal = map[sdStorage.XiaZhuResult]int64{
			sdStorage.SINGLE: 0,
			sdStorage.DOUBLE: 0,
			sdStorage.Red4White0: 0,
			sdStorage.Red0White4: 0,
			sdStorage.Red3White1: 0,
			sdStorage.Red1White3: 0,
		}
		//v.ResultsChipList = []int64{}
		v.ResultsPool = 0
		v.TotalBackYxb = 0
		v.SysProfit = 0
		v.BotProfit = 0
		this.PlayerList[k] = v

		if v.Role == USER || v.Role == Agent{
			info := this.GetPlayerInfo(v.UserID,false)
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil{
				session,_ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(),game.Push, info,protocol.UpdatePlayerInfo,nil)
			}
		}

	}
	this.SwitchRoomState()
	this.SeqExecFlag = true
}
func (this *MyTable) StartGame(){
	this.CountDown = this.GameConf.XiaZhuTime
	this.SwitchRoomState()
	this.SeqExecFlag = true
}