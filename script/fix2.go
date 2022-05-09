package main

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
	"vn/storage/agentStorage"
	"vn/storage/userStorage"
)

type sumStruct struct {
	sum int64
}
type agentIncomeSum struct {
	AgentUid string
	Sum      int64
}

func fixUserInfoSumAgentBalance() {
	c := common.GetMongoDB().C("userInfo")
	db := common.GetMysql().Model(&agentStorage.AgentIncome{})
	db.Select("agent_uid,SUM(amount) sum").Group("agent_uid")
	var agentIncome []agentIncomeSum
	if err := db.Find(&agentIncome).Error; err != nil {
		log.Error(err.Error())
	}
	for i := 0; i < len(agentIncome); i++ {
		agent := agentIncome[i]
		oid := utils.ConvertOID(agent.AgentUid)
		update := bson.M{"$set": bson.M{"SumAgentBalance": agent.Sum}}
		c.Update(bson.M{"_id": oid}, update)
		db := common.GetMysql().Model(&userStorage.UserInfo{})
		db.Where("oid=?", agent.AgentUid).
			UpdateColumn("sum_agent_balance", agent.Sum)
		log.Info("update oid:%v,agentBalance:%v", agent.AgentUid, agent.Sum)
	}
	// if err := c.Find(nil).All(&userInfo); err != nil {
	// 	log.Error(err.Error())
	// 	return
	// }
	// for _, u := range userInfo {
	//
	// 	var sum sumStruct
	// 	db.Where("agent_uid=?", u.Oid.Hex())
	// 	if err := db.Select("SUM(amount) sum").Find(&sum).Error; err != nil && err != gorm.ErrRecordNotFound {
	// 		log.Error(err.Error())
	// 	} else {
	// 		u.SumAgentBalance = sum.sum
	// 		c.Update(bson.M{"_id": u.Oid}, u)
	// 		updateUserInfo2mysql(u)
	// 	}
	// }

}
func checkUser() {
	log.Info("ccc")
	return
	c := common.GetMongoDB().C("user")
	var users []userStorage.User
	if err := c.Find(bson.M{}).All(&users); err != nil {
		log.Error(err.Error())
		return
	}
	log.Info("user len: %v", len(users))
	for _, u := range users {
		var query userStorage.User
		db := common.GetMysql().Model(&userStorage.User{})
		if err := db.Where("oid=?", u.Oid.Hex()).First(&query).Error; err != nil {
			log.Error("err:", err.Error())
		}
		if query.ID == 0 {
			if err := common.GetMysql().Create(&u).Error; err != nil {
				log.Error(err.Error())
			}
		}
	}
	db := common.GetMysql().Model(&userStorage.User{})
	var count int64
	db.Count(&count)
	log.Info("mysql user Count: %v", count)
}
func checkUserInvite() {
	c := common.GetMongoDB().C("userInvite")
	var invites []agentStorage.Invite
	if err := c.Find(bson.M{}).All(&invites); err != nil {
		log.Error(err.Error())
		return
	}
	log.Info("invites len: %v", len(invites))
	for _, invite := range invites {
		var query agentStorage.Invite
		db := common.GetMysql().Model(&agentStorage.Invite{})
		db.Where("oid=?", invite.Oid.Hex()).First(&query)
		if query.ID == 0 {
			common.GetMysql().Create(&invite)
		}
	}
	db := common.GetMysql().Model(&agentStorage.Invite{})
	var count int64
	db.Count(&count)
	log.Info("mysql Invite Count: %v", count)
}
func syncUserInvite() {
	db := common.GetMysql().Model(&userStorage.User{})
	db.Joins("LEFT JOIN user_invite on user_invite.oid=user.oid")
	var users []userStorage.User
	db.Where("user_invite.oid is NULL").Find(&users)
	for _, u := range users {
		invite := agentStorage.Invite{
			Oid:      u.Oid,
			CreateAt: u.CreateAt,
		}
		c := common.GetMongoDB().C("userInvite")
		if err := c.Insert(&invite); err != nil {
			log.Error(err.Error())
		} else {
			db := common.GetMysql()
			if err := db.Create(&invite).Error; err != nil {
				log.Error(err.Error())
			}
		}
	}
}

func fixAgentMember() {
	dayStart, err := utils.StrFormatTime("yyyy-MM-dd", "2021-06-01")
	if err != nil {
		log.Error(err.Error())
		return
	}
	now := utils.Now().Unix()
	for true {
		if now-dayStart.Unix() < 0 {
			return
		}
		date := utils.GetCnDate(dayStart)
		parserOneDay(date)
		dayStart = time.Unix(dayStart.Unix()+86400, 0)
	}

}

func parserOneDay(date string) {
	db := common.GetMysql()
	var oneDayData []agentStorage.AgentMemberData
	db.Raw("SELECT * FROM agent_member_data2 WHERE date = ?", date).Scan(&oneDayData)
	userData := make(map[string]*agentStorage.AgentMemberData, 0)
	for i, data := range oneDayData {
		if amd, ok := userData[data.Uid]; ok {
			parserRaw(amd, data)
		} else {
			userData[data.Uid] = &oneDayData[i]
		}
	}
	for _, raw := range userData {
		if raw.UpdateAt.IsZero() {
			raw.UpdateAt = time.Unix(0, 0)
		}
		if raw.CreateAt.IsZero() {
			raw.CreateAt = time.Unix(0, 0)
		}
		raw.ID = 0
		db.Create(raw)
	}
}

func parserRaw(amd *agentStorage.AgentMemberData, data agentStorage.AgentMemberData) {
	amd.UpdateAt = data.UpdateAt
	amd.TodayCharge += data.TodayCharge
	amd.TodayDouDou += data.TodayDouDou
	amd.TodayBet += data.TodayBet
	amd.TodayIncome += data.TodayIncome
	amd.TodayAgentBalance += data.TodayAgentBalance
	amd.TodayActivity += data.TodayActivity
	amd.VndBalance = data.VndBalance
	amd.AgentBalance = data.AgentBalance
}
