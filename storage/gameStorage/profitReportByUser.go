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
	"vn/storage"
)

type GameProfitByUser struct {
	ID         uint64             `bson:"-" json:"-"`
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Uid        string             `bson:"Uid" json:"Uid"`
	Profit     int64              `bson:"Profit"`     //SystemProfit 系统抽水， 平台明面上的抽水
	BotBalance int64              `bson:"BotBalance"` //系统余额  该余额不够赔付时，要控盘
	BotProfit  int64              `bson:"BotProfit"`  //暗抽金额
	WinLose    int64              `bson:"WinLose"`    //输赢
}
type GameProfitLogByUser struct {
	ID         int64              `bson:"-" json:"-"`
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	AdminID    string             `bson:"AdminID"`
	GameType   game.Type          `bson:"GameType"`
	BotBalance int64              `bson:"BotBalance"`
	CreateAt   time.Time          `bson:"CreateAt"`
}

func (GameProfitByUser) TableName() string {
	return "game_profit_by_User"
}

var (
	cProfitByUser    = "gameProfitByUser"
	cProfitLogByUser = "gameProfitLogByUser"
)

func InitGameProfitByUser() {
	c := common.GetMongoDB().C(cProfitByUser)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().SetUnique(true)); err != nil {
		log.Error("create GameProfitByUser Index: %s", err)
	}
	log.Info("init GameProfitByUser of mongo db")
	//_ = common.GetMysql().AutoMigrate(&GameProfit{})
}
func IncProfitByUser(uid string, amount int64, botAmount int64, botProfit int64, winLose int64) {
	c := common.GetMongoDB().C(cProfitByUser)
	query := bson.M{"Uid": uid}
	update := bson.M{
		"$inc": bson.M{"Profit": amount,
			"BotBalance": botAmount,
			"BotProfit":  botProfit,
			"WinLose":    winLose,
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
func QueryProfitByUser(uid string) *GameProfitByUser {
	c := common.GetMongoDB().C(cProfitByUser)
	query := bson.M{"Uid": uid}
	var gameProfitByUser GameProfitByUser
	if err := c.Find(query).One(&gameProfitByUser); err != nil {
		log.Error(err.Error())
	}
	return &gameProfitByUser
}
func InitGameProfitLogByUser(incDataExpireDay time.Duration) {
	//c := common.GetMongoDB().C(cProfitLog)
	//key := bsonx.Doc{{Key: "CreateAt",Value: bsonx.Int32(1)}}
	//if err := c.CreateIndex(key,options.Index().
	//	SetExpireAfterSeconds(int32(incDataExpireDay/time.Second)));err != nil{
	//	log.Error("create GameProfitLog Index: %s",err)
	//}
	//log.Info("init GameProfitLog of mongo db")
	_ = common.GetMysql().AutoMigrate(&GameProfitLogByUser{})
}
func InsertProfitLogByUser(profitLogByUser GameProfitLogByUser) {
	//c := common.GetMongoDB().C(cProfitLog)
	//if err := c.Insert(&profitLog); err != nil {
	//	log.Error(err.Error())
	//} else {
	//	common.ExecQueueFunc(func() {
	//
	//	})
	//}
	common.GetMysql().Create(&profitLogByUser)
}
func ChargeCalcProfitByUser(uid string, amount int64) {
	gameProfit := QueryProfitByUser(uid)
	chargeIncProfitUserPer, _ := utils.ConvertInt(storage.QueryConf(storage.KChargeIncProfitUserPer))
	botAmount := amount * chargeIncProfitUserPer / 100
	if gameProfit.BotBalance >= botAmount || gameProfit.BotProfit+gameProfit.WinLose > 0 {
		return
	}
	IncProfitByUser(uid, 0, botAmount, -botAmount, 0)
}

//func DouDouCalcProfitByUser(uid string,amount int64){
//	gameProfit := QueryProfitByUser(uid)
//	botAmount := int64(0)
//
//	if gameProfit.WinLose + gameProfit.BotProfit < 0 && gameProfit.BotProfit > 0{//总输超过机器人抽水
//		return
//	}
//	if gameProfit.BotProfit < 0{
//		if gameProfit.WinLose > 0{
//			if amount < gameProfit.WinLose{
//				if gameProfit.BotBalance > amount{
//					botAmount -= amount
//				}else if gameProfit.BotBalance > 0{
//					botAmount -= gameProfit.BotBalance
//				}
//			}else{
//				if gameProfit.BotBalance > gameProfit.WinLose{
//					botAmount -= gameProfit.WinLose
//				}else if gameProfit.BotBalance > 0{
//					botAmount -= gameProfit.BotBalance
//				}
//			}
//		}
//	}else if gameProfit.WinLose + gameProfit.BotProfit > 0{
//		if amount < gameProfit.WinLose + gameProfit.BotProfit{
//			if gameProfit.BotBalance > amount{
//				botAmount -= amount
//			}else if gameProfit.BotBalance > 0{
//				botAmount -= gameProfit.BotBalance
//			}
//		}else{
//			if gameProfit.BotBalance > gameProfit.WinLose + gameProfit.BotProfit{
//				botAmount -= gameProfit.WinLose + gameProfit.BotProfit
//			}else if gameProfit.BotBalance > 0{
//				botAmount -= gameProfit.BotBalance
//			}
//		}
//	}
//	if botAmount != 0{
//		IncProfitByUser(uid,0,botAmount,-botAmount,0)
//	}
//}
