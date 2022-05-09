package data

import (
	"errors"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/game/data/bean"
	"vn/storage/agentStorage"
	"vn/storage/userStorage"
)

type RunAgentData2 struct {
	dayStart time.Time
	dayEnd   time.Time
	uid      string
}

var runningAgentData2 = false

func (s *RunAgentData2) Start() {
	if runningAgentData2 {
		log.Warning("data activity still running...")
		return
	}
	runningAgentData2 = true

	_ = common.GetMysql().AutoMigrate(&bean.AgentMemberData2{})
	err, lastTime := s.queryLastTime()
	if err != nil {
		log.Error(err.Error())
		return
	}
	s.dayStart, _ = utils.GetDayStartTime(lastTime)
	for (utils.Now().Unix() - s.dayStart.Unix()) > 0 {
		s.dayEnd = utils.GetDayEndTimeByStart(s.dayStart)
		log.Info("runningAgentMember2 dayStart: %v , dayEnd: %v", s.dayStart, s.dayEnd)
		s.updateOneDay()
		s.dayStart = time.Unix(s.dayStart.Unix()+86400, 0)
	}

	runningAgentData2 = false
}

func (s *RunAgentData2) updateOneDay() {
	db := common.GetMysql().Model(&agentStorage.AgentIncome{})
	db.Where("create_at BETWEEN ? AND ?", s.dayStart, s.dayEnd)
	var uid []uidStruct
	db.Select("DISTINCT agent_uid,agent_uid AS uid").Find(&uid)
	for _, id := range uid {
		s.uid = id.Uid
		s.ParserOneDay()
		// return
	}
}

type AgentData2 struct {
	SumBet    int64
	SumProfit int64
}

func (s *RunAgentData2) ParserOneDay() {
	data := s.getData()
	invite := agentStorage.QueryInvite(utils.ConvertOID(s.uid))
	if !invite.Oid.IsZero() && !invite.ParentOid.IsZero() {
		baba := userStorage.QueryUserId(invite.ParentOid)
		if baba.Type == userStorage.TypeAgent {
			data.BelongAgent = baba.Account
		}
	}
	dbUser1 := common.GetMysql().Model(&agentStorage.AgentIncome{})
	dbUser1.Where("create_at BETWEEN ? AND ? AND level=? AND agent_uid=?", s.dayStart, s.dayEnd, 1,s.uid)
	var uid []uidStruct
	dbUser1.Select("DISTINCT vip_uid,agent_uid AS uid").Find(&uid)
	data.Sub1Users = len(uid)

	dbUser2 := common.GetMysql().Model(&agentStorage.AgentIncome{})
	dbUser2.Where("create_at BETWEEN ? AND ? AND level=? AND agent_uid=?", s.dayStart, s.dayEnd, 2,s.uid)
	var uid2 []uidStruct
	dbUser2.Select("DISTINCT vip_uid,agent_uid AS uid").Find(&uid2)
	data.Sub2Users = len(uid2)

	dbUser3 := common.GetMysql().Model(&agentStorage.AgentIncome{})
	dbUser3.Where("create_at BETWEEN ? AND ? AND level=? AND agent_uid=?", s.dayStart, s.dayEnd, 3,s.uid)
	var uid3 []uidStruct
	dbUser3.Select("DISTINCT vip_uid,agent_uid AS uid").Find(&uid3)
	data.Sub3Users = len(uid3)

	db1 := common.GetMysql().Model(&agentStorage.AgentIncome{})
	var agentData AgentData2
	db1.Select("sum(`bet_amount`) as sum_bet,sum(`amount`) as sum_profit").
		Where("create_at BETWEEN ? AND ? and agent_uid=? and level=?", s.dayStart, s.dayEnd, s.uid, 1).Find(&agentData)
	data.Sub1Bets = agentData.SumBet
	data.Sub1Profit = agentData.SumProfit

	db2 := common.GetMysql().Model(&agentStorage.AgentIncome{})
	agentData = AgentData2{}
	db2.Select("sum(`bet_amount`) as sum_bet,sum(`amount`) as sum_profit").
		Where("create_at BETWEEN ? AND ? and agent_uid=? and level=?", s.dayStart, s.dayEnd, s.uid, 2).Find(&agentData)
	data.Sub2Bets = agentData.SumBet
	data.Sub2Profit = agentData.SumProfit

	db3 := common.GetMysql().Model(&agentStorage.AgentIncome{})
	agentData = AgentData2{}
	db3.Select("sum(`bet_amount`) as sum_bet,sum(`amount`) as sum_profit").
		Where("create_at BETWEEN ? AND ? and agent_uid=? and level=?", s.dayStart, s.dayEnd, s.uid, 3).Find(&agentData)
	data.Sub3Bets = agentData.SumBet
	data.Sub3Profit = agentData.SumProfit

	data.TotalProfit = data.Sub1Profit + data.Sub2Profit + data.Sub3Profit
	data.Save()
}

func (RunAgentData2) queryLastTime() (error, time.Time) {
	db := common.GetMysql().Model(&bean.AgentMemberData2{})
	var data bean.AgentMemberData2
	db.Order("id desc").First(&data)
	if data.ID != 0 {
		return nil, data.UpdateAt
	}
	db2 := common.GetMysql().Model(&agentStorage.AgentMemberData{})
	var data2 agentStorage.AgentMemberData
	db2.Order("id desc").First(&data2)
	if data2.Uid != "" {
		return nil, data2.CreateAt
	}
	return errors.New("no data"), time.Now()
}

func (s *RunAgentData2) getData() bean.AgentMemberData2 {
	db := common.GetMysql().Model(&bean.AgentMemberData2{})
	date := utils.GetCnDate(s.dayStart)

	var data bean.AgentMemberData2
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
