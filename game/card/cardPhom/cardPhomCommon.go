package cardPhom

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
	"vn/common"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant/gate"
	basegate "vn/framework/mqant/gate/base"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/agentStorage"
	"vn/storage/walletStorage"
)

type PlayerList struct {
	Yxb int64   `bson:"Yxb" json:"Yxb"`//游戏币
	UserID string `bson:"UserID" json:"UserID"`//用户id
	Account string  `bson:"Account" json:"Account"`//账号
	Name string  `bson:"Name" json:"Name"`//用户名
	Head string  `bson:"Head" json:"Head"`//用户头像
	//Sex int8  `bson:"Sex" json:"Sex"`//用户性别
	Role Role  `bson:"Role" json:"Role"`//user 真实用户 robot机器人

	TotalBackYxb int64 `bson:"TotalBackYxb" json:"TotalBackYxb"` //总返回金币

	LastChatTime time.Time //最后发送消息时间
	SysProfit  int64
	BotProfit  int64
	Session gate.Session

	Ready bool //是否准备
	AutoReady bool //是否自动准备
	NotReadyCnt int `bson:"NotReadyCnt" json:"NotReadyCnt"` //累计连续不准备次数
	Hosting bool //托管
	QuitRoom bool

	StraightType StraightType

	FinalScore int64 //最终得分
	EatScore int64
	IsHavePeople bool //是不是有人
	HandPoker []int //手牌
	GivePoker []int //给的牌
	ForbidPutPoker []int //禁止出的牌
	EatData []EatData //
	PutPoker []int//
	CalcPhomData CalcPhomData
}
type EatData struct {
	Poker int
	Score int64
	PreIdx int //被吃
	LastRoundEat bool
}
type CalcPhomData struct {
	maxV int
	Phom [][]int
	State PhomState
}
type RankList struct {
	Idx int
	PointV int
	State  PhomState
}
type PhomState string
const (
	MOM 			PhomState = "Mom"
	UThuong			PhomState = "U"
	UTron			PhomState = "UTron"
	UDen      		PhomState = "UDen"
	VoNo			PhomState = "VoNo"
	UKhan			PhomState = "UKhan"
	XaoKhan			PhomState = "XaoKhan"
	Normal			PhomState = "Normal"
	First			PhomState = "First"
	Second			PhomState = "Second"
	Third			PhomState = "Third"
	Four			PhomState = "Four"
)
type WaitingListState string
const (
	PUTPOKER 		WaitingListState="PutPoker"
	DRAWPOKER 		WaitingListState="DrawPoker"
	EATPOKER 		WaitingListState="EatPoker"
	PHOM			WaitingListState="Phom"
	GivePoker		WaitingListState="GivePoker"
	JieSuan		    WaitingListState="JieSuan"
)
type WaitingList struct {
	Have bool
	Time  time.Time
	PreIdx int
	State	WaitingListState
	EatPoker int
	GivePoker []int
	PhomData PhomData
}
type PhomData struct {
	Poker []int
	State PhomState
}
type StraightScoreData struct {
	Have bool
	Idx int
	Type StraightType
	Score int64
}
type PokerPressList struct {
	WinIdx int
	PressIdx int
	Score int64
}
type PutOverRecord struct {
	Idx int
	Over bool
	LastPutCard []int //最后出的牌
	IsSpring bool //是否是春天
}
type JiesuanData struct {
	RoomState Room_v  `json:"RoomState"`
	CountDown int    `json:"CountDown"`
	PlayerInfo    	 map[int]PlayerInfo
	LastPutCard		 []int
	LastPutIdx       int
}
type PlayerInfo struct {
	StraightType StraightType
	HandPoker    []int
	TotalBackYxb int64
	PhomState PhomState
}
var BaseScoreList = []int64{
	100,500,1000,2000,5000,10000,20000,50000,100000,200000,500000,
}
var NormalTimes = map[int][]int64{
	2:{3},
	3:{2,3},
	4:{1,2,3},
}
type HallConfig struct {
	PlayerNum  int //房间类型 4就是4人房
	BaseScore int64 //底分
	BaseNum int //总人数
	MaxOffset int //最大偏移
	StepNum int
	CurNum int //当前人数
}

type Role string
const (
	USER Role = "user"
	ROBOT Role = "robot"
	Agent Role = "agent"
)

type Room_v uint8
const (
	ROOM_WAITING_START     	Room_v = 1
	ROOM_WAITING_READY      Room_v = 2 //准备
	ROOM_WAITING_XIAZHU    	Room_v = 3 //下注阶段
	ROOM_END			   	Room_v = 4
	ROOM_WAITING_ENTER	   	Room_v = 5
	ROOM_WAITING_JIESUAN	Room_v = 6//结算
	ROOM_WAITING_RESTART	Room_v = 7
	ROOM_WAITING_CLEAR		Room_v = 8
	ROOM_WAITING_SHOWPOKER   Room_v = 9 //亮牌
	ROOM_WAITING_PUTOP 		Room_v = 10
)
type HallType string
const (
	All 	HallType = "ALL"
	FOUR	HallType = "FOUR"
	TWO     HallType = "TWO"
)


