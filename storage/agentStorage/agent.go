package agentStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/storage/userStorage"
)

type Agent struct {
	Oid       primitive.ObjectID `bson:"_id,omitempty" json:"Oid"` //UID
	Level     int                `bson:"Level"`
	Count     int                `bson:"Count"` //活跃会员
	SumIncome int64              `bson:"SumIncome"`
	AdminId   int64              `bson:"AdminId"`
	Remark    string             `bson:"Remark"`
	InviteImg string             `bson:"InviteImg"`
	Theme     int                `bson:"Theme"`
	UpdateAt  time.Time          `bson:"UpdateAt"`
}

type AgentConf struct {
	Level             int `bson:"Level"`
	ProfitPerThousand int `bson:"ProfitPerThousand"`
}

type AgentIncome struct {
	ID        int64              `bson:"-"`
	AgentUid  primitive.ObjectID `bson:"AgentUid"`
	VipUid    primitive.ObjectID `bson:"VipUid"`
	Level     int                `bson:"Level"`
	Amount    int64              `bson:"Amount"`
	BetAmount int64              `bson:"BetAmount"`
	GameType  game.Type          `bson:"GameType"`
	CreateAt  time.Time          `bson:"CreateAt"`
}

func (AgentIncome) TableName() string {
	return "agent_income"
}

var (
	cAgentConf   = "agentConf"
	cAgent       = "agent"
	cAgentIncome = "agentIncome"
	ThemeDefault = 0
	ThemeCustom  = 1
)

func UpsertAgent(id primitive.ObjectID, level int, count int) {
	c := common.GetMongoDB().C(cAgent)
	find := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"Level": level, "UpdateAt": utils.Now(), "Count": count}}
	if _, err := c.Upsert(find, update); err != nil {
		log.Error(err.Error())
	}
}
func InsertAgent(agent *Agent) {
	c := common.GetMongoDB().C(cAgent)
	if err := c.Insert(agent); err != nil {
		log.Error(err.Error())
	}
}
func incAgentIncome(uid primitive.ObjectID, amount int64) {
	c := common.GetMongoDB().C(cAgent)
	find := bson.M{"_id": uid}
	update := bson.M{"$inc": bson.M{"SumIncome": amount}}
	if _, err := c.Upsert(find, update); err != nil {
		log.Info(err.Error())
	}
}
func NewAgentIncome(agentUid primitive.ObjectID, vipUid primitive.ObjectID, amount int64, betAmount int64, gameType game.Type, level int) {
	agentIncome := AgentIncome{
		AgentUid:  agentUid,
		VipUid:    vipUid,
		Amount:    amount,
		BetAmount: betAmount,
		GameType:  gameType,
		Level:     level,
		CreateAt:  utils.Now(),
	}
	c := common.GetMongoDB().C(cAgentIncome)
	if err := c.Insert(&agentIncome); err != nil {
		log.Error(err.Error())
	}
	incAgentIncome(agentUid, amount)
	userStorage.IncUserSumAgentBalance(agentUid, amount)
	common.ExecQueueFunc(func() {
		common.GetMysql().Create(&agentIncome)
	})
}
func DelAgent(id primitive.ObjectID) {

}
func QueryAgent(uid primitive.ObjectID) *Agent {
	c := common.GetMongoDB().C(cAgent)
	find := bson.M{"_id": uid}
	var agent Agent
	if err := c.Find(find).One(&agent); err != nil {
		//log.Error(err.Error())
		return nil
	}
	return &agent
}

func InitAgent(incDataExpireDay time.Duration) {
	c := common.GetMongoDB().C(cAgentIncome)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create AgentIncome Index: %s", err)
	}
	if agentConf := QueryAllAgentConf(); len(*agentConf) == 0 {
		agentConf = newAgentConf()
		insertAgentConf(agentConf)
	}
	_ = common.GetMysql().AutoMigrate(&AgentIncome{})
	_ = common.GetMysql().AutoMigrate(&Invite{})
	_ = common.GetMysql().AutoMigrate(&AgentVipData{})
	_ = common.GetMysql().AutoMigrate(&UserFinanceData{})
	_ = common.GetMysql().AutoMigrate(&AgentMemberData{})
}
func newAgentConf() *[]AgentConf {
	agentConf := []AgentConf{
		{
			Level:             1,
			ProfitPerThousand: 6,
		},
		{
			Level:             2,
			ProfitPerThousand: 3,
		},
		{
			Level:             3,
			ProfitPerThousand: 1,
		},
	}
	return &agentConf
}
func insertAgentConf(agentConf *[]AgentConf) {
	c := common.GetMongoDB().C(cAgentConf)
	var data = make([]interface{}, len(*agentConf))
	for i, d := range *agentConf {
		data[i] = d
	}
	if err := c.InsertMany(data); err != nil {
		log.Error(err.Error())
	}
}
func QueryAllAgentConf() *[]AgentConf {
	c := common.GetMongoDB().C(cAgentConf)
	var agentConf []AgentConf
	if err := c.Find(bson.M{}).Sort("-Level").All(&agentConf); err != nil {
		return nil
	}
	return &agentConf
}
func QueryAgentConf(level int) *AgentConf {
	c := common.GetMongoDB().C(cAgentConf)
	find := bson.M{"Level": level}
	var agentConf AgentConf
	if err := c.Find(find).One(&agentConf); err != nil {
		//log.Error(err.Error())
	}
	return &agentConf
}
