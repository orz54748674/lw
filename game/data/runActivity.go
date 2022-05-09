package data

import (
	"errors"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/game/data/bean"
	"vn/storage/activityStorage"
	"vn/storage/agentStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"

	"gorm.io/gorm"
)

var csBankend activityStorage.ActivityType = "csBankend"

var allActivityTypes []activityStorage.ActivityType = []activityStorage.ActivityType{
	activityStorage.FirstCharge,
	activityStorage.BindPhone,
	activityStorage.GiftCode,
	activityStorage.TotalCharge,
	activityStorage.SignIn,
	activityStorage.Encouragement,
	activityStorage.DayCharge,
	activityStorage.DayGame,
	activityStorage.DayInvite,
	csBankend,
}

type RunActivity struct {
	dayStart time.Time
	dayEnd   time.Time
	uid      string
}

var runningActivity = false

func (s *RunActivity) Start() {
	if runningActivity {
		log.Warning("data activity still running...")
		return
	}
	runningActivity = true

	_ = common.GetMysql().AutoMigrate(&bean.DataActivity{})
	_ = common.GetMysql().AutoMigrate(&bean.DataActivityReport{})
	err, lastTime := s.queryLastTime()
	if err != nil {
		log.Error(err.Error())
		return
	}
	s.dayStart, _ = utils.GetDayStartTime(lastTime)
	for (utils.Now().Unix() - s.dayStart.Unix()) > 0 {
		s.dayEnd = utils.GetDayEndTimeByStart(s.dayStart)
		log.Info("runningActivity dayStart: %v , dayEnd: %v", s.dayStart, s.dayEnd)
		s.updateOneDay()
		overview := RunActivityOverview{
			dayStart: s.dayStart,
			dayEnd:   s.dayEnd,
		}
		overview.updateOneDay()
		s.dayStart = time.Unix(s.dayStart.Unix()+86400, 0)
	}

	runningActivity = false
}

func (s *RunActivity) updateOneDay() {
	db := common.GetMysql().Model(&activityStorage.ActivityRecord{})
	db.Where("create_at BETWEEN ? AND ?", s.dayStart, s.dayEnd)
	var uid []uidStruct
	db.Select("DISTINCT uid").Find(&uid)

	for _, id := range uid {
		s.uid = id.Uid
		s.ParserOneDay()
		// return
	}
}

func (s *RunActivity) ParserOneDay() {
	data := s.getData()
	data.FirstCharge = s.querySumByType(activityStorage.FirstCharge)
	data.BindPhone = s.querySumByType(activityStorage.BindPhone)
	data.GiftCode = s.querySumByType(activityStorage.GiftCode)
	data.TotalCharge = s.querySumByType(activityStorage.TotalCharge)
	data.SignIn = s.querySumByType(activityStorage.SignIn)
	data.Encouragement = s.querySumByType(activityStorage.Encouragement)
	data.DayCharge = s.querySumByType(activityStorage.DayCharge)
	data.DayGame = s.querySumByType(activityStorage.DayGame)
	data.DayInvite = s.querySumByType(activityStorage.DayInvite)
	data.VipChargeGet = s.querySumByType(activityStorage.VipChargeGet)
	data.VipWeek = s.querySumByType(activityStorage.VipWeek)
	data.Vip = s.querySumByType(activityStorage.Vip)
	data.TurnTable = s.querySumByType(activityStorage.TurnTable)
	data.AgentBalance = s.queryDayAgentBalance()
	data.CsBankend = s.getCsBalanceOp()
	data.Sum = data.FirstCharge + data.BindPhone + data.GiftCode +
		data.TotalCharge + data.SignIn + data.Encouragement +
		data.DayCharge + data.DayGame + data.DayInvite + data.VipChargeGet +
		data.VipWeek + data.Vip + data.TurnTable
	data.Save()
}
func (s *RunActivity) getCsBalanceOp() int64 {
	db := common.GetMysql().Model(&payStorage.BalanceChangeLog{})
	db.Where("uid =? AND create_at BETWEEN ? AND ?", s.uid, s.dayStart, s.dayEnd)
	var sum sumTmp
	if err := db.Select("SUM(`amount`) sum_amount").Find(&sum).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err.Error())
	}
	return sum.SumAmount
}