func (this *MyTable)RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return this.Rand.Int63n(max-min) + min
}
func (this *MyTable) sendPackToAll(topic string,in interface{},action string,err *common.Err) error{
	if !this.BroadCast{ //广播功能
		return nil
	}
	body := this.DealProtocolFormat(in,action,err)
	//error := this.NotifyCallBackMsgNR(topic,body)
	for _,v := range this.PlayerList{
		if v.IsHavePeople && v.Role != ROBOT{
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil{
				s,_ := basegate.NewSession(this.app, sb.Session)
				this.SendCallBackMsgByQueue([]string{s.GetSessionID()},topic,body)
			}
		}
	}
	return nil
}
func (this *MyTable) sendPackToPlaying(topic string,in interface{},action string,err *common.Err) error{
	if !this.BroadCast{ //广播功能
		return nil
	}
	body := this.DealProtocolFormat(in,action,err)
	for _,v := range this.PlayerList{
		if v.Ready{
			this.SendCallBackMsgByQueue([]string{this.Players[v.UserID].Session().GetSessionID()},topic,body)
		}
	}
	return nil
}
func (this *MyTable) sendPack(session string,topic string,in interface{},action string,err *common.Err) error{
	body := this.DealProtocolFormat(in,action,err)
	error :=this.SendCallBackMsgByQueue([]string{session},topic,body)
	return error
}
func(this *MyTable) DealProtocolFormat(in interface{},action string,error *common.Err) []byte{
	info := struct {
		Data interface{}
		GameType game.Type
		Action string
		ErrMsg string
		Code int
	}{
		Data: in,
		GameType: game.CardPhom,
		Action: action,
	}
	if error == nil{
		info.Code = 0
		info.ErrMsg = "操作成功"
	}else {
		info.Code = error.Code
		info.ErrMsg = error.SetKey().ErrMsg
	}

	ret,_ := json.Marshal(info)
	return ret
}
func (this *MyTable) sendPackToLobby(session string,topic string,in interface{},action string,err *common.Err) error{
	body := this.DealProtocolFormatToLobby(in,action,err)
	error :=this.SendCallBackMsgByQueue([]string{session},topic,body)
	return error
}
func(this *MyTable) DealProtocolFormatToLobby(in interface{},action string,error *common.Err) []byte{
	info := struct {
		Data interface{}
		GameType game.Type
		Action string
		ErrMsg string
		Code int
	}{
		Data: in,
		GameType: game.Lobby,
		Action: action,
	}
	if error == nil{
		info.Code = 0
		info.ErrMsg = "操作成功"
	}else {
		info.Code = error.Code
		info.ErrMsg = error.SetKey().ErrMsg
	}

	ret,_ := json.Marshal(info)
	return ret
}
func (this *MyTable) PlayerIsTable(uid string) bool {
	for k,v := range this.PlayerList{
		if v.UserID == uid{
			this.PlayerList[k].Hosting = false
			return true
		}
	}
	return false
}
func (this *MyTable) GetPlayerIdx(userID string) int{  //获取玩家Idx
	for k,v := range this.PlayerList{
		if v.UserID == userID{
			return k
		}
	}
	return -1
}
func (this *MyTable) GetTableInfo(userID string) interface{} {
	idx := this.GetPlayerIdx(userID)
	res := make(map[string]interface{})

	tableInfo := make(map[string]interface{})
	tableInfo["RoomState"] = this.RoomState
	tableInfo["CountDown"] = this.CountDown
	tableInfo["EventID"] = this.EventID
	tableInfo["MasterIdx"] = this.GetPlayerIdx(this.Master)
	tableInfo["BaseScore"] = this.BaseScore
	tableInfo["TableID"] = strings.Split(this.tableID,"_")[0]
	tableInfo["ReadyTime"] = this.GameConf.ReadyTime
	tableInfo["PutPokerTime"] = this.GameConf.PutPokerTime
	tableInfo["BottomNum"] = len(this.Bottom)
	if this.RoomState == ROOM_WAITING_PUTOP{
		tableInfo["CurPutIdx"] = this.GetCurIdx()
	}else{
		tableInfo["CurPutIdx"] = -1
	}

	playerInfo := make(map[int]interface{})

	for k,v := range this.PlayerList{
		if k != idx{
			if v.IsHavePeople{
				info := make(map[string]interface{})
				game := make(map[string]interface{})
				game["Ready"] = v.Ready

				playerData := make(map[string]interface{})
				if this.RoomState == ROOM_WAITING_PUTOP || this.RoomState == ROOM_WAITING_JIESUAN{
					if this.PlayerList[idx].Ready && v.Ready{
						info["UserID"] = v.UserID
						info["Head"] = v.Head
						info["Name"] = v.Name
						info["Yxb"] = v.Yxb

						playerData["Info"] = info
					}else{
						playerData["Info"] = nil
					}
					eatPk := make([]int,0)
					for _,v1 := range v.EatData{
						eatPk = append(eatPk,v1.Poker)
					}
					eatPokerTotal := make([]int,len(eatPk))
					copy(eatPokerTotal,eatPk)
					phomPoker := make([]int,0)
					for _,v := range this.PlayerList[k].CalcPhomData.Phom{
						for _,v1 := range v{
							phomPoker = append(phomPoker,v1)
						}
					}
					eatPk = this.RemoveTblInList(eatPk,phomPoker)
					eatPokerPhom := this.RemoveTblInList(eatPokerTotal,eatPk)
					game["EatPokerPhom"] = eatPokerPhom
					game["EatPoker"] = eatPk
					game["PutPoker"] = v.PutPoker
					game["PhomData"] = v.CalcPhomData
				}else{
					playerData["Info"] = nil
					game["EatPokerPhom"] = []int{}
					game["EatPoker"] = []int{}
					game["PutPoker"] = nil
					game["PhomData"] = nil
				}
				playerData["Game"] = game

				playerInfo[k] = playerData

			}
		}
	}

	selfInfo := make(map[string]interface{})
	info := make(map[string]interface{})
	game := make(map[string]interface{})
	info["UserID"] = this.PlayerList[idx].UserID
	info["Head"] = this.PlayerList[idx].Head
	info["Name"] = this.PlayerList[idx].Name
	info["Yxb"] = this.PlayerList[idx].Yxb
	info["Idx"] = idx

	game["Ready"] = this.PlayerList[idx].Ready
	if this.RoomState == ROOM_WAITING_PUTOP || this.RoomState == ROOM_WAITING_JIESUAN{
		if this.PlayerList[idx].Ready{
			game["Poker"] = this.PlayerList[idx].HandPoker
		}else{
			game["Poker"] = []int{}
		}
		eatPk := make([]int,0)
		for _,v := range this.PlayerList[idx].EatData{
			eatPk = append(eatPk,v.Poker)
		}
		eatPokerTotal := make([]int,len(eatPk))
		copy(eatPokerTotal,eatPk)
		phomPoker := make([]int,0)
		for _,v := range this.PlayerList[idx].CalcPhomData.Phom{
			for _,v1 := range v{
				phomPoker = append(phomPoker,v1)
			}
		}
		eatPk = this.RemoveTblInList(eatPk,phomPoker)
		eatPokerPhom := this.RemoveTblInList(eatPokerTotal,eatPk)
		game["EatPokerPhom"] = eatPokerPhom
		game["EatPoker"] = eatPk
		game["PutPoker"] = this.PlayerList[idx].PutPoker
		game["PhomData"] = this.PlayerList[idx].CalcPhomData
	}else{
		game["PhomState"] = ""
		game["Poker"] = []int{}
		game["EatPokerPhom"] = []int{}
		game["EatPoker"] = []int{}
		game["PutPoker"] = nil
		game["PhomData"] = nil
	}
	if userID == this.Master && this.RoomState == ROOM_WAITING_READY{
		if this.GetReadyPlayerNum() >= 2{
			game["CanStartGame"] = true
		}else{
			game["CanStartGame"] = false
		}
	}

	val,_ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if waitingList.Have{
		game["WaitingList"] = waitingList
	}else{
		game["WaitingList"] = nil
	}
	selfInfo["Info"] = info
	selfInfo["Game"] = game


	res["TableInfo"] = tableInfo
	res["PlayerInfo"] = playerInfo
	res["SelfInfo"] = selfInfo
	if this.RoomState == ROOM_WAITING_JIESUAN && this.PlayerList[idx].Ready{
		res["JieSuanData"] = this.JieSuanData
	}

	return res

}
func (this *MyTable) GetPlayerInfo(userID string) interface{} {
	idx := this.GetPlayerIdx(userID)
	res := make(map[string]interface{})

	tableInfo := make(map[string]interface{})
	tableInfo["RoomState"] = this.RoomState
	tableInfo["CountDown"] = this.CountDown
	tableInfo["EventID"] = this.EventID
	tableInfo["MasterIdx"] = this.GetPlayerIdx(this.Master)

	playerInfo := make(map[int]interface{})

	for k,v := range this.PlayerList{
		if k != idx{
			if v.IsHavePeople{
				info := make(map[string]interface{})
				game := make(map[string]interface{})
				game["Ready"] = v.Ready

				playerData := make(map[string]interface{})
				if (this.RoomState == ROOM_WAITING_PUTOP || this.RoomState == ROOM_WAITING_JIESUAN) && this.PlayerList[idx].Ready && v.Ready{
					info["UserID"] = v.UserID
					info["Head"] = v.Head
					info["Name"] = v.Name
					info["Yxb"] = v.Yxb
					val,_ := this.WaitingList.Load(k)
					waitingList := val.(WaitingList)
					game["ShowPoker"] = waitingList

					playerData["Info"] = info
				}else{
					playerData["Info"] = nil
				}
				playerData["Game"] = game

				playerInfo[k] = playerData

			}
		}
	}

	res["TableInfo"] = tableInfo
	res["PlayerInfo"] = playerInfo

	return res
}
func (this *MyTable) Shuffle() []int{
	pool := make([]int,len(card))
	copy(pool,card)
	n := len(pool)
	for i := 0;i < n;i++{
		idx := this.RandInt64(1,int64(n + 1)) - 1
		tmpIdx := this.RandInt64(1,int64(n + 1)) - 1
		pool[idx],pool[tmpIdx] = pool[tmpIdx],pool[idx]
	}

	for i := 0;i < n;i++{
		idx := this.RandInt64(1,int64(n + 1)) - 1
		tmpIdx := this.RandInt64(1,int64(n + 1)) - 1
		pool[idx],pool[tmpIdx] = pool[tmpIdx],pool[idx]
	}

	for i := 0;i < n;i++{
		idx := this.RandInt64(1,int64(n + 1)) - 1
		tmpIdx := this.RandInt64(1,int64(n + 1)) - 1
		pool[idx],pool[tmpIdx] = pool[tmpIdx],pool[idx]
	}

	return pool
}
func (this *MyTable) DealPhomPoker(idx int){
	nextIdx := this.GetNextPutIdx(this.FirstPhom)
	val,_ := this.WaitingList.Load(idx)
	waitingList := val.(WaitingList)
	if len(this.PlayerList[this.FirstPhom].PutPoker) > 0{
		movePoker := this.PlayerList[this.FirstPhom].PutPoker[len(this.PlayerList[this.FirstPhom].PutPoker) - 1]
		srcIdx := this.FirstPhom
		desIdx := waitingList.PreIdx

		if srcIdx != desIdx{
			this.PlayerList[srcIdx].PutPoker = append(this.PlayerList[srcIdx].PutPoker[:len(this.PlayerList[srcIdx].PutPoker)-1])
			this.PlayerList[desIdx].PutPoker = append(this.PlayerList[desIdx].PutPoker,movePoker)

			res := make(map[string]interface{})
			res["srcIdx"] = srcIdx
			res["desIdx"] = desIdx
			res["movePoker"] = movePoker
			this.sendPackToAll(game.Push,res,protocol.MovePoker,nil)
		}
	}
	this.FirstPhom = nextIdx
}
func (this *MyTable) DealForbidPutPoker(idx int){
	eatPoker := make([]int,0)
	for _,v := range this.PlayerList[idx].EatData{
		eatPoker =append(eatPoker,v.Poker)
	}
	phomPoker := make([]int,0)
	for _,v := range this.PlayerList[idx].CalcPhomData.Phom{
		for _,v1 := range v{
			phomPoker = append(phomPoker,v1)
		}
	}
	eatPoker = this.RemoveTblInList(eatPoker,phomPoker)
	handPoker :=this.FindNotInArrayList(eatPoker,this.PlayerList[idx].HandPoker)
	forbidPutPoker := make([]int,len(eatPoker))
	copy(forbidPutPoker,eatPoker)
	for _,v := range handPoker{
		comp := this.FindNotInArrayList([]int{v},handPoker)
		forbid := false
		for _,v1 := range eatPoker{
			res := this.CheckEatPoker(comp,[]int{},v1)
			if len(res) == 0{
				forbid = true
				break
			}
		}
		if forbid{
			forbidPutPoker = append(forbidPutPoker,v)
		}
	}

	this.PlayerList[idx].ForbidPutPoker = forbidPutPoker
}
func(this *MyTable) CalcPokerVal(poker []int) int{
	val := 0
	for _,v := range poker{
		val += v % 0x10
	}
	return val
}
func(this *MyTable) CalcPhomPoker(idx,phomNum int,poker []int,phomRes [][]int){
	resPhom := make([][]int,len(phomRes))
	copy(resPhom,phomRes)
	pk := make([]int,len(poker))
	copy(pk,poker)
	sort.Slice(pk, func(i, j int) bool { //升序排序
		if pk[i] % 0x10 == pk[j] % 0x10{
			return pk[i] < pk[j]
		}
		return pk[i] % 0x10 < pk[j] % 0x10
	})
	straight := this.CompStraight(pk)
	three := this.CompThreeOfAKind(pk)
	four := this.CompFourOfAKind(pk)
	phom := make([][]int,0)
	for _,v := range straight{
		phom = append(phom,v)
	}
	for _,v := range three{
		phom = append(phom,v)
	}
	for _,v := range four{
		phom = append(phom,v)
	}
	if len(phom) == 0 {
		eatPk := make([]int,0)
		for _,v := range this.PlayerList[idx].EatData{
			eatPk = append(eatPk,v.Poker)
		}
		//if len(eatPk) > 0{
		//	log.Info("'")
		//}
		//phomPoker := make([]int,0)
		//for _,v := range this.PlayerList[idx].CalcPhomData.Phom{
		//	for _,v1 := range v{
		//		phomPoker = append(phomPoker,v1)
		//	}
		//}
		//eatPk = this.RemoveTblInList(eatPk,phomPoker)
		phomPk := make([]int,0)
		for _,v := range resPhom{
			for _,v1 := range v{
				phomPk = append(phomPk,v1)
			}
		}
		if this.IsContainArray(eatPk,phomPk) {
			if phomNum == 3 {
				val := phomNum*0xFFFFFF - this.CalcPokerVal(pk)*0xFFFF - len(pk)*0xFFF - pk[len(pk)-1]%0x10*0xFF - pk[len(pk)-1]/0x10
				if val > this.CalcPhomData.maxV {
					this.CalcPhomData.maxV = val
					this.CalcPhomData.Phom = resPhom
				}
			} else {
				val := 0xFFFFFF - this.CalcPokerVal(pk)*0xFFFF - len(pk)*0xFFF - pk[len(pk)-1]%0x10*0xFF - pk[len(pk)-1]/0x10
				if val > this.CalcPhomData.maxV {
					this.CalcPhomData.maxV = val
					this.CalcPhomData.Phom = resPhom
				}
			}
		}
		return
	}
	phomNum++
	for _,v := range phom{
		remainPk := this.FindNotInArrayList(v,pk)
		resPhom = append(resPhom,v)
		if len(remainPk) == 0{
			val := 0xFFFFFF
			if phomNum == 3{
				val = phomNum*0xFFFFFF + 1
			}
			if val > this.CalcPhomData.maxV {
				this.CalcPhomData.maxV = val
				this.CalcPhomData.Phom = resPhom
			}
			resPhom = append(resPhom[:len(resPhom)-1])
			//continue
			return
		}
		this.CalcPhomPoker(idx,phomNum,remainPk,resPhom)
		resPhom = append(resPhom[:len(resPhom)-1])
	}
}

