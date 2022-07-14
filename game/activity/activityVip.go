package activity

import (
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/game"
	gate2 "vn/gate"
	"vn/storage/activityStorage"
	"vn/storage/agentStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

func dealVipUpGradeByUid(uid string) { //处理是否升级VIP
	uOid := utils.ConvertOID(uid)
	userInfo := userStorage.QueryUserInfo(uOid)
	vipConf := activityStorage.QueryActivityVipConfByType(activityStorage.ChargeNeed)
	aUserInfo := activityStorage.QueryActivityUserInfo(uid, activityStorage.Vip)
	curLevel := userInfo.VipLevel
	levelSwitchVipTable := []int64{
		vipConf.Vip0, vipConf.Vip1, vipConf.Vip2, vipConf.Vip3, vipConf.Vip4, vipConf.Vip5, vipConf.Vip6, vipConf.Vip7, vipConf.Vip8, vipConf.Vip9,
	}
	for i := len(levelSwitchVipTable) - 1; i >= 1; i-- {
		if curLevel >= i { //最高了
			return
		}
		if aUserInfo.SumCharge >= levelSwitchVipTable[i] {
			userStorage.SetUserInfoVipLevel(uOid, i) //更新VIP等级
			if userInfo.SafeStatus == 0 {            //保险箱状态
				userStorage.SetUserInfoSafeStatus(uOid, 1)
			}
			activityStorage.SetActivityWeekHaveUpGrade(uid, activityStorage.Vip, true) //本周升过级置为true
			activityStorage.SetActivityUserCharge(uid, activityStorage.VipWeek, 0)     //每周彩金的充值要求置为0
			start := curLevel
			if curLevel == 0 {
				start = 1
			}
			for j := start; j <= i; j++ { //更新前面所有礼包的状态
				vipRecord := activityStorage.QueryVipByLevel(uid, j)
				if vipRecord == nil || vipRecord.Status == activityStorage.Undo { //没有或者未完成
					getGold := activityStorage.QueryActivityVipConfByType(activityStorage.VipGetGold)
					getPoints := activityStorage.QueryActivityVipConfByType(activityStorage.VipGetPoints)
					levelSwitchGoldTable := []int64{
						vipConf.Vip0, getGold.Vip1, getGold.Vip2, getGold.Vip3, getGold.Vip4, getGold.Vip5, getGold.Vip6, getGold.Vip7, getGold.Vip8, getGold.Vip9,
					}
					levelSwitchPointsTable := []int64{
						vipConf.Vip0, getPoints.Vip1, getPoints.Vip2, getPoints.Vip3, getPoints.Vip4, getPoints.Vip5, getPoints.Vip6, getPoints.Vip7, getPoints.Vip8, getPoints.Vip9,
					}
					if levelSwitchGoldTable[j] > 0 || levelSwitchPointsTable[j] > 0 {
						vipRecord = &activityStorage.ActivityVip{
							ActivityID: getGold.Oid.Hex(),
							Uid:        uid,
							Level:      j,
							GetGold:    levelSwitchGoldTable[j],
							GetPoints:  float64(levelSwitchPointsTable[j]),
							BetTimes:   getGold.BetTimes,
							Status:     activityStorage.Done,
							UpdateAt:   utils.Now(),
							CreateAt:   utils.Now(),
						}
						activityStorage.UpsertActivityVip(vipRecord)
					}
				}
			}
			NotifyVipActivityNum(uid)
			//通知客户端升级成功
			sb := gate2.QuerySessionBean(uid)
			if sb != nil {
				msg := make(map[string]interface{})
				msg["Data"] = i
				msg["Action"] = actionUpGradeSuccess
				msg["GameType"] = game.Lobby
				protocol.SendPack(uid, game.Push, msg)
			}
			return
		}
	}
}
func dealVipDownGradeAllUid() { //处理降级
	allUserInfo := userStorage.QueryAllUserInfo(bson.M{"VipLevel": bson.M{"$gt": 0}})
	vipConf := activityStorage.QueryActivityVipConfByType(activityStorage.KeepGradeNeed)
	levelSwitchVipTable := []int64{
		vipConf.Vip0, vipConf.Vip1, vipConf.Vip2, vipConf.Vip3, vipConf.Vip4, vipConf.Vip5, vipConf.Vip6, vipConf.Vip7, vipConf.Vip8, vipConf.Vip9,
	}
	chargeConf := activityStorage.QueryActivityVipConfByType(activityStorage.ChargeNeed)
	chargeSwitchVipTable := []int64{
		chargeConf.Vip0, chargeConf.Vip1, chargeConf.Vip2, chargeConf.Vip3, chargeConf.Vip4, chargeConf.Vip5, chargeConf.Vip6, chargeConf.Vip7, chargeConf.Vip8, chargeConf.Vip9,
	}
	for _, v := range allUserInfo {
		info := activityStorage.QueryActivityUserInfo(v.Oid.Hex(), activityStorage.Vip)
		if v.VipLevel > 0 {
			if !info.WeekHaveUpGrade && info.WeekBets < levelSwitchVipTable[v.VipLevel] { //没升过级且没达到保级要求 就会降级
				userStorage.SetUserInfoVipLevel(v.Oid, v.VipLevel-1)                                                        //降一级
				activityStorage.SetActivityUserCharge(v.Oid.Hex(), activityStorage.Vip, chargeSwitchVipTable[v.VipLevel-1]) //累计充值置为降级的起始值
				vipRecord := activityStorage.QueryVipByLevel(v.Oid.Hex(), v.VipLevel)
				if vipRecord.Status == activityStorage.Done { //未领取的重置为未完成
					vipRecord.Status = activityStorage.Undo
					activityStorage.UpsertActivityVip(vipRecord)
					NotifyVipActivityNum(v.Oid.Hex())
				}
			}
		}
		if info.WeekHaveUpGrade {
			activityStorage.SetActivityWeekHaveUpGrade(v.Oid.Hex(), activityStorage.Vip, false) //本周升过级置为false
		}
		activityStorage.ResetActivityWeekBets(v.Oid.Hex(), activityStorage.Vip) //重置本周流水
	}
}
func dealVipWeekByUid(uid string) { //处理每周彩金
	uOid := utils.ConvertOID(uid)
	userInfo := userStorage.QueryUserInfo(uOid)
	vipConf := activityStorage.QueryActivityVipConfByType(activityStorage.WeekGet)
	aUserInfo := activityStorage.QueryActivityUserInfo(uid, activityStorage.VipWeek)
	curLevel := userInfo.VipLevel
	levelSwitchVipWeekTable := []int64{
		vipConf.Vip0, vipConf.Vip1, vipConf.Vip2, vipConf.Vip3, vipConf.Vip4, vipConf.Vip5, vipConf.Vip6, vipConf.Vip7, vipConf.Vip8, vipConf.Vip9,
	}

	if aUserInfo.SumCharge >= levelSwitchVipWeekTable[curLevel] && levelSwitchVipWeekTable[curLevel] > 0 { //达到该vip每周彩金要求
		vipRecord := activityStorage.QueryVipWeekByLevel(uid, curLevel)
		if vipRecord == nil || vipRecord.Status == activityStorage.Undo { //没有或者未完成
			vipRecord = &activityStorage.ActivityVipWeek{
				ActivityID: vipConf.Oid.Hex(),
				Uid:        uid,
				Level:      curLevel,
				GetGold:    levelSwitchVipWeekTable[curLevel],
				BetTimes:   vipConf.BetTimes,
				Status:     activityStorage.Done,
				UpdateAt:   utils.Now(),
				CreateAt:   utils.Now(),
			}
			activityStorage.UpsertActivityVipWeek(vipRecord)
			NotifyVipActivityNum(uid)
		}
	}
}
func dealVipChargeGiftByUid(order *payStorage.Order) { //处理充值优惠
	uOid := order.UserId
	uid := uOid.Hex()
	userInfo := userStorage.QueryUserInfo(uOid)
	vipConf := activityStorage.QueryActivityVipConfByType(activityStorage.ChargeGetPerThousand)
	curLevel := userInfo.VipLevel
	levelSwitchVipWeekTable := []int64{
		vipConf.Vip0, vipConf.Vip1, vipConf.Vip2, vipConf.Vip3, vipConf.Vip4, vipConf.Vip5, vipConf.Vip6, vipConf.Vip7, vipConf.Vip8, vipConf.Vip9,
	}
	if levelSwitchVipWeekTable[curLevel] > 0 {
		get := levelSwitchVipWeekTable[curLevel] * order.GotAmount / 1000
		douDouBet := get * vipConf.BetTimes
		userStorage.IncUserDouDouBet(uOid, douDouBet)
		bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventVipChargeGift, uid+"_"+vipConf.Oid.Hex(), get)
		walletStorage.OperateVndBalance(bill)
		notifyWallet(uid)
		activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
			Type:       activityStorage.VipChargeGet,
			ActivityID: vipConf.Oid.Hex(),
			Uid:        uid,
			Charge:     order.GotAmount,
			Get:        get,
			GetPoints:  0,
			BetTimes:   vipConf.BetTimes,
			UpdateAt:   utils.Now(),
			CreateAt:   utils.Now(),
		})
		agentStorage.OnActivityData(uid, get)
	}
}
func dealVipActivityWeekBets(uid string, bets int64) {
	if activityStorage.QueryActivityIsOpen(activityStorage.Vip) { //开启该活动
		activityStorage.IncActivityWeekBets(uid, activityStorage.Vip, bets)
	}
}
