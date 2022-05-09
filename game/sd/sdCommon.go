package sd

import (
	"encoding/json"
	"sort"
	"time"
	"vn/common"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/agentStorage"
	"vn/storage/sdStorage"
	"vn/storage/walletStorage"
)
const ResultsRecordNum = 32 //最近多少条开奖记录
//
////-----------------机器人---
const MaxOffset = 20 //最大偏移量
const StepNum = 3 //每次变换值

type Room_v uint8
const (
	ROOM_WAITING_START     	Room_v = 1
	ROOM_WAITING_READY      Room_v = 2 //准备阶段，摇盆
	ROOM_WAITING_XIAZHU    	Room_v = 3 //下注阶段
	ROOM_END			   	Room_v = 4
	ROOM_WAITING_ENTER	   	Room_v = 5
	ROOM_WAITING_JIESUAN	Room_v = 6//结算
	ROOM_WAITING_RESTART	Room_v = 7
	ROOM_WAITING_CLEAR		Room_v = 8
)
type Role string
const (
	USER Role = "user"
	ROBOT Role = "robot"
	Agent Role = "agent"
)
type JiesuanData struct {
	RoomState     Room_v `json:"RoomState"`
	CountDown     int    `json:"CountDown"`
	PrizePool     int64  `json:"PrizePool"`
	Results       map[string]sdStorage.Result
	PrizeResults  []sdStorage.XiaZhuResult
	XiaZhuTime    int
	JieSuanTime   int
	ReadyGameTime int
	ResultsRecord sdStorage.ResultsRecord

	PositionInfo []PositionInfo
	TotalBackYxb int64
}
type PositionInfo struct {
	TotalBackYxb int64
	Yxb int64
	UserID string
}
type PlayerList struct {
	Yxb int64   `bson:"Yxb" json:"Yxb"`//游戏币
	UserID string `bson:"UserID" json:"UserID"`//用户id
	Name string  `bson:"Name" json:"Name"`//用户名
	Head string  `bson:"Head" json:"Head"`//用户头像
	//Sex int8  `bson:"Sex" json:"Sex"`//用户性别
	Role Role  `bson:"Role" json:"Role"`//user 真实用户 robot机器人
	DoubleState bool `bson:"DoubleState" json:"DoubleState"`//加倍下注状态
	LastState bool `bson:"LastState" json:"LastState"`//上轮下注状态

	XiaZhuResult map[sdStorage.XiaZhuResult] []int64 `bson:"XiaZhuResult" json:"XiaZhuResult"`//单注下注结果
	XiaZhuResultTotal map[sdStorage.XiaZhuResult] int64 `bson:"XiaZhuResultTotal" json:"XiaZhuResultTotal"`//下注总结果
	LastXiaZhuResult map[sdStorage.XiaZhuResult] []int64 `bson:"LastXiaZhuResult" json:"LastXiaZhuResult"`//上次下注结果
	TotalBackYxb int64 `bson:"TotalBackYxb" json:"TotalBackYxb"` //总返回金币

	ResultsPool int64 `bson:"ResultsPool" json:"ResultsPool"` //奖池中奖金额

	NotXiaZhuCnt int `bson:"NotXiaZhuCnt" json:"NotXiaZhuCnt"` //累计连续不下注次数
	LastChatTime time.Time //最后发送消息时间
	SysProfit  int64
	BotProfit  int64
}
type RobotXiaZhuList struct {
	XiaZhu map[string][]int64
}

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
	error := this.NotifyCallBackMsgNR(topic,body)
	return error
}
func (this *MyTable) sendPack(session string,topic string,in interface{},action string,err *common.Err) error{
	body := this.DealProtocolFormat(in,action,err)
	error :=this.SendCallBackMsgNR([]string{session},topic,body)
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
		GameType: game.SeDie,
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
func (this *MyTable) UpdatePlayerList(){  //交换位置
	for k,v := range this.PlayerList{
		if v.Role == USER || v.Role == Agent{
			wallet := walletStorage.QueryWallet(utils.ConvertOID(v.UserID))
			this.PlayerList[k].Yxb = wallet.VndBalance
		}
	}
	sort.Slice(this.PlayerList, func(i, j int) bool {
		return this.PlayerList[i].Yxb > this.PlayerList[j].Yxb
	})
	this.PositionList = []PlayerList{}
	for k,v := range this.PlayerList{
		if k < this.SeatNum{
			this.PositionList = append(this.PositionList,v)
		}
	}
	_ = this.sendPackToAll(game.Push,this.PositionList,protocol.UpdatePlayerList,nil)
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
	return info
}
func (this *MyTable) GetPlayerIdx(userID string) int{  //获取玩家Idx
	for k,v := range this.PlayerList{
		if v.UserID == userID{
			return k
		}
	}
	return -1
}
func (this *MyTable) GetPlayerInfo(userID string,enter bool) interface{} {
	type PlayerInfo struct {
		ChipsList []int64
		DoubleState bool
		LastState bool
		XiaZhuResultTotal map[sdStorage.XiaZhuResult] int64
		//Idx int
		UserID string
		Yxb int64
		Name string
		Head string
	}
	var info PlayerInfo
	idx := this.GetPlayerIdx(userID)
	info.UserID = userID
	if idx >= 0 {
		if enter{
			info.ChipsList = this.ChipsList
		}else{
			info.ChipsList = nil
		}
		info.DoubleState = this.PlayerList[idx].DoubleState
		info.LastState = this.PlayerList[idx].LastState
		info.XiaZhuResultTotal = this.PlayerList[idx].XiaZhuResultTotal
		info.Name = this.PlayerList[idx].Name
		info.Yxb = this.PlayerList[idx].Yxb
		info.Head = this.PlayerList[idx].Head
	}
	return info
}
func (this *MyTable) GetTableInfo(countDown bool) interface{} {
	//发送前端的数据结构
	type PositionInfo struct {
		TotalBackYxb int64
		Yxb int64
		UserID string
	}
	var positionInfo []PositionInfo
	for k,v := range this.PlayerList{
		if k < this.SeatNum{
			info := PositionInfo{
				UserID: v.UserID,
				Yxb: v.Yxb,
			}
			positionInfo = append(positionInfo,info)
		}else{
			break
		}
	}
	type Info struct {
		RoomState Room_v  `json:"RoomState"`
		CountDown int	`json:"CountDown"`
		PrizePool int64	`json:"PrizePool"`
		PlayerNum int
		//ResultsChipList []map[sdStorage.XiaZhuResult] int64
		Results JiesuanData
		PrizeResults []sdStorage.XiaZhuResult  //桌上中奖结果
		XiaZhuTotal map[sdStorage.XiaZhuResult]int64//map[sdStorage.XiaZhuResult] int64
		XiaZhuTime int
		JieSuanTime int
		ReadyGameTime int
		ResultsRecord sdStorage.ResultsRecord //路单
		PositionInfo []PositionInfo
		EventID string
		Hundred bool
	}

	var info Info

	info.RoomState = this.RoomState
	info.Hundred = this.Hundred
	if countDown{
		info.CountDown = this.CountDown
		resultRecord := sdStorage.GetResultsRecord(this.tableID)
		info.ResultsRecord = resultRecord
	}else{
		info.CountDown = -1
		info.ResultsRecord.ResultsRecordNum = -1
	}

	info.PlayerNum = this.PlayerNum
	//info.ResultsChipList = tableInfo.ResultsChipList
	if this.RoomState == ROOM_WAITING_JIESUAN { //结算数据
		info.Results = this.JieSuanData
	}
	info.PositionInfo = positionInfo
	info.XiaZhuTotal = map[sdStorage.XiaZhuResult]int64{}
	this.XiaZhuTotal.Range(func(key, value interface{}) bool { //启动table队列
		info.XiaZhuTotal[key.(sdStorage.XiaZhuResult)] = value.(int64)
		return true
	})
	info.XiaZhuTime = this.GameConf.XiaZhuTime
	info.JieSuanTime = this.GameConf.JieSuanTime
	info.ReadyGameTime = this.GameConf.ReadyGameTime
	info.EventID = this.EventID
	return info
}

func (this *MyTable) PlayerIsTable(uid string) bool {
	for _,v := range this.PlayerList{
		if v.UserID == uid{
			return true
		}
	}
	return false
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
func (this *MyTable)DeepCopyPlayerList(in []PlayerList) []PlayerList{
	var out []PlayerList
	for _,v := range in{
		out = append(out,v)
	}
	return out
}