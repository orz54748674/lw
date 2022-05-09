// Copyright 2014 loolgame Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package room

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/yireyun/go-queue"
	"reflect"
	"runtime"
	"sync"
)

type QueueMsg struct {
	Func   string
	Params []interface{}
}
type QueueReceive interface {
	Receive(msg *QueueMsg, index int)
}
type QueueTable struct {
	opts            Options
	functions       map[string]reflect.Value
	receive         QueueReceive
	queue0          *queue.EsQueue
	queue1          *queue.EsQueue
	current_w_queue int //当前写的队列
	lock            *sync.RWMutex
}

func (self *QueueTable) QueueInit(opts ...Option) {
	self.opts = newOptions(opts...)
	self.functions = map[string]reflect.Value{}
	self.queue0 = queue.NewQueue(self.opts.Capaciity)
	self.queue1 = queue.NewQueue(self.opts.Capaciity)
	self.current_w_queue = 0
	self.lock = new(sync.RWMutex)
}
func (self *QueueTable) SetReceive(receive QueueReceive) {
	self.receive = receive
}
func (self *QueueTable) Register(id string, f interface{}) {

	if _, ok := self.functions[id]; ok {
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	self.functions[id] = reflect.ValueOf(f)
}

/**
协成安全,任意协成可调用
*/
func (self *QueueTable) PutQueue(_func string, params ...interface{}) error {
	q := self.wqueue()
	self.lock.Lock()
	ok, quantity := q.Put(&QueueMsg{
		Func:   _func,
		Params: params,
	})
	self.lock.Unlock()
	if !ok {
		return fmt.Errorf("Put Fail, quantity:%v\n", quantity)
	} else {
		return nil
	}

}

/**
切换并且返回读的队列
*/
func (self *QueueTable) switchqueue() *queue.EsQueue {
	self.lock.Lock()
	if self.current_w_queue == 0 {
		self.current_w_queue = 1
		self.lock.Unlock()
		return self.queue0
	} else {
		self.current_w_queue = 0
		self.lock.Unlock()
		return self.queue1
	}

}
func (self *QueueTable) wqueue() *queue.EsQueue {
	self.lock.Lock()
	if self.current_w_queue == 0 {
		self.lock.Unlock()
		return self.queue0
	} else {
		self.lock.Unlock()
		return self.queue1
	}

}

/**
【每帧调用】执行队列中的所有事件
*/
func (self *QueueTable) ExecuteEvent(arge interface{}) {
	ok := true
	queue := self.switchqueue()
	index := 0
	for ok {
		val, _ok, _ := queue.Get()
		index++
		if _ok {
			if self.receive != nil {
				self.receive.Receive(val.(*QueueMsg), index)
			} else {
				msg := val.(*QueueMsg)
				function, ok := self.functions[msg.Func]
				if !ok {
					//fmt.Println(fmt.Sprintf("Remote function(%s) not found", msg.Func))
					if self.opts.NoFound != nil {
						fc, err := self.opts.NoFound(msg)
						if err != nil {
							self.opts.RecoverHandle(msg, err)
							continue
						}
						function = fc
					} else {
						if self.opts.RecoverHandle != nil {
							self.opts.RecoverHandle(msg, errors.Errorf("Remote function(%s) not found", msg.Func))
						}
						continue
					}
				}
				f := function
				in := make([]reflect.Value, len(msg.Params))
				for k, _ := range in {
					switch v2 := msg.Params[k].(type) { //多选语句switch
					case nil:
						in[k] = reflect.Zero(f.Type().In(k))
					default:
						in[k] = reflect.ValueOf(v2)
					}
					//in[k] = reflect.ValueOf(msg.Params[k])
				}
				_runFunc := func() {
					defer func() {
						if r := recover(); r != nil {
							buff := make([]byte, 1024)
							runtime.Stack(buff, false)
							if self.opts.RecoverHandle != nil {
								self.opts.RecoverHandle(msg, errors.New(string(buff)))
							}
						}
					}()
					out := f.Call(in)
					if self.opts.ErrorHandle != nil {
						if len(out) == 1 {
							value, ok := out[0].Interface().(error)
							if ok {
								if value != nil {
									self.opts.ErrorHandle(msg, value)
								}
							}
						}
					}
				}
				_runFunc()
			}
		}
		ok = _ok
	}
}
