package lobby

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	"vn/game/activity"
	"vn/game/lobby/lobbyImpl"
	pk "vn/game/mini/poker"
	gate2 "vn/gate"
	"vn/http/loginImpl"
	"vn/storage/agentStorage"
	"vn/storage/apiStorage"
	"vn/storage/botStorage"
	"vn/storage/gameStorage"
	"vn/storage/gbsStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

type Impl struct {
	app      module.App
	settings *conf.ModuleSettings
}

func (s *Impl) Login(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	//ip := utils.GetIPBySession(session)
	//log.Info("params: %s", params)
	_, err = utils.CheckParams2(params, []string{"token", "uid"})
	if err != nil {
		return nil, err
	}
	token := params["token"].(string)
	res, uid, err := lobbyImpl.TcpLogin(s.app, token, session)
	if err != nil {
		go func() {
			time.Sleep(2 * time.Second)
			session.Close()
		}()
		log.Info(err.Error())
		return errCode.Forbidden.SetData(token).GetMap(), err
	}
	uOid := utils.ConvertOID(uid)
	res["wallet"] = walletStorage.GetWallet(uOid)
	res["user"] = userStorage.QueryUserId(uOid)
	res["userInfo"] = userStorage.QueryUserInfo(uOid)
	//userStorage.UpdateTokenTime(utils.ConvertOID(uid))
	RefreshLobbyBubble(uOid)
	//TODO 返回用户状态
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) GetUserInfo(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	uOid := utils.ConvertOID(session.GetUserID())
	res := make(map[string]interface{})
	res["wallet"] = walletStorage.GetWallet(uOid)
	res["user"] = userStorage.QueryUserId(uOid)
	res["userInfo"] = userStorage.QueryUserInfo(uOid)
	//TODO 返回用户状态
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) Info(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	//log.Info("params: %s,ip: %s")
	_, err = utils.CheckParams2(params, []string{"key"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	keysP := params["key"].([]interface{})
	var keyStr []string
	for _, k := range keysP {
		keyStr = append(keyStr, k.(string))
	}
	res := lobbyImpl.GetLobbyInfo(utils.ConvertOID(session.GetUserID()), keyStr)
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) Page(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	_, err := utils.CheckParams2(params, []string{"path"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	path := params["path"].(string)
	uid := utils.ConvertOID(session.GetUserID())
	gate2.UpdateSessionPage(uid, path)
	updateUserPage(uid, path)
	return errCode.Success(nil).GetI18nMap(), nil
}

func (s *Impl) AgentInfo(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	uOid := utils.ConvertOID(uid)
	agent := agentStorage.QueryAgent(uOid)
	if agent == nil {
		agent = &agentStorage.Agent{}
	}
	inviteData := agentStorage.QueryAgentInviteData(uOid)
	agentInfo := &agentStorage.AgentInfo{
		TodayProfit:  agentStorage.QueryTodayProfit(uOid, 0),
		SumProfit:    agent.SumIncome,
		InvitedCount: inviteData.Count,
		AliveCount:   agent.Count,
		BetAmount1:   agentStorage.QueryTodayProfitBet(uOid, 1),
		BetAmount2:   agentStorage.QueryTodayProfitBet(uOid, 2),
		BetAmount3:   agentStorage.QueryTodayProfitBet(uOid, 3),
		Profit1:      agentStorage.QueryTodayProfit(uOid, 1),
		Profit2:      agentStorage.QueryTodayProfit(uOid, 2),
		Profit3:      agentStorage.QueryTodayProfit(uOid, 3),
	}
	res := make(map[string]interface{}, 2)
	wallet := walletStorage.QueryWallet(utils.ConvertOID(session.GetUserID()))
	res["wallet"] = wallet
	res["agentInfo"] = agentInfo
	return errCode.Success(res).GetI18nMap(), nil
}

func (s *Impl) BindPhone(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	_, err := utils.CheckParams2(params, []string{"area", "phone", "code"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	if user.Phone != 0 {
		return errCode.UserAlreadyBind.GetI18nMap(), nil
	}
	if len(params["phone"].(string)) != 9 && len(params["phone"].(string)) != 10 {
		return errCode.PhoneFormatErr.GetI18nMap(), nil
	}
	area, _ := utils.ConvertInt(params["area"])
	phone, length := utils.ParsePhone(params["phone"])
	if length != 9 {
		return errCode.PhoneFormatErr.GetI18nMap(), nil
	}
	code, _ := params["code"].(string)
	event := lobbyStorage.EventBind
	if check := loginImpl.CheckoutCode(area, phone, event, code); check {
		if query := userStorage.QueryUser(bson.M{"Area": area, "Phone": phone}); query != nil {
			return errCode.PhoneAlreadyBind.GetI18nMap(), nil
		}
		reward := lobbyImpl.BindPhone(utils.ConvertOID(uid), area, phone)
		res := make(map[string]interface{}, 1)
		res["reward"] = reward
		return errCode.Success(res).GetI18nMap(), nil
	}
	return errCode.SmsCodeErr.GetI18nMap(), nil
}
func (s *Impl) myBill(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	//uid := session.GetUserID()
	_, err := utils.CheckParams2(params, []string{"area", "phone", "code"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	return errCode.Success(nil).GetI18nMap(), nil
}
func (s *Impl) TestPush(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {

	res := make(map[string]interface{})
	//res["ok"] = banners

	return res, nil
}
func (s *Impl) PrizePool(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {

	res := map[string]interface{}{}

	res["gbsPoolInfo"] = gbsStorage.GetGameConf()
	res["miniPokerPool"] = pk.GetPrizePool()

	return errCode.Success(res).GetI18nMap(), nil
}
func CheckNickName(nickName string) (string, error) {
	if len(nickName) < 5 || len(nickName) > 16 {
		return nickName, fmt.Errorf("nickName format error")
	}
	if strings.Contains(nickName, " ") {
		return nickName, fmt.Errorf("nickName format error")
	}
	patternList := []string{`[a-zA-Z]+`, `^[a-zA-Z0-9]+$`}
	for _, pattern := range patternList {
		match, _ := regexp.MatchString(pattern, nickName)
		if !match {
			return nickName, fmt.Errorf("nickName format error")
		}
	}
	return nickName, nil
}
func CheckNickName2(nickName string) (string, error) {
	users := userStorage.QueryUsers()
	for _, v := range users {
		if v.NickName == nickName || v.Account == nickName {
			return nickName, fmt.Errorf("nickName exist error")
		}
	}
	bots := botStorage.QueryBots()
	for _, v := range bots {
		if strings.Contains(v.NickName, nickName) {
			return nickName, fmt.Errorf("nickName exist error")
		}
	}
	return nickName, nil
}
func (s *Impl) SetNickName(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	_, err := utils.CheckParams2(params, []string{"NickName"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	nickName := params["NickName"].(string)
	if _, ok := CheckNickName(nickName); ok != nil {
		return errCode.NameFormatError.GetI18nMap(), nil
	}
	if _, ok := CheckNickName2(nickName); ok != nil {
		return errCode.NickNameExistError.GetI18nMap(), nil
	}
	user.NickName = nickName
	userStorage.UpdateUser(user)
	res := make(map[string]interface{}, 1)
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) SetAvatar(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	_, err := utils.CheckParams2(params, []string{"Avatar"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	Avatar := params["Avatar"].(string)
	user.Avatar = Avatar
	userStorage.UpdateUser(user)
	return errCode.Success(nil).GetI18nMap(), nil
}
func CheckModifyPwd(pwd string) (string, error) {
	if len(pwd) < 6 || len(pwd) > 16 {
		return pwd, fmt.Errorf("password format error")
	}
	if strings.Contains(pwd, " ") {
		return pwd, fmt.Errorf("password format error")
	}
	return pwd, nil
}

func (s *Impl) ModifyPassword(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	login := userStorage.QueryLogin(utils.ConvertOID(uid))
	_, err := utils.CheckParams2(params, []string{"OldPwd", "NewPwd", "ConfirmPwd"})
	if err != nil || login == nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	OldPwd := params["OldPwd"].(string)
	NewPwd := params["NewPwd"].(string)
	ConfirmPwd := params["ConfirmPwd"].(string)

	if OldPwd != login.Password {
		return errCode.OldPwdError.GetI18nMap(), nil
	}
	if NewPwd != ConfirmPwd {
		return errCode.PwdNotSameError.GetI18nMap(), nil
	}
	if _, ok := CheckModifyPwd(NewPwd); ok != nil {
		return errCode.PwdFormatError.GetI18nMap(), nil
	}
	login.Password = NewPwd
	userStorage.UpsertLogin(login)
	res := make(map[string]interface{}, 1)
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) agentConf(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	agentConf := agentStorage.QueryAllAgentConf()
	res := make(map[string]interface{}, 2)
	res["agentConf"] = agentConf
	uid := session.GetUserID()
	res["inviteUrl"] = lobbyImpl.GetInviteUrl(uid)
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) QueryBetRecord(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	_, err := utils.CheckParams2(params, []string{"Offset", "PageSize"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	Offset, _ := utils.ConvertInt(params["Offset"])
	PageSize, _ := utils.ConvertInt(params["PageSize"])
	RecordType := params["RecordType"].(string)
	res := make(map[string]interface{}, 2)
	betRecord := gameStorage.QueryBetRecord(uid, int(Offset), int(PageSize), RecordType)
	res["BetRecord"] = betRecord
	res["TotalNum"] = gameStorage.QueryBetRecordTotal(uid, RecordType)
	res["RecordType"] = RecordType
	return errCode.Success(res).GetI18nMap(), nil
}
func (this *Lobby) notifyWallet(uid string) {
	sb := gate2.QuerySessionBean(uid)
	if sb == nil {
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	msg := make(map[string]interface{})
	msg["Wallet"] = wallet
	msg["Action"] = "wallet"
	msg["GameType"] = game.All
	b, _ := json.Marshal(msg)
	this.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, b)
}

func (this *Lobby) AdminNotice(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	adminId, _ := utils.ConvertInt(params["adminId"])
	title := params["title"].(string)
	content := params["content"].(string)
	res := make(map[string]interface{})
	res["Code"] = 0
	res["Action"] = "globalNotice"
	res["ErrMsg"] = "操作成功"
	res["GameType"] = "lobby"
	notice := make(map[string]interface{}, 2)
	notice["title"] = title
	notice["content"] = content
	res["data"] = notice
	ret, _ := json.Marshal(res)
	this.push.NotifyAllPlayersNR(game.Push, ret)
	Notice := &lobbyStorage.Notice{
		AdminId:    adminId,
		Title:      title,
		Content:    content,
		CreateTime: utils.Now(),
	}
	lobbyStorage.InsertNotice(Notice)
	return errCode.Success(nil).GetI18nMap(), nil
}

type api struct {
	GameType   string
	Topic      string
	ScreenType int8
}

func (s *Impl) ApiConf(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	_, err := utils.CheckParams2(params, []string{"type"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	t, _ := params["type"].(string)
	env := s.app.GetSettings().Settings["env"].(string)
	res := []*api{}
	mApiConfig := &apiStorage.ApiConfig{}
	apis, err := mApiConfig.GetApis(t, env)
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	for _, item := range apis {
		res = append(res, &api{
			GameType:   item.ApiTypeName,
			Topic:      fmt.Sprintf("%v&ProductType=%v", item.Topic, item.ProductType),
			ScreenType: item.ScreenType,
		})
	}
	data := map[string]interface{}{
		"apis": res,
		"url":  "apiLogin?Token=%s&Topic=%s&GameType=%s&DriveType=%d",
	}

	return errCode.Success(data).GetI18nMap(), nil
}
func (s *Impl) GetWindowVnd(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	uid := utils.ConvertOID(session.GetUserID())
	wallet := walletStorage.QueryWallet(uid)
	inRoomNeedVnd := gameStorage.QueryGameCommonData(uid.Hex()).InRoomNeedVnd
	res := make(map[string]interface{}, 1)
	res["Vnd"] = wallet.VndBalance - inRoomNeedVnd
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) wallet(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	userID := session.GetUserID()
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	userInfo := userStorage.QueryUserInfo(utils.ConvertOID(userID))
	res := map[string]interface{}{
		"wallet":     wallet,
		"safeStatus": userInfo.SafeStatus,
	}
	return errCode.Success(res).GetMap(), nil
}

func RefreshLobbyBubble(uid primitive.ObjectID) {
	activity.RefreshNormalActivity(uid)
	activity.RefreshDayActivity(uid)
}
func (s *Impl) GetRankList(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	_, err := utils.CheckParams2(params, []string{"gameType"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	gameType := params["gameType"].(string)
	res := make(map[string]interface{}, 1)
	res["RankList"] = gameStorage.GetGameWinLoseRank(game.Type(gameType), 20)
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) GameInviteRecord(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	res := gameStorage.QueryGameInviteRecord(uid)
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) SetSafeStatus(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	_, err := utils.CheckParams2(params, []string{"Status", "PassWord"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	uid := session.GetUserID()
	login := userStorage.QueryLogin(utils.ConvertOID(uid))
	if login == nil {
		return errCode.Forbidden.GetI18nMap(), nil
	}
	userInfo := userStorage.QueryUserInfo(utils.ConvertOID(uid))
	if userInfo.SafeStatus == 0 {
		return errCode.PleaseActivationSafe.GetI18nMap(), nil
	}
	password := params["PassWord"].(string)
	if password != login.Password {
		return errCode.PasswordErr.GetI18nMap(), nil
	}
	status, _ := utils.ConvertInt(params["Status"])
	userStorage.SetUserInfoSafeStatus(utils.ConvertOID(uid), int(status))
	res := make(map[string]interface{}, 1)
	res["Status"] = status
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) GetMaxJackpotAll(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	res := lobbyImpl.GetMaxJackpotAll()
	return errCode.Success(res).GetI18nMap(), nil
}