type GivePokerData struct {
	GiveIdx int
	GetIdx int
	GetPhomIdx int
	GetPhomPk []int
	Poker []int
}
func(this *MyTable) CheckGivePoker(idx int,poker []int) []GivePokerData{
	res := make([]GivePokerData,0)
	haveGive := make([]int,0)
	for _,v := range poker {
		find := false
		for k1, v1 := range this.PlayerList {
			if k1 != idx && len(v1.CalcPhomData.Phom) > 0 && v1.CalcPhomData.State != "" {
				for k2, v2 := range v1.CalcPhomData.Phom {
					if len(v2) > 0 && len(this.CheckGivePokerFour(v2, []int{}, v)) > 0 {
						find = true
						phomPk := append(v2, v)
						res = append(res, GivePokerData{
							GiveIdx:    idx,
							GetIdx:     k1,
							GetPhomIdx: k2,
							GetPhomPk:  phomPk,
							Poker:      []int{v},
						})
						haveGive = append(haveGive,v)
						break
					}
				}
			}
			if find {
				break
			}
		}
	}
	for _,v := range poker{
		find := false
		if this.IsContainElement(v,haveGive){
			continue
		}
		for k1,v1 := range this.PlayerList{
			if k1 != idx && len(v1.CalcPhomData.Phom) > 0{
				for k2,v2 := range v1.CalcPhomData.Phom{
					if len(v2) > 0 && len(this.CheckGivePokerStraight(v2,[]int{},v)) > 0{
						find = true
						phomPk := append(v2,v)
						findRes := -1
						for k3,v3 := range res{
							if v3.GetPhomIdx == k2 && v3.GetIdx == k1{
								findRes = k3
								break
							}
						}
						if findRes >= 0 {
							nearGive := this.CheckNearGivePoker(poker,haveGive,v)
							for _,v3 := range nearGive{
								res[findRes].Poker = append(res[findRes].Poker,v3)
								res[findRes].GetPhomPk = append(res[findRes].GetPhomPk,v3)
								haveGive = append(haveGive,v3)
							}
							res[findRes].Poker = append(res[findRes].Poker,v)
							res[findRes].GetPhomPk = append(res[findRes].GetPhomPk,v)
							haveGive = append(haveGive,v)
						}else{
							pk := make([]int,0)
							pk =append(pk,v)
							nearGive := this.CheckNearGivePoker(poker,haveGive,v)
							for _,v3 := range nearGive{
								pk= append(pk,v3)
								phomPk = append(phomPk,v3)
								haveGive = append(haveGive,v3)
							}
							res = append(res,GivePokerData{
								GiveIdx: idx,
								GetIdx: k1,
								GetPhomPk: phomPk,
								GetPhomIdx: k2,
								Poker: pk,
							})
							haveGive = append(haveGive,v)
						}
						break
					}
				}
			}
			if find{
				break
			}
		}
	}
	return res
}
func(this *MyTable) CheckPhomPoker(poker []int,phomRes [][]int) (bool,[][]int){
	resPhom := make([][]int,len(phomRes))
	copy(resPhom,phomRes)
	pk := make([]int,len(poker))
	copy(pk,poker)
	straight := this.CompStraight(pk)
	three := this.CompThreeOfAKind(pk)
	four := this.CompFourOfAKind(pk)
	phom := make([][]int,0)
	for _,v := range straight{
		phom = append(phom,v)
	}
	for _,v := range three{
		phom = append(phom,v)
	}
	for _,v := range four{
		phom = append(phom,v)
	}
	if len(phom) == 0{
		if len(pk) == 0{
			return true,resPhom
		}
		return false,resPhom
	}
	for _,v := range phom{
		remainPk := this.FindNotInArrayList(v,pk)
		resPhom = append(resPhom,v)
		ret,res := this.CheckPhomPoker(remainPk,resPhom)
		if ret{
			return ret,res
		}
		resPhom = append(resPhom[:len(resPhom)-1])
	}
	return false,resPhom
}
func (this *MyTable) CheckNotStraight(poker []int) bool{
	pk := make([]int,len(poker))
	copy(pk,poker)
	sort.Slice(pk, func(i, j int) bool { //升序排序
		return pk[i] < pk[j]
	})
	for _,v := range pk{
		remainPk := this.FindNotInArrayList([]int{v},pk)
		for _,v1 := range remainPk{
			if v % 0x10 == v1 % 0x10{
				return false
			}

			if v - v1 <= 2 && v - v1 >= -2{
				return false
			}
		}
	}
	return true
}
func (this *MyTable) GetStraightScoreIdx() StraightScoreData{
	idx := this.FirstPhom
	res := StraightScoreData{}
	for true{
		this.PlayerList[idx].CalcPhomData = CalcPhomData{}
		this.CalcPhomPoker(idx,0,this.PlayerList[idx].HandPoker,[][]int{})
		if len(this.CalcPhomData.Phom) == 3{
			for _,v := range this.CalcPhomData.Phom{
				this.PlayerList[idx].HandPoker = this.RemoveTblInList(this.PlayerList[idx].HandPoker,v)
			}
			res.Have = true
			res.Idx = idx
			res.Type = ThreePhom
			if len(this.PlayerList[idx].HandPoker) == 0{
				res.Score = this.BaseScore * 10
			}else{
				res.Score = this.BaseScore * 5
			}
			break
		}

		if this.CheckNotStraight(this.PlayerList[idx].HandPoker){
			res.Have = true
			res.Idx = idx
			res.Type = NotStraight
			if len(this.PlayerList[idx].HandPoker) == 10{
				res.Score = this.BaseScore * 10
			}else{
				res.Score = this.BaseScore * 5
			}
			this.PlayerList[idx].CalcPhomData.State = UKhan
			break
		}

		idx = this.GetNextPutIdx(idx)
		if idx == this.FirstPhom{
			break
		}
	}

	return res
}
func (this *MyTable) DealPoker(pool []int){
	for k,_ := range this.PlayerList{
		this.PlayerList[k].HandPoker = []int{}
	}
	for i := 0;i < 9;i++{
		for k,v := range this.PlayerList{
			if v.Ready{
				this.PlayerList[k].HandPoker = append(this.PlayerList[k].HandPoker,pool[len(pool) - 1])
				pool = append(pool[:len(pool)-1])
			}
		}
	}
	for true{
		idx := this.RandInt64(1,int64(this.TotalPlayerNum + 1)) - 1
		if this.PlayerList[idx].Ready{
			this.FirstPhom = int(idx)
			this.PlayerList[idx].HandPoker = append(this.PlayerList[idx].HandPoker,pool[len(pool) - 1])
			pool = append(pool[:len(pool)-1])
			break
		}
	}
	this.Bottom = []int{}
	if this.PlayingNum == 4{
		for i := 0;i < 15;i++{
			this.Bottom = append(this.Bottom,pool[i])
		}
	}else if this.PlayingNum == 3{
		for i := 0;i < 11;i++{
			this.Bottom = append(this.Bottom,pool[i])
		}
	}else if this.PlayingNum == 2{
		for i := 0;i < 7;i++{
			this.Bottom = append(this.Bottom,pool[i])
		}
	}

	this.Pool = this.RemoveTblInList(this.Pool,this.Bottom)

	if this.NeedControl{
		this.ControlDealerPoker()
		for k,v := range this.PlayerList{ //
			if v.Ready && len(v.HandPoker) == 10{
				this.FirstPhom = k
				break
			}
		}
	}
	//this.FirstPhom = 3
	//this.Bottom = []int{73,28,53,61,17,34,57,24,52,70,36,74,42,41,20}
	//this.PlayerList[0].HandPoker = []int{22,69,77,55,21,33,56,23,49}
	//this.PlayerList[1].HandPoker = []int{75,54,68,58,39,19,59,66,35}
	//this.PlayerList[2].HandPoker = []int{67,76,29,25,18,45,65,43,38}
	//this.PlayerList[3].HandPoker = []int{27,60,40,44,26,37,72,51,71,50}

	//for k,v := range this.PlayerList{
	//	if v.Ready{
	//		log.Info("----------------------------deal pooker--idx = %d----------------------- ",k)
	//		res,_ := json.Marshal(v.HandPoker)
	//		log.Info("10进制 deal poker %s",res)
	//		//pk := ""
	//		//x := fmt.Sprintf("%x",v.HandPoker)
	//		//pk += x
	//		//log.Info("16进制 deal poker %s",pk)
	//	}
	//
	//}
	//res,_ := json.Marshal(this.Bottom)
	//log.Info("--------------10进制 bottom poker %s",res)
	//res,_ = json.Marshal(pool)
	//log.Info("10进制 deal poker %s",res)
}
func(this *MyTable) GetCurIdx() int{
	for k,v := range this.PlayerList{
		if v.Ready{
			val,_ := this.WaitingList.Load(k)
			waitingList := val.(WaitingList)
			if waitingList.Have{
				return k
			}
		}
	}
	return -1
}
func (this *MyTable) GetRealReadyPlayerNum() int{
	playerNum := 0
	for _,v := range this.PlayerList{
		if v.IsHavePeople && v.Role == USER && v.Ready{
			playerNum++
		}
	}
	return playerNum
}
func (this *MyTable) SliceOutOfOrder(in []int) []int{
	n := len(in)
	for i := 0;i < n;i++{
		idx := this.RandInt64(1,int64(n + 1)) - 1
		tmpIdx := this.RandInt64(1,int64(n + 1)) - 1
		in[idx],in[tmpIdx] = in[tmpIdx],in[idx]
	}
	return in
}
func(this *MyTable) SwitchRoomState() interface{}{ //切换房间状态
	info := struct {
		RoomState Room_v
		CountDown int
		EventID string
	}{
		RoomState: this.RoomState,
		CountDown: this.CountDown,
		EventID: this.EventID,
	}
	_ = this.sendPackToAll(game.Push,info,protocol.SwitchRoomState,nil)
	//time.Sleep(500 * time.Millisecond)
	return info
}
func (this *MyTable)notifyWallet(uid string) {
	sb := vGate.QuerySessionBean(uid)
	if sb == nil{
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	msg := make(map[string]interface{})
	msg["Wallet"] = wallet
	msg["Action"] = "wallet"
	msg["GameType"] = game.All
	b, _ := json.Marshal(msg)
	this.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, b)
	this.onlinePush.ExecuteCallBackMsg(this.Trace())
	agentStorage.OnWalletChange(uid)
}
func (this *MyTable) Combinations(iterable []int, r int) [][]int{
	var res [][]int
	res = [][]int{}
	pool := iterable
	n := len(pool)

	if r > n {
		return res
	}

	indices := make([]int, r)
	for i := range indices {
		indices[i] = i
	}

	result := make([]int, r)
	result1 := make([]int, r)
	for i, el := range indices {
		result[i] = pool[el]
	}
	copy(result1,result)
	res = append(res,result1)
	for {
		i := r - 1
		result2 := make([]int, r)
		for ; i >= 0 && indices[i] == i+n-r; i -= 1 {
		}

		if i < 0 {
			return res
		}

		indices[i] += 1
		for j := i + 1; j < r; j += 1 {
			indices[j] = indices[j-1] + 1
		}

		for ; i < len(indices); i += 1 {
			result[i] = pool[indices[i]]
		}
		copy(result2,result)
		res = append(res,result2)
	}

}
func (this *MyTable) GetPlayerNum() int{
	num := 0
	for _,v := range this.PlayerList{
		if v.IsHavePeople{
			num++
		}
	}
	return num
}
func (this *MyTable)FindNotInArrayList(child []int,parent []int) []int{ //
	res := make([]int,0)
	for _,v := range parent{
		find := false
		for _,v1 := range child{
			if v1 == v {
				find = true
				break
			}
		}
		if !find{
			res = append(res,v)
		}
	}
	return res
}
func (this *MyTable)IsContainArray(child []int,parent []int) bool{ //
	for _,v := range child{
		find := false
		for _,v1 := range parent{
			if v1 == v {
				find = true
				break
			}
		}
		if !find{
			return false
		}
	}
	return true
}
func (this *MyTable)IsContainElement(child int,parent []int) bool{ //
	for _,v1 := range parent{
		if v1 == child {
			return true
		}
	}
	return false
}
func (this *MyTable)CopyMap(src map[int]int) map[int]int{
	dst := map[int]int{}
	for k,v := range src{
		dst[k] = v
	}
	return dst
}
func (this *MyTable) GetRobotNum() int{
	num := 0
	for _,v := range this.PlayerList{
		if v.IsHavePeople && v.Role == ROBOT{
			num++
		}
	}
	return num
}
func (this *MyTable) GetTablePlayerNum() int{
	playerNum := 0
	for _,v := range this.PlayerList{
		if v.IsHavePeople{
			playerNum++
		}
	}
	return playerNum
}
func (this *MyTable) GetReadyPlayerNum() int{
	num := 0
	for _,v := range this.PlayerList{
		if v.Ready{
			num++
		}
	}
	return num
}