func (s *RunActivity) queryDayAgentBalance() int64 {
	db := common.GetMysql().Model(&agentStorage.AgentMemberData{})
	date := utils.GetCnDate(s.dayStart)
	var agentData agentStorage.AgentMemberData
	db.Where("date =? AND uid =?", date, s.uid)
	if err := db.First(&agentData).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err.Error())
	}
	return agentData.TodayAgentBalance
}
func (s *RunActivity) querySumByType(t activityStorage.ActivityType) int64 {
	db := common.GetMysql().Model(&activityStorage.ActivityRecord{})
	db.Where("create_at BETWEEN ? AND ? AND type =? AND uid=?",
		s.dayStart, s.dayEnd, t, s.uid)
	var sum sumTmp
	if err := db.Select("SUM(`get`) sum_amount").Find(&sum).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err.Error())
	}
	return sum.SumAmount
}

func (RunActivity) queryLastTime() (error, time.Time) {
	db := common.GetMysql().Model(&bean.DataActivity{})
	var data bean.DataActivity
	db.Order("id desc").First(&data)
	if data.ID != 0 {
		return nil, data.UpdateAt
	}
	db2 := common.GetMysql().Model(&activityStorage.ActivityRecord{})
	var record activityStorage.ActivityRecord
	db2.First(&record)
	if record.Uid != "" {
		return nil, record.CreateAt
	}
	return errors.New("no data"), time.Now()
}

func (s *RunActivity) getData() bean.DataActivity {
	db := common.GetMysql().Model(&bean.DataActivity{})
	date := utils.GetCnDate(s.dayStart)

	var data bean.DataActivity
	db.Where("date = ? AND uid = ?",
		date, s.uid).First(&data)
	if data.ID == 0 {
		user := userStorage.QueryUserId(utils.ConvertOID(s.uid))
		data.Date = date
		data.Uid = s.uid
		data.Account = user.Account
		data.CreateAt = utils.Now()
		data.UpdateAt = data.CreateAt
	}
	//if !utils.IsToday(s.dayEnd) {
	if utils.Now().Unix()-s.dayEnd.Unix() > 0 {
		data.UpdateAt = s.dayEnd
	} else {
		data.UpdateAt = utils.Now()
	}
	return data
}

type RunActivityOverview struct {
	dayStart time.Time
	dayEnd   time.Time
	data     *bean.DataActivityReport
}

