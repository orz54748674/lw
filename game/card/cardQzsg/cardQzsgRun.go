package cardQzsg

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
	"vn/storage/cardStorage/cardQzsgStorage"
	"vn/storage/gameStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) ClearTable() { //
	cardQzsgStorage.RemoveTableInfo(this.tableID)

	myRoom := (this.module).(*Room)
	myRoom.DestroyTable(this.tableID)
}
func (this *MyTable) TableInit(module module.RPCModule, app module.App, tableID string) {
	this.Players = map[string]room.BasePlayer{}
	this.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	this.GameConf = cardQzsgStorage.GetRoomConf()

	this.tableIDTail = strings.Split(this.tableID, "_")[1] + "_" + strings.Split(this.tableID, "_")[2] + "_" + strings.Split(this.tableID, "_")[3]

	this.BaseScore, _ = strconv.ParseInt(strings.Split(this.tableID, "_")[1], 10, 64)
	botNum, _ := strconv.ParseInt(strings.Split(this.tableID, "_")[2], 10, 64)
	this.RobotNum = int(botNum)
	totalNum, _ := strconv.ParseInt(strings.Split(this.tableID, "_")[3], 10, 64)
	this.TotalPlayerNum = int(totalNum)
	this.AutoCreate = true
	this.MinEnterTable = this.BaseScore * int64(this.GameConf.MinEnterTableOdds)

	tableInfo := cardQzsgStorage.GetTableInfo(tableID)
	tableInfo.BaseScore = this.BaseScore
	tableInfo.RobotNum = this.RobotNum
	tableInfo.TotalPlayerNum = this.TotalPlayerNum
	tableInfo.TableID = tableID
	tableInfo.ServerID = module.GetServerID()
	cardQzsgStorage.UpsertTableInfo(tableInfo, tableID)

	this.onlinePush = &vGate.OnlinePush{
		App:       app,
		TraceSpan: this.Trace(),
	}
	this.MinEnterTable = this.BaseScore * int64(this.GameConf.MinEnterTableOdds)
	this.onlinePush.OnlinePushInit(this, 512)
	this.SeqExecFlag = true
	this.RoomState = ROOM_WAITING_ENTER

	this.RobotGenerate(this.RobotNum)
	this.PlayerList = make([]PlayerList, this.TotalPlayerNum)
	go func() {
		c := cron.New()
		c.AddFunc("*/1 * * * * ?", this.OnTimer)
		c.Start()
	}()

}
func (this *MyTable) OnTimer() { //????????????
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
		if playerNum < this.TotalPlayerNum && playerNum > 0 { //????????????
			if this.BaseScore < 10000 {
				rand = this.RandInt64(1, 2000)
			} else {
				rand = this.RandInt64(1, this.BaseScore/2)
			}
			if rand == 1 && this.PlayerList[0].Role != "" {
				record := gameStorage.GameInviteRecord{
					GameType:        game.CardQzsg,
					GameName:        common2.I18str(string(game.CardQzsg)),
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
		if this.GetReadyPlayerNum() > 1 && this.CountDown < 0 { //????????????????????????????????????????????????
			this.RoomState = ROOM_WAITING_SHOWPOKER
			this.PutQueue(protocol.ReadyGame)
		} else {
			this.SeqExecFlag = true
		}
	}

	if this.RoomState == ROOM_WAITING_GRABDEALER && this.SeqExecFlag {
		this.SeqExecFlag = false
		if this.CountDown < 0 {
			for k, v := range this.PlayerList {
				waitingList, _ := this.WaitingList.Load(k)
				if v.Ready && !waitingList.(bool) {
					res := make(map[string]interface{})
					res["GrabDealer"] = -1
					this.PutQueue(protocol.GrabDealer, v.UserID, res)
					time.Sleep(100 * time.Millisecond)
				}
			}
			this.OnlyExecOne = true
		} else {
			for k, v := range this.PlayerList {
				waitingList, _ := this.WaitingList.Load(k)
				if v.Ready && !waitingList.(bool) && v.Role == ROBOT {
					rand := this.RandInt64(1, 3)
					if rand == 1 {
						res := make(map[string]interface{})
						res["GrabDealer"] = 1
						this.PutQueue(protocol.GrabDealer, v.UserID, res)
						break
					} else {
						res := make(map[string]interface{})
						res["GrabDealer"] = -1
						this.PutQueue(protocol.GrabDealer, v.UserID, res)
						break
					}
				}
			}
			this.SeqExecFlag = true
		}
	}
	if this.RoomState == ROOM_WAITING_XIAZHU && this.SeqExecFlag {
		this.SeqExecFlag = false
		if this.CountDown < 0 {
			for k, v := range this.PlayerList {
				waitingList, _ := this.WaitingList.Load(k)
				if v.Ready && !waitingList.(bool) {
					res := make(map[string]interface{})
					res["betV"] = int(ChipsList[0] * this.BaseScore)
					this.PutQueue(protocol.XiaZhu, v.UserID, res)
					time.Sleep(100 * time.Millisecond)
				}
			}
			this.OnlyExecOne = true
		} else {
			if this.CountDown < this.GameConf.XiaZhuTime-2 {
				for k, v := range this.PlayerList {
					waitingList, _ := this.WaitingList.Load(k)
					if v.Ready && !waitingList.(bool) && v.Role == ROBOT {
						rand := this.RandInt64(1, 5)
						if rand == 1 {
							betRand := this.RandInt64(1, int64(len(ChipsList)+1)) - 1
							betV := ChipsList[betRand] * this.BaseScore
							res := make(map[string]interface{})
							res["betV"] = int(betV)
							this.PutQueue(protocol.XiaZhu, v.UserID, res)
							break
						}
					}
				}
			}
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
						this.PutQueue(protocol.QuitTable, v.UserID)
						continue
					}
					rand := this.RandInt64(1, 6)
					if v.Role == ROBOT {
						if rand == 1 || v.Yxb < this.MinEnterTable {
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
	this.GameConf = cardQzsgStorage.GetRoomConf()

	this.JieSuanData = JiesuanData{}
	this.JieSuanData.PlayerInfo = map[int]PlayerInfo{}
	this.EventID = string(game.CardQzsg) + this.tableID + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	this.PlayingNum = 0
	this.GrabDealerList = []int{}
	this.DealerIdx = -1
	for k, v := range this.PlayerList {
		this.WaitingList.Store(k, false)
		this.PlayerList[k].TotalBackYxb = 0

		this.PlayerList[k].PokerType = PokerType(0)
		this.PlayerList[k].PokerVal = 0
		this.PlayerList[k].FinalScore = 0
		this.PlayerList[k].PressScore = 0
		this.PlayerList[k].BetVal = 0
		this.PlayerList[k].HandPoker = []int{}

		if v.Ready {
			this.PlayingNum++
		}
	}
	this.PutQueue(protocol.StartGame)
}
func (this *MyTable) StartGame() { //????????????
	if this.GetReadyPlayerNum() <= 1 {
		this.RoomState = ROOM_WAITING_READY
		this.SwitchRoomState()
		this.SeqExecFlag = true
		return
	}
	this.RobotNum = int(this.RandInt64(1, int64(this.TotalPlayerNum)))
	this.RoomState = ROOM_WAITING_GRABDEALER
	this.CountDown = this.GameConf.QiangZhuangTime
	this.SwitchRoomState()
	//this.DealPoker(this.Shuffle())
	//info := make(map[string]interface{})
	//info["MasterIdx"] = this.GetPlayerIdx(this.Master)

	for k, v := range this.PlayerList {
		if v.IsHavePeople && v.Ready {
			playerInfo := this.GetPlayerInfo(v.UserID)
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				s, _ := basegate.NewSession(this.app, sb.Session)
				_ = this.sendPack(s.GetSessionID(), game.Push, playerInfo, protocol.UpdatePlayerInfo, nil)
			}
			this.PlayerList[k].NotReadyCnt = 0
			if v.Role != ROBOT {
				this.PlayerList[k].Hosting = true
				activityStorage.UpsertGameDataInBet(v.UserID, game.CardQzsg, 1)
			}
		} else if v.IsHavePeople {
			this.PlayerList[k].NotReadyCnt++
		}
	}
	//for _,v := range this.PlayerList{
	//	if v.Ready{
	//		//info["HandPoker"] = v.HandPoker
	//		sb := vGate.QuerySessionBean(v.UserID)
	//		if sb != nil{
	//			session,_ := basegate.NewSession(this.app, sb.Session)
	//			this.sendPack(session.GetSessionID(),game.Push,info,protocol.StartGame,nil)
	//		}
	//	}
	//}

	this.SeqExecFlag = true
}
