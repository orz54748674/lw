package activity

import (
	"encoding/json"
	"math/rand"
	"runtime"
	"sort"
	"strconv"
	"time"
	common2 "vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/game"
	common3 "vn/game/common"
	gate2 "vn/gate"
	"vn/storage/activityStorage"
	"vn/storage/agentStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

type Impl struct {
	push *gate2.OnlinePush
}
func (s *Impl) RemoveAllData() {
	activityStorage.RemoveAllTotalCharge()
	activityStorage.RemoveAllEncouragement()

	activityStorage.RemoveAllDayCharge()
	activityStorage.RemoveAllDayGame()
	activityStorage.RemoveAllDayInvite()
}
func (s *Activity) RobotTurnTable() {
	if activityStorage.QueryActivityIsOpen(activityStorage.TurnTable){
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		bots := common3.RandBotN(61,r)
		turnTableConf := activityStorage.QueryActivityTurnTableConf()
		for _,v := range bots{
			cnt := utils.RandInt64(1,10,r)
			activityStorage.IncActivityTurnTableCnt(v.Oid.Hex(),v.NickName,"robot",cnt)
			gold := utils.RandInt64(10000,cnt * 10000,r) / 1000 * 1000
			rdAward := utils.RandInt64(1,1000,r)
			userInfo := activityStorage.QueryTurnTableInfo(v.Oid.Hex())
			rd := utils.RandInt64(80,120,r)
			sumCnt := userInfo.TurnTableCnt / int(rd)//80-120之间次加次大奖
			sumGet := userInfo.SumGetGold / turnTableConf.GetAward
			if (rdAward == 1 && userInfo.TurnTableCnt > 10) || (sumCnt > 0 && int(sumGet) < sumCnt){ //加个大奖
				if sumCnt > 0 && int(sumGet) < sumCnt{
					rd = utils.RandInt64(300000,3000000,r)
					activityStorage.IncTurnTableSumGetGold(v.Oid.Hex(),turnTableConf.GetAward + rd)
				}else{
					activityStorage.IncTurnTableSumGetGold(v.Oid.Hex(),turnTableConf.GetAward + 200000)
				}
			}
			activityStorage.IncTurnTableSumGetGold(v.Oid.Hex(),gold)
		}
	}
}
func (s *Impl) OnZeroRefreshData() {
	defer func() {
		if r := recover(); r != nil {
			buff := make([]byte, 1024)
			runtime.Stack(buff, false)
			log.Error("RefreshData panic(%v)\n info:%s", r, string(buff))
		}
	}()
	for {
		now := time.Now()
		// 计算下一个零点
		next := now.Add(time.Hour * 24)
		next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
		t := time.NewTimer(next.Sub(now))
		<-t.C
		//以下为定时执行的操作
		s.RemoveAllData()

		if time.Now().Weekday() == time.Monday{//周一
			if activityStorage.QueryActivityIsOpen(activityStorage.Vip) { //开启该活动
				dealVipDownGradeAllUid() //处理降级
			}
			activityStorage.RemoveAllVipWeek()//清掉每周彩金
		}
	}
}
func (s *Impl)DealInfoFormat() map[string]interface{} {
	res := make(map[string]interface{})
	res["Code"] = 0
	res["Action"] = "HD_info"
	res["ErrMsg"] = "操作成功"
	res["GameType"] = "activity"
	return res
}
func (s *Impl) notifyWallet(uid string) {
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
	s.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, b)
}
func (s *Impl) JoinFirstCharge(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	uid := session.GetUserID()
	if !activityStorage.QueryActivityIsOpen(activityStorage.FirstCharge){//未开启首充活动
		return errCode.ServerError.GetI18nMap(), nil
	}
	userFirstCharge := activityStorage.QueryFirstChargeByUid(uid)
	if len(userFirstCharge) != 0 && userFirstCharge[0].Status != activityStorage.Undo{ //已经完成首充活动或者已经报名
		return errCode.ServerError.GetI18nMap(), nil
	}
	conf := activityStorage.QueryActivityFistChargeConf()
	if len(conf) == 0{
		return errCode.ServerError.GetI18nMap(), nil
	}
	activityStorage.UpsertActivityFirstCharge(&activityStorage.ActivityFirstCharge{
		ActivityID: conf[0].Oid.Hex(),
		Uid: uid,
		GetPer:conf[0].GetPer,
		GetMax: conf[0].GetMax,
		BetTimes: conf[0].BetTimes,
		Status: activityStorage.Done,
		CreateAt: utils.Now(),
		UpdateAt: utils.Now(),
	})
	return errCode.Success(nil).GetI18nMap(), nil
}
func (s *Impl) GetActivityConf(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	uid := session.GetUserID()
	res := make(map[string]interface{})

	for _,v := range activityStorage.ActivityNormalList{
		conf := activityStorage.QueryActivityConfByType(v)
		if conf != nil && conf.Status != 0{
			activity := make(map[string]interface{})
			activity["haveReceiveNum"] = QueryHaveReceiveActivity(uid,v)
			activity["conf"] = conf
			res[string(v)] = activity
		}
	}
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) GetTotalChargeActivity(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	//log.Info("params: %s,ip: %s")
	uid := utils.ConvertOID(session.GetUserID())
	res := make(map[string]interface{})
	resActivity,totalCharge := RefreshTotalChargeActivity(uid)
	sort.Slice(resActivity, func(i, j int) bool { //升序排序
		return resActivity[i].Charge < resActivity[j].Charge
	})
	nextGoalGet := int64(0)
	needGoal := int64(0)
	for _,v := range resActivity{
		if totalCharge < v.Charge{
			nextGoalGet = v.Get
			needGoal = v.Charge - totalCharge
			break
		}
	}

	res["totalCharge"] = totalCharge
	res["activity"] = resActivity
	res["nextGoalGet"] = nextGoalGet
	res["needGoal"] = needGoal
	res["haveReceiveNum"] = QueryHaveReceiveActivity(uid.Hex(),activityStorage.TotalCharge)
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) ReceiveTotalChargeActivity(session gate.Session,activityID string) (r map[string]interface{}, err error) {
	//log.Info("params: %s,ip: %s")
	userID := session.GetUserID()
	activity := activityStorage.QueryTodayTotalCharge(userID,activityID)
	if activity == nil || activity.Status != activityStorage.Done{
		return errCode.ActivityReceiveError.GetI18nMap(), nil
	}
	activity.Status = activityStorage.Received
	activity.UpdateAt = time.Now()
	activityStorage.UpsertActivityTotalCharge(activity)
	douDouBet := activity.Get * activity.BetTimes
	userStorage.IncUserDouDouBet(utils.ConvertOID(userID), douDouBet)

	bill := walletStorage.NewBill(userID,walletStorage.TypeIncome,walletStorage.EventTotalCharge,userID+"_"+activity.ActivityID,activity.Get)
	walletStorage.OperateVndBalance(bill)
	s.notifyWallet(userID)
	activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
		Type: activityStorage.TotalCharge,
		ActivityID: activity.ActivityID,
		Uid: userID,
		Charge: activity.Charge,
		Get: activity.Get,
		BetTimes: activity.BetTimes,
		UpdateAt: utils.Now(),
		CreateAt: utils.Now(),
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
	agentStorage.OnActivityData(userID,activity.Get)
	NotifyNormalActivityNum(userID)
	return errCode.Success(nil).GetMap(), nil
}
func (s *Impl) GetSignInActivity(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	//log.Info("params: %s,ip: %s")
	uid := utils.ConvertOID(session.GetUserID())
	res := make(map[string]interface{})
	resActivity := RefreshSignInActivity(uid)
	sort.Slice(resActivity, func(i, j int) bool { //升序排序
		return resActivity[i].Day < resActivity[j].Day
	})
	res["activity"] = resActivity
	res["haveReceiveNum"] = QueryHaveReceiveActivity(uid.Hex(),activityStorage.SignIn)
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) GetEncouragementConf(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	res := activityStorage.QueryActivityEncouragementConf()
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) ReceiveSignInActivity(session gate.Session,activityID string) (r map[string]interface{}, err error) {
	//log.Info("params: %s,ip: %s")
	userID := session.GetUserID()
	activity := activityStorage.QuerySignInById(userID,activityID)
	if activity == nil || activity.Status != activityStorage.Done || !activityStorage.QueryActivityIsOpen(activityStorage.SignIn){
		return errCode.ActivityReceiveError.GetI18nMap(), nil
	}
	totalCharge := payStorage.QueryTodayChargeByUid(utils.ConvertOID(userID))
	if activity.Day == 1 && ((activity.Charge == 0 && totalCharge == 0)||totalCharge < activity.Charge){
		return errCode.ActivityNeedChargeError.GetI18nMap(), nil
	}
	activity.Status = activityStorage.Received
	activity.UpdateAt = time.Now()
	activityStorage.UpsertActivitySignIn(activity)
	douDouBet := activity.Get * activity.BetTimes
	userStorage.IncUserDouDouBet(utils.ConvertOID(userID), douDouBet)

	bill := walletStorage.NewBill(userID,walletStorage.TypeIncome,walletStorage.EventSignIn,userID+"_"+activity.ActivityID,activity.Get)
	walletStorage.OperateVndBalance(bill)
	s.notifyWallet(userID)
	activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
		Type: activityStorage.SignIn,
		ActivityID: activity.ActivityID,
		Uid: userID,
		Charge: activity.Charge,
		Get: activity.Get,
		BetTimes: activity.BetTimes,
		UpdateAt: utils.Now(),
		CreateAt: utils.Now(),
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
	agentStorage.OnActivityData(userID,activity.Get)
	NotifyNormalActivityNum(userID)
	return errCode.Success(nil).GetMap(), nil
}

