package hello

import (
	"fmt"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
)

var Module = func() module.Module {
	this := new(Hello)
	return this
}

type Hello struct {
	basemodule.BaseModule
	room       *room.Room
	curTableID string
}

func (self *Hello) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "hello"
}
func (self *Hello) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}
func (s *Hello) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings)
	s.curTableID = "11"
	s.room = room.NewRoom(s.App)
	_, err := s.room.CreateById(s.App, s.curTableID, s.NewTable)
	if err != nil {
		log.Error(err.Error())
	}
	//self.GetServer().RegisterGO("/say/hi", self.say) //handler
	//hdLogin := &game.Hook{Fun:impl.HdLogin}
	//self.GetServer().RegisterGO("HD_login", hdLogin.NoLoginHook)
	s.GetServer().RegisterGO("/listener/onLogin", s.onLogin)
	common.AddListener(s.GetServerID(), common.EventLogin, "/listener/onLogin")
	s.GetServer().RegisterGO("/listener/onDisconnect", s.onDisconnect)
	common.AddListener(s.GetServerID(), common.EventDisconnect, "/listener/onDisconnect")

	hook := game.NewHook(s.GetType())
	hook.RegisterAndCheckLogin(s.GetServer(), "HD_hello", s.hello)
	s.GetServer().RegisterGO("HD_play", s.TableQueue)
}
func (self *Hello) NewTable(app module.App, tableId string) (room.BaseTable, error) {
	table := NewTable(
		self, app,
		room.TableId(tableId),
		room.Router(func(TableId string) string {
			return fmt.Sprintf("%v://%v/%v", self.GetType(), self.GetServerId(), tableId)
		}),
		room.Capaciity(2048),
		//room.DestroyCallbacks(func(table room.BaseTable) error {
		//	log.Info("回收了房间: %v", table.TableId())
		//	_ = self.room.DestroyTable(table.TableId())
		//	return nil
		//}),
	)
	return table, nil
}

func (self *Hello) TableQueue(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	//	table_id := msg["table_id"].(string)
	action := msg["action"].(string)
	table := self.room.GetTable(self.curTableID) //

	if table == nil {
		log.Info("--------------- table not exist---")
		return errCode.ServerError.GetI18nMap(), nil
	}
	erro := table.PutQueue(action, session, msg)
	if erro != nil {
		log.Info("--------------- table.PutQueue error---tableID = %s", self.curTableID, "---error = %s", erro)
		return errCode.ServerError.GetI18nMap(), nil
	}
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}

func (s *Hello) hello(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	log.Info("hello.................. %v, ip:%v", msg, session.GetIP())
	time.Sleep(500 * time.Millisecond)
	return errCode.Success(nil).GetI18nMap(), nil
}

func (self *Hello) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())

	<-closeSig
	log.Info("%v模块已停止...", self.GetType())
}

func (self *Hello) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}
func (s *Hello) onLogin(uid string) (interface{}, error) {
	log.Info("serverId: %s, uid: %s", s.GetServerID(), uid)
	return nil, nil
}
func (s *Hello) onDisconnect(uid string) (interface{}, error) {
	log.Info("onDisconnect serverId: %s, uid: %s", s.GetServerID(), uid)
	return nil, nil
}
