package bean

import (
	"time"
	"vn/common"
	"vn/framework/mqant/log"
)

type Risk struct {
	ID                    uint64
	Date                  string
	Uid                   string
	Account               string
	ParentAccount         string
	Uuid                  string
	Ip                    string
	DeviceAccountCount    int
	DeviceAccount         string //逗号分割
	IpAccountCount        int
	IpAccount             string
	RecentlyGameCount     int64 //玩了几个游戏
	RecentlyBetCount      int64
	RecentlyBetAmount     int64
	RecentlyChargeCount   int64
	RecentlyChargeAmount  int64
	RecentlyDouDouCount   int64
	RecentlyDouDouAmount  int64
	RecentlyDouDouAccount string
	RecentlyLoginAccount  string
	RegisterTime          time.Time
	CreateAt              time.Time
	UpdateAt              time.Time
}

func (Risk) TableName() string {
	return "data_risk"
}

func (s *Risk) Save() {
	if err := common.GetMysql().Save(s).Error; err != nil {
		log.Error(err.Error())
	}
}
