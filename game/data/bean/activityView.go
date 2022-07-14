package bean

import (
	"time"
	"vn/common"
	"vn/framework/mqant/log"
)

type DataActivity struct {
	ID            uint64
	Date          string
	Account       string
	Uid           string
	FirstCharge   int64
	BindPhone     int64
	GiftCode      int64
	TotalCharge   int64
	SignIn        int64
	Encouragement int64
	DayCharge     int64
	DayGame       int64
	DayInvite     int64
	AgentBalance  int64
	CsBankend     int64
	VipChargeGet  int64
	VipWeek       int64
	Vip           int64
	TurnTable     int64
	Sum           int64
	CreateAt      time.Time
	UpdateAt      time.Time
}

func (s *DataActivity) Save() {
	if err := common.GetMysql().Save(s).Error; err != nil {
		log.Error(err.Error())
	}
}
func (DataActivity) TableName() string {
	return "data_activity"
}

type DataActivityReport struct {
	ID                  uint64
	Date                string
	FirstCharge         int64
	FirstChargePeople   int64
	BindPhone           int64
	BindPhonePeople     int64
	GiftCode            int64
	GiftCodePeople      int64
	TotalCharge         int64
	TotalChargePeople   int64
	TotalCharge1        int64
	TotalCharge1People  int64
	TotalCharge2        int64
	TotalCharge2People  int64
	TotalCharge3        int64
	TotalCharge3People  int64
	TotalCharge4        int64
	TotalCharge4People  int64
	TotalCharge5        int64
	TotalCharge5People  int64
	TotalCharge6        int64
	TotalCharge6People  int64
	SignIn              int64
	SignInPeople        int64
	Encouragement       int64
	EncouragementPeople int64
	EncouragementCount  int64
	DayCharge           int64
	DayChargePeople     int64
	DayGame             int64
	DayGamePeople       int64
	DayGameCount        int64
	DayInvite           int64
	DayInvitePeople     int64
	VipChargeGet        int64
	VipChargeGetPeople  int64
	VipWeek             int64
	VipWeekPeople       int64
	Vip                 int64
	VipPeople           int64
	TurnTable           int64
	TurnTablePeople     int64
	AgentBalance        int64
	CsBankend           int64
	Sum                 int64
	CreateAt            time.Time
	UpdateAt            time.Time
}

func (s *DataActivityReport) Save() {
	if err := common.GetMysql().Save(s).Error; err != nil {
		log.Error(err.Error())
	}
}
func (DataActivityReport) TableName() string {
	return "data_activity_report"
}
