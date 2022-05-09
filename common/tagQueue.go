package common

import "vn/framework/mqant/log"

type (
	//Queue 队列
	Queue struct {
		top    *node
		rear   *node
		length int
	}
	//双向链表节点
	node struct {
		pre   *node
		next  *node
		value interface{}
	}
)

// Create a new queue
func NewQueue() *Queue {
	return &Queue{nil, nil, 0}
}

//获取队列长度
func (this *Queue) Len() int {
	return this.length
}

//返回true队列不为空
func (this *Queue) Any() bool {
	return this.length > 0
}

//返回队列顶端元素
func (this *Queue) Peek() interface{} {
	if this.top == nil {
		return nil
	}
	return this.top.value
}

//入队操作
func (this *Queue) Push(v interface{}) {
	n := &node{nil, nil, v}
	if this.length == 0 {
		this.top = n
		this.rear = this.top
	} else {
		n.pre = this.rear
		this.rear.next = n
		this.rear = n
	}
	this.length++
}

//出队操作
func (this *Queue) Pop() interface{} {
	if this.length == 0 {
		return nil
	}
	n := this.top
	if this.top.next == nil {
		this.top = nil
	} else {
		this.top = this.top.next
		this.top.pre.next = nil
		this.top.pre = nil
	}
	this.length--
	return n.value
}

type FuncQueue struct {
	queue *Queue

	running bool
}

var tagQueue = NewMapRWMutex()

func NewFuncQueue() *FuncQueue {
	return &FuncQueue{
		NewQueue(),
		false,
	}
}
func (s *FuncQueue) length() int {
	return s.queue.length
}
func (s *FuncQueue) Add(f func()) {
	s.queue.Push(f)
	s.Run()
}
func AddQueueByTag(tag string, f func()) {
	queueValue := tagQueue.Get(tag)
	if queueValue == nil {
		queueValue = NewFuncQueue()
		tagQueue.Set(tag, queueValue)
	}
	fQueue := queueValue.(*FuncQueue)
	fQueue.Add(f)
	if fQueue.length() == 0 {
		tagQueue.Remove(tag)
	}
}
func Tt() {
	log.Info("map len:%v", tagQueue.Len())
}
func (s *FuncQueue) Run() {
	if s.running {
		return
	}
	s.running = true
	v := s.queue.Pop()
	for v != nil {
		f := v.(func())
		f()
		v = s.queue.Pop()
	}
	s.running = false
}
