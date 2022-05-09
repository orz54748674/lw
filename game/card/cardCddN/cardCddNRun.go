package cardCddN

import (
	"github.com/robfig/cron"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
	common2 "vn/common"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/module"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/activityStorage"
	"vn/storage/cardStorage/cardCddNStorage"
	"vn/storage/gameStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) ClearTable() { //
	cardCddNStorage.RemoveTableInfo(this.tableID)

	myRoom := (this.module).(*Room)
	myRoom.DestroyTable(this.tableID)
}
func (this *MyTable) TableInit(module module.RPCModule,app module.App,tableID string){
	this.Players = map[string]room.BasePlayer{}
	this.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	this.GameConf = cardCddNStorage.GetRoomConf()

	this.tableIDTail = strings.Split(this.tableID,"_")[1] + "_" + strings.Split(this.tableID,"_")[2] + "_" + strings.Split(this.tableID,"_")[3]

	this.BaseScore,_ = strconv.ParseInt(strings.Split(this.tableID,"_")[1], 10, 64)
	botNum,_ := strconv.ParseInt(strings.Split(this.tableID,"_")[2], 10, 64)
	this.RobotNum = int(botNum)
	totalNum,_ := strconv.ParseInt(strings.Split(this.tableID,"_")[3], 10, 64)
	this.TotalPlayerNum = int(totalNum)
	this.AutoCreate = true
	this.MinEnterTable =this.BaseScore * int64(this.GameConf.MinEnterTableOdds)

	tableInfo := cardCddNStorage.GetTableInfo(tableID)
	tableInfo.BaseScore = this.BaseScore
	tableInfo.RobotNum = this.RobotNum
	tableInfo.TotalPlayerNum = this.TotalPlayerNum
	tableInfo.TableID = tableID
	tableInfo.ServerID = module.GetServerID()
	cardCddNStorage.UpsertTableInfo(tableInfo,tableID)

	this.onlinePush = &vGate.OnlinePush{
		App:       app,
		TraceSpan: this.Trace(),
	}
	this.MinEnterTable =this.BaseScore * int64(this.GameConf.MinEnterTableOdds)
	this.onlinePush.OnlinePushInit(this, 512)
	this.SeqExecFlag = true
	this.RoomState = ROOM_WAITING_ENTER

	this.RobotGenerate(this.RobotNum)
	this.PlayerList = make([]PlayerList,this.TotalPlayerNum)
	this.LastData = LastData{}
	this.InitWaitingList()
	go func() {
		c := cron.New()
		c.AddFunc("*/1 * * * * ?",this.OnTimer)
		c.Start()
	}()

}
func (this *MyTable) OnTimer() { //定时任务
	if !this.SeqExecFlag{
		return
	}
	if this.RoomState == ROOM_END && this.SeqExecFlag{
		this.SeqExecFlag = false
		for _,v := range this.PlayerList{
			if v.IsHavePeople && v.Role != ROBOT{
				this.PutQueue(protocol.QuitTable, v.UserID)
			}
		}
		this.PutQueue(protocol.ClearTable)
		return
	}
	if this.RoomState != ROOM_END{
		robotNum := this.RobotNum - this.GetRobotNum()
		rand := this.RandInt64(1,4)
		realPlayerNum := this.GetTableRealPlayerNum()
		if robotNum > 0 && rand == 1 && realPlayerNum < 2{
			this.RobotGenerate(robotNum)
		}
		if this.RobotNum == 1{
			this.RefreshTime++
		}else{
			this.RefreshTime = 0
		}
		if this.RefreshTime >= 10{
			this.RobotNum = int(this.RandInt64(1,int64(this.TotalPlayerNum)))
		}
		playerNum := this.GetTablePlayerNum()
		if playerNum < this.TotalPlayerNum && playerNum > 0{ //邀请玩家
			if this.BaseScore < 10000{
				rand = this.RandInt64(1,2000)
			}else{
				rand = this.RandInt64(1,this.BaseScore / 2)
			}
			if rand == 1 && this.PlayerList[0].Role != ""{
				record := gameStorage.GameInviteRecord{
					GameType: game.CardCddN,
					GameName: common2.I18str(string(game.CardCddN)),
					InvitorNickName: this.PlayerList[0].Account,
					RoomId: this.tableID,
					BaseScore: this.BaseScore,
					ServerId: this.module.GetServerID(),
					UpdateAt: utils.Now(),
				}
				myRoom := (this.module).(*Room)
				myRoom.NotifyGameInviteToOnlineUsers(record)
			}
		}
	}
	if this.RoomState == ROOM_WAITING_READY && this.SeqExecFlag{
		this.SeqExecFlag = false
		if this.GetTableRealPlayerNum() <= 0 && !this.AutoCreate{
			this.RoomState = ROOM_END
			this.SeqExecFlag = true
			return
		}
		for _,v := range this.PlayerList {
			if v.IsHavePeople {
				if  v. Role == ROBOT && !v.Ready && v.UserID != this.Master {
					rand := this.RandInt64(1,4)
					if rand != 1{
						this.PutQueue(protocol.RobotReady, v.UserID)
					}
				}
			}
		}
		masterIdx := this.GetPlayerIdx(this.Master)
		if masterIdx >= 0 && this.PlayerList[masterIdx].Role == ROBOT &&
			this.GetReadyPlayerNum() > 1 &&
			this.CountDown >= 0 &&
			this.GetReadyPlayerNum() == this.GetPlayerNum() {
			this.CountDown = -1
		}
		if this.GetReadyPlayerNum() > 1 && this.CountDown < 0{ //有两人准备，直接进入游戏准备状态
			this.RoomState = ROOM_WAITING_PUTOP
			this.PutQueue(protocol.ReadyGame)
		}else{
			this.SeqExecFlag = true
		}
	}

	if this.RoomState == ROOM_WAITING_PUTOP && this.SeqExecFlag{
		this.SeqExecFlag = false
		for k,v := range this.PlayerList{
			if v.Ready && v.Role == ROBOT{
				val,_ := this.WaitingList.Load(k)
				waitingList := val.(WaitingList)
				if waitingList.Have{
					rand := this.RandInt64(1,int64(this.GameConf.PutPokerTime) - 6)
					if this.GameConf.PutPokerTime - this.CountDown >= int(rand) || this.CountDown < 0 {
						if waitingList.FirstRound{
							this.InitBestCardData()
							this.GetPutCardList(v.HandPoker,CalcCardData{})
							this.AutoPutPoker(v.UserID,this.BestHandCard.Poker)
						}else{
							this.GetPutCardListLimit(v.UserID)
						}
						return
					}
				}
			}
		}
		if this.CountDown < 0{
			for k,v := range this.PlayerList{
				val,_ := this.WaitingList.Load(k)
				waitingList := val.(WaitingList)
				if v.Ready && waitingList.Have && v.Role != ROBOT{
					if waitingList.FirstRound{
						minPk := this.GetMinPoker(v.HandPoker)
						this.AutoPutPoker(v.UserID,[]int{minPk})
					}else{
						this.PutQueue(protocol.CheckPoker,v.UserID)
					}
					return
				}
			}
			this.OnlyExecOne = true
		}else{
			this.SeqExecFlag = true
		}
	}
	if this.RoomState == ROOM_WAITING_JIESUAN && this.SeqExecFlag{
		this.SeqExecFlag = false
		if this.OnlyExecOne{
			this.OnlyExecOne = false
		}

		if this.CountDown <= 0 {
			this.CountDown = this.GameConf.ReadyTime
			this.RoomState = ROOM_WAITING_READY
			this.SwitchRoomState()
			realPlayerNum := this.GetTableRealPlayerNum()
			for k,v := range this.PlayerList{
				if v.IsHavePeople {
					if v.UserID != this.Master {
						this.PlayerList[k].Ready = false
					}
					sb := vGate.QuerySessionBean(v.UserID)
					if v.Role != ROBOT{
						wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
						v.Yxb = wallet.VndBalance
					}
					if v.Role != ROBOT && (sb == nil || v.NotReadyCnt >= 3 || v.Yxb < this.MinEnterTable || v.Hosting || v.QuitRoom){
						this.PutQueue(protocol.QuitTable, v.UserID)
						continue
					}
					rand := this.RandInt64(1,5)
					if v.Role == ROBOT {
						if rand == 1 || v.Yxb < this.MinEnterTable || realPlayerNum >= 2{
							this.PutQueue(protocol.RobotQuitTable, v.UserID)
							continue
						}
					}
					if v.AutoReady && v.Role != ROBOT {
						this.PutQueue(protocol.Ready, v.UserID)
					}
					if v.Role == ROBOT && v.UserID != this.Master {
						if rand == 1 {
							this.PutQueue(protocol.RobotReady, v.UserID)
						}
					}
				}
			}
			this.OnlyExecOne = true
			this.SeqExecFlag = true
		}else{
			this.SeqExecFlag = true
		}
	}
	if this.RoomState == ROOM_WAITING_ENTER && this.SeqExecFlag{
		this.SeqExecFlag = false
		this.RoomState = ROOM_WAITING_READY
		this.SwitchRoomState()
		//if this.tableID == "101"{
		//	time.Sleep(time.Second * 5)
		//	this.RoomState = ROOM_WAITING_SHOWPOKER
		//	this.StartGame()
		//}
		this.SeqExecFlag = true
	}
	this.CountDown--
}
func (this *MyTable) ReadyGame() {
	if this.GetReadyPlayerNum() <= 1{
		this.RoomState = ROOM_WAITING_READY
		this.SwitchRoomState()
		this.SeqExecFlag = true
		return
	}
	this.GameConf = cardCddNStorage.GetRoomConf()

	this.InitWaitingList()
	this.JieSuanData = JiesuanData{}
	this.StraightScoreData = StraightScoreData{}
	this.JieSuanData.PlayerInfo = map[int]PlayerInfo{}
	this.JieSuanData.LastPutCard = []int{}
	this.LastData.LastPutCard = []int{}
	this.LastData.LastPutIdx = 0
	this.PutOverRecord = []PutOverRecord{}
	this.EventID = string(game.CardCddN) + this.tableID + "_" + strconv.FormatInt(time.Now().Unix(),10)
	this.PlayingNum = 0
	this.CurRoundList = map[int]bool{}
	this.CurRoundPressList = PokerPressList{
		Single2: make([]int,0),
	}
	this.PokerPressList = []PokerPressList{}

	compPlayList := make([]string,0)
	for k,v := range this.PlayerList{
		this.PlayerList[k].TotalBackYxb = 0

		this.PlayerList[k].StraightType = StraightType(0)
		this.PlayerList[k].FinalScore = 0
		this.PlayerList[k].PressScore = 0
		this.PlayerList[k].HandPoker = []int{}
		this.PlayerList[k].OriginPoker = []int{}

		if v.Ready{
			this.PlayingNum++
			this.CurRoundList[k] = true

			compPlayList = append(compPlayList,v.UserID)
		}
	}
	if this.LastData.LastFirstIdx < 0 || !reflect.DeepEqual(this.LastData.LastPlayList,compPlayList){
		this.IsBlack3First = true
	}else{
		this.IsBlack3First = false
	}
	this.LastData.LastPlayList = compPlayList

	this.PutQueue(protocol.StartGame)
}
func (this *MyTable) StartGame() { //开始游戏
	if this.GetReadyPlayerNum() <= 1{
		this.RoomState = ROOM_WAITING_READY
		this.SwitchRoomState()
		this.SeqExecFlag = true
		return
	}
	this.RoomState = ROOM_WAITING_PUTOP
	this.CountDown = this.GameConf.PutPokerTime
	this.SwitchRoomState()
	this.DealPoker(this.Shuffle())
	//start := time.Now().UnixNano()
	////this.InitBestCardData()
	////this.GetPutCardList(this.PlayerList[0].HandPoker,CalcCardData{})
	//end := time.Now().UnixNano()
	//log.Info("total cost time = %d",time.Duration(end -start) / time.Millisecond)
	//res,_ :=json.Marshal(this.BestHandCard)
	//log.Info("------ %s ------",res)
	info := make(map[string]interface{})
	info["MasterIdx"] = this.GetPlayerIdx(this.Master)
	info["FirstPutIdx"] = this.FirstPutIdx
	idx,_type := this.GetStraightScoreIdx()
	info["StraightIdx"] = -1
	info["StraightType"] = -1
	if _type > 0{
		info["StraightIdx"] = idx
		info["StraightType"] = _type
		this.PlayerList[idx].StraightType = _type
		this.StraightScoreData = StraightScoreData{
			Idx: idx,
			Type: _type,
		}
	}
	for k,v := range this.PlayerList {
		if v.IsHavePeople && v.Ready{
			playerInfo := this.GetPlayerInfo(v.UserID)
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil{
				s,_ := basegate.NewSession(this.app, sb.Session)
				_ = this.sendPack(s.GetSessionID(),game.Push,playerInfo,protocol.UpdatePlayerInfo,nil)
			}
			this.PlayerList[k].NotReadyCnt = 0
			if v.Role != ROBOT {
				if  _type <= 0{
					this.PlayerList[k].Hosting = true
				}
				activityStorage.UpsertGameDataInBet(v.UserID,game.CardCddN,1)
			}
		} else if v.IsHavePeople {
			this.PlayerList[k].NotReadyCnt++
		}
	}
	for _,v := range this.PlayerList{
		if v.Ready{
			info["HandPoker"] = v.HandPoker
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil{
				session,_ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(),game.Push,info,protocol.StartGame,nil)
			}
		}
	}

	if _type > 0{
		time.Sleep(time.Second)
		this.RoomState = ROOM_WAITING_JIESUAN
		this.PutQueue(protocol.JieSuan)
		this.SeqExecFlag = true
		return
	}
	this.WaitingList.Store(this.FirstPutIdx,WaitingList{
		Time: time.Now(),
		FirstRound: true,
		Have: true,
		CanPut: true,
		BigPk:[][]int{},
	})
	this.SendState(this.WaitingList)

	this.SeqExecFlag = true
}
