package cardSss

import (
	"github.com/robfig/cron"
	"math/rand"
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
	"vn/storage/cardStorage/cardSssStorage"
	"vn/storage/gameStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) ClearTable() { //
	cardSssStorage.RemoveTableInfo(this.tableID)

	myRoom := (this.module).(*Room)
	myRoom.DestroyTable(this.tableID)
}
func (this *MyTable) TableInit(module module.RPCModule, app module.App, tableID string) {
	this.Players = map[string]room.BasePlayer{}
	this.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	this.GameConf = cardSssStorage.GetRoomConf()
	//this.InitBots()
	this.StraightScore = map[StraightType]int64{
		QingLong:       100, //青龙
		YiTiaoLong:     50,  //一条龙
		SameColor:      30,  //清一色
		Pair5Three1:    10,  //五对加三条
		Flush3:         8,   //三同花
		Straight3:      8,   //三顺子
		StraightFlush3: 40,  //三同花顺
		Pair6:          8,   //6对
	}

	//rand_basescore_robotnum_totalnum
	this.tableIDTail = strings.Split(this.tableID, "_")[1] + "_" + strings.Split(this.tableID, "_")[2] + "_" + strings.Split(this.tableID, "_")[3]

	tableInfo := cardSssStorage.GetTableInfo(tableID)
	this.BaseScore, _ = strconv.ParseInt(strings.Split(this.tableID, "_")[1], 10, 64)
	botNum, _ := strconv.ParseInt(strings.Split(this.tableID, "_")[2], 10, 64)
	this.RobotNum = int(botNum)
	totalNum, _ := strconv.ParseInt(strings.Split(this.tableID, "_")[3], 10, 64)
	this.TotalPlayerNum = int(totalNum)
	this.AutoCreate = true
	this.MinEnterTable = this.BaseScore * int64(this.GameConf.MinEnterTableOdds)

	tableInfo.BaseScore = this.BaseScore
	tableInfo.RobotNum = this.RobotNum
	tableInfo.TotalPlayerNum = this.TotalPlayerNum
	tableInfo.TableID = tableID
	tableInfo.ServerID = module.GetServerID()
	cardSssStorage.UpsertTableInfo(tableInfo, tableID)

	this.onlinePush = &vGate.OnlinePush{
		App:       app,
		TraceSpan: this.Trace(),
	}
	this.onlinePush.OnlinePushInit(this, 512)
	this.RobotGenerate(this.RobotNum)
	this.SeqExecFlag = true
	this.RoomState = ROOM_WAITING_READY

	this.PlayerList = make([]PlayerList, this.TotalPlayerNum)
	go func() {
		c := cron.New()
		c.AddFunc("*/1 * * * * ?", this.OnTimer)
		c.Start()
	}()

}
func (this *MyTable) OnTimer() { //定时任务
	if !this.SeqExecFlag {
		return
	}
	if this.RoomState == ROOM_END && this.SeqExecFlag {
		this.SeqExecFlag = false
		for _, v := range this.PlayerList {
			if v.IsHavePeople && v.Role != ROBOT {
				this.PutQueue(protocol.QuitTable, v.UserID)
			}
		}
		this.PutQueue(protocol.ClearTable)
		return
	}

	if this.RoomState != ROOM_END {
		robotNum := this.RobotNum - this.GetRobotNum()
		rand := this.RandInt64(1, 4)
		if robotNum > 0 && rand == 1 {
			this.RobotGenerate(robotNum)
		}
		if this.RobotNum == 1 {
			this.RefreshTime++
		} else {
			this.RefreshTime = 0
		}
		if this.RefreshTime >= 10 {
			this.RobotNum = int(this.RandInt64(1, int64(this.TotalPlayerNum)))
		}
		playerNum := this.GetTablePlayerNum()
		if playerNum < this.TotalPlayerNum && playerNum > 0 { //邀请玩家
			if this.BaseScore < 10000 {
				rand = this.RandInt64(1, 2000)
			} else {
				rand = this.RandInt64(1, this.BaseScore/2)
			}
			if rand == 1 && this.PlayerList[0].Role != "" {
				record := gameStorage.GameInviteRecord{
					GameType:        game.CardSss,
					GameName:        common2.I18str(string(game.CardSss)),
					InvitorNickName: this.PlayerList[0].Account,
					RoomId:          this.tableID,
					BaseScore:       this.BaseScore,
					ServerId:        this.module.GetServerID(),
					UpdateAt:        utils.Now(),
				}
				myRoom := (this.module).(*Room)
				myRoom.NotifyGameInviteToOnlineUsers(record)
			}
		}
	}

	if this.RoomState == ROOM_WAITING_READY && this.SeqExecFlag {
		this.SeqExecFlag = false
		if this.GetTableRealPlayerNum() <= 0 && !this.AutoCreate {
			this.RoomState = ROOM_END
			this.SeqExecFlag = true
			return
		}
		for _, v := range this.PlayerList {
			if v.IsHavePeople {
				if v.Role == ROBOT && !v.Ready && v.UserID != this.Master {
					rand := this.RandInt64(1, 4)
					if rand != 1 {
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
		if this.GetReadyPlayerNum() > 1 && this.CountDown < 0 { //有两人准备，直接进入游戏准备状态
			this.RoomState = ROOM_WAITING_SHOWPOKER
			this.PutQueue(protocol.ReadyGame)
		} else {
			this.SeqExecFlag = true
		}
	}
	if this.RoomState == ROOM_WAITING_SHOWPOKER && this.SeqExecFlag {
		this.SeqExecFlag = false
		for k, v := range this.PlayerList {
			if v.Ready {
				waitingList, _ := this.WaitingList.Load(k)
				if !waitingList.(bool) {
					rand := int64(0)
					if this.GetRobotNum() == 3 {
						rand = this.RandInt64(1, 21)
					} else if this.GetRobotNum() == 2 {
						rand = this.RandInt64(1, 26)
					} else {
						rand = this.RandInt64(1, 31)
					}

					if v.Role == ROBOT && (rand == 1 || this.CountDown <= 0) {
						res := this.SortPokerFunc(v.UserID, v.HandPoker)
						this.PutQueue(protocol.DealShowPoker, v.UserID, res)
						time.Sleep(100 * time.Millisecond)
					} else if v.Role != ROBOT && this.CountDown <= 0 {
						this.PutQueue(protocol.DealShowPoker, v.UserID, v.HandPoker)
					}
				}
			}
		}
		if this.CountDown <= 0 {
			this.OnlyExecOne = true
		} else {
			this.SeqExecFlag = true
		}
	}
	if this.RoomState == ROOM_WAITING_JIESUAN && this.SeqExecFlag {
		this.SeqExecFlag = false
		if this.OnlyExecOne {
			this.OnlyExecOne = false
		}

		if this.CountDown <= 0 {
			this.CountDown = this.GameConf.ReadyTime
			this.RoomState = ROOM_WAITING_READY
			this.SwitchRoomState()
			for k, v := range this.PlayerList {
				if v.IsHavePeople {
					if v.UserID != this.Master {
						this.PlayerList[k].Ready = false
					}
					sb := vGate.QuerySessionBean(v.UserID)
					if v.Role != ROBOT {
						wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
						v.Yxb = wallet.VndBalance
					}
					if v.Role != ROBOT && (sb == nil || v.NotReadyCnt >= 3 || v.Yxb < this.MinEnterTable || v.Hosting || v.QuitRoom) {
						if v.UserID != "" {
							this.PutQueue(protocol.QuitTable, v.UserID)
						}
						continue
					}
					rand := this.RandInt64(1, 5)
					if v.Role == ROBOT {
						if rand == 1 {
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
		} else {
			this.SeqExecFlag = true
		}
	}
	if this.RoomState == ROOM_WAITING_ENTER && this.SeqExecFlag {
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
	if this.GetReadyPlayerNum() <= 1 {
		this.RoomState = ROOM_WAITING_READY
		this.SwitchRoomState()
		this.SeqExecFlag = true
		return
	}
	this.GameConf = cardSssStorage.GetRoomConf()

	this.ShooterList = []int{}
	this.ShotList = []int{}
	this.HomeRun = -1
	this.JieSuanData = JiesuanData{}
	this.JieSuanData.PlayerInfo = map[int]PlayerInfo{}
	this.JieSuanData.ShooterList = []int{}
	this.JieSuanData.ShotList = []int{}
	this.EventID = string(game.CardSss) + this.tableID + "_" + strconv.FormatInt(time.Now().Unix(), 10)

	for k, _ := range this.PlayerList {
		this.WaitingList.Store(k, false)
		this.PlayerList[k].TotalBackYxb = 0

		this.PlayerList[k].StraightType = StraightType(-1)
		this.PlayerList[k].PokerVal = make([]int, 3)
		this.PlayerList[k].PokerType = make([]PokerType, 3)
		this.PlayerList[k].Oolong = false
		this.PlayerList[k].ResultScore = make([]int64, 3)
		this.PlayerList[k].FinalScore = 0
		this.PlayerList[k].ShotScore = 0
		this.PlayerList[k].HomeRunScore = 0
		this.PlayerList[k].HandPoker = []int{}
		this.PlayerList[k].SysProfit = 0
	}
	this.PutQueue(protocol.StartGame)
}
func (this *MyTable) StartGame() { //开始游戏
	if this.GetReadyPlayerNum() <= 1 {
		this.RoomState = ROOM_WAITING_READY
		this.SwitchRoomState()
		this.SeqExecFlag = true
		return
	}
	this.RobotNum = int(this.RandInt64(1, int64(this.TotalPlayerNum)))
	this.CountDown = this.GameConf.ShowPokerTime
	this.SwitchRoomState()

	this.DealPoker(this.Shuffle())
	info := make(map[string]interface{})
	info["MasterIdx"] = this.GetPlayerIdx(this.Master)
	for _, v := range this.PlayerList {
		if v.IsHavePeople && v.Ready {
			playerInfo := this.GetPlayerInfo(v.UserID)
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				s, _ := basegate.NewSession(this.app, sb.Session)
				_ = this.sendPack(s.GetSessionID(), game.Push, playerInfo, protocol.UpdatePlayerInfo, nil)
			}
		}
	}
	for k, v := range this.PlayerList {
		if v.Ready {
			info["Poker"] = this.PlayerList[k].HandPoker
			straightType := this.CheckStraightScore(this.PlayerList[k].HandPoker)
			info["StraightType"] = straightType //直接得分的类型
			if straightType > 0 {
				this.PlayerList[k].StraightType = straightType
			}
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				session, _ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(), game.Push, info, protocol.StartGame, nil)
			}
			this.PlayerList[k].NotReadyCnt = 0
			if v.Role != ROBOT {
				if straightType <= 0 {
					this.PlayerList[k].Hosting = true
				}
				activityStorage.UpsertGameDataInBet(v.UserID, game.CardSss, 1)
			}
		} else if v.IsHavePeople {
			this.PlayerList[k].NotReadyCnt++
		}
	}

	this.SeqExecFlag = true

}
