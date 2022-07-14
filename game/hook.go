package game

import (
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/server"
	"vn/storage/userStorage"
)

type Func func(session gate.Session, params map[string]interface{}) (map[string]interface{}, error)

type hook struct {
	//funMap map[string]hookFunc
	ModelType string
}
type hookFunc struct {
	fun       Func
	modelType string
	action    string
}

func NewHook(modelType string) *hook {
	return &hook{
		//funMap:make(map[string]hookFunc),
		ModelType: modelType,
	}
}
func (s *hook) RegisterAndCheckLogin(server server.Server, action string, fun Func) *hook {
	hookFun := hookFunc{fun: fun, action: action, modelType: s.ModelType}
	//s.funMap[action] = hookFun
	server.RegisterGO(action, hookFun.NeedLogin)
	return s
}
func (s *hook) RegisterAndNoLogin(server server.Server, action string, fun Func) *hook {
	hookFun := &hookFunc{fun: fun, action: action, modelType: s.ModelType}
	//s.funMap[action] = *hookFun
	server.RegisterGO(action, hookFun.NoLogin)
	return s
}
func (s *hook) RegisterAdminInterface(server server.Server, action string, fun Func) *hook {
	hookFun := &hookFunc{fun: fun, action: action, modelType: s.ModelType}
	//s.funMap[action] = *hookFun
	server.RegisterGO(action, hookFun.checkAdmin)
	return s
}

func (s *hookFunc) NeedLogin(session gate.Session, params map[string]interface{}) (interface{}, error) {
	//log.Info("NeedLogin:%s ", s.action)
	uid := session.GetUserID()
	if uid == "" {
		s.closeSession(session)
		return s.noLoginResult()
	}
	token := userStorage.QueryTokenByUid(utils.ConvertOID(uid))
	if token == nil || token.SessionId != session.GetSessionID() {
		s.closeSession(session)
		return s.accountChanged(token.AccessToken)
	}
	user := userStorage.QueryUserId(token.Oid)
	if user.Status != userStorage.StatusNormal {
		s.closeSession(session)
		return s.accountBlack(token.Oid.Hex())
	}
	start := time.Now()
	//userStorage.QueryTokenByUid(utils.ConvertOID(uid))
	res, err := s.fun(session, params)
	s.parseResponse(res)
	spent := time.Since(start)
	if spent > 400*time.Millisecond {
		log.Warning("slow action %v,gameType:%s, spent:%v", s.action, s.modelType, spent)
	}
	return res, err
}
func (s *hookFunc) closeSession(session gate.Session) {
	go func() {
		time.Sleep(2 * time.Second)
		session.Close()
	}()
}
func (s *hookFunc) noLoginResult() (interface{}, error) {
	nologin := errCode.Forbidden.GetI18nMap()
	s.parseResponse(nologin)
	return nologin, errCode.Forbidden.GetError()
}
func (s *hookFunc) accountChanged(token string) (interface{}, error) {
	accountChanged := errCode.AccountChanged.SetData(token).GetI18nMap()
	s.parseResponse(accountChanged)
	return accountChanged, errCode.Forbidden.GetError()
}
func (s *hookFunc) accountBlack(uid string) (interface{}, error) {
	accountChanged := errCode.ConnectCustomerService.SetData(uid).GetI18nMap()
	s.parseResponse(accountChanged)
	return accountChanged, errCode.Forbidden.GetError()
}
func (s *hookFunc) NoLogin(session gate.Session, params map[string]interface{}) (interface{}, error) {
	//log.Info("nologin action: %s", s.action)
	res, err := s.fun(session, params)
	s.parseResponse(res)
	return res, err
}
func (s *hookFunc) checkAdmin(session gate.Session, params map[string]interface{}) (interface{}, error) {
	ip := utils.GetIPBySession(session.GetIP())
	adminIpConf := common.App.GetSettings().Settings["adminIp"].(string)
	//if strings.HasPrefix(ip,"172")||
	//	strings.HasPrefix(ip,"127")||
	//	strings.HasPrefix(ip,"192"){
	if ip == "127.0.0.1" || ip == adminIpConf {
		//session.Bind("000000000000000000000001")
		res, err := s.fun(session, params)
		s.parseResponse(res)
		return res, err
	}
	forbidden := errCode.Forbidden.SetErr("forbidden admin ip:" + ip).GetI18nMap()
	s.parseResponse(forbidden)
	return forbidden, nil
}
func (s *hookFunc) parseResponse(res map[string]interface{}) {
	if res != nil {
		if res["Action"] == nil || res["Action"] == "" {
			res["Action"] = s.action
		}
		res["GameType"] = s.modelType
	}
}
