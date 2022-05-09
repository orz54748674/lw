package basegate

import (
	"github.com/yireyun/go-queue"
	"sync"
)

var userQueue = newMapRWMutex()
type userSyncExec struct {
	queue *queue.EsQueue
	running bool
}

func getQueue(sessionId string) *userSyncExec {
	q := userQueue.Get(sessionId)
	if q != nil {
		return q.(*userSyncExec)
	}else{
		exec := &userSyncExec{
			queue: queue.NewQueue(64),
		}
		userQueue.Set(sessionId,exec)
		return exec
	}
}

func (s *userSyncExec) execQueue()  {
	if s.running{
		return
	}
	s.running = true
	ok := true
	for ok{
		val,_ok,_ := s.queue.Get()
		if _ok{
			f := val.(func())
			f()
		}
		ok = _ok
	}
	s.running = false
}

type mapRWMutex struct {
	Data map[string]interface{}
	Lock *sync.RWMutex
}

func newMapRWMutex() *mapRWMutex {
	m := &mapRWMutex{}
	m.Data = map[string]interface{}{}
	m.Lock = new(sync.RWMutex)
	return m
}
func (d mapRWMutex) Get(k string) interface{}{
	d.Lock.RLock()
	defer d.Lock.RUnlock()
	return d.Data[k]
}

func (d mapRWMutex) Set(k string,v interface{}) {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	d.Data[k] = v
}

func (d mapRWMutex) Remove(k string) {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	delete(d.Data,k)
}
func (d mapRWMutex) size() int {
	return len(d.Data)
}