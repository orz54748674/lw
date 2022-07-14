package slotSex

import (
	"encoding/json"
	"vn/common"
	"vn/common/utils"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/agentStorage"
	"vn/storage/slotStorage/slotSexStorage"
	"vn/storage/walletStorage"
)

type Config struct {
	OddsList map[slotSexStorage.Symbol]map[int]int64
}

const TotalRows = 3
const MinWinLine = 3   //最小三连线才中奖
const BaseCoinNum = 50 //50个硬币一次
var InitJackpot = []int64{10000000, 20000000, 30000000}

const InitPoolScaleThousand = 10

type JieSuanData struct {
	ResultPositions []int64 //转轴最后的位置
	Result          []Result
	TotalBackScore  int64
	GetJackpot      bool
	FreeRemainTimes int //剩余次数
	FreeGame        bool
	BonusGame       bool
	BonusTimes      []int64
	MusicType       MusicType
	TrialData       TrialData
	CoinNum         int64
	CoinValue       int64
	AnimationData   AnimationData
}

type AnimationType string

const (
	BigAnimation1 AnimationType = "BigAnimation1"
	ColAnimation1 AnimationType = "ColAnimation1"
	ColAnimation2 AnimationType = "ColAnimation2"
	ColAnimation3 AnimationType = "ColAnimation3"
	ColAnimation4 AnimationType = "ColAnimation4"
)

type Role string

const (
	USER  Role = "user"
	ROBOT Role = "robot"
	Agent Role = "agent"
)

type AnimationData struct {
	AnimationType AnimationType
	PlayCol       int //在哪一列播放
}

var BigAnimationList = []AnimationType{ //
	BigAnimation1,
}
var ColAnimationList = []AnimationType{ //
	ColAnimation1, ColAnimation2, ColAnimation3, ColAnimation4,
}

type TrialData struct {
	VndBalance int64 `bson:"VndBalance" json:"VndBalance"`
}
type BonusGameData struct {
	ClickNum         int64
	State            int //1  点击图标  2 点击 倍数
	CurSymbolScore   int64
	TotalSymbolScore int64
	Serial           int
	IsTimeOut        bool
}
type MiniGameData struct {
	ClickNum         int64
	TotalSymbolScore int64
	Serial           int
	SymbolList       []int
	CurSymbol        int
	State            int
}

type TrialModeConf struct { //试玩模式
	VndBalance int64
}
type ModeType string

const (
	NORMAL ModeType = "normal"
	TRIAL  ModeType = "trial"
)

type Result struct {
	LineType     int                   //连线的类型  3连 4连 5连
	WinPositions []map[int64]int64     //出现的位置
	Symbol       slotSexStorage.Symbol //图案
	SymbolScore  int64                 //总得分
	CoinValue    int64                 //硬币值
	LineSerial   int64                 //第几条线
}
type MusicType string

const (
	WinNormal  MusicType = "NORMAL"
	WinJackPot MusicType = "JACKPOT"
	Win500     MusicType = "500"
)

type BonusSymbol int

const (
	HULU   BonusSymbol = 1  //葫芦
	JADE   BonusSymbol = 2  //玉
	FAN    BonusSymbol = 3  //扇子
	COIN   BonusSymbol = 4  //钱币
	FISH   BonusSymbol = 5  //鱼
	FROG   BonusSymbol = 6  //青蛙
	TREE   BonusSymbol = 7  //树
	TEAPOT BonusSymbol = 8  //茶壶
	GUN    BonusSymbol = 9  //炮
	BOWL   BonusSymbol = 10 //碗
	PACKET BonusSymbol = 11 //钱袋
	CAT    BonusSymbol = 12 //猫
)

var BonusSymbolList = []BonusSymbol{
	HULU,
	JADE,
	FAN,
	COIN,
	FISH,
	FROG,
	TREE,
	TEAPOT,
	GUN,
	BOWL,
	PACKET,
	CAT,
}
var OddsList = map[slotSexStorage.Symbol]map[int]int64{
	slotSexStorage.S1: {
		3: 2,
		4: 8,
		5: 80,
	},
	slotSexStorage.S2: {
		3: 4,
		4: 16,
		5: 160,
	},
	slotSexStorage.S3: {
		3: 6,
		4: 24,
		5: 240,
	},
	slotSexStorage.S4: {
		3: 8,
		4: 32,
		5: 320,
	},
	slotSexStorage.S5: {
		3: 10,
		4: 40,
		5: 400,
	},
	slotSexStorage.S6: {
		3: 12,
		4: 48,
		5: 480,
	},
	slotSexStorage.S7: {
		3: 16,
		4: 64,
		5: 640,
	},
	slotSexStorage.S8: {
		3: 20,
		4: 80,
		5: 800,
	},
	slotSexStorage.JACKPOT: {
		2: 10,
		3: 50,
		4: 500,
		5: 5000,
	},
}
var ScatterTimes = map[int]int{ //免费次数
	3: 10,
	4: 15,
	5: 20,
}
var BonusScoreList = []int64{ //分数
	75, 150, 225, 300,
}
var BonusTimes = map[int][]int64{ //倍数
	3: {1, 2, 3},
	4: {2, 3, 4},
	5: {3, 4, 5},
}

