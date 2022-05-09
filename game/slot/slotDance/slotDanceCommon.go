package slotDance

import (
	"encoding/json"
	"vn/common"
	"vn/common/utils"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/agentStorage"
	"vn/storage/slotStorage/slotDanceStorage"
	"vn/storage/walletStorage"
)

type Config struct {
	OddsList map[slotDanceStorage.Symbol]map[int]int64

}

const TotalRows = 3
const MinWinLine = 3 //最小三连线才中奖
const BaseCoinNum = 50 //50个硬币一次

type JieSuanData struct {
	WildPositions []map[int64]int64
	ScatterPositions []map[int64]int64
	ResultPositions []int64 //转轴最后的位置
	WildTimes int64
	Result []Result
	TotalBackScore int64

	FreeData  FreeData
	MusicType MusicType
	AnimationType AnimationType
	TrialData TrialData
	CoinNum  int64
	CoinValue int64
}
type FreeData struct {
	FreeRemainTimes int //剩余次数
	FreeGame        bool//是否中freeGame
	FreeTimes       int64//当前倍数
	FreeStepTimes   []int64 //每次增加的倍数
	FreeTotalScore  int64 //freeGame总得分
	FreeUsedTimes   int64 //用了多少次
}
type TrialData struct {
	VndBalance     int64   `bson:"VndBalance" json:"VndBalance"`
}
type TrialModeConf struct { //试玩模式
	VndBalance     int64
}
type Role string
const (
	USER Role = "user"
	ROBOT Role = "robot"
	Agent Role = "agent"
)
type ModeType string
const (
	NORMAL 	ModeType = "normal"
	TRIAL	ModeType = "trial"
)
type Result struct {
	SymbolPositions []map[int64]int64    //出现的位置
	LineType        int                  //连线的类型  3连 4连 5连
	Symbol          slotDanceStorage.Symbol //图案
	SymbolScore     int64                //总得分
	CoinValue       int64                //硬币值
	HaveWild        bool                 //是否有wild
	GroupNum        int64        //第一列的组数
}
type MusicType string
const (
	WIN1 	MusicType = "win1"
	WIN2 	MusicType = "win2"
	WIN3 	MusicType = "win3"
	WINBig	MusicType = "winBig"
)
type AnimationType string
const (
	BigAnimation1 	AnimationType = "BigAnimation1"
	BigAnimation2 	AnimationType = "BigAnimation2"
)
var OddsList = map[slotDanceStorage.Symbol]map[int]int64{
	slotDanceStorage.D1: {
		3:10,
		4:20,
		5:100,
	},
	slotDanceStorage.D2: {
		3:10,
		4:20,
		5:100,
	},
	slotDanceStorage.D3: {
		3:15,
		4:30,
		5:125,
	},
	slotDanceStorage.D4: {
		3:15,
		4:30,
		5:125,
	},
	slotDanceStorage.D5: {
		3:30,
		4:100,
		5:200,
	},
	slotDanceStorage.D6: {
		3:40,
		4:100,
		5:250,
	},
	slotDanceStorage.D7: {
		3:50,
		4:150,
		5:300,
	},
	slotDanceStorage.D8: {
		3:75,
		4:150,
		5:400,
	},
}

var CoinValue = []int64{
	10,50,200,1000,5000,
}
var CoinNum = []int64{
	50,100,200,300,500,
}
func (this *MyTable) sendPackToAll(topic string,in interface{},action string,err *common.Err) error{
	if !this.BroadCast{ //广播功能
		return nil
	}
	body := this.DealProtocolFormat(in,action,err)
	//error := this.NotifyCallBackMsgNR(topic,body)
	//return error
	this.onlinePush.NotifyCallBackMsgNR(topic, body)
	this.onlinePush.ExecuteCallBackMsg(this.Trace())
	return nil
}
func (this *MyTable) sendPack(session string,topic string,in interface{},action string,err *common.Err) error{
	body := this.DealProtocolFormat(in,action,err)
	//error :=this.SendCallBackMsgNR([]string{session},topic,body)
	//return error
	this.onlinePush.SendCallBackMsgNR([]string{session}, topic, body)
	this.onlinePush.ExecuteCallBackMsg(this.Trace())
	return nil
}
func (this *MyTable)RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return this.Rand.Int63n(max-min) + min
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
		GameType: game.SlotDance,
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
func (this *MyTable) GetTableInfo() interface{} {
	type Info struct {
		CoinValue       []int64
		CoinNum         []int64
		XiaZhuV         int64
		ReelsList       [][]slotDanceStorage.Symbol
		ReelsListTrial   [][]slotDanceStorage.Symbol

		InitSymbol      []int64 //初始页面
		JieSuanData     JieSuanData
		ServerId        string
	}

	var info Info
	info.CoinValue = CoinValue
	info.CoinNum = CoinNum
	info.ReelsList = this.ReelsList
	info.ReelsListTrial = this.ReelsListTrial

	info.InitSymbol = make([]int64, len(this.ReelsList))
	for k,v := range this.ReelsList{
		rand := this.RandInt64(1,int64(len(v) + 1))
		rand = rand -1
		info.InitSymbol[k] = rand
	}

	info.JieSuanData = this.JieSuanData
	info.XiaZhuV = this.CoinNum * this.CoinValue
	info.ServerId = this.module.GetServerID()
	return info
}

func (this *MyTable) PlayerIsTable(uid string) bool {
	if this.UserID == uid{
		return true
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
func (this *MyTable)IsInFreeGame() bool{
	if this.JieSuanData.FreeData.FreeRemainTimes > 0 {
		return true
	}
	return false
}
func (this *MyTable)DealGameResultRecord(pos []int64,reelList [][]slotDanceStorage.Symbol) string{
	res := make([][]slotDanceStorage.Symbol,3)
	for i := 0;i < 3;i++{
		res[i] = make([]slotDanceStorage.Symbol,5)
		for j := 0;j < 5;j++{
			idx := int(pos[j]) + i
			if idx >= len(reelList[j]) {
				idx = idx - len(reelList[j])
			}
			res[i][j] = reelList[j][idx]
		}
	}
	ret, _ := json.Marshal(res)
	return string(ret)
}