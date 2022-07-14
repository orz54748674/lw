package cardPhom

import (
	"errors"
	"math/rand"
	"reflect"
	"runtime"
	"sync"
	"time"
	"vn/common/protocol"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	vGate "vn/gate"
	"vn/storage/cardStorage/cardPhomStorage"
)

type MyTable struct {
	room.QTable
	module module.RPCModule
	app    module.App

	onlinePush  *vGate.OnlinePush
	tableID     string
	tableIDTail string //
	Players     map[string]room.BasePlayer
	BroadCast   bool //广播标志

	SeqExecFlag bool //顺序执行标记
	OnlyExecOne bool //执行一次标志
	GameConf    *cardPhomStorage.Conf
	CountDown   int //倒计时

	Rand *rand.Rand

	EventID        string
	BaseScore      int64
	TotalPlayerNum int          //总人数
	PlayerList     []PlayerList `bson:"PlayerList" json:"PlayerList"`

	Bottom []int //底牌

	Master    string //房主
	RoomState Room_v

	RobotNum   int  //机器人数量
	AutoCreate bool //自動創建

	WaitingList sync.Map

	FirstPhom       int
	FirstPut        int
	PlayingNum      int
	MinEnterTable   int64
	WinIdx          int
	RankList        []RankList
	LastRoundEatIdx int

	StraightScoreData StraightScoreData
	JieSuanData       JiesuanData

	CalcPhomData CalcPhomData
	NeedControl  bool
	Pool         []int
	RefreshTime  int
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
	log.Info("cardPhom Table OnCreate")
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

func NewTable(module module.RPCModule, app module.App, tableID string, opts ...room.Option) *MyTable {
	this := &MyTable{
		module:  module,
		app:     app,
		tableID: tableID,
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
	this.TableInit(module, app, tableID)
	this.Register(protocol.Enter, this.Enter)               //进入房间
	this.Register(protocol.InviteEnter, this.InviteEnter)   //进入房间
	this.Register(protocol.GetEnterData, this.GetEnterData) //
	this.Register(protocol.Ready, this.Ready)               //
	this.Register(protocol.RobotReady, this.RobotReady)
	this.Register(protocol.AutoReady, this.AutoReady)
	this.Register(protocol.MasterStartGame, this.MasterStartGame)
	this.Register(protocol.QuitTable, this.QuitTable)           //退出房间
	this.Register(protocol.RobotEnter, this.RobotEnter)         //
	this.Register(protocol.RobotQuitTable, this.RobotQuitTable) //
	this.Register(protocol.StartGame, this.StartGame)
	this.Register(protocol.ReadyGame, this.ReadyGame)
	this.Register(protocol.JieSuan, this.JieSuan)
	this.Register(protocol.ClearTable, this.ClearTable) //
	this.Register(protocol.PutPoker, this.PutPoker)     //
	this.Register(protocol.EatPoker, this.EatPoker)     //
	//this.Register(protocol.GetPhomPoker, this.GetPhomPoker)             //
	this.Register(protocol.PhomPoker, this.PhomPoker) //
	this.Register(protocol.GivePoker, this.GivePoker) //
	this.Register(protocol.DrawPoker, this.DrawPoker) //
	this.Register(protocol.SortPoker, this.SortPoker)
	return this
}
