package admin

import (
	"time"
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
	this := new(Admin)
	return this
}

type Admin struct {
	basemodule.BaseModule
	room       *room.Room
	curTableID string
}

func (self *Admin) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "admin"
}
func (self *Admin) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}
func (s *Admin) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings)
	//s.curTableID = "11"
	//s.room = room.NewRoom(s.App)
	//_, err := s.room.CreateById(s.App, s.curTableID, s.NewTable)
	//if err != nil {
	//	log.Error(err.Error())
	//}

	hook := game.NewHook(s.GetType())
	hook.RegisterAdminInterface(s.GetServer(), "HD_login", s.login)
	//s.GetServer().RegisterGO("HD_play", s.TableQueue)
}

const AdminUid = "605d84730356a24829e88c23"
func (s *Admin) login(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	session.Bind(AdminUid)
	return errCode.Success(nil).GetI18nMap(), nil
}
func (s *Admin) hello(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	log.Info("hello.................. %v, ip:%v", msg, session.GetIP())
	time.Sleep(500 * time.Millisecond)
	return errCode.Success(nil).GetI18nMap(), nil
}

func (self *Admin) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())

	<-closeSig
	log.Info("%v模块已停止...", self.GetType())
}

func (self *Admin) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}
func (s *Admin) onLogin(uid string) (interface{}, error) {
	log.Info("serverId: %s, uid: %s", s.GetServerID(), uid)
	return nil, nil
}
func (s *Admin) onDisconnect(uid string) (interface{}, error) {
	log.Info("onDisconnect serverId: %s, uid: %s", s.GetServerID(), uid)
	return nil, nil
}
