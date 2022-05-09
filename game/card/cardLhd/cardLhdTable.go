package cardLhd

import (
	"errors"
	"math/rand"
	"reflect"
	"runtime"
	"time"
	"vn/common/protocol"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/cardStorage/cardLhdStorage"
)

type MyTable struct {
	room.QTable
	module  module.RPCModule
	app module.App


	onlinePush *vGate.OnlinePush
	tableID string
	Players map[string] room.BasePlayer
	BroadCast bool  //广播标志
	SeatNum int  //座位数量

	GameConf *cardLhdStorage.Conf
	RobotXiaZhuList map[string]RobotXiaZhuList
	SeqExecFlag bool //顺序执行标记
	OnlyExecOne bool //执行一次标志
	CountDown int  //倒计时
	RoomState Room_v //房间状态
	PlayerList []PlayerList `bson:"PlayerList" json:"PlayerList"`
	ResultsChipList map[cardLhdStorage.XiaZhuResult] []int64  `bson:"XiaZhuResult" json:"XiaZhuResult"`//桌上下注筹码结果
	Results map[string]int  `bson:"Results" json:"Results"`//开奖结果图案

	XiaZhuTotal map[cardLhdStorage.XiaZhuResult] int64 `bson:"XiaZhuTotal" json:"XiaZhuTotal"`//桌上下注结果总数
	RealXiaZhuTotal map[cardLhdStorage.XiaZhuResult]int64 `bson:"RealXiaZhuTotal" json:"RealXiaZhuTotal"` //每轮真实玩家下注总和

	ShortCut map[string][]game.ShortCutMode `bson:"ShortCut" json:"ShortCut"` //快捷语
	PositionNum int
	PositionList []PlayerList `bson:"PositionList" json:"PositionList"`
	Rand *rand.Rand
	PlayerNum int    //人数
	EventID string
	RobotYxbConf []cardLhdStorage.Robot
	JieSuanData JiesuanData
}
func (this *MyTable) GetSeats() map[string]room.BasePlayer {
	return this.Players
}
func (this *MyTable) GetModule() module.RPCModule {
	return this.module
}
func (this *MyTable) GetApp() module.App {
	return this.app
}
func (this *MyTable) OnCreate() {
	//可以加载数据
	log.Info("Lhd Table OnCreate")
	//一定要调用QTable.OnCreate()


	this.QTable.OnCreate()
}

/**
每帧都会调用
*/
func (this *MyTable) Update(ds time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			buff := make([]byte, 1024)
			runtime.Stack(buff, false)
			log.Error("Update panic(%v)\n info:%s", r, string(buff))
			this.Finish()
		}
	}()
}

func NewTable(module module.RPCModule,app module.App,tableID string,opts ...room.Option) *MyTable {
	this := &MyTable{
		module:  module,
		app: app,
		tableID:tableID,
	}
	opts = append(opts, room.TimeOut(0))
	opts = append(opts, room.Update(this.Update))
	opts = append(opts, room.NoFound(func(msg *room.QueueMsg) (value reflect.Value, e error) {
		//return reflect.ValueOf(this.doSay), nil
		return reflect.Zero(reflect.ValueOf("").Type()), errors.New("no found handler")
	}))
	opts = append(opts, room.SetRecoverHandle(func(msg *room.QueueMsg, err error) {
		log.Error("Recover %v Error: %v", msg.Func, err.Error())
	}))
	opts = append(opts, room.SetErrorHandle(func(msg *room.QueueMsg, err error) {
		log.Error("Error %v Error: %v", msg.Func, err.Error())
	}))
	this.OnInit(this, opts...)
	//this.OnCreate()

	this.TableInit(module,app,tableID)
	this.Register(protocol.Empty, this.Empty)             //下注
	this.Register(protocol.XiaZhu, this.XiaZhu)             //下注
	this.Register(protocol.LastXiaZhu, this.LastXiaZhu)     //上轮下注
	this.Register(protocol.DoubleXiaZhu, this.DoubleXiaZhu) //加倍下注

	this.Register(protocol.Enter, this.Enter)             //进入房间
	this.Register(protocol.QuitTable, this.QuitTable)             //进入房间

	//队列函数
	this.Register(protocol.StartGame, this.StartGame)             //下注
	this.Register(protocol.RobotXiaZhu, this.RobotXiaZhu)             //下注
	this.Register(protocol.ReadyGame, this.ReadyGame)             //
	this.Register(protocol.JieSuan, this.JieSuan)             //
	this.Register(protocol.UpdatePlayerList, this.UpdatePlayerList)             //
	this.Register(protocol.ClearTable, this.ClearTable)             //
	this.Register(protocol.GetShortCutList, this.GetShortCutList)             //
	this.Register(protocol.SendShortCut, this.SendShortCut)             //
	this.Register(protocol.RobotEnter, this.RobotEnter)             //
	this.Register(protocol.RobotQuitTable, this.RobotQuitTable)             //
	this.Register(protocol.RobotBetCalc, this.RobotBetCalc)             //
	return this
}