func (s *Impl) getDayChargeActivity(uid primitive.ObjectID) []activityStorage.ActivityDayCharge{
	resActivity := RefreshDayChargeActivity(uid)
	sort.Slice(resActivity, func(i, j int) bool { //升序排序
		return resActivity[i].Charge < resActivity[j].Charge
	})
	return resActivity
}
func (s *Impl) getDayGameActivity(uid primitive.ObjectID) []activityStorage.ActivityDayGame{
	resActivity,_ := RefreshDayGameActivity(uid)
	sort.Slice(resActivity, func(i, j int) bool { //升序排序
		return resActivity[i].NeedBet < resActivity[j].NeedBet
	})
	for k,v := range resActivity{
		resActivity[k].GameTypeLanguage = common2.I18str(string(v.GameType))
	}
	return resActivity
}

func (s *Impl) getDayInviteActivity(uid primitive.ObjectID) []activityStorage.ActivityDayInvite{
	resActivity,_ := RefreshDayInviteActivity(uid)
	sort.Slice(resActivity, func(i, j int) bool { //升序排序
		return resActivity[i].InviteNum < resActivity[j].InviteNum
	})
	return resActivity
}
func (s *Impl) GetDayActivity(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	//log.Info("params: %s,ip: %s")
	uid := utils.ConvertOID(session.GetUserID())
	res := make(map[string]interface{})
	res[string(activityStorage.DayCharge)] = s.getDayChargeActivity(uid)
	res[string(activityStorage.DayGame)] = s.getDayGameActivity(uid)
	res[string(activityStorage.DayInvite)] = s.getDayInviteActivity(uid)
	res["haveReceiveNum"] = GetDayActivityNum(uid)
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) receiveDayChargeActivity(userID string,activityID string) (r map[string]interface{}, err error){
	activity := activityStorage.QueryTodayDayCharge(userID,activityID)
	if activity == nil || activity.Status != activityStorage.Done{
		return errCode.ActivityReceiveError.GetI18nMap(), nil
	}
	activity.Status = activityStorage.Received
	activity.UpdateAt = time.Now()
	activityStorage.UpsertActivityDayCharge(activity)
	if activity.Get > 0{
		douDouBet := activity.Get * activity.BetTimes
		userStorage.IncUserDouDouBet(utils.ConvertOID(userID), douDouBet)

		bill := walletStorage.NewBill(userID,walletStorage.TypeIncome,walletStorage.EventDayCharge,userID+"_"+activity.ActivityID,activity.Get)
		walletStorage.OperateVndBalance(bill)
		s.notifyWallet(userID)
		activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
			Type: activityStorage.DayCharge,
			ActivityID: activity.ActivityID,
			Uid: userID,
			Charge: activity.Charge,
			Get: activity.Get,
			GetPoints: activity.GetPoints,
			BetTimes: activity.BetTimes,
			UpdateAt: utils.Now(),
			CreateAt: utils.Now(),
		})
		agentStorage.OnActivityData(userID,activity.Get)
	}
	if activity.GetPoints > 0{
		activityStorage.IncTurnTableInfoPoints(userID,activity.GetPoints)
	}
	NotifyDayActivityNum(userID)
	return errCode.Success(nil).GetMap(), nil
}
func (s *Impl) receiveDayGameActivity(userID string,activityID string) (r map[string]interface{}, err error){
	activity := activityStorage.QueryTodayDayGame(userID,activityID)
	if activity == nil || activity.Status != activityStorage.Done{
		return errCode.ActivityReceiveError.GetI18nMap(), nil
	}
	activity.Status = activityStorage.Received
	activity.UpdateAt = time.Now()
	activityStorage.UpsertActivityDayGame(activity)
	if activity.Get > 0{
		douDouBet := activity.Get * activity.BetTimes
		userStorage.IncUserDouDouBet(utils.ConvertOID(userID), douDouBet)

		bill := walletStorage.NewBill(userID,walletStorage.TypeIncome,walletStorage.EventDayGame,userID+"_"+activity.ActivityID,activity.Get)
		walletStorage.OperateVndBalance(bill)
		s.notifyWallet(userID)
		activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
			Type: activityStorage.DayGame,
			ActivityID: activity.ActivityID,
			Uid: userID,
			Charge: 0,
			Get: activity.Get,
			GetPoints: activity.GetPoints,
			BetTimes: activity.BetTimes,
			UpdateAt: utils.Now(),
			CreateAt: utils.Now(),
		})
		agentStorage.OnActivityData(userID,activity.Get)
	}
	if activity.GetPoints > 0{
		activityStorage.IncTurnTableInfoPoints(userID,activity.GetPoints)
	}
	NotifyDayActivityNum(userID)
	return errCode.Success(nil).GetMap(), nil
}
func (s *Impl) receiveDayInviteActivity(userID string,activityID string) (r map[string]interface{}, err error){
	activity := activityStorage.QueryTodayDayInvite(userID,activityID)
	if activity == nil || activity.Status != activityStorage.Done{
		return errCode.ActivityReceiveError.GetI18nMap(), nil
	}
	activity.Status = activityStorage.Received
	activity.UpdateAt = time.Now()
	activityStorage.UpsertActivityDayInvite(activity)
	if activity.Get > 0{
		douDouBet := activity.Get * activity.BetTimes
		userStorage.IncUserDouDouBet(utils.ConvertOID(userID), douDouBet)

		bill := walletStorage.NewBill(userID,walletStorage.TypeIncome,walletStorage.EventDayInvite,userID+"_"+activity.ActivityID,activity.Get)
		walletStorage.OperateVndBalance(bill)
		s.notifyWallet(userID)
		activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
			Type: activityStorage.DayInvite,
			ActivityID: activity.ActivityID,
			Uid: userID,
			Charge: 0,
			Get: activity.Get,
			GetPoints: activity.GetPoints,
			BetTimes: activity.BetTimes,
			UpdateAt: utils.Now(),
			CreateAt: utils.Now(),
		})
		agentStorage.OnActivityData(userID,activity.Get)
	}
	if activity.GetPoints > 0{
		activityStorage.IncTurnTableInfoPoints(userID,activity.GetPoints)
	}
	NotifyDayActivityNum(userID)
	return errCode.Success(nil).GetMap(), nil
}
func (s *Impl) ReceiveDayActivity(session gate.Session,activityID string,activityType activityStorage.ActivityType) (r map[string]interface{}, err error) {
	//log.Info("params: %s,ip: %s")
	userID := session.GetUserID()
	if activityType == activityStorage.DayCharge{
		return s.receiveDayChargeActivity(userID,activityID)
	}else if activityType == activityStorage.DayGame{
		return  s.receiveDayGameActivity(userID,activityID)
	}else if activityType == activityStorage.DayInvite{
		return s.receiveDayInviteActivity(userID,activityID)
	}
	return errCode.Success(nil).GetMap(), nil
}

