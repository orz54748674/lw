package bean

import (
	"time"
	"vn/common"
	"vn/framework/mqant/log"
	"vn/game"
)

type Overview struct {
	ID                   uint64
	Date                 string
	Platform             string
	Game                 game.Type
	Channel              string
	DeviceNew            int64
	GameStartPeople      int64 //启动设备
	Dnu                  int64 //新增用户
	Dau                  int64 //活跃用户
	Charge               int64 //充值金额
	ChargePeople         int64 //充值人数
	FirstDayChargePeople int64 //首次充值人数
	NewCharge int64 //新增充值金额
	NewChargePeople int64 //新增充值人数
	DouDou               int64
	DouDouPeople         int64
	ActivityGive         int64 //奖励金额
	Pur                  int   //付费率 百分比
	Arpu                 int64 //平均用户付费
	Arpp                 int64 //平均付费用户付费金额
	Pcu                  int64 //最高同时在线数
	Acu                  int64 //平均在线数
	Urr                  int   // 次日留存
	Urr3                 int   //非连续3留
	Urr7                 int
	BetAmount            int64
	Income               int64
	WinRate              int64 //胜率，总局数的系统赢次数/总局数
	GamePlayPeople       int64 //游戏玩耍人数
	GameCount            int64 //游戏玩耍次数
	UpdateAt             time.Time
	CreateAt             time.Time
}

func (Overview) TableName() string {
	return "data_overview"
}

func (s *Overview) Save() {
	if err := common.GetMysql().Save(s).Error; err != nil {
		log.Error(err.Error())
	}
}
