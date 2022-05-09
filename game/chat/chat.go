package chat

import (
	"github.com/robfig/cron"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	gate2 "vn/gate"
	"vn/storage/chatStorage"
)

var Module = func() module.Module {
	this := new(Chat)
	return this
}

type Chat struct {
	basemodule.BaseModule
	impl *Impl
	push *gate2.OnlinePush
	botsChat map[game.Type][]chatStorage.ChatBotMsgList
}

func (self *Chat) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.Chat)
}
func (self *Chat) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

const (
	actionSend      = "HD_send"
	actionJoinGroup = "HD_joinGroup"
	actionExitGroup = "HD_exitGroup"
	actionMsgLog    = "HD_msgLog"
)

func (self *Chat) OnInit(app module.App, settings *conf.ModuleSettings) {
	self.BaseModule.OnInit(self, app, settings)
	hook := game.NewHook(self.GetType())
	hook.RegisterAndCheckLogin(self.GetServer(), actionSend, self.send)
	hook.RegisterAndCheckLogin(self.GetServer(), actionJoinGroup, self.joinGroup)
	hook.RegisterAndCheckLogin(self.GetServer(), actionExitGroup, self.exitGroup)
	hook.RegisterAndCheckLogin(self.GetServer(), actionMsgLog, self.msgLog)

	self.GetServer().RegisterGO("/chat/onDisconnect", self.onDisconnect)
	common.AddListener(self.GetServerID(), common.EventDisconnect, "/chat/onDisconnect")
	self.push = &gate2.OnlinePush{
		TraceSpan: log.CreateRootTrace(),
		App:       app,
	}
	self.push.OnlinePushInit(nil, 2048)
	self.impl = &Impl{push: self.push}
	incDataExpireDay := time.Duration(
		app.GetSettings().Settings["mongoIncDataExpireDay"].(float64)) * 24 * time.Hour
	chatStorage.Init(incDataExpireDay)

	self.InitBotsChat()
	go func() {
		c := cron.New()
		c.AddFunc("*/1 * * * * ?",self.OnTimer)
		c.Start()
	}()

}

func (self *Chat) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	self.push.Run(100 * time.Millisecond)
	<-closeSig
	log.Info("%v模块已停止...", self.GetType())
}

func (self *Chat) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}
func (s *Chat) send(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"groupId", "msgId", "content"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), ok
	}
	groupId := params["groupId"].(string)
	msgId := params["msgId"].(string)
	content := params["content"].(string)
	s.impl.send(session.GetUserID(), msgId, groupId, content)
	return errCode.Success(nil).GetI18nMap(), nil
}
func (s *Chat) joinGroup(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"groupId"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), ok
	}
	groupId := params["groupId"].(string)
	if !utils.StrInArray(groupId, game.ChatGroups) {
		return errCode.ChatGroupNotFind.GetI18nMap(), nil
	}
	s.impl.addGroup(session.GetUserID(), groupId)
	return errCode.Success(nil).GetI18nMap(), nil
}
func (s *Chat) exitGroup(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"groupId"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), ok
	}
	groupId := params["groupId"].(string)
	s.impl.exitGroup(session.GetUserID(), groupId)
	return errCode.Success(nil).GetI18nMap(), nil
}
func (s *Chat) onDisconnect(uid string) (interface{}, error) {
	log.Info("onDisconnect serverId: %s, uid: %s", s.GetServerID(), uid)
	s.impl.disconnect(uid)
	return nil, nil
}
func (s *Chat) msgLog(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"groupId", "size"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), ok
	}
	groupId, _ := params["groupId"].(string)
	size, _ := utils.ConvertInt(params["size"])
	if size > 100 {
		return errCode.PageSizeErr.GetI18nMap(), nil
	}
	msgList := s.impl.getGroupMsgList(groupId, size)
	res := make(map[string]interface{}, 1)
	res["msgList"] = msgList
	msg := make(map[string]interface{}, 1)
	msg["GroupId"] = groupId
	res["msg"] = msg
	return errCode.Success(res).GetI18nMap(), nil
}
