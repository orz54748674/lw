package common

import "sync"

type MapRWMutex struct {
	Data map[string]interface{}
	Lock *sync.RWMutex
}

func NewMapRWMutex() *MapRWMutex {
	m := &MapRWMutex{}
	m.Data = map[string]interface{}{}
	m.Lock = new(sync.RWMutex)
	return m
}
func (d MapRWMutex) Get(k string) interface{} {
	d.Lock.RLock()
	defer d.Lock.RUnlock()
	return d.Data[k]
}

func (d MapRWMutex) Set(k string, v interface{}) {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	d.Data[k] = v
}

func (d MapRWMutex) Len() int {
	return len(d.Data)
}
func (d MapRWMutex) Remove(k string) {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	delete(d.Data, k)
}
