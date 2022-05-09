package lobbyImpl

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/gate"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	mqrpc "vn/framework/mqant/rpc"
	"vn/game"
	"vn/game/activity"
	pk "vn/game/mini/poker"
	gate2 "vn/gate"
	"vn/storage"
	"vn/storage/activityStorage"
	"vn/storage/agentStorage"
	"vn/storage/gameStorage"
	"vn/storage/gbsStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/slotStorage/slotCsStorage"
	"vn/storage/slotStorage/slotLsStorage"
	"vn/storage/slotStorage/slotSexStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
	"vn/storage/yxxStorage"
)

func TcpLogin(app module.App, token string, a gate.Session) (map[string]interface{}, string, error) {
	tokenObj := userStorage.QueryToken(token)
	if tokenObj == nil {
		log.Info("token not found: %s", token)
		return nil, "", errCode.Forbidden.GetError()
	}
	uid := tokenObj.Oid.Hex()
	if a.GetSessionID() != tokenObj.SessionId {
		sessionBean := gate2.QuerySessionBean(uid)
		if sessionBean != nil {
			session, err := basegate.NewSession(app, sessionBean.Session)
			if err != nil {
				log.Error(err.Error())
			} else {
				if err := session.SendNR(game.Push, getAccountChangedResponse()); err != "" {
					log.Error(err)
				}
			}
		}
	}
	//if ip != tokenObj.Ip{
	//	return errCode.Illegal.GetError()
	//}
	tokenObj.SessionId = a.GetSessionID()
	tokenObj.UpdateTime = utils.Now()
	if err := userStorage.UpsertToken(tokenObj); err != nil {
		return nil, "", err.GetError()
	}
	login := userStorage.QueryLogin(tokenObj.Oid)
	if login == nil{
		return nil, "", errCode.Forbidden.GetError()
	}
	login.LastTime = utils.Now()
	login.LastIp = tokenObj.Ip
	userStorage.UpsertLogin(login)
	a.Bind(uid)
	return onLogin(app, uid), uid, nil
}

func onLogin(app module.App, uid string) map[string]interface{} {
	listeners := common.QueryListener(common.EventLogin)
	res := make(map[string]interface{})
	for _, listener := range *listeners {
		ctx, _ := context.WithTimeout(context.TODO(), time.Second*3)
		result, err := app.Call(ctx, listener.ServerId, listener.ServerRegister, mqrpc.Param(uid))
		if err != "" {
			log.Error("call onLogin error: %s,server:%s", err, listener.ServerId)
		}
		if result != nil {
			re := result.(map[string]interface{})
			for k, v := range re {
				res[k] = v
			}
		}
	}
	return res
}

func GetLobbyInfo(uid primitive.ObjectID, keys []string) map[string]interface{} {
	res := make(map[string]interface{}, 3)
	if utils.IsContainStr(keys, "customerService") {
		res["customerService"] = storage.QueryCustomerInfo()
	}
	if utils.IsContainStr(keys, "sumGrade") {
		sumGrade := make(map[string]*lobbyStorage.SumGrade, 3)
		sumGrade["sumGradeYxx"] = getSumGrade(uid, game.YuXiaXie)
		sumGrade["sumGradeDx"] = getSumGrade(uid, game.BiDaXiao)
		res["sumGrade"] = sumGrade
	}
	if utils.IsContainStr(keys, "bannerArray") {
		res["bannerArray"] = getBanners()
	}
	if utils.IsContainStr(keys, "lobbyNotice") {
		res["lobbyNotice"] = storage.QueryLobbyNotice()
	}
	if utils.IsContainStr(keys, "mailUnreadNum") {
		//user := userStorage.QueryUserId(utils.ConvertOID(uid.Hex()))
		res["mailUnreadNum"] = lobbyStorage.QueryLobbyBubble(uid.Hex(), lobbyStorage.Mail).Num //gameStorage.QueryMailUnreadNum(user.Account, gameStorage.MailAll)
	}
	if utils.IsContainStr(keys, "yxxPrizePool") {
		tableInfo := yxxStorage.GetTableInfo("000000")
		res["yxxPrizePool"] = tableInfo.PrizePool
	}
	if utils.IsContainStr(keys, "slotLsPrizePool") {
		goldJackpot, silverJackpot := slotLsStorage.GetJackpot()
		info := make(map[string][]int64, 2)
		info["GoldJackpot"] = goldJackpot
		info["SilverJackpot"] = silverJackpot
		res["slotLsPrizePool"] = info
	}
	if utils.IsContainStr(keys, "slotCsPrizePool") {
		jackpot := slotCsStorage.GetJackpot()
		info := make(map[string][]int64, 2)
		info["Jackpot"] = jackpot
		res["slotCsPrizePool"] = info
	}
	if utils.IsContainStr(keys, "slotSexPrizePool") {
		jackpot := slotSexStorage.GetJackpot()
		info := make(map[string][]int64, 2)
		info["Jackpot"] = jackpot
		res["slotSexPrizePool"] = info
	}
	if utils.IsContainStr(keys, "dxInfo") {
		res["dxInfo"] = getDxInfo()
	}
	if utils.IsContainStr(keys, "reconnectInfo") {
		serverID := gameStorage.QueryGameReconnect(uid.Hex())
		info := make(map[string]string, 1)
		info["serverID"] = serverID
		res["reconnectInfo"] = info
	}
	if utils.IsContainStr(keys, "gbsPoolInfo") {
		res["gbsPoolInfo"] = gbsStorage.GetGameConf()
	}
	if utils.IsContainStr(keys, "miniPokerPool") {
		res["miniPokerPool"] = pk.GetPrizePool()
	}
	if utils.IsContainStr(keys, "lobbyGameLayout") {
		res["lobbyGameLayout"] = lobbyStorage.QueryLobbyGameLayout()
	}
	if utils.IsContainStr(keys, "firstChargeStatus") {
		res["firstChargeStatus"] = activity.QueryHaveFirstCharge(uid)
	}
	if utils.IsContainStr(keys, "normalActivityNum") {
		//activity.RefreshNormalActivity(uid)
		res["normalActivityNum"] = lobbyStorage.QueryLobbyBubble(uid.Hex(), lobbyStorage.NormalActivity).Num //activity.GetNormalActivityNum(uid)
	}
	if utils.IsContainStr(keys, "dayActivityNum") {
		//activity.RefreshDayActivity(uid)
		res["dayActivityNum"] = lobbyStorage.QueryLobbyBubble(uid.Hex(), lobbyStorage.DayActivity).Num //activity.GetDayActivityNum(uid)
	}
	if utils.IsContainStr(keys, "vipActivityNum") {
		//activity.RefreshDayActivity(uid)
		res["vipActivityNum"] = lobbyStorage.QueryLobbyBubble(uid.Hex(), lobbyStorage.VipActivity).Num //activity.GetDayActivityNum(uid)
	}
	if utils.IsContainStr(keys, "vipLevel") {
		res["vipLevel"] = userStorage.QueryUserInfo(uid).VipLevel
	}
	return res
}
func getDxInfo() interface{} {
	ctx, _ := context.WithTimeout(context.TODO(), time.Second*3)
	result, err := common.App.Call(ctx, string(game.BiDaXiao), "/dx/dxInfo", mqrpc.Param())
	if err == "" {
		return result
	}
	log.Error(err)
	return map[string]interface{}{}
}
func getSumGrade(uid primitive.ObjectID, gameType game.Type) *lobbyStorage.SumGrade {
	sumGrade := lobbyStorage.QuerySumGrade(uid, gameType)
	if sumGrade == nil {
		sumGrade = lobbyStorage.NewSumGrade(uid, gameType)
		rank, _ := lobbyStorage.GetWinRank(sumGrade)
		sumGrade.WinRank = rank
	}
	return sumGrade
}