func (s *Impl) GetVipActivityConf(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	res := make(map[string]interface{})
	res["conf"] = activityStorage.QueryActivityVipConf()
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) GetVipActivity(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	uid := session.GetUserID()
	uOid := utils.ConvertOID(uid)
	userInfo := userStorage.QueryUserInfo(uOid)
	//VIP升级
	vipConf := activityStorage.QueryActivityVipConfByType(activityStorage.ChargeNeed)
	aUserInfo := activityStorage.QueryActivityUserInfo(uid,activityStorage.Vip)
	curLevel := userInfo.VipLevel
	levelSwitchVipTable := []int64{
		vipConf.Vip0,vipConf.Vip1,vipConf.Vip2,vipConf.Vip3,vipConf.Vip4,vipConf.Vip5,vipConf.Vip6,vipConf.Vip7,vipConf.Vip8,vipConf.Vip9,
	}
	curVipCharge := aUserInfo.SumCharge
	goalVipCharge := int64(0)
	if curLevel >= len(levelSwitchVipTable) - 1{
		curVipCharge = levelSwitchVipTable[curLevel]
		goalVipCharge = levelSwitchVipTable[curLevel]
	}else{
		goalVipCharge = levelSwitchVipTable[curLevel + 1]
	}

	//VIP保级
	vipConf = activityStorage.QueryActivityVipConfByType(activityStorage.KeepGradeNeed)
	levelSwitchVipTable = []int64{
		vipConf.Vip0,vipConf.Vip1,vipConf.Vip2,vipConf.Vip3,vipConf.Vip4,vipConf.Vip5,vipConf.Vip6,vipConf.Vip7,vipConf.Vip8,vipConf.Vip9,
	}
	curVipKeepBets := aUserInfo.WeekBets
	goalVipKeepBets := levelSwitchVipTable[curLevel]
	keepIsOk := false
	if aUserInfo.WeekHaveUpGrade || curVipKeepBets >= goalVipKeepBets{
		keepIsOk = true
	}
	//每周彩金
	vipConf = activityStorage.QueryActivityVipConfByType(activityStorage.WeekGet)
	aUserInfo = activityStorage.QueryActivityUserInfo(uid,activityStorage.VipWeek)
	levelSwitchVipTable = []int64{
		vipConf.Vip0,vipConf.Vip1,vipConf.Vip2,vipConf.Vip3,vipConf.Vip4,vipConf.Vip5,vipConf.Vip6,vipConf.Vip7,vipConf.Vip8,vipConf.Vip9,
	}
	vipRecord := activityStorage.QueryVipWeekByLevel(uid, curLevel)
	weekGet := levelSwitchVipTable[curLevel]
	var weekStatus activityStorage.ActivityStatus
	if vipRecord != nil{
		weekStatus = vipRecord.Status
	}else{
		weekStatus = activityStorage.Undo
	}
	weekNeedCharge := weekGet - aUserInfo.SumCharge
	weekCountDown := int(7 - time.Now().Weekday() + time.Monday)
	if weekCountDown > 7{
		weekCountDown -= 7
	}
	//VIP receive 数量
	vipDoneNum := len(activityStorage.QueryVipAllDone(uid))

	res := make(map[string]interface{})
	res["CurVipLevel"] = curLevel
	res["CurVipCharge"] = curVipCharge
	res["GoalVipCharge"] = goalVipCharge

	res["KeepIsOk"] = keepIsOk
	res["CurVipKeepBets"] = curVipKeepBets
	res["GoalVipKeepBets"] = goalVipKeepBets

	res["WeekStatus"] = weekStatus
	res["WeekGet"] = weekGet
	res["WeekNeedCharge"] = weekNeedCharge
	res["WeekCountDown"] = weekCountDown

	res["VipDoneNum"] = vipDoneNum
	if weekStatus == activityStorage.Done{
		res["AllReceiveNum"] = vipDoneNum + 1
	}else{
		res["AllReceiveNum"] = vipDoneNum
	}

	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) GetVipGift(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	uid := session.GetUserID()
	record := activityStorage.QueryVipAll(uid)
	res := make([]activityStorage.ActivityVip,0)
	vipConf := activityStorage.QueryActivityVipConfByType(activityStorage.ChargeNeed)
	levelSwitchVipTable := []int64{
		vipConf.Vip0,vipConf.Vip1,vipConf.Vip2,vipConf.Vip3,vipConf.Vip4,vipConf.Vip5,vipConf.Vip6,vipConf.Vip7,vipConf.Vip8,vipConf.Vip9,
	}
	for i := 1;i < len(levelSwitchVipTable);i++{
		find := false
		for _,v := range record{
			if v.Level == i{
				find = true
				res = append(res,v)
				break
			}
		}
		if !find{
			res = append(res,activityStorage.ActivityVip{
				Level: i,
				Status: activityStorage.Undo,
			})
		}
	}
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) ReceiveVipGift(session gate.Session,level int) (r map[string]interface{}, err error) {
	//log.Info("params: %s,ip: %s")
	userID := session.GetUserID()
	activity := activityStorage.QueryVipByLevel(userID,level)
	if activity == nil || activity.Status != activityStorage.Done || !activityStorage.QueryActivityIsOpen(activityStorage.Vip){
		return errCode.ActivityReceiveError.GetI18nMap(), nil
	}
	activity.Status = activityStorage.Received
	activity.UpdateAt = time.Now()
	activityStorage.UpsertActivityVip(activity)
	if activity.GetGold > 0 || activity.GetPoints > 0{
		if activity.GetGold > 0{
			douDouBet := activity.GetGold * activity.BetTimes
			userStorage.IncUserDouDouBet(utils.ConvertOID(userID), douDouBet)
			bill := walletStorage.NewBill(userID,walletStorage.TypeIncome,walletStorage.EventVip,userID+"_"+activity.ActivityID,activity.GetGold)
			walletStorage.OperateVndBalance(bill)
			s.notifyWallet(userID)
			activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
				Type: activityStorage.Vip,
				ActivityID: activity.ActivityID,
				Uid: userID,
				Charge: int64(activity.Level),
				Get: activity.GetGold,
				GetPoints: activity.GetPoints,
				BetTimes: activity.BetTimes,
				UpdateAt: utils.Now(),
				CreateAt: utils.Now(),
			})
			agentStorage.OnActivityData(userID,activity.GetGold)
		}
		if activity.GetPoints > 0{
			activityStorage.IncTurnTableInfoPoints(userID,activity.GetPoints)
		}
	}
	res := make(map[string]interface{},2)
	res["GetGold"] = activity.GetGold
	res["GetPoints"] = activity.GetPoints
	NotifyVipActivityNum(userID)
	return errCode.Success(res).GetMap(), nil
}
func (s *Impl) ReceiveVipGiftAll(session gate.Session) (r map[string]interface{}, err error) {
	//log.Info("params: %s,ip: %s")
	userID := session.GetUserID()
	record := activityStorage.QueryVipAll(userID)
	gold := int64(0)
	points := float64(0)
	douDouBet := int64(0)
	for _,v := range record{
		if v.Status == activityStorage.Done{
			gold += v.GetGold
			points += v.GetPoints
			douDouBet += v.GetGold * v.BetTimes
			v.Status = activityStorage.Received
			v.UpdateAt = time.Now()
			activityStorage.UpsertActivityVip(&v)
			if v.GetGold > 0 || v.GetPoints > 0{
				if v.GetGold > 0{
					douDouBet := v.GetGold * v.BetTimes
					userStorage.IncUserDouDouBet(utils.ConvertOID(userID), douDouBet)
					bill := walletStorage.NewBill(userID,walletStorage.TypeIncome,walletStorage.EventVip,userID+"_"+v.ActivityID,v.GetGold)
					walletStorage.OperateVndBalance(bill)
					s.notifyWallet(userID)
					activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
						Type: activityStorage.Vip,
						ActivityID: v.ActivityID,
						Uid: userID,
						Charge: int64(v.Level),
						Get: v.GetGold,
						GetPoints: v.GetPoints,
						BetTimes: v.BetTimes,
						UpdateAt: utils.Now(),
						CreateAt: utils.Now(),
					})
					agentStorage.OnActivityData(userID,v.GetGold)
				}
				if v.GetPoints > 0{
					activityStorage.IncTurnTableInfoPoints(userID,v.GetPoints)
				}
			}
		}
	}
	res := make(map[string]interface{},2)
	res["GetGold"] = gold
	res["GetPoints"] = points
	NotifyVipActivityNum(userID)
	return errCode.Success(res).GetMap(), nil
}
func (s *Impl) ReceiveVipWeekGift(session gate.Session,level int) (r map[string]interface{}, err error) {
	//log.Info("params: %s,ip: %s")
	userID := session.GetUserID()
	userInfo := userStorage.QueryUserInfo(utils.ConvertOID(userID))
	if userInfo.VipLevel != level{
		return errCode.ActivityReceiveError.GetI18nMap(), nil
	}
	activity := activityStorage.QueryVipWeekById(userID,userInfo.VipLevel)
	if activity == nil || activity.Status != activityStorage.Done || !activityStorage.QueryActivityIsOpen(activityStorage.Vip){
		return errCode.ActivityReceiveError.GetI18nMap(), nil
	}
	activity.Status = activityStorage.Received
	activity.UpdateAt = time.Now()
	activityStorage.UpsertActivityVipWeek(activity)
	if activity.GetGold > 0{
		if activity.GetGold > 0{
			douDouBet := activity.GetGold * activity.BetTimes
			userStorage.IncUserDouDouBet(utils.ConvertOID(userID), douDouBet)
			bill := walletStorage.NewBill(userID,walletStorage.TypeIncome,walletStorage.EventVipWeek,userID+"_"+activity.ActivityID,activity.GetGold)
			walletStorage.OperateVndBalance(bill)
			s.notifyWallet(userID)
		}
		activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
			Type: activityStorage.Vip,
			ActivityID: activity.ActivityID,
			Uid: userID,
			Charge: int64(activity.Level),
			Get: activity.GetGold,
			GetPoints: 0,
			BetTimes: activity.BetTimes,
			UpdateAt: utils.Now(),
			CreateAt: utils.Now(),
		})
		agentStorage.OnActivityData(userID,activity.GetGold)
	}
	res := make(map[string]interface{},1)
	res["GetGold"] = activity.GetGold
	NotifyVipActivityNum(userID)
	return errCode.Success(res).GetMap(), nil
}
func (s *Impl) GetTurnTableConf(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	uid := session.GetUserID()
	//uOid := utils.ConvertOID(uid)
	// turnTable list
	turnTableList := activityStorage.QueryActivityTurnTableList()
	turnTableConf := activityStorage.QueryActivityTurnTableConf()
	userInfo := activityStorage.QueryTurnTableInfo(uid)
	conf := make(map[string]interface{})
	conf["GetAward"] = turnTableConf.GetAward
	conf["TurnNeedPoints"] = turnTableConf.TurnNeedPoints
	conf["LNum"] = userInfo.L
	conf["UNum"] = userInfo.U
	conf["CNum"] = userInfo.C
	conf["KNum"] = userInfo.K
	conf["YNum"] = userInfo.Y
	conf["Points"] = userInfo.Points
	res := make(map[string]interface{})
	res["TurnTableList"] = turnTableList
	res["Conf"] = conf
	return errCode.Success(res).GetMap(), nil
}
func (s *Impl) StartTurnTable(session gate.Session, params map[string]interface{}) (ret map[string]interface{}, err error) {
	uid := session.GetUserID()
	turnInfo := activityStorage.QueryTurnTableInfo(uid)
	conf := activityStorage.QueryActivityTurnTableConf()
	tableList := activityStorage.QueryActivityTurnTableList()
	if turnInfo.Points < conf.TurnNeedPoints{//积分不足
		return errCode.PointsNotEnough.GetI18nMap(), nil
	}
	activityStorage.IncTurnTableInfoPoints(uid,-conf.TurnNeedPoints)//

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	control := activityStorage.QueryActivityTurnTableControl()
	var resNoList activityStorage.ActivityTurnTableList
	if control == nil{
		no := utils.RandInt64(1,int64(len(tableList)) + 1,r)
		resNoList = activityStorage.QueryActivityTurnTableListByNo(int(no))
	}else{
		controlTotal := control.L + control.U + control.C + control.K + control.Y + control.TRUOT
		v := utils.RandInt64(1,controlTotal,r)
		var wordType activityStorage.WordType
		if v <= control.L{
			wordType = activityStorage.L
		}else if v <= control.L + control.U{
			wordType = activityStorage.U
		}else if v <= control.L + control.U + control.C{
			wordType = activityStorage.C
		}else if v <= control.L + control.U + control.C + control.K{
			wordType = activityStorage.K
		}else if v <= control.L + control.U + control.C + control.K + control.Y{
			wordType = activityStorage.Y
		}else{
			wordType = activityStorage.TRUOT
		}
		list := activityStorage.QueryActivityTurnTableListByWordType(wordType)
		if len(list) == 0{
			no := utils.RandInt64(1,int64(len(tableList)) + 1,r)
			resNoList = activityStorage.QueryActivityTurnTableListByNo(int(no))
		}else{
			idx := utils.RandInt64(1,int64(len(list)) + 1,r) - 1
			resNoList = list[idx]
		}
	}
	getVnd := int64(0)
	getVnd += resNoList.GetVnd
	getPoints := float64(0)
	getPoints += resNoList.GetPoints
	userInfo := activityStorage.QueryTurnTableInfo(uid)
	lNum,uNum,cNum,kNum,yNum := userInfo.L,userInfo.L,userInfo.C,userInfo.K,userInfo.Y
	if resNoList.WordType == activityStorage.L{
		lNum += 1
	}else if resNoList.WordType == activityStorage.U{
		uNum += 1
	}else if resNoList.WordType == activityStorage.C{
		cNum += 1
	}else if resNoList.WordType == activityStorage.K{
		kNum += 1
	}else if resNoList.WordType == activityStorage.Y{
		yNum += 1
	}
	isJackpot := false
	if lNum > 0 && uNum > 0 && cNum > 0 && kNum > 0 && yNum > 0{//中大奖
		isJackpot = true
		getVnd += conf.GetAward
	}

	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	activityStorage.IncActivityTurnTableCnt(uid,user.NickName,"user",1)
	activityStorage.IncActivityWordNum(uid,resNoList.WordType,1)
	activityStorage.IncTurnTableSumGetGold(uid,getVnd)
	if isJackpot{//清除所有集字
		activityStorage.ResetActivityWordNum(uid)
	}
	recordNo := "#" + strconv.FormatInt(time.Now().Unix(),10)
	activityStorage.InsertActivityTurnTableRecord(&activityStorage.ActivityTurnTableRecord{
		Uid: uid,
		No: recordNo,
		WordType: resNoList.WordType,
		GetVnd: getVnd,
		IsJackPot: isJackpot,
		BetTimes: conf.BetTimes,
		UpdateAt: utils.Now(),
		CreateAt: utils.Now(),
	})

	if getVnd > 0{
		douDouBet := getVnd * conf.BetTimes
		userStorage.IncUserDouDouBet(utils.ConvertOID(uid), douDouBet)
		bill := walletStorage.NewBill(uid,walletStorage.TypeIncome,walletStorage.EventTurnTable,uid+"_"+ recordNo,getVnd)
		walletStorage.OperateVndBalance(bill)
		//notifyWallet(uid)
		activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
			Type: activityStorage.TurnTable,
			ActivityID: recordNo,
			Uid: uid,
			Charge: int64(conf.TurnNeedPoints),
			Get: getVnd,
			GetPoints: getPoints,
			BetTimes: conf.BetTimes,
			UpdateAt: utils.Now(),
			CreateAt: utils.Now(),
		})
		agentStorage.OnActivityData(uid,getVnd)
	}
	if getPoints > 0{
		activityStorage.IncTurnTableInfoPoints(uid,getPoints)
	}

	res := make(map[string]interface{})
	res["GetVnd"] = getVnd
	res["GetPoints"] = getPoints
	res["IsJackpot"] = isJackpot
	res["No"] = resNoList.No
	return errCode.Success(res).GetMap(), nil
}
func (s *Impl) TurnTableRankList(session gate.Session, params map[string]interface{}) (ret map[string]interface{}, err error) {
	res := activityStorage.QueryTurnTableInfoAll(20)
	return errCode.Success(res).GetMap(), nil
}
func (s *Impl) TurnTableRecord(session gate.Session, params map[string]interface{}) (ret map[string]interface{}, err error) {
	_, err = utils.CheckParams2(params, []string{"Offset", "PageSize"})
	if err != nil {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	Offset, _ := utils.ConvertInt(params["Offset"])
	PageSize, _ := utils.ConvertInt(params["PageSize"])
	uid := session.GetUserID()
	turnTableConf := activityStorage.QueryActivityTurnTableConf()
	userInfo := activityStorage.QueryTurnTableInfo(uid)
	conf := make(map[string]interface{})
	conf["GetAward"] = turnTableConf.GetAward
	conf["LNum"] = userInfo.L
	conf["UNum"] = userInfo.U
	conf["CNum"] = userInfo.C
	conf["KNum"] = userInfo.K
	conf["YNum"] = userInfo.Y
	record := activityStorage.QueryTurnTableRecordByUid(uid,int(Offset),int(PageSize))
	res := make(map[string]interface{})
	res["Conf"] =conf
	res["Record"] = record
	res["TotalNum"] = activityStorage.QueryTurnTableRecordByUidTotal(uid)
	return errCode.Success(res).GetMap(), nil
}
func (s *Impl) updateVipLevel(session gate.Session, params map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	uid, _ := params["uid"].(string)
	uOid := utils.ConvertOID(uid)
	res := make(map[string]interface{})
	level, _ := utils.ConvertInt(params["level"])
	//info := activityStorage.QueryActivityUserInfo(uid,activityStorage.Vip)
	userInfo := userStorage.QueryUserInfo(uOid)
	vipConf := activityStorage.QueryActivityVipConfByType(activityStorage.ChargeNeed)
	curLevel := userInfo.VipLevel
	if curLevel == int(level){
		return errCode.Success(res).GetI18nMap(), nil
	}
	levelSwitchVipTable := []int64{
		vipConf.Vip0,vipConf.Vip1,vipConf.Vip2,vipConf.Vip3,vipConf.Vip4,vipConf.Vip5,vipConf.Vip6,vipConf.Vip7,vipConf.Vip8,vipConf.Vip9,
	}
	if int(level) > curLevel{//升级
		activityStorage.SetActivityUserCharge(uid,activityStorage.Vip,levelSwitchVipTable[level])//累计充值置为升级的起始值
		dealVipUpGradeByUid(uid)
	}else{ //降级
		activityStorage.SetActivityUserCharge(uid,activityStorage.Vip,levelSwitchVipTable[level])//累计充值置为降级的起始值
		userStorage.SetUserInfoVipLevel(uOid,int(level))//降级
		for i := int(level) + 1;i <= curLevel;i++{
			vipRecord := activityStorage.QueryVipByLevel(uid, i)
			if vipRecord.Status == activityStorage.Done{//未领取的重置为未完成
				vipRecord.Status = activityStorage.Undo
				activityStorage.UpsertActivityVip(vipRecord)
			}
		}
		NotifyVipActivityNum(uid)
		//if info.WeekHaveUpGrade{
		//	activityStorage.SetActivityWeekHaveUpGrade(uid,activityStorage.Vip,false)//本周升过级置为false
		//}
	}
	return errCode.Success(res).GetI18nMap(), nil
}