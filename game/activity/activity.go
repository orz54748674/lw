package activity

import (
	"github.com/robfig/cron"
	"time"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/framework/mqant/module/base"
	"vn/game"
	gate2 "vn/gate"
	"vn/storage/activityStorage"
)

var Module = func() module.Module {
	this := new(Activity)
	return this
}

type Activity struct {
	basemodule.BaseModule
	push *gate2.OnlinePush
	impl *Impl
}

func (s *Activity) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.Activity)
}
func (s *Activity) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

const (
	actionJoinFirstCharge           = "HD_JoinFirstCharge"

	actionGetActivityConf           = "HD_GetActivityConf"
	actionGetTotalChargeActivity          = "HD_GetTotalChargeActivity"
	actionReceiveTotalChargeActivity          = "HD_ReceiveTotalChargeActivity"
	actionGetSignInActivity          = "HD_GetSignInActivity"
	actionReceiveSignInActivity          = "HD_ReceiveSignInActivity"
	actionTodayEncourageFinish          = "HD_TodayEncourageFinish"
	actionGetEncouragementConf         = "HD_GetEncouragementConf"
	actionEncouragementChargeHint         = "HD_EncouragementChargeHint"
	actionEncouragementReceiveHint         = "HD_EncouragementReceiveHint"
	//每日任务
	actionGetDayActivity          = "HD_GetDayActivity"
	actionReceiveDayActivity          = "HD_ReceiveDayActivity"
	//VIP
	actionGetVipActivityConf         = "HD_GetVipActivityConf"
	actionUpGradeSuccess         = "HD_UpGradeSuccess"
	actionGetVipActivity         = "HD_GetVipActivity"
	actionGetVipGift         = "HD_GetVipGift"
	actionReceiveVipGift         = "HD_ReceiveVipGift"
	actionReceiveVipGiftAll         = "HD_ReceiveVipGiftAll"
	actionReceiveVipWeekGift         = "HD_ReceiveVipWeekGift"
	actionUpdateVipLevel        = "HD_UpdateVipLevel"
	//转盘
	actionGetTurnTableConf         = "HD_GetTurnTableConf"
	actionStartTurnTable         = "HD_StartTurnTable"
	actionTurnTableRankList         = "HD_TurnTableRankList"
	actionTurnTableRecord        = "HD_TurnTableRecord"
)
func (s *Activity) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings)

	hook := game.NewHook(s.GetType())
	hook.RegisterAndCheckLogin(s.GetServer(), actionJoinFirstCharge, s.impl.JoinFirstCharge)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetActivityConf, s.impl.GetActivityConf)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetTotalChargeActivity, s.impl.GetTotalChargeActivity)
	hook.RegisterAndCheckLogin(s.GetServer(), actionReceiveTotalChargeActivity, s.ReceiveTotalChargeActivity)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetSignInActivity, s.impl.GetSignInActivity)
	hook.RegisterAndCheckLogin(s.GetServer(), actionReceiveSignInActivity, s.ReceiveSignInActivity)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetEncouragementConf, s.impl.GetEncouragementConf)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetDayActivity, s.impl.GetDayActivity)
	hook.RegisterAndCheckLogin(s.GetServer(), actionReceiveDayActivity, s.ReceiveDayActivity)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetVipActivityConf, s.impl.GetVipActivityConf)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetVipActivity, s.impl.GetVipActivity)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetVipGift, s.impl.GetVipGift)
	hook.RegisterAndCheckLogin(s.GetServer(), actionReceiveVipGift, s.ReceiveVipGift)
	hook.RegisterAndCheckLogin(s.GetServer(), actionReceiveVipGiftAll, s.ReceiveVipGiftAll)
	hook.RegisterAndCheckLogin(s.GetServer(), actionReceiveVipWeekGift, s.ReceiveVipWeekGift)

	hook.RegisterAndCheckLogin(s.GetServer(), actionGetTurnTableConf, s.impl.GetTurnTableConf)
	hook.RegisterAndCheckLogin(s.GetServer(), actionStartTurnTable, s.impl.StartTurnTable)
	hook.RegisterAndCheckLogin(s.GetServer(), actionTurnTableRankList, s.impl.TurnTableRankList)
	hook.RegisterAndCheckLogin(s.GetServer(), actionTurnTableRecord, s.impl.TurnTableRecord)

	hook.RegisterAdminInterface(s.GetServer(),actionUpdateVipLevel, s.impl.updateVipLevel)
	s.push = &gate2.OnlinePush{
		TraceSpan: log.CreateRootTrace(),
		App:       app,
	}
	s.push.OnlinePushInit(nil, 128)
	s.impl = &Impl{push: s.push}

	incDataExpireDay := time.Duration(
		app.GetSettings().Settings["mongoIncDataExpireDay"].(float64)) * 24 * time.Hour

	activityStorage.InitActivityFirstChargeConf()
	activityStorage.InitActivityFirstCharge()

	activityStorage.InitGameDataInBet()
	activityStorage.RemoveAllGameDataInBet()

	activityStorage.InitActivityReceiveRecord(incDataExpireDay)
	activityStorage.InitActivityUserInfo()

	activityStorage.InitActivityConf()
	activityStorage.InitActivityTotalChargeConf()
	activityStorage.InitActivityTotalCharge()

	activityStorage.InitActivitySignInConf()
	activityStorage.InitActivitySignIn()

	activityStorage.InitActivityEncouragementConf()
	activityStorage.InitActivityEncouragement(time.Duration(2) * 24 * time.Hour)

	activityStorage.InitActivityDayChargeConf()
	activityStorage.InitActivityDayCharge()
	activityStorage.InitActivityDayGameConf()
	activityStorage.InitActivityDayGame()
	activityStorage.InitActivityDayInviteConf()
	activityStorage.InitActivityDayInvite()

	activityStorage.InitActivityVipConf()
	activityStorage.InitActivityVip()
	activityStorage.InitActivityVipWeek()

	activityStorage.InitActivityTurnTableConf()
	activityStorage.InitActivityTurnTableList()
	activityStorage.InitActivityTurnTableRecord()
	activityStorage.InitActivityTurnTableInfo()
	activityStorage.InitActivityTurnTableControl()

	go s.impl.OnZeroRefreshData()
	go func() {
		c := cron.New()
		c.AddFunc("*/61 * * * * ?", s.RobotTurnTable)
		c.Start()
	}()
}

