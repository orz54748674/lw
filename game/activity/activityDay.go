package activity

import (
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/game"
	"vn/storage/activityStorage"
	"vn/storage/agentStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
)

func dealDayGameActivity(uid string, gameType game.Type) {
	have := false
	dayGameConf := activityStorage.QueryActivityDayGameConf()
	for _, v := range dayGameConf {
		if v.GameType == gameType {
			have = true
			break
		}
	}
	if !have {
		return
	}
	_, change := RefreshDayGameActivity(utils.ConvertOID(uid))
	if !change {
		return
	}
	NotifyDayActivityNum(uid)
}
func dealDayInviteActivity(uid string) {
	invite := agentStorage.QueryInvite(utils.ConvertOID(uid))
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	if user.Type != userStorage.TypeNormal { //陪玩号不计算
		return
	}
	if invite.Oid.IsZero() {
		return
	}
	agentUid := invite.ParentOid
	_, change := RefreshDayInviteActivity(agentUid)
	if !change {
		return
	}
	NotifyDayActivityNum(agentUid.Hex())
}
func RefreshDayActivity(uid primitive.ObjectID) {
	RefreshDayChargeActivity(uid)
	RefreshDayGameActivity(uid)
	RefreshDayInviteActivity(uid)
	NotifyDayActivityNum(uid.Hex())
}
func GetDayActivityNum(uid primitive.ObjectID) int {
	num := 0
	for _, v := range activityStorage.ActivityDayList {
		num += QueryHaveReceiveActivity(uid.Hex(), v)
	}
	return num
}
func RefreshDayChargeActivity(uid primitive.ObjectID) []activityStorage.ActivityDayCharge {
	if !activityStorage.QueryActivityIsOpen(activityStorage.DayCharge) {
		return []activityStorage.ActivityDayCharge{}
	}
	totalCharge := payStorage.QueryTodayChargeByUid(uid)
	resActivity := make([]activityStorage.ActivityDayCharge, 0)
	dayChargeConf := activityStorage.QueryActivityDayChargeConf()
	for _, v := range dayChargeConf {
		activity := activityStorage.QueryTodayDayCharge(uid.Hex(), v.Oid.Hex())
		if activity == nil {
			status := activityStorage.Undo
			if totalCharge > 0 && totalCharge >= v.TotalCharge {
				status = activityStorage.Done
			}
			newActivity := &activityStorage.ActivityDayCharge{
				ActivityID: v.Oid.Hex(),
				Uid:        uid.Hex(),
				CurCharge:  totalCharge,
				Charge:     v.TotalCharge,
				Get:        v.Get,
				GetPoints:        v.GetPoints,
				BetTimes:   v.BetTimes,
				Status:     status,
				CreateAt:   utils.Now(),
				UpdateAt:   utils.Now(),
			}
			activityStorage.UpsertActivityDayCharge(newActivity)
			resActivity = append(resActivity, *newActivity)
		} else {
			if totalCharge > 0 && totalCharge >= v.TotalCharge && activity.Status == activityStorage.Undo {
				activity.Status = activityStorage.Done
				activity.UpdateAt = utils.Now()
				activityStorage.UpsertActivityDayCharge(activity)
			}
			if activity.CurCharge != totalCharge {
				activity.UpdateAt = utils.Now()
				activity.CurCharge = totalCharge
				activityStorage.UpsertActivityDayCharge(activity)
			}

			resActivity = append(resActivity, *activity)
		}
	}
	return resActivity
}
func RefreshDayGameActivity(uid primitive.ObjectID) ([]activityStorage.ActivityDayGame, bool) {
	if !activityStorage.QueryActivityIsOpen(activityStorage.DayGame) {
		return []activityStorage.ActivityDayGame{}, false
	}
	resActivity := make([]activityStorage.ActivityDayGame, 0)
	change := false
	dayGameConf := activityStorage.QueryActivityDayGameConf()
	for _, v := range dayGameConf {
		betNum := activityStorage.QueryTodayBetRecordTotal(uid.Hex(), v.GameType)
		activity := activityStorage.QueryTodayDayGame(uid.Hex(), v.Oid.Hex())
		if activity == nil {
			status := activityStorage.Undo
			if betNum > 0 && betNum >= v.NeedBet {
				status = activityStorage.Done
				change = true
			}
			newActivity := &activityStorage.ActivityDayGame{
				ActivityID: v.Oid.Hex(),
				Uid:        uid.Hex(),
				GameType:   v.GameType,
				CurBet:     betNum,
				NeedBet:    v.NeedBet,
				Get:        v.Get,
				GetPoints:        v.GetPoints,
				BetTimes:   v.BetTimes,
				Status:     status,
				CreateAt:   utils.Now(),
				UpdateAt:   utils.Now(),
			}
			activityStorage.UpsertActivityDayGame(newActivity)
			resActivity = append(resActivity, *newActivity)
		} else {
			if betNum > 0 && betNum >= v.NeedBet && activity.Status == activityStorage.Undo {
				activity.Status = activityStorage.Done
				activity.UpdateAt = utils.Now()
				activityStorage.UpsertActivityDayGame(activity)
				change = true
			}
			if activity.CurBet != betNum {
				activity.UpdateAt = utils.Now()
				activity.CurBet = betNum
				activityStorage.UpsertActivityDayGame(activity)
			}
			resActivity = append(resActivity, *activity)
		}
	}
	return resActivity, change
}
func RefreshDayInviteActivity(uid primitive.ObjectID) ([]activityStorage.ActivityDayInvite, bool) {
	if !activityStorage.QueryActivityIsOpen(activityStorage.DayInvite) {
		return []activityStorage.ActivityDayInvite{}, false
	}
	resActivity := make([]activityStorage.ActivityDayInvite, 0)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	inviteData := agentStorage.QueryAgentInviteDataByDate(uid, thatTime)
	uids := convertOid2Str(inviteData.Users)
	count := activityStorage.QueryBetRecordByUsers(uids, thatTime)
	change := false
	dayInviteConf := activityStorage.QueryActivityDayInviteConf()
	for _, v := range dayInviteConf {
		activity := activityStorage.QueryTodayDayInvite(uid.Hex(), v.Oid.Hex())
		if activity == nil {
			status := activityStorage.Undo
			if count > 0 && count >= v.InviteNum {
				status = activityStorage.Done
				change = true
			}
			newActivity := &activityStorage.ActivityDayInvite{
				ActivityID: v.Oid.Hex(),
				Uid:        uid.Hex(),
				CurInvite:  count,
				InviteNum:  v.InviteNum,
				Get:        v.Get,
				GetPoints:        v.GetPoints,
				BetTimes:   v.BetTimes,
				Status:     status,
				CreateAt:   utils.Now(),
				UpdateAt:   utils.Now(),
			}
			activityStorage.UpsertActivityDayInvite(newActivity)
			resActivity = append(resActivity, *newActivity)
		} else {
			if count > 0 && count >= v.InviteNum && activity.Status == activityStorage.Undo {
				activity.Status = activityStorage.Done
				activity.CurInvite = count
				activity.UpdateAt = utils.Now()
				activityStorage.UpsertActivityDayInvite(activity)
				change = true
			}
			if activity.InviteNum != count {
				activity.UpdateAt = utils.Now()
				activity.CurInvite = count
				activityStorage.UpsertActivityDayInvite(activity)
			}
			resActivity = append(resActivity, *activity)
		}
	}
	return resActivity, change
}