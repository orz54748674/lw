package http

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
	mqrpc "vn/framework/mqant/rpc"
	"vn/http/loginImpl"
	"vn/storage"
	"vn/storage/dataStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/userStorage"
)

type UserController struct {
	BaseController
}

func (s *UserController) phoneBind(w http.ResponseWriter, r *http.Request) {

}

func (s *UserController) tokenBind(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	p := r.Form
	if check, ok := utils.CheckParams(p,
		[]string{"token"}); ok != nil {
		s.response(w, errCode.ErrParams.SetKey(check))
		return
	}
	token := p["token"][0]
	tokenObj := userStorage.QueryToken(token)
	if tokenObj == nil {
		s.response(w, errCode.Forbidden)
		return
	}
	login := userStorage.QueryLogin(tokenObj.Oid)
	if login == nil {
		return
	}
	ip := utils.GetIP(r)
	tokenObj.Ip = ip
	userStorage.UpsertToken(tokenObj)

	login.LastIp = tokenObj.Ip
	//userStorage.UpsertLogin(login)
	userStorage.NewLoginLog(r, *login)
	common.ExecQueueFunc(func() {
		ipInfo := dataStorage.IpInfo{Ip: ip}
		if ipInfo.Exists() {
			return
		}
		if err := ipInfo.RequestIpInfo(); err == nil {
			ipInfo.Create()
		}
	})
	s.response(w, errCode.Success(nil))
}
func (s *UserController) login(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	p := r.Form
	if check, ok := utils.CheckParams(p,
		[]string{"account", "password", "platform", "uuid", "uuid_web"}); ok != nil {
		s.response(w, errCode.ErrParams.SetKey(check))
		return
	}
	account := strings.ToLower(p["account"][0])
	errCode := loginImpl.LoginAccount(s.App, account, p["password"][0],
		p["platform"][0], p["uuid"][0], p["uuid_web"][0], r)
	s.response(w, errCode)
}
func CheckRegisterName(params url.Values) (string, error) {
	accout := params["account"][0]
	if len(accout) < 5 || len(accout) > 16 {
		return accout, fmt.Errorf("accout format error")
	}
	if strings.Contains(accout, " ") {
		return accout, fmt.Errorf("accout format error")
	}
	//patternList := []string{`[~!@#$%^&*?_-]+`}
	//for _, pattern := range patternList {
	//	match, _ := regexp.MatchString(pattern, accout)
	//	if match {
	//		return accout,fmt.Errorf("accout format error")
	//	}
	//}
	patternList := []string{`[a-zA-Z]+`, `^[a-zA-Z0-9]+$`}
	for _, pattern := range patternList {
		match, _ := regexp.MatchString(pattern, accout)
		if !match {
			return accout, fmt.Errorf("accout format error")
		}
	}
	return accout, nil
}
func CheckRegisterPwd(params url.Values) (string, error) {
	pwd := params["password"][0]
	if len(pwd) < 6 || len(pwd) > 16 {
		return pwd, fmt.Errorf("password format error")
	}
	if strings.Contains(pwd, " ") {
		return pwd, fmt.Errorf("password format error")
	}
	return pwd, nil
}

