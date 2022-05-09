package bean

import (
	"time"
	"vn/common"
	"vn/framework/mqant/log"
)

type AgentMemberData2 struct {
	ID                int64
	Date              string `gorm:"index:idx_member"`
	Uid               string `gorm:"index:idx_member"`
	BelongAgent       string //属于推广
	Account           string
	Sub1Users	  	  int  //下1级人数
	Sub1Bets	      int64  //下1级下注流水
	Sub1Profit	      int64  //下1级佣金
	Sub2Users	      int  //下2级人数
	Sub2Bets	      int64  //下2级下注流水
	Sub2Profit	      int64  //下2级佣金
	Sub3Users	      int  //下3级人数
	Sub3Bets	      int64  //下3级下注流水
	Sub3Profit	      int64  //下3级佣金
	TotalProfit		  int64  //总佣金
	UpdateAt          time.Time
	CreateAt          time.Time
}

func (s *AgentMemberData2) Save() {
	if err := common.GetMysql().Save(s).Error; err != nil {
		log.Error(err.Error())
	}
}
func (s *AgentMemberData2) TableName() string {
	return "agent_member_data_2"
}

