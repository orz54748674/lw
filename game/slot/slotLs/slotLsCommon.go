package slotLs

import (
	"encoding/json"
	"vn/common"
	"vn/common/utils"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/agentStorage"
	"vn/storage/slotStorage/slotLsStorage"
	"vn/storage/walletStorage"
)

type Config struct {
	OddsList map[slotLsStorage.Symbol]map[int]int64
}

const TotalRows = 3
const MinWinLine = 3   //最小三连线才中奖
const BaseCoinNum = 50 //50个硬币一次
var InitGoldJackpot = []int64{10000000, 20000000, 30000000, 40000000, 50000000}
var InitSilverJackpot = []int64{1000000, 2000000, 3000000, 4000000, 5000000}

const InitPoolScaleThousand = 10

type JieSuanData struct {
	WildPositions    []map[int64]int64
	ResultPositions  []int64 //转轴最后的位置
	JackpotPositions []map[int64]int64
	WildTimes        int64
	Result           []Result
	TotalBackScore   int64
	FreeGameTimes    int
	FreeRemainTimes  int //剩余次数
	MusicType        MusicType
	TrialData        TrialData
	CoinNum          int64
	CoinValue        int64
}
type TrialData struct {
	GoldJackpot   []int64 `bson:"GoldJackpot" json:"GoldJackpot"`     //金奖池
	SilverJackpot []int64 `bson:"SilverJackpot" json:"SilverJackpot"` //银奖池
	VndBalance    int64   `bson:"VndBalance" json:"VndBalance"`
}
type TrialModeConf struct { //试玩模式
	GoldJackpot   []int64 `bson:"GoldJackpot" json:"GoldJackpot"`     //金奖池
	SilverJackpot []int64 `bson:"SilverJackpot" json:"SilverJackpot"` //银奖池
	VndBalance    int64
}
type Role string

const (
	USER  Role = "user"
	ROBOT Role = "robot"
	Agent Role = "agent"
)

type ModeType string

const (
	NORMAL    ModeType = "normal"
	Free      ModeType = "free"
	TRIAL     ModeType = "trial"
	TRIALFREE ModeType = "trialFree"
)

type Result struct {
	SymbolPositions []map[int64]int64    //出现的位置
	LineType        int                  //连线的类型  3连 4连 5连
	Symbol          slotLsStorage.Symbol //图案
	SymbolScore     int64                //总得分
	CoinValue       int64                //硬币值
	HaveWild        bool                 //是否有wild
	GroupNum        int64                //第一列的组数
}
type MusicType string

const (
	WIN1  MusicType = "win1"
	WIN2  MusicType = "win2"
	WIN3  MusicType = "win3"
	WIN4  MusicType = "win4"
	WIN5  MusicType = "win5"
	BET   MusicType = "bet"
	BET3  MusicType = "bet3"
	BET4  MusicType = "bet4"
	BET5  MusicType = "bet5"
	BET6  MusicType = "bet6"
	BET10 MusicType = "bet10"
	BET40 MusicType = "bet40"
)

var OddsList = map[slotLsStorage.Symbol]map[int]int64{
	slotLsStorage.NINE: {
		3: 5,
		4: 10,
		5: 100,
	},
	slotLsStorage.TEN: {
		3: 5,
		4: 15,
		5: 100,
	},
	slotLsStorage.J: {
		3: 10,
		4: 15,
		5: 100,
	},
	slotLsStorage.Q: {
		3: 10,
		4: 15,
		5: 100,
	},
	slotLsStorage.K: {
		3: 10,
		4: 20,
		5: 200,
	},
	slotLsStorage.A: {
		3: 10,
		4: 30,
		5: 200,
	},
	slotLsStorage.PACKET: {
		3: 15,
		4: 35,
		5: 300,
	},
	slotLsStorage.TORTOISE: {
		3: 20,
		4: 50,
		5: 300,
	},
	slotLsStorage.FISH: {
		3: 30,
		4: 100,
		5: 800,
	},
	slotLsStorage.LION: {
		3: 35,
		4: 100,
		5: 800,
	},
	slotLsStorage.PHOENIX: {
		3: 50,
		4: 100,
		5: 1000,
	},
	slotLsStorage.SCATTER: {
		3: 2,
		4: 5,
		5: 20,
	},
}

type FreeType string //免费游戏的类型
const (
	WHITE  FreeType = "WHITE"
	RED    FreeType = "RED"
	BLACK  FreeType = "BLACK"
	PURPLE FreeType = "PURPLE"
	BLUE   FreeType = "BLUE"
	GOLD   FreeType = "GOLD"
	GREEN  FreeType = "GREEN"
)

type FreeGameConf struct {
	Times      []int //倍数
	NumOfTimes int   //次数
}
type FreeTypeConf struct {
	Times      []int //倍数
	NumOfTimes int   //次数
}

var FreeSelectList = map[FreeType]FreeTypeConf{
	WHITE: {
		Times:      []int{2, 3, 5},
		NumOfTimes: 25,
	},
	RED: {
		Times:      []int{3, 5, 8},
		NumOfTimes: 20,
	},
	BLACK: {
		Times:      []int{5, 8, 10},
		NumOfTimes: 15,
	},
	BLUE: {
		Times:      []int{8, 10, 15},
		NumOfTimes: 13,
	},
	GOLD: {
		Times:      []int{10, 15, 30},
		NumOfTimes: 10,
	},
	GREEN: {
		Times:      []int{15, 30, 40},
		NumOfTimes: 6,
	},
}

type FreeRandConf struct { //选择随机
	Times      [][]int //倍数
	NumOfTimes []int   //次数
}