type LineCoordinate struct {
	Row int64
	Col int64
}

var WildReplaceList = []slotSexStorage.Symbol{
	slotSexStorage.JACKPOT,
	slotSexStorage.S1,
	slotSexStorage.S2,
	slotSexStorage.S3,
	slotSexStorage.S4,
	slotSexStorage.S5,
	slotSexStorage.S6,
	slotSexStorage.S7,
	slotSexStorage.S8,
}
var LineCoordinates = map[int64][]LineCoordinate{
	1:  {{1, 0}, {1, 1}, {1, 2}, {1, 3}, {1, 4}},
	2:  {{0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}},
	3:  {{2, 0}, {2, 1}, {2, 2}, {2, 3}, {2, 4}},
	4:  {{0, 0}, {1, 1}, {2, 2}, {1, 3}, {0, 4}},
	5:  {{2, 0}, {1, 1}, {0, 2}, {1, 3}, {2, 4}},
	6:  {{0, 0}, {1, 1}, {1, 2}, {1, 3}, {0, 4}},
	7:  {{2, 0}, {1, 1}, {1, 2}, {1, 3}, {2, 4}},
	8:  {{1, 0}, {0, 1}, {0, 2}, {0, 3}, {1, 4}},
	9:  {{1, 0}, {2, 1}, {2, 2}, {2, 3}, {1, 4}},
	10: {{0, 0}, {0, 1}, {1, 2}, {2, 3}, {2, 4}},
	11: {{2, 0}, {2, 1}, {1, 2}, {0, 3}, {0, 4}},
	12: {{1, 0}, {0, 1}, {1, 2}, {2, 3}, {1, 4}},
	13: {{1, 0}, {2, 1}, {1, 2}, {0, 3}, {1, 4}},
	14: {{0, 0}, {1, 1}, {0, 2}, {1, 3}, {0, 4}},
	15: {{2, 0}, {1, 1}, {2, 2}, {1, 3}, {2, 4}},
	16: {{1, 0}, {1, 1}, {0, 2}, {1, 3}, {1, 4}},
	17: {{1, 0}, {1, 1}, {2, 2}, {1, 3}, {1, 4}},
	18: {{0, 0}, {0, 1}, {2, 2}, {0, 3}, {0, 4}},
	19: {{2, 0}, {2, 1}, {0, 2}, {2, 3}, {2, 4}},
	20: {{1, 0}, {0, 1}, {2, 2}, {0, 3}, {1, 4}},
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
	1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
}
var CoinNum = []int64{
	100, 1000, 10000,
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
		GameType: game.SlotSex,
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
		CoinValue      []int64
		CoinNum        []int64
		XiaZhuV        int64
		ReelsList      [][]slotSexStorage.Symbol
		ReelsListTrial [][]slotSexStorage.Symbol

		InitSymbol     []int64 //初始页面
		InitSymbolFree []int64 //初始页面
		//JieSuanData     JieSuanData
		FreeRemainTimes int
		ServerId        string
		BonusTimeOut    int

		LineCoordinates map[int64][]LineCoordinate
	}

	var info Info
	info.CoinValue = CoinValue
	info.CoinNum = CoinNum
	info.ReelsList = this.ReelsList
	info.ReelsListTrial = this.ReelsListTrial

	info.InitSymbol = make([]int64, len(this.ReelsList))
	for k, v := range this.ReelsList {
		rand := this.RandInt64(1, int64(len(v)+1))
		rand = rand - 1
		info.InitSymbol[k] = rand
	}
	info.InitSymbolFree = make([]int64, len(this.ReelsList))

	info.FreeRemainTimes = this.JieSuanData.FreeRemainTimes
	//info.JieSuanData = this.JieSuanData
	info.XiaZhuV = this.CoinNum * this.CoinValue
	info.ServerId = this.module.GetServerID()
	info.LineCoordinates = LineCoordinates
	info.BonusTimeOut = slotSexStorage.GetRoomConf().BonusTimeOut
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
	if this.JieSuanData.FreeRemainTimes > 0 {
		return true
	}
	return false
}

func (this *MyTable) DealGameResultRecord(pos []int64, reelList [][]slotSexStorage.Symbol) string {
	res := make([][]slotSexStorage.Symbol, 3)
	for i := 0; i < 3; i++ {
		res[i] = make([]slotSexStorage.Symbol, 5)
		for j := 0; j < 5; j++ {
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
