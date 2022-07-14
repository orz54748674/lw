package activity

import (
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/game"
	gate2 "vn/gate"
	"vn/storage/activityStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

func NotifyNormalActivityNum(uid string) {
	sb := gate2.QuerySessionBean(uid)
	if sb == nil {
		return
	}
	data := make(map[string]interface{})
	num := GetNormalActivityNum(utils.ConvertOID(uid))
	data["normalActivityNum"] = num
	ret := protocol.DealProtocolFormat(data, game.Lobby, "HD_info", nil)
	protocol.SendPack(uid, game.Push, ret)

	lobbyStorage.UpsertLobbyBubble(lobbyStorage.LobbyBubble{
		Uid:        uid,
		BubbleType: lobbyStorage.NormalActivity,
		Num:        num,
		UpdateAt:   utils.Now(),
	})
}
func NotifyDayActivityNum(uid string) {

	sb := gate2.QuerySessionBean(uid)
	if sb == nil {
		return
	}
	data := make(map[string]interface{})
	num := GetDayActivityNum(utils.ConvertOID(uid))
	data["dayActivityNum"] = num
	ret := protocol.DealProtocolFormat(data, game.Lobby, "HD_info", nil)
	protocol.SendPack(uid, game.Push, ret)
	lobbyStorage.UpsertLobbyBubble(lobbyStorage.LobbyBubble{
		Uid:        uid,
		BubbleType: lobbyStorage.DayActivity,
		Num:        num,
		UpdateAt:   utils.Now(),
	})

}
func NotifyVipActivityNum(uid string) {

	data := make(map[string]interface{})
	num := GetVipActivityNum(utils.ConvertOID(uid))
	lobbyStorage.UpsertLobbyBubble(lobbyStorage.LobbyBubble{
		Uid:        uid,
		BubbleType: lobbyStorage.VipActivity,
		Num:        num,
		UpdateAt:   utils.Now(),
	})
	data["vipActivityNum"] = num
	ret := protocol.DealProtocolFormat(data, game.Lobby, "HD_info", nil)
	protocol.SendPack(uid, game.Push, ret)
}
func notifyFirstChargeStatus(uid string) {
	data := make(map[string]interface{})
	data["firstChargeStatus"] = QueryHaveFirstCharge(utils.ConvertOID(uid))
	ret := protocol.DealProtocolFormat(data, game.Lobby, "HD_info", nil)
	protocol.SendPack(uid, game.Push, ret)
}
func notifyWallet(uid string) {
	sb := gate2.QuerySessionBean(uid)
	if sb == nil {
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	msg := make(map[string]interface{})
	msg["Wallet"] = wallet
	msg["Action"] = "wallet"
	msg["GameType"] = game.All
	protocol.SendPack(uid, game.Push, msg)
}
func NotifyDealChargeActivity(order *payStorage.Order) {
	dealFirstChargeActivity(order)
	upsertUserInfoActivity(order)
	RefreshTotalChargeActivity(order.UserId)
	RefreshDayChargeActivity(order.UserId)

	NotifyNormalActivityNum(order.UserId.Hex())
	NotifyDayActivityNum(order.UserId.Hex())

	dealTurnTableChargeActivity(order)
}
func NotifyBetActivity(uid string, gameType game.Type, bets int64) {
	dealDayGameActivity(uid, gameType)
	dealDayInviteActivity(uid)
	dealVipActivityWeekBets(uid, bets)
	dealTurnTableBetsActivity(uid, bets)
}
func upsertUserInfoActivity(order *payStorage.Order) {
	if activityStorage.QueryActivityIsOpen(activityStorage.Encouragement) { //开启该活动
		activityStorage.IncActivityUserCharge(order.UserId.Hex(), activityStorage.Encouragement, order.GotAmount)
	}
	if activityStorage.QueryActivityIsOpen(activityStorage.Vip) { //开启该活动
		activityStorage.IncActivityUserCharge(order.UserId.Hex(), activityStorage.Vip, order.GotAmount)
		dealVipUpGradeByUid(order.UserId.Hex()) //升级VIP
		activityStorage.IncActivityUserCharge(order.UserId.Hex(), activityStorage.VipWeek, order.GotAmount)
		dealVipWeekByUid(order.UserId.Hex()) //处理每周彩金
		dealVipChargeGiftByUid(order)        //Vip充值优惠赠送
	}
}

func QueryHaveFirstCharge(uid primitive.ObjectID) activityStorage.ActivityStatus {
	if !activityStorage.QueryActivityIsOpen(activityStorage.FirstCharge) { //未开启首充活动
		return activityStorage.Close
	}
	userFirstCharge := activityStorage.QueryFirstChargeByUid(uid.Hex())
	if len(userFirstCharge) != 0 { //
		return userFirstCharge[0].Status
	}
	conf := activityStorage.QueryActivityFistChargeConf()
	if len(conf) == 0 {
		return activityStorage.Close
	}
	return activityStorage.Undo
}

func convertOid2Str(ids []primitive.ObjectID) []string {
	strArray := make([]string, 0)
	for _, oid := range ids {
		strArray = append(strArray, oid.Hex())
	}
	return strArray
}

func LobbyIsHaveActivityNotify(uid primitive.ObjectID) bool { //true 红点提示  否则没有
	return false
}

//查询是否有可领取的活动
func QueryHaveReceiveActivity(uid string, activityType activityStorage.ActivityType) int {
	if !activityStorage.QueryActivityIsOpen(activityType) {
		return 0
	}
	num := 0
	if activityType == activityStorage.TotalCharge {
		query := activityStorage.QueryTodayTotalChargeByUid(uid)
		for _, v := range query {
			if v.Status == activityStorage.Done {
				num++
			}
		}
	} else if activityType == activityStorage.SignIn {
		query := activityStorage.QuerySignInAll(uid)
		for _, v := range query {
			if v.Status == activityStorage.Done || (v.Day == 1 && v.Status == activityStorage.Undo) {
				num++
			}
		}
	} else if activityType == activityStorage.DayCharge {
		query := activityStorage.QueryTodayDayChargeByUid(uid)
		for _, v := range query {
			if v.Status == activityStorage.Done {
				num++
			}
		}
	} else if activityType == activityStorage.DayGame {
		query := activityStorage.QueryTodayDayGameByUid(uid)
		for _, v := range query {
			if v.Status == activityStorage.Done {
				num++
			}
		}
	} else if activityType == activityStorage.DayInvite {
		query := activityStorage.QueryTodayDayInviteByUid(uid)
		for _, v := range query {
			if v.Status == activityStorage.Done {
				num++
			}
		}
	} else if activityType == activityStorage.BindPhone {
		user := userStorage.QueryUserId(utils.ConvertOID(uid))
		if user.Phone == 0 {
			num++
		}
	} else if activityType == activityStorage.Vip {
		userInfo := userStorage.QueryUserInfo(utils.ConvertOID(uid))
		curLevel := userInfo.VipLevel
		vipRecord := activityStorage.QueryVipWeekByLevel(uid, curLevel)
		if vipRecord != nil && vipRecord.Status == activityStorage.Done {
			num++
		}
		num += len(activityStorage.QueryVipAllDone(uid))
	}

	return num
}
