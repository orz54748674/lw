package slotLs

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
	vGate "vn/gate"
	"vn/storage/slotStorage/slotLsStorage"
)

type MyTable struct {
	room.QTable
	module  module.RPCModule
	app module.App


	onlinePush *vGate.OnlinePush
	tableID string
	Players map[string] room.BasePlayer
	BroadCast bool  //广播标志

	GameConf *slotLsStorage.Conf

	Rand *rand.Rand

	CoinValue int64
	CoinNum int64
	XiaZhuV int64
	ReelsList [][]slotLsStorage.Symbol
	ReelsListFree [][]slotLsStorage.Symbol
	ReelsListTrial [][]slotLsStorage.Symbol
	ReelsListTrialFree [][]slotLsStorage.Symbol

	UserID string
	Role Role

	JieSuanData     JieSuanData
	JieSuanDataFree JieSuanData
	JieSuanDataTrial JieSuanData
	JieSuanDataTrialFree JieSuanData
	FreeGameConf    FreeGameConf
	EventID         string
	ResultsPool     int64  //奖池中奖金额
	Name            string
	IsInCheckout    bool
	FreeType        FreeType

	TrialModeConf TrialModeConf
	ModeType	ModeType
	TrialData TrialData
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
	log.Info("slotLs Table OnCreate")
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
	this.Register(protocol.Enter, this.Enter)             //进入房间
	this.Register(protocol.QuitTable, this.QuitTable)             //退出房间
	this.Register(protocol.Spin, this.Spin)             //
	this.Register(protocol.SpinFree, this.SpinFree)             //
	this.Register(protocol.SpinTrial, this.SpinTrial)             //
	this.Register(protocol.SpinTrialFree, this.SpinTrialFree)             //
	this.Register(protocol.ClearTable, this.ClearTable)
	return this
}



