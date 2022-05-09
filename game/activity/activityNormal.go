package activity

import (
	"strconv"
	"time"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/game"
	gate2 "vn/gate"
	"vn/storage/activityStorage"
	"vn/storage/agentStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

func dealFirstChargeActivity(order *payStorage.Order) {

	if !activityStorage.QueryActivityIsOpen(activityStorage.FirstCharge) { //未开启首充活动
		return
	}
	userFirstCharge := activityStorage.QueryFirstChargeByUid(order.UserId.Hex())
	if len(userFirstCharge) == 0 || userFirstCharge[0].Status != activityStorage.Done { //已经完成首充活动或者没有报名
		return
	}
	conf := activityStorage.QueryActivityFistChargeConf()
	if len(conf) == 0 {
		return
	}
	getAmout := order.GotAmount * conf[0].GetPer / 100
	if getAmout > conf[0].GetMax {
		getAmout = conf[0].GetMax
	}
	bill := walletStorage.NewBill(order.UserId.Hex(), walletStorage.TypeIncome,
		walletStorage.EventFirstCharge, order.Oid.Hex(), getAmout)
	if err := walletStorage.OperateVndBalance(bill); err != nil {
		return
	}
	activityRecord := &activityStorage.ActivityRecord{
		Type:       activityStorage.FirstCharge,
		ActivityID: "",
		Uid:        order.UserId.Hex(),
		Charge:     order.Amount,
		Get:        getAmout,
		BetTimes:   conf[0].BetTimes,
		UpdateAt:   time.Now(),
		CreateAt:   time.Now(),
	}
	betTimes := conf[0].BetTimes
	douDouBet := (order.GotAmount+getAmout)*betTimes - order.GotAmount //后面已经加了一倍流水
	userStorage.IncUserDouDouBet(order.UserId, douDouBet)
	agentStorage.OnActivityData(order.UserId.Hex(), activityRecord.Get)
	activityStorage.InsertActivityReceiveRecord(activityRecord)
	activityStorage.UpsertActivityFirstCharge(&activityStorage.ActivityFirstCharge{
		ActivityID: conf[0].Oid.Hex(),
		Uid:        order.UserId.Hex(),
		GetPer:     conf[0].GetPer,
		GetMax:     conf[0].GetMax,
		BetTimes:   conf[0].BetTimes,
		Status:     activityStorage.Received,
		CreateAt:   utils.Now(),
		UpdateAt:   utils.Now(),
	})
	notifyFirstChargeStatus(order.UserId.Hex())
}
func RefreshNormalActivity(uid primitive.ObjectID) {
	RefreshTotalChargeActivity(uid)
	RefreshSignInActivity(uid)
	NotifyNormalActivityNum(uid.Hex())
}
func GetNormalActivityNum(uid primitive.ObjectID) int {
	num := 0
	for _, v := range activityStorage.ActivityNormalList {
		num += QueryHaveReceiveActivity(uid.Hex(), v)
	}
	return num
}
func GetVipActivityNum(uid primitive.ObjectID) int {
	num := 0
	for _, v := range activityStorage.ActivityVipList {
		num += QueryHaveReceiveActivity(uid.Hex(), v)
	}
	return num
}
func RefreshTotalChargeActivity(uid primitive.ObjectID) ([]activityStorage.ActivityTotalCharge, int64) {
	if !activityStorage.QueryActivityIsOpen(activityStorage.TotalCharge) {
		return []activityStorage.ActivityTotalCharge{}, 0
	}
	totalCharge := payStorage.QueryTodayChargeByUid(uid)
	resActivity := make([]activityStorage.ActivityTotalCharge, 0)
	totalChargeConf := activityStorage.QueryActivityTotalChargeConf()
	for _, v := range totalChargeConf {
		activity := activityStorage.QueryTodayTotalCharge(uid.Hex(), v.Oid.Hex())
		if activity == nil {
			status := activityStorage.Undo
			if totalCharge > 0 && totalCharge >= v.TotalCharge {
				status = activityStorage.Done
			}
			newActivity := &activityStorage.ActivityTotalCharge{
				ActivityID: v.Oid.Hex(),
				Uid:        uid.Hex(),
				Charge:     v.TotalCharge,
				Get:        v.Get,
				BetTimes:   v.BetTimes,
				Status:     status,
				CreateAt:   utils.Now(),
				UpdateAt:   utils.Now(),
			}
			activityStorage.UpsertActivityTotalCharge(newActivity)
			resActivity = append(resActivity, *newActivity)
		} else {
			if totalCharge > 0 && totalCharge >= v.TotalCharge && activity.Status == activityStorage.Undo {
				activity.Status = activityStorage.Done
				activity.UpdateAt = utils.Now()
				activityStorage.UpsertActivityTotalCharge(activity)
			}
			resActivity = append(resActivity, *activity)
		}
	}
	return resActivity, totalCharge
}
func RefreshSignInActivity(uid primitive.ObjectID) []activityStorage.ActivitySignIn {
	if !activityStorage.QueryActivityIsOpen(activityStorage.SignIn) {
		return []activityStorage.ActivitySignIn{}
	}
Again:
	resActivity := make([]activityStorage.ActivitySignIn, 0)
	activity := activityStorage.QuerySignInAll(uid.Hex())
	receiveCnt := 0
	doneCnt := 0
	undoCnt := 0
	haveDoneOrReceive := false //是否有完成或者领取
	for _, v := range activity {
		if v.Status == activityStorage.Received {
			receiveCnt++
		} else if v.Status == activityStorage.Done {
			doneCnt++
		} else if v.Status == activityStorage.Undo {
			undoCnt++
		}
		if (v.Day == 1 && v.Status == activityStorage.Undo) || v.Status == activityStorage.Done || (v.Status == activityStorage.Received && utils.IsSameDay(time.Now(), v.UpdateAt)) {
			haveDoneOrReceive = true
		}
	}
	if len(activity) == 0 || undoCnt == len(activity) || (len(activity) == receiveCnt && !utils.IsSameDay(time.Now(), activity[len(activity)-1].UpdateAt)) { //刚开始或者所有签到领取完毕之后
		if len(activity) > 0 && len(activity) == receiveCnt && !utils.IsSameDay(time.Now(), activity[len(activity)-1].UpdateAt) {
			activityStorage.RemoveAllSignInByUid(uid.Hex())
		}
		status := activityStorage.Undo
		signInConf := activityStorage.QueryActivitySignInConf()
		for _, v := range signInConf {
			newActivity := &activityStorage.ActivitySignIn{
				ActivityID: v.Oid.Hex(),
				Uid:        uid.Hex(),
				Type:       v.Type,
				Day:        v.Day,
				Charge:     v.TotalCharge,
				Get:        v.Get,
				BetTimes:   v.BetTimes,
				Status:     status,
				CreateAt:   utils.Now(),
				UpdateAt:   utils.Now(),
			}
			if v.Day == 1 {
				totalCharge := payStorage.QueryTodayChargeByUid(uid)
				if totalCharge > 0 && totalCharge >= newActivity.Charge {
					newActivity.Status = activityStorage.Done
				}
			}
			activityStorage.UpsertActivitySignIn(newActivity)
			resActivity = append(resActivity, *newActivity)
		}
	} else {
		if !haveDoneOrReceive {
			for k, v := range activity {
				if v.Status == activityStorage.Undo && v.Day != 1 {
					activity[k].Status = activityStorage.Done
					activity[k].UpdateAt = utils.Now()
					activityStorage.UpsertActivitySignIn(&activity[k])
					break
				}
			}
		}
		resActivity = activity
	}

	//判断是否有中断天数领取
	idx := 0
	for k, v := range resActivity {
		if v.Status != activityStorage.Undo {
			idx = k
		}
	}
	if idx != 0 && !utils.IsSameDay(resActivity[idx].UpdateAt, time.Now()) {
		activityStorage.RemoveAllSignInByUid(uid.Hex())
		goto Again
	}
	return resActivity
}
//计算鼓励金
func CalcEncouragementFunc(uid string) {

	if !activityStorage.QueryActivityIsOpen(activityStorage.Encouragement) { //没开启该活动
		return
	}
	gameData := activityStorage.QueryGameDataInBetAll(uid)
	if len(gameData) > 0 { //还有未结算的游戏
		return
	}
	go func() {
		time.Sleep(2 * time.Second)
		conf := activityStorage.QueryActivityEncouragementConf()
		wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
		if wallet.VndBalance + wallet.SafeBalance >= conf.MinVnd { //余额大于鼓励金的最低限制
			return
		}
		encouragement := activityStorage.QueryEncouragementAllByUid(uid)
		remainReceiveCnt := 0

		sb := gate2.QuerySessionBean(uid)
		if len(encouragement) > conf.ChargeGetCnt { //已经弹过领取提示了
			return
		} else if len(encouragement) == conf.ChargeGetCnt { //提示今日鼓励金发放完毕
			activityStorage.InsertActivityEncouragement(&activityStorage.ActivityEncouragement{ //为了区分提示是否弹出
				ActivityID: conf.Oid.Hex(),
				Uid:        uid,
				Get:        conf.Get,
				BetTimes:   conf.BetTimes,
				Status:     activityStorage.Received,
				UpdateAt:   utils.Now(),
				CreateAt:   utils.Now(),
			})
			if sb == nil {
				return
			}
			msg := make(map[string]interface{})
			msg["Data"] = ""
			msg["Action"] = actionTodayEncourageFinish
			msg["GameType"] = game.Activity
			msg["remainReceiveCnt"] = 0
			protocol.SendPack(uid,game.Push,msg)
			return
		}

		sumCharge := activityStorage.QueryActivityUserInfo(uid, activityStorage.Encouragement).SumCharge
		if sumCharge < conf.TotalCharge {
			remainReceiveCnt = conf.UnChargeGetCnt - len(encouragement) - 1
		} else {
			remainReceiveCnt = conf.ChargeGetCnt - len(encouragement) - 1
		}
		if sumCharge < conf.TotalCharge && len(encouragement) >= conf.UnChargeGetCnt { //未达到累计充值
			if sb == nil {
				return
			}
			msg := make(map[string]interface{})
			data := make(map[string]interface{})
			data["needCharge"] = conf.TotalCharge - sumCharge
			data["get"] = int64(conf.ChargeGetCnt) * conf.Get
			data["alreadyReceiveCnt"] = len(encouragement)
			data["remainReceiveCnt"] = remainReceiveCnt
			msg["Data"] = data
			msg["Action"] = actionEncouragementChargeHint
			msg["GameType"] = game.Activity
			protocol.SendPack(uid,game.Push,msg)
			return
		}

		douDouBet := conf.Get * conf.BetTimes
		userStorage.IncUserDouDouBet(utils.ConvertOID(uid), douDouBet)

		bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventEncouragement, uid+"_"+strconv.FormatInt(time.Now().Unix(), 10), conf.Get)
		walletStorage.OperateVndBalance(bill)
		wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))
		notifyWallet(uid)
		activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
			Type:       activityStorage.Encouragement,
			ActivityID: uid + "_" + strconv.FormatInt(time.Now().Unix(), 10),
			Uid:        uid,
			Charge:     sumCharge,
			Get:        conf.Get,
			BetTimes:   conf.BetTimes,
			UpdateAt:   utils.Now(),
			CreateAt:   utils.Now(),
		})
		//sb := gate2.QuerySessionBean(userID)
		//if sb != nil{
		//	res := s.DealInfoFormat()
		//	activity := make(map[string]interface{},1)
		//	activity["activityNum"] = gameStorage.QueryActivityNum(userID)
		//	res["Data"] = activity
		//
		//	ret,_ := json.Marshal(res)
		//	s.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push,ret)
		//}
		agentStorage.OnActivityData(uid, conf.Get)
		if sb == nil {
			return
		}
		msg := make(map[string]interface{})
		data := make(map[string]interface{})
		data["get"] = conf.Get
		data["alreadyReceiveCnt"] = len(encouragement)
		data["remainReceiveCnt"] = remainReceiveCnt
		msg["Data"] = data
		msg["Action"] = actionEncouragementReceiveHint
		msg["GameType"] = game.Activity
		protocol.SendPack(uid,game.Push,msg)
		activityStorage.InsertActivityEncouragement(&activityStorage.ActivityEncouragement{ //
			ActivityID: conf.Oid.Hex(),
			Uid:        uid,
			Get:        conf.Get,
			BetTimes:   conf.BetTimes,
			Status:     activityStorage.Received,
			UpdateAt:   utils.Now(),
			CreateAt:   utils.Now(),
		})
	}()

}