func BindPhone(uid primitive.ObjectID, area int64, phone int64) int64 {
	user := userStorage.QueryUser(bson.M{"_id": uid})
	var amount int64 = 0
	if user.Area == 0 && user.Phone == 0 {
		user.Area = area
		user.Phone = phone
		user.UpdateAt = utils.Now()
		userStorage.UpdateUser(*user)
		//发奖
		common.ExecQueueFunc(func() {
			amount = getRewardAmount()
			bill := walletStorage.NewBill(uid.Hex(), walletStorage.TypeIncome,
				walletStorage.EventBindPhone, "", amount)
			if err := walletStorage.OperateVndBalance(bill); err == nil {
				walletStorage.NotifyUserWallet(uid.Hex())
				agentStorage.OnActivityData(uid.Hex(), amount)
				activityRewardBetTimes, _ := utils.ConvertInt(storage.QueryConf(storage.KActivityRewardBetTimes))
				douDouBet := amount * activityRewardBetTimes
				userStorage.IncUserDouDouBet(uid, douDouBet)
				gameStorage.IncProfitByUser(uid.Hex(), 0, amount, -amount, 0)
				activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
					Type:       activityStorage.BindPhone,
					ActivityID: "",
					Uid:        uid.Hex(),
					Charge:     0,
					Get:        amount,
					BetTimes:   activityRewardBetTimes,
					UpdateAt:   time.Now(),
					CreateAt:   time.Now(),
				})
				activity.NotifyNormalActivityNum(uid.Hex())
			}
		})
	}
	return amount
}
func getRewardAmount() int64 {
	bindPhoneConf := storage.QueryConf(storage.KBindPhoneReward).(string)
	conf := strings.Split(bindPhoneConf, ",")
	max, _ := utils.ConvertInt(conf[0])
	mini, _ := utils.ConvertInt(conf[1])
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	reward := utils.RandInt64(mini, max,r) * 100
	return reward
}

func GetInviteUrl(uid string) string {
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	inviteCode := utils.Base58encode(int(user.ShowId))
	inviteUrl := ""
	agent := agentStorage.QueryAgent(utils.ConvertOID(uid))
	cdnHost := storage.QueryConf(storage.KCdnHost).(string)
	cdnHost2 := storage.QueryConf(storage.KCdnHost2).(string)
	channel := user.Channel
	if !strings.Contains(user.Channel, "_i") {
		channel = fmt.Sprintf("%s_i", user.Channel)
	}
	if agent != nil {
		if agent.Theme == agentStorage.ThemeCustom {
			imgUrl := url.QueryEscape(agent.InviteImg)
			inviteUrl = fmt.Sprintf("%s/invite_c.html?invite=%s&c=%s&img=%s",
				cdnHost2, inviteCode, channel, imgUrl)
		}
	}
	if inviteUrl == "" {
		inviteUrl = fmt.Sprintf("%s/invite.html?invite=%s&c=%s",
			cdnHost2, inviteCode, channel)
	}
	tmp := url.QueryEscape(inviteUrl)
	jumpUrl := fmt.Sprintf("%s/j.html?url=%s", cdnHost, tmp)
	return jumpUrl
}
func getAccountChangedResponse() []byte {
	res := errCode.AccountChanged.SetAction("HD_login").GetI18nMap()
	res["GameType"] = game.Lobby
	b, _ := json.Marshal(res)
	return b
}
