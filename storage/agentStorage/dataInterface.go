package agentStorage

import (
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
)

type AgentInfo struct {
	TodayProfit  int64
	SumProfit    int64
	InvitedCount int64
	AliveCount   int
	BetAmount1   int64
	BetAmount2   int64
	BetAmount3   int64
	Profit1      int64
	Profit2      int64
	Profit3      int64
}

type InviteData struct { //搜索结果集
	Oid   primitive.ObjectID   `bson:"_id,omitempty" json:"Oid"`
	Count int64                `bson:"Count"`
	Users []primitive.ObjectID `bson:"Users"`
}

func QueryTodayProfit(agentUid primitive.ObjectID, level int) int64 { //0是总计
	c := common.GetMongoDB().C(cAgentIncome)
	var pipe []bson.D
	if level == 0 {
		pipe = mongo.Pipeline{
			{{"$match", bson.M{"CreateAt": bson.M{"$gte": utils.GetTodayTime()}, "AgentUid": agentUid}}},
			{{"$group", bson.M{"_id": "$AgentUid", "SumProfit": bson.M{"$sum": "$Amount"}}}},
		}
	} else {
		pipe = mongo.Pipeline{
			{{"$match", bson.M{"CreateAt": bson.M{"$gte": utils.GetTodayTime()}, "AgentUid": agentUid, "Level": level}}},
			{{"$group", bson.M{"_id": "$AgentUid", "SumProfit": bson.M{"$sum": "$Amount"}}}},
		}
	}

	var res map[string]interface{}
	if err := c.Pipe(pipe).One(&res); err != nil {
		//log.Error(err.Error())
		return 0
	}
	sumProfit, _ := utils.ConvertInt(res["SumProfit"])
	return sumProfit
}
func QueryTodayProfitBet(agentUid primitive.ObjectID, level int) int64 { //0是总计  查询今日佣金下注
	c := common.GetMongoDB().C(cAgentIncome)
	var pipe []bson.D
	if level == 0 {
		pipe = mongo.Pipeline{
			{{"$match", bson.M{"CreateAt": bson.M{"$gte": utils.GetTodayTime()}, "AgentUid": agentUid}}},
			{{"$group", bson.M{"_id": "$AgentUid", "SumBet": bson.M{"$sum": "$BetAmount"}}}},
		}
	} else {
		pipe = mongo.Pipeline{
			{{"$match", bson.M{"CreateAt": bson.M{"$gte": utils.GetTodayTime()}, "AgentUid": agentUid, "Level": level}}},
			{{"$group", bson.M{"_id": "$AgentUid", "SumBet": bson.M{"$sum": "$BetAmount"}}}},
		}
	}

	var res map[string]interface{}
	if err := c.Pipe(pipe).One(&res); err != nil {
		//log.Error(err.Error())
		return 0
	}
	sumBet, _ := utils.ConvertInt(res["SumBet"])
	return sumBet
}
