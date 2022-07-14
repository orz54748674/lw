package loginImpl

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	"vn/storage"
	"vn/storage/agentStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

func Register(app module.App, user *userStorage.User, login *userStorage.Login, inviteCode string) *common.Err {
	lastUid := userStorage.NewUserGlobalId()
	user.ShowId = getUserShowId(lastUid)
	user.NickName = "" //fmt.Sprintf("%s%d","user",user.ShowId)
	newUser, err := userStorage.InsertUser(user)
	if err != nil {
		return err
	}
	login.Oid = newUser.Oid
	if err := userStorage.UpsertLogin(login); err != nil {
		return err
	}
	token := userStorage.NewToken(newUser.Oid, login.LastIp)
	log.Info("new user token:%v ,uid: %s", token.AccessToken, newUser.Oid.Hex())
	if err := userStorage.UpsertToken(token); err != nil {
		return err
	}
	hasInvite := false
	if inviteCode != "" {
		parentShowId := utils.Base58decode(inviteCode)
		parent := userStorage.QueryUser(bson.M{"ShowId": parentShowId})
		if parent != nil {
			hasInvite = true
			agentStorage.InsertInvite(token.Oid, parent.Oid)
		}
	}
	if !hasInvite {
		var zero primitive.ObjectID
		agentStorage.InsertInvite(token.Oid, zero)
	}
	newUserGet := int64(app.GetSettings().Settings["newUserBalance"].(float64))
	wallet := walletStorage.GetWallet(newUser.Oid)
	if newUserGet > 0 {
		wallet.VndBalance = newUserGet
		walletStorage.UpsertWallet(wallet)
	}
	agentStorage.OnRegister(newUser.Oid.Hex())
	result := map[string]interface{}{"user": newUser, "token": token, "tcpInfo": getTcpInfo(app)}
	return errCode.Success(result)
}
func LoginAccount(app module.App, account string, password string,
	platform string, uuid string, uuidWeb string, r *http.Request) *common.Err {
	user := userStorage.QueryUser(bson.M{"Account": account})
	if user == nil {
		return errCode.AccountNotExist.SetKey()
	}
	login := userStorage.QueryLogin(user.Oid)
	if login == nil {
		return errCode.Forbidden.SetKey()
	}
	if password != login.Password {
		return errCode.PasswordErr.SetKey()
	}
	ip := utils.GetIP(r)
	if user.Status != userStorage.StatusNormal || userStorage.DeviceIsBlack(login.Uuid) || userStorage.IpIsBlack(ip) {
		return errCode.ConnectCustomerService.SetKey()
	}
	login.LastPlatform = platform
	login.LastIp = ip
	login.Uuid = uuid
	login.UuidWeb = uuidWeb
	//login.LastTime = utils.Now()
	if err := userStorage.UpsertLogin(login); err != nil {
		return err
	}
	//oldToken := userStorage.QueryTokenByUid(user.Oid)
	//if oldToken != nil && oldToken.SessionId != ""{
	//	sessionBean := gate2.QuerySessionId(oldToken.SessionId)
	//	if sessionBean != nil{
	//		session,err := basegate.NewSession(app, sessionBean.Session)
	//		if err != nil{
	//			log.Error(err.Error())
	//		}else{
	//			if err := session.SendNR(game.Push, getAccountChangedResponse());err != ""{
	//				log.Error(err)
	//			}
	//		}
	//	}
	//}
	token := userStorage.NewToken(user.Oid, ip)
	if err := userStorage.UpsertToken(token); err != nil {
		return err
	}
	userStorage.NewLoginLog(r, *login)
	result := map[string]interface{}{"user": user, "token": token, "tcpInfo": getTcpInfo(app)}
	return errCode.Success(result).SetKey()
}
func getUserShowId(lastUid int64) int64 {
	showId := generateId(lastUid)
	if user := userStorage.QueryUser(bson.M{"ShowId": showId}); user != nil {
		return getUserShowId(lastUid)
	}
	return showId
}
func generateId(lastUid int64) int64 {
	l := len(strconv.Itoa(int(lastUid)))
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return utils.RandomNum(l+5, r)
}
func getTcpInfo(app module.App) *TcpInfo {
	//tcpPort := int(s.App.GetSettings().Settings["tcpPort"].(float64))
	tcpHost := storage.QueryConf(storage.KTcpHost).(string)
	wssPort := int(app.GetSettings().Settings["wssPort"].(float64))
	return &TcpInfo{
		TcpHost: tcpHost,
		WssPort: wssPort,
	}
}

type TcpInfo struct {
	TcpHost string
	WssPort int
}

func getAccountChangedResponse() []byte {
	res := errCode.AccountChanged.SetAction("HD_login").GetI18nMap()
	res["GameType"] = game.Lobby
	b, _ := json.Marshal(res)
	return b
}
