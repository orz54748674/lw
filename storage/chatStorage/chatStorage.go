package chatStorage

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/storage/userStorage"
)

type Message struct {
	Oid         primitive.ObjectID    `bson:"_id,omitempty" json:"Oid"`
	MsgId    string           `bson:"MsgId"`
	GroupId  string           `bson:"GroupId"`
	Content  string           `bson:"Content"`
	FromUid  string           `bson:"FromUid"`
	FromUser userStorage.User `bson:"FromUser"`
	CreateAt time.Time        `bson:"CreateAt"`
}

type Group struct {
	Uid     string `bson:"Oid"`
	GroupId string `bson:"GroupId"`
}
type ChatBotMsgList struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	GameType game.Type           `bson:"GameType"`
	Msg      string              `bson:"Msg"`
}
var (
	cChatGroup = "chatGroup"
	cChatMsg   = "chatMsg"
	cChatBotMsgList = "chatBotMsgList"
)

func AddGroup(uid string, groupId string) {
	c := common.GetMongoDB().C(cChatGroup)
	query := bson.M{"Oid": uid, "GroupId": groupId}
	update := bson.M{"Oid": uid, "GroupId": groupId}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
}
func QueryGroup(groupId string) []string {
	c := common.GetMongoDB().C(cChatGroup)
	query := bson.M{"GroupId": groupId}
	var groups []Group
	if err := c.Find(query).All(&groups); err != nil {
		log.Error(err.Error())
	}
	res := make([]string,0)
	for _, g := range groups {
		res = append(res, g.Uid)
	}
	return res
}
func ExitGroup(uid string, groupId string) {
	c := common.GetMongoDB().C(cChatGroup)
	query := bson.M{"Oid": uid, "GroupId": groupId}
	if err := c.Remove(query); err != nil {
		log.Info(err.Error())
	}
}

func Disconnect(uid string) {
	c := common.GetMongoDB().C(cChatGroup)
	find := bson.M{"Oid": uid}
	if _, err := c.RemoveAll(find); err != nil {
		log.Error(err.Error())
	}
}
func SaveMsg(msg *Message) {
	c := common.GetMongoDB().C(cChatMsg)
	//query := bson.M{"MsgId": msg.MsgId}
	//if _, err := c.Upsert(query, msg); err != nil {
	//	log.Error(err.Error())
	//}
	if err := c.Insert(msg); err != nil {
		log.Error(err.Error())
	}
}
func QueryMsgList(groupId string, size int64) *[]Message {
	c := common.GetMongoDB().C(cChatMsg)
	query := bson.M{"GroupId": groupId}
	var msgList []Message
	if err := c.Find(query).Sort("-_id").Limit(int(size)).All(&msgList); err != nil {
		log.Error(err.Error())
	}
	return &msgList
}

func Init(incDataExpireDay time.Duration) {
	c := common.GetMongoDB().C(cChatGroup)
	_, _ = c.RemoveAll(bson.M{})

	c2 := common.GetMongoDB().C(cChatMsg)
	key := bsonx.Doc{{Key: "CreateAt",Value: bsonx.Int32(1)}}
	if err := c2.CreateIndex(key,options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second)));err != nil{
		log.Error("create ChatMsg Index: %s",err)
	}
	key = bsonx.Doc{{Key: "GroupId",Value: bsonx.Int32(1)}}
	if err := c2.CreateIndex(key,options.Index());err != nil{
		log.Error("create ChatMsg Index: %s",err)
	}
}
func InitChatBotMsgList() {
	c := common.GetMongoDB().C(cChatBotMsgList)
	count, err := c.Find(bson.M{}).Count()
	if err != nil || count == 0 {
		readConf2Db()
	}
	log.Info("bot count: %v", count)
}
func readConf2Db() {
	c := common.GetMongoDB().C(cChatBotMsgList)
	path := utils.GetProjectAbsPath()
	for _,v := range game.ChatBotGame{
		f, err := ioutil.ReadFile(filepath.Join(path, fmt.Sprintf("bin/chat/%s.txt",v)))
		if err != nil {
			log.Error("read fail: %v", err)
		}else{
			msg := string(f)
			res := strings.Split(msg, "\n")
			var data1 []ChatBotMsgList
			for _, msg := range res {
				if msg != ""{
					data1 = append(data1,ChatBotMsgList{
						GameType: v,
						Msg: msg,
					})
				}
			}
			var data2 = make([]interface{},len(data1))
			for k1,v1 := range data1{
				data2[k1] = v1
			}
			if err := c.InsertMany(data2); err != nil {
				log.Error(err.Error())
			}
			log.Info("init chat msg count: %v", len(res))
		}
	}
}
func RandomBotChatN(gameType game.Type,n int) []ChatBotMsgList {
	c := common.GetMongoDB().C(cChatBotMsgList)
	pipe := c.Pipe(mongo.Pipeline{
		{{"$sample", bson.M{"size": n}}},
		{{"$match", bson.M{"GameType": gameType}}},
	})
	var all []ChatBotMsgList
	if err := pipe.All(&all); err != nil {
		log.Error(err.Error())
	}
	return all
}
func QueryBotsChat() []ChatBotMsgList {
	c := common.GetMongoDB().C(cChatBotMsgList)
	var botsChat []ChatBotMsgList
	if err := c.Find(nil).All(&botsChat); err != nil {
		log.Info("not found bots: ,err: %v", err)
		return nil
	}
	return botsChat
}