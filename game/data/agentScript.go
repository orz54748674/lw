package data

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	"vn/storage"
	"vn/storage/agentStorage"
	"vn/storage/gameStorage"
	"vn/storage/userStorage"
)

type AgentScript struct {
}
func (s *AgentScript)UpdateUser(user userStorage.User) {
	c := common.GetMongoDB().C("user")
	query := bson.M{"_id":user.Oid}
	if err := c.Update(query, &user);err !=nil{
		log.Error(err.Error())
	}
	common.GetMysql().Where("oid=?",user.Oid.Hex()).Updates(&user)
}
func (s *AgentScript) start() {
	start := time.Now()
	query := agentStorage.QueryInviteData()
	agentVipExpire, err := utils.ConvertInt(storage.QueryConf(storage.KAgentVipExpire))
	if err != nil {
		log.Error(err.Error())
		return
	}
	now := utils.Now()
	thatTime := time.Unix(now.Unix()-agentVipExpire*86400, 0)
	//size := len(*agentConf)
	for _, inviteData := range *query {
		uids := convertOid2Str(inviteData.Users)
		count := gameStorage.QueryBetRecordByUsers(uids, thatTime)
		agent := agentStorage.QueryAgent(inviteData.Oid)
		if agent == nil {
			agent = &agentStorage.Agent{
				Oid:      inviteData.Oid,
				Level:    1,
				Count:    count,
				UpdateAt: utils.Now(),
			}
			agentStorage.InsertAgent(agent)
		}else{
			agentStorage.UpsertAgent(inviteData.Oid, 1, count)
		}
	}
	end := time.Now()
	log.Info("finished agent work spent time: %v", end.Sub(start))
}

func convertOid2Str(ids []primitive.ObjectID) []string {
	var strArray []string
	for _, oid := range ids {
		strArray = append(strArray, oid.Hex())
	}
	return strArray
}

func (AgentScript) parseMyVipData()  {
	//now := time.Now()
	//todayStr := utils.GetDateStr(now)
	//allAgent := agentStorage.
	//myVipData := agentStorage.QueryAgentVipData(todayStr)
	//1,查出所有 代理。

}