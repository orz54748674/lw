package gameStorage

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

type GameProfit struct {
	ID         uint64    `bson:"-" json:"-"`
	GameType   game.Type `bson:"GameType"`
	Profit     int64     `bson:"Profit"`     //SystemProfit 系统抽水， 平台明面上的抽水
	BotBalance int64     `bson:"BotBalance"` //系统余额  该余额不够赔付时，要控盘
	BotProfit  int64     `bson:"BotProfit"`  //暗抽金额
}
type GameProfitLog struct {
	ID         int64              `bson:"-" json:"-"`
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	AdminID    string             `bson:"AdminID"`
	GameType   game.Type          `bson:"GameType"`
	BotBalance int64              `bson:"BotBalance"`
	CreateAt   time.Time          `bson:"CreateAt"`
}

func (GameProfit) TableName() string {
	return "game_profit"
}

var (
	cProfit    = "gameProfit"
	cProfitLog = "gameProfitLog"
)

func InitGameProfit() {
	c := common.GetMongoDB().C(cProfit)
	key := bsonx.Doc{{Key: "GameType", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().SetUnique(true)); err != nil {
		log.Error("create GameProfit Index: %s", err)
	}
	log.Info("init GameProfit of mongo db")
	//_ = common.GetMysql().AutoMigrate(&GameProfit{})
}
func IncProfit(uid string, gameType game.Type, amount int64, botAmount int64, botProfit int64) {
	if uid != "" { //uid为空，表示外面已经把陪玩号去掉了,可以用来在外面计算出总和，一次性存进去
		user := userStorage.QueryUserId(utils.ConvertOID(uid))
		if user.Type != userStorage.TypeNormal { //陪玩号不计算
			return
		}
	}
	gameProfit := QueryProfit(gameType)
	betGame := false
	for _, v := range game.BetGame {
		if gameType == v {
			betGame = true
			break
		}
	}
	if gameProfit.BotBalance < -botAmount && botAmount < 0 && betGame { //防止机器人余额为负
		botAmount = 0
		botProfit = 0
	}
	c := common.GetMongoDB().C(cProfit)
	query := bson.M{"GameType": gameType}
	update := bson.M{
		"$inc": bson.M{"Profit": amount,
			"BotBalance": botAmount,
			"BotProfit":  botProfit,
		},
	}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
	//q := QueryProfit(gameType)
	//var gameProfit GameProfit
	//common.GetMysql().FirstOrCreate(&gameProfit,
	//	"game_type=?",gameType)
	//q.ID = gameProfit.ID
	//common.GetMysql().Save(&q)
}

//func UpsertProfit(gameType game.Type, gameProfit *GameProfit) {
//	c := common.GetMongoDB().C(cProfit)
//	query := bson.M{"GameType": gameType}
//	update := structs.Map(gameProfit)
//	if _, err := c.Upsert(query, update); err != nil {
//		log.Error(err.Error())
//	}
//}
func QueryProfit(gameType game.Type) *GameProfit {
	c := common.GetMongoDB().C(cProfit)
	query := bson.M{"GameType": gameType}
	var gameProfit GameProfit
	if err := c.Find(query).One(&gameProfit); err != nil {
		log.Error(err.Error())
	}
	return &gameProfit
}
func InitGameProfitLog(incDataExpireDay time.Duration) {
	//c := common.GetMongoDB().C(cProfitLog)
	//key := bsonx.Doc{{Key: "CreateAt",Value: bsonx.Int32(1)}}
	//if err := c.CreateIndex(key,options.Index().
	//	SetExpireAfterSeconds(int32(incDataExpireDay/time.Second)));err != nil{
	//	log.Error("create GameProfitLog Index: %s",err)
	//}
	//log.Info("init GameProfitLog of mongo db")
	_ = common.GetMysql().AutoMigrate(&GameProfitLog{})
}
func InsertProfitLog(profitLog GameProfitLog) {
	//c := common.GetMongoDB().C(cProfitLog)
	//if err := c.Insert(&profitLog); err != nil {
	//	log.Error(err.Error())
	//} else {
	//	common.ExecQueueFunc(func() {
	//
	//	})
	//}
	common.GetMysql().Create(&profitLog)
}
