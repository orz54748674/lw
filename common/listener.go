package common

import (
	"time"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
)

type Listener struct {
	Oid            primitive.ObjectID `bson:"_id,omitempty"`
	ServerId       string             `bson:"ServerId"`
	ServerRegister string             `bson:"ServerRegister"`
	Event          string             `bson:"Event"`
}
type OnLogin func(uid string)
type OnDisconnect func(uid string)

var (
	cListener       = "listener"
	EventLogin      = "OnLogin"
	EventDisconnect = "OnDisconnect"
)

func InitListener(app module.App) {
	go func() {
		time.Sleep(10 * time.Second)
		doInit(app)
	}()
	go func() {
		time.Sleep(20 * time.Second)
		doInit(app)
	}()
	go func() {
		time.Sleep(1 * time.Minute)
		doInit(app)
	}()
}
func doInit(app module.App) {
	c := GetMongoDB().C(cListener)
	var listeners []Listener
	_ = c.Find(bson.M{}).All(&listeners)
	serverOid := make([]primitive.ObjectID, 0)
	for _, l := range listeners {
		if _, err := app.GetServerByID(l.ServerId); err != nil {
			serverOid = append(serverOid, l.Oid)
		}
	}
	c.RemoveAll(bson.M{"_id": bson.M{"$in": serverOid}})
	log.Info("initListener")
}
func AddListener(serverId string, event string, serverRegister string) {
	log.Info("AddListener")
	s := newListener(serverId, event, serverRegister)
	c := GetMongoDB().C(cListener)
	selector := bson.M{
		"ServerId":       s.ServerId,
		"Event":          s.Event,
		"ServerRegister": serverRegister,
	}
	if _, err := c.Upsert(selector, selector); err != nil {
		log.Error(err.Error())
	}
}
func QueryListener(event string) *[]Listener {
	c := GetMongoDB().C(cListener)
	selector := bson.M{
		"Event": event,
	}
	var listeners []Listener
	err := c.Find(selector).All(&listeners)
	if err != nil {
		log.Info(err.Error())
	}
	return &listeners
}
func newListener(serverId string, event string, serverRegister string) *Listener {
	listener := &Listener{
		ServerId:       serverId,
		Event:          event,
		ServerRegister: serverRegister,
	}
	return listener
}
