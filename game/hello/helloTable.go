package hello

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"time"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
)

type TestTable struct {
	room.QTable
	app        module.App
	module     module.RPCModule
	players    map[string]room.BasePlayer
}

func (this *TestTable) GetSeats() map[string]room.BasePlayer {
	return this.players
}
func (this *TestTable) GetModule() module.RPCModule {
	return this.module
}
func (this *TestTable) GetApp() module.App {
	return this.app
}

func (this *TestTable) OnCreate() {
	//可以加载数据
	log.Info("DxTable OnCreate")
	//一定要调用QTable.OnCreate()
	this.QTable.OnCreate()
}

/**
每帧都会调用
*/
func (this *TestTable) Update(ds time.Duration) {
	//log.Info("Update %v", ds)
	defer func() {
		if r := recover(); r != nil {
			buff := make([]byte, 1024)
			runtime.Stack(buff, false)
			log.Error("Update panic(%v)\n info:%s", r, string(buff))
			this.Finish()
		}
	}()

}

func NewTable(module module.RPCModule, app module.App, opts ...room.Option) *TestTable {
	this := &TestTable{
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
	this.Register("empty", this.empty)
	this.Register(actionBet, this.Bet)
	this.Register(actionTest, this.Test)
	//this.Register("info", this.info)

	return this
}

func (this *TestTable) empty(session gate.Session, msg map[string]interface{}) (err error) {
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
	actionTest = "Test"
)

func (s *TestTable) Test(session gate.Session, params map[string]interface{}) error {
	log.Info("test: %v" , params)
	return nil
}
func (s *TestTable) Bet(session gate.Session, params map[string]interface{}) error {
	log.Info("test Bet")
	uid := session.GetUserID()
	player := &room.BasePlayerImp{}
	player.Bind(session)
	player.OnRequest(session)
	s.players[uid] = player
	var r []map[string]interface{}
	for i :=0 ;i<100 ;i++{
		res := map[string]interface{}{
			fmt.Sprintf("%d",i):utils.RandomString(int(utils.RandInt64(1,999999))),
		}
		r = append(r,res)
	}
	//for _,res := range r{
		//body,_ := json.Marshal(res)
		//_ = s.SendCallBackMsgByQueue([]string{session.GetSessionID()},game.Push,body)
		//_ = s.SendCallBackMsgNR([]string{session.GetSessionID()},game.Push,body)
	//}
	return nil
}

func (s *TestTable) sendResponse(session gate.Session,res map[string]interface{}){
	res["GameType"] = game.BiDaXiao
	b,_ := json.Marshal(res)
	session.SendNR(game.Push,b)
	//_ = s.onlinePush.SendCallBackMsgNR([]string{session.GetSessionID()}, game.Push,b)
}
func (s *TestTable) close(session gate.Session, msg map[string]interface{}) error {

	return nil
}
func (s *TestTable) chat(session gate.Session, msg map[string]interface{}) error {

	return nil
}

//func (s *DxTable)toResult(session gate.Session,action string,errCode *common.Err){
//	res := errCode.GetMap()
//	res["Action"] = action
//	json,_ := json.Marshal(res)
//	session.Send(dxResponseTopic,json)
//}