func (this *MyTable) GetTableRealPlayerNum() int{
	playerNum := 0
	for _,v := range this.PlayerList{
		if v.IsHavePeople && v.Role != ROBOT{
			playerNum++
		}
	}
	return playerNum
}
func (this *MyTable) SendState(stateList sync.Map){
	stateList.Range(func(k, v interface{}) bool { //
		if v.(WaitingList).Have{
			sb := vGate.QuerySessionBean(this.PlayerList[k.(int)].UserID)
			if sb != nil{
				session,_ := basegate.NewSession(this.app, sb.Session)
				this.sendPack(session.GetSessionID(),game.Push,v.(WaitingList),protocol.NotifyWaitingState,nil)
			}
		}
		return true
	})
}
func (this *MyTable)RemoveTblInList(list []int,tbl []int) []int{ //
	listCopy := make([]int,len(list))
	copy(listCopy,list)
	tblCopy := make([]int,len(tbl))
	copy(tblCopy,tbl)

	for _,v := range tblCopy{
		for k1,v1 := range listCopy{
			if v == v1{
				listCopy = append(listCopy[:k1],listCopy[k1 + 1:]...)
				break
			}
		}
	}
	return listCopy
}
func (this *MyTable)InitWaitingList(){
	for k,_ := range this.PlayerList{
		this.WaitingList.Store(k,WaitingList{})
	}
}
func (this *MyTable)GetNextPutIdx(idx int) int{ //
	nextIdx := idx
	for true{
		nextIdx++
		if nextIdx >= this.TotalPlayerNum {
			nextIdx -= this.TotalPlayerNum
		}
		if this.PlayerList[nextIdx].Ready {
			break
		}
	}
	return nextIdx
}
func (this *MyTable)GetNextPutIdxAll(idx int) int{ //
	nextIdx := idx
	for true{
		nextIdx++
		if nextIdx >= this.TotalPlayerNum {
			nextIdx -= this.TotalPlayerNum
		}
		if this.PlayerList[nextIdx].Ready{
			break
		}
	}
	return nextIdx
}
func (this *MyTable) SliceRemoveDuplicates(slice []int) []int {
	sort.Ints(slice)
	i:= 0
	var j int
	for{
		if i >= len(slice)-1 {
			break
		}

		for j = i + 1; j < len(slice) && slice[i] == slice[j]; j++ {
		}
		slice= append(slice[:i+1], slice[j:]...)
		i++
	}
	return slice
}

func (this *MyTable) GetMinPoker(pk []int) int{
	min := -1
	minV := -1
	for _,v := range pk{
		if min < 0{
			min = v
			minV = v % 0x10 * 0xFF + v / 0x10
		}else if minV > (v % 0x10 * 0xFF + v / 0x10){
			min = v
			minV = v % 0x10 * 0xFF + v / 0x10
		}
	}
	return min
}
func (this *MyTable)GetNextIdx(idx int) int{ //
	nextIdx := idx
	if this.GetTablePlayerNum() == 0{
		return nextIdx
	}
	for true{
		nextIdx++
		if nextIdx >= this.TotalPlayerNum {
			nextIdx -= this.TotalPlayerNum
		}
		if this.PlayerList[nextIdx].IsHavePeople{
			break
		}
		if nextIdx == idx{
			break
		}
	}
	return nextIdx
}