func (s *Activity) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", s.GetType())
	s.push.Run(100 * time.Millisecond)
	<-closeSig
	log.Info("%v模块已停止...", s.GetType())
}
func (s *Activity) OnDestroy() {
	//一定别忘了继承
	s.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", s.GetType())
}
func (s *Activity) ReceiveTotalChargeActivity(session gate.Session, msg map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	if msg["ActivityID"] == nil{
		return errCode.ServerError.GetI18nMap(), nil
	}
	activityID := msg["ActivityID"].(string)
	return s.impl.ReceiveTotalChargeActivity(session,activityID)
}
func (s *Activity) ReceiveSignInActivity(session gate.Session, msg map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	if msg["ActivityID"] == nil{
		return errCode.ServerError.GetI18nMap(), nil
	}
	activityID := msg["ActivityID"].(string)
	return s.impl.ReceiveSignInActivity(session,activityID)
}
func (s *Activity) ReceiveDayActivity(session gate.Session, msg map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	if msg["ActivityID"] == nil || msg["ActivityType"] == nil{
		return errCode.ServerError.GetI18nMap(), nil
	}
	activityID := msg["ActivityID"].(string)
	activityType := activityStorage.ActivityType(msg["ActivityType"].(string))
	return s.impl.ReceiveDayActivity(session,activityID,activityType)
}
func (s *Activity) ReceiveVipGift(session gate.Session, msg map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	if msg["Level"] == nil{
		return errCode.ServerError.GetI18nMap(), nil
	}
	level,_ := utils.ConvertInt(msg["Level"])
	return s.impl.ReceiveVipGift(session,int(level))
}
func (s *Activity) ReceiveVipGiftAll(session gate.Session, msg map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	return s.impl.ReceiveVipGiftAll(session)
}
func (s *Activity) ReceiveVipWeekGift(session gate.Session, msg map[string]interface{}) (r map[string]interface{}, err error) {
	log.Info("params: %s,ip: %s")
	if msg["level"] == nil{
		return errCode.ServerError.GetI18nMap(), nil
	}
	level,_ := utils.ConvertInt(msg["level"])
	return s.impl.ReceiveVipWeekGift(session,int(level))
}