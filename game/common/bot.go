package common

import (
	"math/rand"
	"sync"
	"vn/common"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
)

type Bot struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	ShowId   int64              `bson:"ShowId"`
	NickName string             `bson:"NickName"`
	Avatar   string             `bson:"Avatar"`
}

var bots []Bot
var onceBot sync.Once
var cBot = "bot"

func RandInt64(min, max int64, rand *rand.Rand) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return rand.Int63n(max-min) + min
}
func RandBotN(num int, rand *rand.Rand) []Bot {
	botRes := make([]Bot, 0)
	if len(bots) < num {
		log.Info("-------bot num error---", len(bots), num)
		return botRes
	}
	for num > 0 {
		r := RandInt64(1, int64(len(bots))+1, rand) - 1
		find := false
		for _, v := range botRes {
			if v.NickName == bots[r].NickName {
				find = true
				break
			}
		}
		if !find {
			botRes = append(botRes, bots[r])
			num--
		}
	}
	return botRes
}
func RandomAndNotIn(num int, ids []primitive.ObjectID, rand *rand.Rand) []Bot {
	botRes := make([]Bot, 0)
	if len(bots) < num {
		log.Info("-------bot num error---", len(bots), num)
		return botRes
	}
	if ids == nil {
		ids = []primitive.ObjectID{}
	}
	for num > 0 {
		r := RandInt64(1, int64(len(bots))+1, rand) - 1
		find := false
		for _, v := range ids {
			if bots[r].Oid == v {
				find = true
				break
			}
		}
		if !find {
			for _, v := range botRes {
				if v.NickName == bots[r].NickName {
					find = true
					break
				}
			}
		}
		if !find {
			botRes = append(botRes, bots[r])
			num--
		}
	}

	return botRes
}
func InitBots() {
	onceBot.Do(func() {
		bots = QueryBots()
	})
}
func QueryBots() []Bot {
	c := common.GetMongoDB().C(cBot)
	var bots []Bot
	if err := c.Find(nil).All(&bots); err != nil {
		log.Info("not found bots: ,err: %v", err)
		return nil
	}
	return bots
}
