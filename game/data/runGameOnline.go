package data

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/log"
)

type GameOnlineLog struct {
	ID           uint64
	Game         string
	OnlinePeople int64
	CreateAt     time.Time
}

func (GameOnlineLog) TableName() string {
	return "data_game_online_log"
}

type RunGameOnline struct {
}

var cUserOnlinePage = "UserOnlinePage"

type gameCount struct {
	GameType string `bson:"_id,omitempty"`
	Count    int64  `bson:"Count"`
}

func (RunGameOnline) Start() {
	c := common.GetMongoDB().C(cUserOnlinePage)
	pipe := mongo.Pipeline{
		{{
			"$group", bson.M{"_id": "$GameType", "Count": bson.M{"$sum": 1}},
		}},
	}
	var onlineCount []gameCount
	if err := c.Pipe(pipe).All(&onlineCount); err != nil {
		log.Error(err.Error())
	}
	for _, online := range onlineCount {
		onlineLog := GameOnlineLog{
			Game:         online.GameType,
			OnlinePeople: online.Count,
			CreateAt:     utils.Now(),
		}
		common.GetMysql().Create(&onlineLog)
	}
	count, _ := c.Find(bson.M{}).Count()
	if count != 0 {
		onlineLog := GameOnlineLog{
			Game:         "all",
			OnlinePeople: count,
			CreateAt:     utils.Now(),
		}
		common.GetMysql().Create(&onlineLog)
	}
}
