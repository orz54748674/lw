package botStorage

import (
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strings"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	"vn/storage/userStorage"
)

type Bot struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	ShowId   int64              `bson:"ShowId"`
	NickName string             `bson:"NickName"`
	Avatar   string             `bson:"Avatar"`
}

var (
	cBot = "bot"
)

func InitBot() {
	c := common.GetMongoDB().C(cBot)
	count, err := c.Find(bson.M{}).Count()
	if err != nil || count == 0 {
		readConf2Db()
	}
	log.Info("bot count: %v", count)
}
func newBot(name string) Bot {
	bot := Bot{
		ShowId:   getBotUid(),
		NickName: name,
		Avatar:   userStorage.GetSystemAvatar(),
	}
	return bot
}
func getBotUid() int64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	uid := utils.RandomNum(5, r)
	c := common.GetMongoDB().C(cBot)
	var bot Bot
	if err := c.Find(bson.M{"ShowId": uid}).One(&bot); err != nil {
		return uid
	}
	return getBotUid()
}
func insertBot(bot *Bot) {
	c := common.GetMongoDB().C(cBot)
	if err := c.Insert(bot); err != nil {
		log.Error(err.Error())
	}
}
func insertAllBot(bot []Bot) {
	c := common.GetMongoDB().C(cBot)
	var data = make([]interface{}, len(bot))
	for i, v := range bot {
		data[i] = v
	}
	if err := c.InsertMany(data); err != nil {
		log.Error(err.Error())
	}
}
func readConf2Db() {
	path := utils.GetProjectAbsPath()
	f, err := ioutil.ReadFile(filepath.Join(path, "bin/botName.txt"))
	if err != nil {
		log.Error("read fail: %v", err)
		return
	}
	names := string(f)
	res := strings.Split(names, "\n")
	var data []Bot
	for _, name := range res {
		if name != "" {
			data = append(data, newBot(name))
		}
	}
	insertAllBot(data)
	log.Info("init bot name count: %v", len(res))
}

//func RandomN(n int) []Bot {
//	c := common.GetMongoDB().C(cBot)
//	pipe := c.Pipe(mongo.Pipeline{
//		{{"$sample", bson.M{"size": n}}},
//	})
//	var all []Bot
//	if err := pipe.All(&all); err != nil {
//		log.Error(err.Error())
//	}
//	return all
//}
//func RandomAndNotIn(n int, ids []primitive.ObjectID) []Bot {
//	c := common.GetMongoDB().C(cBot)
//	if ids == nil{
//		ids = []primitive.ObjectID{}
//	}
//	query := mongo.Pipeline{
//		{{"$match", bson.M{"_id": bson.M{"$nin": ids}}}},
//		{{"$sample", bson.M{"size": n}}},
//	}
//	var all []Bot
//	if err := c.Pipe(query).All(&all); err != nil {
//		log.Error(err.Error())
//	}
//	return all
//}
func Query(uid int64) *Bot {
	c := common.GetMongoDB().C(cBot)
	var bot Bot
	if err := c.Find(bson.M{"ShowId": uid}).One(&bot); err != nil {
		log.Error(err.Error())
	}
	return &bot
}
func QueryBotByUid(uids []int64) map[int64]Bot {
	c := common.GetMongoDB().C(cBot)
	query := bson.M{"ShowId": bson.M{"$in": uids}}
	var bots []Bot
	if err := c.Find(query).All(&bots); err != nil {
		log.Error(err.Error())
	}
	res := make(map[int64]Bot, len(bots))
	for _, b := range bots {
		res[b.ShowId] = b
	}
	return res
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
