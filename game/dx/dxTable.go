package dx

import (
	"encoding/json"
	"errors"
	"reflect"
	"runtime"
	"sort"
	"sync"
	"time"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	gate2 "vn/gate"
)

type DxTable struct {
	room.QTable
	app        module.App
	module     module.RPCModule
	players    map[string]room.BasePlayer
	onlinePush *gate2.OnlinePush
	dxRun      *dxRun
	tableOnce  sync.Once
}

func (this *DxTable) GetSeats() map[string]room.BasePlayer {
	return this.players
}
func (this *DxTable) GetModule() module.RPCModule {
	return this.module
}
func (this *DxTable) GetApp() module.App {
	return this.app
}
func (this *DxTable) Start() {

	this.tableOnce.Do(func() {
		go func() {
			this.dxRun.Run()
		}()
	})
}
func (this *DxTable) OnCreate() {
	//可以加载数据
	log.Info("DxTable OnCreate")
	//一定要调用QTable.OnCreate()
	this.QTable.OnCreate()
}

/**
每帧都会调用
*/
func (this *DxTable) Update(ds time.Duration) {
	//log.Info("Update %v", ds)
	defer func() {
		if r := recover(); r != nil {
			buff := make([]byte, 1024)
			runtime.Stack(buff, false)
			log.Error("Update panic(%v)\n info:%s", r, string(buff))
			this.Finish()
		}
	}()
	this.onlinePush.ExecuteCallBackMsg(this.Trace())
}

func NewTable(module module.RPCModule, app module.App, opts ...room.Option) *DxTable {
	this := &DxTable{
		module:  module,
		app:     app,
		players: map[string]room.BasePlayer{},
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
	this.onlinePush = &gate2.OnlinePush{
		App:       app,
		TraceSpan: this.Trace(),
	}
	this.dxRun = &dxRun{
		app:      app,
		settings: this.GetModule().GetModuleSettings(),
		table:    this,
	}
	this.onlinePush.OnlinePushInit(this, 2048)
	this.Register("empty", this.empty)
	this.Register(actionBet, this.Bet)
	//this.Register("info", this.info)

	return this
}

func (this *DxTable) empty(session gate.Session, msg map[string]interface{}) (err error) {
	//player := this.FindPlayer(session)
	//if player == nil {
	//	return errors.New("no join")
	//}
	//player.OnRequest(session)
	//_ = this.NotifyCallBackMsg("/room/say", []byte(fmt.Sprintf("say hi from %v", msg["name"])))
	return nil
}

var (
	actionBet = "Bet"
)

func (s *DxTable) Bet(session gate.Session, params map[string]interface{}) error {
	if check, ok := utils.CheckParams2(params,
		[]string{"big", "small"}); ok != nil {
		s.sendResponse(session, errCode.ErrParams.SetKey(check).GetMap())
		return nil
	}
	uid := session.GetUserID()
	big, _ := utils.ConvertInt(params["big"])
	small, _ := utils.ConvertInt(params["small"])
	res := s.dxRun.Bet(uid, big, small)
	res["Action"] = actionBet
	s.sendResponse(session, res)
	return nil
}

func (s *DxTable) sendResponse(session gate.Session, res map[string]interface{}) {
	res["GameType"] = game.BiDaXiao
	b, _ := json.Marshal(res)
	session.SendNR(game.Push, b)
	//_ = s.onlinePush.SendCallBackMsgNR([]string{session.GetSessionID()}, game.Push,b)
}
func (s *DxTable) close(session gate.Session, msg map[string]interface{}) error {

	return nil
}
func (s *DxTable) chat(session gate.Session, msg map[string]interface{}) error {

	return nil
}
func (s *DxTable) GetCurRoundBet() []CurRoundBet {
	sort.Slice(s.dxRun.curRoundBet, func(i, j int) bool {
		return s.dxRun.curRoundBet[i].Bets > s.dxRun.curRoundBet[j].Bets
	})
	var res []CurRoundBet
	if len(s.dxRun.curRoundBet) > curRoundBetMax {
		res = append(s.dxRun.curRoundBet[:0], s.dxRun.curRoundBet[:curRoundBetMax]...)
	} else {
		res = s.dxRun.curRoundBet
	}
	return res
}

//func (s *DxTable)toResult(session gate.Session,action string,errCode *common.Err){
//	res := errCode.GetMap()
//	res["Action"] = action
//	json,_ := json.Marshal(res)
//	session.Send(dxResponseTopic,json)
//}