func (s *RunActivityOverview) updateOneDay() {
	data := s.getData()
	s.data = &data
	s.parserData()
	data.FirstChargePeople = s.queryPeopleByType(activityStorage.FirstCharge)
	data.BindPhonePeople = s.queryPeopleByType(activityStorage.BindPhone)
	data.GiftCodePeople = s.queryPeopleByType(activityStorage.GiftCode)
	data.TotalChargePeople = s.queryPeopleByType(activityStorage.TotalCharge)
	data.SignInPeople = s.queryPeopleByType(activityStorage.SignIn)
	data.EncouragementPeople = s.queryPeopleByType(activityStorage.Encouragement)
	data.DayChargePeople = s.queryPeopleByType(activityStorage.DayCharge)
	data.DayGamePeople = s.queryPeopleByType(activityStorage.DayGame)
	data.DayInvitePeople = s.queryPeopleByType(activityStorage.DayInvite)
	data.VipPeople = s.queryPeopleByType(activityStorage.Vip)
	data.VipWeekPeople = s.queryPeopleByType(activityStorage.VipWeek)
	data.VipChargeGetPeople = s.queryPeopleByType(activityStorage.VipChargeGet)
	data.TurnTablePeople = s.queryPeopleByType(activityStorage.TurnTable)
	data.EncouragementCount = s.queryCountByType(activityStorage.Encouragement)
	data.DayGameCount = s.queryCountByType(activityStorage.DayGame)
	//data.TotalCharge1, data.TotalCharge1People = s.queryTotalCharge(1)
	//data.TotalCharge2, data.TotalCharge2People = s.queryTotalCharge(2)
	//data.TotalCharge3, data.TotalCharge3People = s.queryTotalCharge(3)
	//data.TotalCharge4, data.TotalCharge4People = s.queryTotalCharge(4)
	//data.TotalCharge5, data.TotalCharge5People = s.queryTotalCharge(5)
	//data.TotalCharge6, data.TotalCharge6People = s.queryTotalCharge(6)
	data.Save()
}
func (s *RunActivityOverview) queryTotalCharge(no int) (sumAmount, people int64) {
	allTotalChargeConf := getALlTotalChargeConf()
	if no-1 >= len(allTotalChargeConf) {
		log.Error("RunActivityOverview query activity TotalChargeConf error. len:%v", len(allTotalChargeConf))
		return
	}
	conf := allTotalChargeConf[no-1]
	db := common.GetMysql().Model(&activityStorage.ActivityRecord{})
	db.Where("activity_id=? AND create_at BETWEEN ? AND ?", conf.Oid.Hex(), s.dayStart, s.dayEnd)
	var uids []uidStruct
	if err := db.Select("DISTINCT uid").Find(&uids).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err.Error())
	}
	people = int64(len(uids))
	var sum sumTmp
	if err := db.Select("SUM(`get`) sum_amount").Find(&sum).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err.Error())
	}
	sumAmount = sum.SumAmount
	return
}
func (s *RunActivityOverview) parserData() {
	db := common.GetMysql().Model(&bean.DataActivity{})
	date := utils.GetCnDate(s.dayStart)
	db.Where("date =?", date)
	var sumData bean.DataActivity
	db.Select("SUM(`first_charge`) first_charge,SUM(`bind_phone`) bind_phone,SUM(`gift_code`) gift_code,SUM(`total_charge`) total_charge," +
		"SUM(`sign_in`) sign_in,SUM(`encouragement`) encouragement,SUM(`day_charge`) day_charge,SUM(`day_game`) day_game,SUM(`day_invite`) day_invite," +
		"SUM(`agent_balance`) agent_balance,SUM(`cs_bankend`) cs_bankend,SUM(`vip_charge_get`) vip_charge_get,SUM(`vip_week`) vip_week,SUM(`sum`) sum," +
		"SUM(`vip`) vip,SUM(`turn_table`) turn_table").
		Find(&sumData)
	s.data.FirstCharge = sumData.FirstCharge
	s.data.BindPhone = sumData.BindPhone
	s.data.GiftCode = sumData.GiftCode
	s.data.TotalCharge = sumData.TotalCharge
	s.data.SignIn = sumData.SignIn
	s.data.Encouragement = sumData.Encouragement
	s.data.DayCharge = sumData.DayCharge
	s.data.DayGame = sumData.DayGame
	s.data.DayInvite = sumData.DayInvite
	s.data.Vip = sumData.Vip
	s.data.VipWeek = sumData.VipWeek
	s.data.VipChargeGet = sumData.VipChargeGet
	s.data.TurnTable = sumData.TurnTable
	s.data.AgentBalance = sumData.AgentBalance
	s.data.CsBankend = sumData.CsBankend
	s.data.Sum = sumData.Sum
}
func (s *RunActivityOverview) getData() bean.DataActivityReport {
	db := common.GetMysql().Model(&bean.DataActivityReport{})
	date := utils.GetCnDate(s.dayStart)

	var data bean.DataActivityReport
	db.Where("date = ?",
		date).First(&data)
	if data.ID == 0 {
		data.Date = date
		data.CreateAt = utils.Now()
		data.UpdateAt = data.CreateAt
	}
	if utils.Now().Unix()-s.dayEnd.Unix() > 0 {
		data.UpdateAt = s.dayEnd
	} else {
		data.UpdateAt = utils.Now()
	}
	return data
}

func (s *RunActivityOverview) queryPeopleByType(t activityStorage.ActivityType) int64 {
	db := common.GetMysql().Model(&activityStorage.ActivityRecord{})
	db.Where("create_at BETWEEN ? AND ? AND type =?",
		s.dayStart, s.dayEnd, t)
	var uids []uidStruct
	if err := db.Select("DISTINCT uid").Find(&uids).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err.Error())
	}
	return int64(len(uids))
}

func (s *RunActivityOverview) queryCountByType(t activityStorage.ActivityType) int64 {
	db := common.GetMysql().Model(&activityStorage.ActivityRecord{})
	db.Where("create_at BETWEEN ? AND ? AND type =?",
		s.dayStart, s.dayEnd, t)
	var c int64
	db.Count(&c)
	return c
}

var totalChargeConfAll []activityStorage.ActivityTotalChargeConf

func getALlTotalChargeConf() []activityStorage.ActivityTotalChargeConf {
	if len(totalChargeConfAll) == 0 {
		totalChargeConfAll = activityStorage.QueryActivityTotalChargeConf()
	}
	return totalChargeConfAll
}