func (s *UserController) register(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	p := r.PostForm
	if check, ok := utils.CheckParams(p,
		[]string{"account", "password", "platform", "channel", "uuid", "uuid_web"}); ok != nil {
		s.response(w, errCode.ErrParams.SetKey(check))
		return
	}
	if userStorage.DeviceIsBlack(p["uuid"][0]) || userStorage.IpIsBlack(utils.GetIP(r)) {
		s.response(w, errCode.ConnectCustomerService.SetKey())
		return
	}
	registerLimit := storage.QueryConf(storage.KRegisterLimit).(string)
	conf := strings.Split(registerLimit, ",")
	days, _ := utils.ConvertInt(conf[0])
	uuidNum, _ := utils.ConvertInt(conf[1])
	ipNum, _ := utils.ConvertInt(conf[2])
	if days != 0 && uuidNum != 0 {
		limitTime := utils.Now().AddDate(0, 0, int(-days))
		num := userStorage.QueryUserNums(bson.M{"RegisterUuid": p["uuid"][0], "CreateAt": bson.M{"$gt": limitTime}})
		if num >= uuidNum {
			s.response(w, errCode.RegisterLimit.SetKey())
			return
		}
	}
	if days != 0 && ipNum != 0 {
		limitTime := utils.Now().AddDate(0, 0, int(-days))
		num := userStorage.QueryUserNums(bson.M{"RegisterIp": utils.GetIP(r), "CreateAt": bson.M{"$gt": limitTime}})
		if num >= ipNum {
			s.response(w, errCode.RegisterLimit.SetKey())
			return
		}
	}
	if check, ok := CheckRegisterName(p); ok != nil {
		s.response(w, errCode.NameFormatError.SetKey(check))
		return
	}
	if check, ok := CheckRegisterPwd(p); ok != nil {
		s.response(w, errCode.PwdFormatError.SetKey(check))
		return
	}
	account := strings.ToLower(p["account"][0])
	user, login := userStorage.NewUser(account,
		p["password"][0], p["platform"][0], p["channel"][0], p["uuid"][0], p["uuid_web"][0], utils.GetIP(r))

	inviteCode := ""
	if code, ok := p["inviteCode"]; ok {
		inviteCode = code[0]
	}
	res := loginImpl.Register(s.App, user, login, inviteCode)
	//log.Info("register request PostForm: %s ", p)
	userStorage.NewLoginLog(r, *login)
	s.response(w, res)
}
func (s *UserController) smsSend(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	p := r.Form
	if check, ok := utils.CheckParams(p, []string{"phone", "area", "event"}); ok != nil {
		s.response(w, errCode.ErrParams.SetKey(check))
		return
	}
	if len(p["phone"][0]) != 9 && len(p["phone"][0]) != 10 {
		s.response(w, errCode.PhoneFormatErr.SetKey())
		return
	}
	area, _ := utils.ConvertInt(p["area"][0])
	phone, length := utils.ParsePhone(p["phone"][0])
	if length != 9 {
		s.response(w, errCode.PhoneFormatErr.SetKey())
		return
	}
	event := p["event"][0]
	if event != lobbyStorage.EventBind {
		s.response(w, errCode.ErrParams.SetKey("event"))
		return
	}
	if user := userStorage.QueryUser(bson.M{"Phone": phone, "Area": area}); user != nil {
		s.response(w, errCode.PhoneAlreadyBind.SetKey())
		return
	}
	smsCodeExpireSecond := int64(s.App.GetSettings().Settings["smsCodeExpireSecond"].(float64))
	sms := lobbyStorage.QuerySms(area, phone, event)
	if sms != nil {
		now := time.Now().Unix()
		waite := now - sms.CreateAt.Unix()
		if waite < smsCodeExpireSecond {
			s.response(w, errCode.SmsSentTooFast.SetKey(waite))
			return
		}
	}
	r1 := rand.New(rand.NewSource(time.Now().UnixNano()))
	newCode := int(utils.RandomNum(5, r1))
	newSms := &lobbyStorage.Sms{
		Area:     area,
		Phone:    phone,
		Code:     strconv.Itoa(newCode),
		Event:    lobbyStorage.EventBind,
		CreateAt: utils.Now(),
	}
	loginImpl.SendSmsCode(newSms)
	s.response(w, errCode.Success(nil).SetKey())
	return
}

func (s *UserController) apiLogin(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	token := params.Get("Token")
	gameType := params.Get("GameType")
	action := params.Get("Topic")
	driveType := params.Get("DriveType")
	productType := params.Get("ProductType")
	tokenObj := userStorage.QueryToken(token)
	if tokenObj == nil {
		s.response(w, errCode.Success(map[string]interface{}{"err": "user not login"}).SetKey())
		return
	}
	user := userStorage.QueryUserId(tokenObj.Oid)
	if user.Type != userStorage.TypeNormal {
		s.response(w, errCode.Success(map[string]interface{}{"err": "user type does not allow login"}).SetKey())
		return
	}

	data := map[string]interface{}{
		"token":       token,
		"gameType":    gameType,
		"action":      action,
		"driveType":   driveType,
		"uid":         tokenObj.Oid.Hex(),
		"productType": productType,
	}
	ctx, _ := context.WithTimeout(context.TODO(), time.Second*3)
	res, err := mqrpc.String(
		s.App.Call(
			ctx,
			gameType,                   //要访问的moduleType
			fmt.Sprintf("/%s", action), //访问模块中handler路径
			mqrpc.Param(data),
		),
	)

	if err != nil {
		log.Error("apilogin rpc err：%s ", err.Error())
		s.response(w, errCode.Success(map[string]interface{}{"err": err.Error()}).SetKey())
		return
	}
	log.Debug("Redirect url:%s", res)
	http.Redirect(w, r, res, http.StatusTemporaryRedirect)
}

func (s *UserController) apiLoginV1(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	token := params.Get("Token")
	gameType := params.Get("GameType")
	action := params.Get("Topic")
	params.Del("Token")
	params.Del("GameType")
	params.Del("Topic")
	tokenObj := userStorage.QueryToken(token)
	if tokenObj == nil {
		s.response(w, errCode.Success(map[string]interface{}{"err": "user not login"}).SetKey())
		return
	}
	user := userStorage.QueryUserId(tokenObj.Oid)
	if user.Type != userStorage.TypeNormal {
		s.response(w, errCode.Success(map[string]interface{}{"err": "user type does not allow login"}).SetKey())
		return
	}

	data := map[string]interface{}{
		"token":  token,
		"uid":    tokenObj.Oid.Hex(),
		"params": params,
	}
	ctx, _ := context.WithTimeout(context.TODO(), time.Second*3)
	res, err := mqrpc.String(
		s.App.Call(
			ctx,
			gameType,                   //要访问的moduleType
			fmt.Sprintf("/%s", action), //访问模块中handler路径
			mqrpc.Param(data),
		),
	)

	if err != nil {
		log.Error("apiLoginV1 rpc err：%s ", err.Error())
		s.response(w, errCode.Success(map[string]interface{}{"err": err.Error()}).SetKey())
		return
	}
	log.Debug("apiLoginV1 Redirect url:%s", res)
	http.Redirect(w, r, res, http.StatusTemporaryRedirect)
}