var FreeGameRandConfig = FreeRandConf{
	Times: [][]int{
		{2, 3, 5},
		{3, 5, 8},
		{5, 8, 10},
		{8, 10, 15},
		{10, 15, 30},
		{15, 30, 40},
	},
	NumOfTimes: []int{
		6, 10, 13, 15, 20, 25,
	},
}
var WildRandList = map[int64]map[int64]int64{
	1: {
		1: 200,
	},
	2: {
		200: 400,
	},
	3: {
		400: 600,
	},
	5: {
		600: 800,
	},
	8: {
		800: 900,
	},
	10: {
		900: 940,
	},
	15: {
		940: 960,
	},
	30: {
		960: 980,
	},
	40: {
		980: 1000,
	},
}
var JackpotRandList = map[int64]map[int64]int64{
	0: {
		1: 200,
	},
	1: {
		200: 700,
	},
	2: {
		700: 950,
	},
	3: {
		950: 1000,
	},
}

var CoinValue = []int64{
	10, 50, 200, 1000, 5000,
}
var CoinNum = []int64{
	50, 100, 200, 300, 500,
}

func (this *MyTable) sendPackToAll(topic string, in interface{}, action string, err *common.Err) error {
	if !this.BroadCast { //广播功能
		return nil
	}
	body := this.DealProtocolFormat(in, action, err)
	//error := this.NotifyCallBackMsgNR(topic,body)
	//return error
	this.onlinePush.NotifyCallBackMsgNR(topic, body)
	this.onlinePush.ExecuteCallBackMsg(this.Trace())
	return nil
}
func (this *MyTable) sendPack(session string, topic string, in interface{}, action string, err *common.Err) error {
	body := this.DealProtocolFormat(in, action, err)
	//error :=this.SendCallBackMsgNR([]string{session},topic,body)
	//return error
	this.onlinePush.SendCallBackMsgNR([]string{session}, topic, body)
	this.onlinePush.ExecuteCallBackMsg(this.Trace())
	return nil
}
func (this *MyTable) RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return this.Rand.Int63n(max-min) + min
}
func (this *MyTable) DealProtocolFormat(in interface{}, action string, error *common.Err) []byte {
	info := struct {
		Data     interface{}
		GameType game.Type
		Action   string
		ErrMsg   string
		Code     int
	}{
		Data:     in,
		GameType: game.SlotLs,
		Action:   action,
	}
	if error == nil {
		info.Code = 0
		info.ErrMsg = "操作成功"
	} else {
		info.Code = error.Code
		info.ErrMsg = error.SetKey().ErrMsg
	}

	ret, _ := json.Marshal(info)
	return ret
}
func (this *MyTable) GetTableInfo() interface{} {
	type Info struct {
		CoinValue     []int64
		CoinNum       []int64
		XiaZhuV       int64
		ReelsList     [][]slotLsStorage.Symbol
		ReelsListFree [][]slotLsStorage.Symbol

		ReelsListTrial     [][]slotLsStorage.Symbol
		ReelsListTrialFree [][]slotLsStorage.Symbol

		InitSymbol           []int64 //初始页面
		InitSymbolFree       []int64 //初始页面
		JieSuanData          JieSuanData
		JieSuanDataFree      JieSuanData
		JieSuanDataTrial     JieSuanData
		JieSuanDataTrialFree JieSuanData
		ServerId             string
		FreeType             FreeType
	}

	var info Info
	info.CoinValue = CoinValue
	info.CoinNum = CoinNum
	info.ReelsList = this.ReelsList
	info.ReelsListFree = this.ReelsListFree

	info.ReelsListTrial = this.ReelsListTrial
	info.ReelsListTrialFree = this.ReelsListTrialFree

	info.InitSymbol = make([]int64, len(this.ReelsList))
	for k, v := range this.ReelsList {
		rand := this.RandInt64(1, int64(len(v)+1))
		rand = rand - 1
		info.InitSymbol[k] = rand
	}
	info.InitSymbolFree = make([]int64, len(this.ReelsList))
	for k, v := range this.ReelsListFree {
		rand := this.RandInt64(1, int64(len(v)+1))
		rand = rand - 1
		info.InitSymbolFree[k] = rand
	}
	info.JieSuanData = this.JieSuanData
	info.JieSuanDataFree = this.JieSuanDataFree
	info.JieSuanDataTrial = this.JieSuanDataTrial
	info.JieSuanDataTrialFree = this.JieSuanDataTrialFree
	info.XiaZhuV = this.CoinNum * this.CoinValue
	info.ServerId = this.module.GetServerID()
	info.FreeType = this.FreeType
	return info
}

func (this *MyTable) PlayerIsTable(uid string) bool {
	if this.UserID == uid {
		return true
	}
	return false
}
func (this *MyTable) notifyWallet(uid string) {
	sb := vGate.QuerySessionBean(uid)
	if sb == nil {
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
func (this *MyTable) IsInFreeGame() bool {
	if this.JieSuanDataFree.FreeGameTimes > 0 || this.JieSuanDataFree.FreeRemainTimes > 0 {
		return true
	}
	return false
}
func (this *MyTable) DealGameResultRecord(pos []int64, reelList [][]slotLsStorage.Symbol, jackPos []map[int64]int64, wildTimes int64) string {
	res := make(map[string]interface{})
	resSymbol := make([][]slotLsStorage.Symbol, 3)
	for i := 0; i < 3; i++ {
		resSymbol[i] = make([]slotLsStorage.Symbol, 5)
		for j := 0; j < 5; j++ {
			idx := int(pos[j]) + i
			if idx >= len(reelList[j]) {
				idx = idx - len(reelList[j])
			}
			resSymbol[i][j] = reelList[j][idx]
		}
	}
	res["symbolPos"] = resSymbol
	res["jackPos"] = jackPos
	res["wildTimes"] = wildTimes
	ret, _ := json.Marshal(res)
	return string(ret)
}
