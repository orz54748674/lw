package gate

import (
	"context"
	"fmt"
	"github.com/yireyun/go-queue"
	"runtime"
	"strings"
	"time"
	"vn/framework/mqant-modules/room"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
)

type OnlinePush struct {
	App           module.App
	TraceSpan     log.TraceSpan
	queue_message *queue.EsQueue
	tableimp      room.TableImp
}
type CallBackMsg struct {
	notify    bool     //是否是广播
	needReply bool     //是否需要回复
	allUsers  bool     //全网用户
	players   []string //如果不是广播就指定session
	topic     *string
	body      *[]byte
}

func (this *OnlinePush) OnlinePushInit(table room.TableImp, Capaciity uint32) {
	this.queue_message = queue.NewQueue(Capaciity)
	this.tableimp = table
}

func (this *OnlinePush) SendCallBackMsg(players []string, topic string, body []byte) error {
	ok, quantity := this.queue_message.Put(&CallBackMsg{
		notify:    false,
		needReply: true,
		allUsers:  false,
		players:   players,
		topic:     &topic,
		body:      &body,
	})
	if !ok {
		return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
	} else {
		return nil
	}
}

func (this *OnlinePush) NotifyCallBackMsg(topic string, body []byte) error {
	ok, quantity := this.queue_message.Put(&CallBackMsg{
		notify:    true,
		needReply: true,
		allUsers:  false,
		players:   nil,
		topic:     &topic,
		body:      &body,
	})
	if !ok {
		return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
	} else {
		return nil
	}
}

func (this *OnlinePush) SendCallBackMsgNR(players []string, topic string, body []byte) error {
	ok, quantity := this.queue_message.Put(&CallBackMsg{
		notify:    false,
		needReply: false,
		allUsers:  false,
		players:   players,
		topic:     &topic,
		body:      &body,
	})
	if !ok {
		return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
	} else {
		return nil
	}
}

func (this *OnlinePush) NotifyCallBackMsgNR(topic string, body []byte) error {
	ok, quantity := this.queue_message.Put(&CallBackMsg{
		notify:    true,
		needReply: false,
		allUsers:  false,
		players:   nil,
		topic:     &topic,
		body:      &body,
	})
	if !ok {
		return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
	} else {
		return nil
	}
}
func (this *OnlinePush) NotifyAllPlayersNR(topic string, body []byte) error {
	ok, quantity := this.queue_message.Put(&CallBackMsg{
		notify:    true,
		needReply: false,
		allUsers:  true,
		players:   nil,
		topic:     &topic,
		body:      &body,
	})
	if !ok {
		return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
	} else {
		return nil
	}
}

/**
合并玩家所在网关
*/
func (this *OnlinePush) mergeGate() map[string][]string {
	merge := map[string][]string{}
	for _, role := range this.tableimp.GetSeats() {
		if role != nil && role.Session() != nil {
			//未断网
			if _, ok := merge[role.Session().GetServerID()]; ok {
				merge[role.Session().GetServerID()] = append(merge[role.Session().GetServerID()], role.Session().GetSessionID())
			} else {
				merge[role.Session().GetServerID()] = []string{role.Session().GetSessionID()}
			}
		}
	}
	return merge
}
func (this *OnlinePush) mergeAllGate() map[string][]string {
	merge := map[string][]string{}
	allSession := QueryAllSession()
	if allSession == nil {
		return merge
	}
	for _, sessionBean := range *allSession {
		if _, ok := merge[sessionBean.ServerId]; ok {
			merge[sessionBean.ServerId] = append(merge[sessionBean.ServerId], sessionBean.SessionId)
		} else {
			merge[sessionBean.ServerId] = []string{sessionBean.SessionId}
		}
	}
	return merge
}

/**
【每帧调用】统一发送所有消息给各个客户端
*/
func (this *OnlinePush) ExecuteCallBackMsg(span log.TraceSpan) {
	var merge map[string][]string
	ok := true
	queue := this.queue_message
	var index = 0
	for ok {
		val, _ok, _ := queue.Get()
		index++
		if _ok {
			msg := val.(*CallBackMsg)
			if msg.notify {
				if merge == nil {
					if msg.allUsers {
						merge = this.mergeAllGate()
					} else {
						merge = this.mergeGate()
					}
				}
				for serverid, plist := range merge {
					sessionids := strings.Join(plist, ",")
					server, e := this.App.GetServerByID(serverid)
					if e != nil {
						log.Warning("SendBatch error %v", e)
						return
					}
					if msg.needReply {
						ctx, _ := context.WithTimeout(context.TODO(), time.Second*3)
						result, err := server.Call(ctx, "SendBatch", span, sessionids, *msg.topic, *msg.body)
						if err != "" {
							log.Warning("SendBatch error %v %v", serverid, err)
						} else {
							if int(result.(int64)) < len(plist) {
								//有连接断了
							}
						}
					} else {
						err := server.CallNR("SendBatch", span, sessionids, *msg.topic, *msg.body)
						if err != nil {
							log.Warning("SendBatch error %v %v", serverid, err.Error())
						}
					}

				}
			} else {
				for _, sessionId := range msg.players {
					sessionBean := QuerySessionId(sessionId)
					if sessionBean != nil {
						session, err := basegate.NewSession(this.App, sessionBean.Session)
						if err != nil {
							log.Error(err.Error())
						} else {
							if err := session.SendNR(*msg.topic, *msg.body); err != "" {
								log.Error(err)
							}
						}
					}
					//for _, role := range this.tableimp.GetSeats() {
					//	if role != nil {
					//		if (role.Session() != nil) && (role.Session().GetServerID() == sessionId) {
					//			if msg.needReply {
					//				e := role.Session().Send(*msg.topic, *msg.body)
					//				if e == "" {
					//					role.OnResponse(role.Session())
					//				}
					//			} else {
					//				_ = role.Session().SendNR(*msg.topic, *msg.body)
					//			}
					//		}
					//	}
					//}
				}
			}
		}
		ok = _ok
	}
}

func (this *OnlinePush) Run(w time.Duration) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buff := make([]byte, 1024)
				runtime.Stack(buff, false)
				log.Error("online push panic(%v)\n info:%s", r, string(buff))
			}
		}()
		for {
			time.Sleep(w)
			this.ExecuteCallBackMsg(this.TraceSpan)
		}
	}()
}
