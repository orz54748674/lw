package activityStorage

import (
	"github.com/liangdas/mqant/utils/fatih/structs"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/game"
)

//每日任务

//充值类任务
type ActivityDayChargeConf struct {
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	TotalCharge int64              `bson:"TotalCharge"`
	Get         int64              `bson:"Get"`
	GetPoints   float64            `bson:"GetPoints"`
	BetTimes    int64              `bson:"BetTimes"`
	CreateAt    time.Time          `bson:"CreateAt"`
	UpdateAt    time.Time          `bson:"UpdateAt"`
}
type ActivityDayCharge struct {
	ActivityID string         `bson:"ActivityID",json:"ActivityID"`
	Uid        string         `bson:"Uid",json:"Uid"`
	CurCharge  int64          `bson:"CurCharge"`
	Charge     int64          `bson:"Charge"`
	Get        int64          `bson:"Get",json:"Get"`
	GetPoints  float64        `bson:"GetPoints"`
	BetTimes   int64          `bson:"BetTimes",json:"BetTimes"`
	Status     ActivityStatus `bson:"Status",json:"Status"` //"1"可领取 "2"已领取
	UpdateAt   time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time      `bson:"CreateAt",json:"CreateAt"`
}

//游戏类任务
type ActivityDayGameConf struct {
	Oid       primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	GameType  game.Type          `bson:"GameType"`
	NeedBet   int64              `bson:"NeedBet"`
	Get       int64              `bson:"Get"`
	GetPoints float64            `bson:"GetPoints"`
	BetTimes  int64              `bson:"BetTimes"`
	CreateAt  time.Time          `bson:"CreateAt"`
	UpdateAt  time.Time          `bson:"UpdateAt"`
}
type ActivityDayGame struct {
	ActivityID       string         `bson:"ActivityID",json:"ActivityID"`
	Uid              string         `bson:"Uid",json:"Uid"`
	GameType         game.Type      `bson:"GameType"`
	GameTypeLanguage string         `bson:"GameTypeLanguage"` //游戏名
	CurBet           int64          `bson:"CurBet"`
	NeedBet          int64          `bson:"NeedBet"`
	Get              int64          `bson:"Get",json:"Get"`
	GetPoints        float64        `bson:"GetPoints"`
	BetTimes         int64          `bson:"BetTimes",json:"BetTimes"`
	Status           ActivityStatus `bson:"Status",json:"Status"` //
	UpdateAt         time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt         time.Time      `bson:"CreateAt",json:"CreateAt"`
}

//邀请类任务
type ActivityDayInviteConf struct {
	Oid       primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	InviteNum int                `bson:"InviteNum"`
	Get       int64              `bson:"Get"`
	GetPoints float64            `bson:"GetPoints"`
	BetTimes  int64              `bson:"BetTimes"`
	CreateAt  time.Time          `bson:"CreateAt"`
	UpdateAt  time.Time          `bson:"UpdateAt"`
}
type ActivityDayInvite struct {
	ActivityID string         `bson:"ActivityID",json:"ActivityID"`
	Uid        string         `bson:"Uid",json:"Uid"`
	CurInvite  int            `bson:"CurInvite"`
	InviteNum  int            `bson:"InviteNum"`
	Get        int64          `bson:"Get",json:"Get"`
	GetPoints  float64        `bson:"GetPoints"`
	BetTimes   int64          `bson:"BetTimes",json:"BetTimes"`
	Status     ActivityStatus `bson:"Status",json:"Status"` //
	UpdateAt   time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time      `bson:"CreateAt",json:"CreateAt"`
}

//----------------------每日任务充值类-----------------------
func InitActivityDayChargeConf() {
	log.Info("init ActivityDayChargeConf of mongo db")
	c2 := common.GetMongoDB().C(cActivityDayChargeConf)
	count2, err := c2.Find(bson.M{}).Count()
	if err == nil && count2 == 0 {
		initActivityDayChargeConf()
	}
}
func initActivityDayChargeConf() {
	insertActivityDayChargeConf(&ActivityDayChargeConf{TotalCharge: 0, Get: 500, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityDayChargeConf(&ActivityDayChargeConf{TotalCharge: 1000000, Get: 2000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
}
func insertActivityDayChargeConf(conf *ActivityDayChargeConf) {
	c := common.GetMongoDB().C(cActivityDayChargeConf)
	if err := c.Insert(conf); err != nil {
		log.Error(err.Error())
	}
}
func QueryActivityDayChargeConf() []ActivityDayChargeConf {
	c := common.GetMongoDB().C(cActivityDayChargeConf)
	query := bson.M{}
	var activityConf []ActivityDayChargeConf
	if err := c.Find(query).All(&activityConf); err != nil {
		log.Error(err.Error())
	}
	return activityConf
}

func InitActivityDayCharge() {
	c := common.GetMongoDB().C(cActivityDayChargeConf)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}, {Key: "ActivityID", Value: bsonx.Int32(1)}, {Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create ActivityTotalCharge Index: %s", err)
	}
}
func UpsertActivityDayCharge(activity *ActivityDayCharge) {
	c := common.GetMongoDB().C(cActivityDayCharge)
	selector := bson.M{"Uid": activity.Uid, "ActivityID": activity.ActivityID}
	update := structs.Map(activity)
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("insert ActivityDayCharge error: %s", err)
	}
}
func RemoveAllDayCharge() {
	c := common.GetMongoDB().C(cActivityDayCharge)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}
func QueryTodayDayCharge(uid string, activityID string) *ActivityDayCharge {
	c := common.GetMongoDB().C(cActivityDayCharge)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	var activity ActivityDayCharge
	query := bson.M{"Uid": uid, "ActivityID": activityID, "CreateAt": bson.M{"$gt": thatTime}}
	if err := c.Find(query).One(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return &activity
}
func QueryTodayDayChargeByUid(uid string) []ActivityDayCharge {
	c := common.GetMongoDB().C(cActivityDayCharge)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	var activity []ActivityDayCharge
	query := bson.M{"Uid": uid, "CreateAt": bson.M{"$gt": thatTime}}
	if err := c.Find(query).All(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return activity
}

//----------------------每日任务游戏类-----------------------
func InitActivityDayGameConf() {
	log.Info("init ActivityDayGameConf of mongo db")
	c2 := common.GetMongoDB().C(cActivityDayGameConf)
	count2, err := c2.Find(bson.M{}).Count()
	if err == nil && count2 == 0 {
		initActivityDayGameConf()
	}
}
func initActivityDayGameConf() {
	insertActivityDayGameConf(&ActivityDayGameConf{GameType: game.BiDaXiao, NeedBet: 5000000, Get: 1000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
}
func insertActivityDayGameConf(conf *ActivityDayGameConf) {
	c := common.GetMongoDB().C(cActivityDayGameConf)
	if err := c.Insert(conf); err != nil {
		log.Error(err.Error())
	}
}
func QueryActivityDayGameConf() []ActivityDayGameConf {
	c := common.GetMongoDB().C(cActivityDayGameConf)
	query := bson.M{}
	var activityConf []ActivityDayGameConf
	if err := c.Find(query).All(&activityConf); err != nil {
		log.Error(err.Error())
	}
	return activityConf
}

func InitActivityDayGame() {
	c := common.GetMongoDB().C(cActivityDayGame)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}, {Key: "ActivityID", Value: bsonx.Int32(1)}, {Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create ActivityDayGameConf Index: %s", err)
	}
}
func UpsertActivityDayGame(activity *ActivityDayGame) {
	c := common.GetMongoDB().C(cActivityDayGame)
	selector := bson.M{"Uid": activity.Uid, "ActivityID": activity.ActivityID}
	update := structs.Map(activity)
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("insert ActivityDayGame error: %s", err)
	}
}
func RemoveAllDayGame() {
	c := common.GetMongoDB().C(cActivityDayGame)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}
func QueryTodayDayGame(uid string, activityID string) *ActivityDayGame {
	c := common.GetMongoDB().C(cActivityDayGame)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	var activity ActivityDayGame
	query := bson.M{"Uid": uid, "ActivityID": activityID, "CreateAt": bson.M{"$gt": thatTime}}
	if err := c.Find(query).One(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return &activity
}
func QueryTodayDayGameByUid(uid string) []ActivityDayGame {
	c := common.GetMongoDB().C(cActivityDayGame)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	var activity []ActivityDayGame
	query := bson.M{"Uid": uid, "CreateAt": bson.M{"$gt": thatTime}}
	if err := c.Find(query).All(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return activity
}

//----------------------每日任务邀请类-----------------------
func InitActivityDayInviteConf() {
	log.Info("init ActivityDayInviteConf of mongo db")
	c2 := common.GetMongoDB().C(cActivityDayInviteConf)
	count2, err := c2.Find(bson.M{}).Count()
	if err == nil && count2 == 0 {
		initActivityDayInviteConf()
	}
}
func initActivityDayInviteConf() {
	insertActivityDayInviteConf(&ActivityDayInviteConf{InviteNum: 1, Get: 500, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
}
func insertActivityDayInviteConf(conf *ActivityDayInviteConf) {
	c := common.GetMongoDB().C(cActivityDayInviteConf)
	if err := c.Insert(conf); err != nil {
		log.Error(err.Error())
	}
}
func QueryActivityDayInviteConf() []ActivityDayInviteConf {
	c := common.GetMongoDB().C(cActivityDayInviteConf)
	query := bson.M{}
	var activityConf []ActivityDayInviteConf
	if err := c.Find(query).All(&activityConf); err != nil {
		log.Error(err.Error())
	}
	return activityConf
}

func InitActivityDayInvite() {
	c := common.GetMongoDB().C(cActivityDayInvite)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}, {Key: "ActivityID", Value: bsonx.Int32(1)}, {Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create ActivityDayInvite Index: %s", err)
	}
}
func UpsertActivityDayInvite(activity *ActivityDayInvite) {
	c := common.GetMongoDB().C(cActivityDayInvite)
	selector := bson.M{"Uid": activity.Uid, "ActivityID": activity.ActivityID}
	update := structs.Map(activity)
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("insert ActivityDayInvite error: %s", err)
	}
}
func RemoveAllDayInvite() {
	c := common.GetMongoDB().C(cActivityDayInvite)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}
func QueryTodayDayInvite(uid string, activityID string) *ActivityDayInvite {
	c := common.GetMongoDB().C(cActivityDayInvite)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	var activity ActivityDayInvite
	query := bson.M{"Uid": uid, "ActivityID": activityID, "CreateAt": bson.M{"$gt": thatTime}}
	if err := c.Find(query).One(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return &activity
}
func QueryTodayDayInviteByUid(uid string) []ActivityDayInvite {
	c := common.GetMongoDB().C(cActivityDayInvite)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	var activity []ActivityDayInvite
	query := bson.M{"Uid": uid, "CreateAt": bson.M{"$gt": thatTime}}
	if err := c.Find(query).All(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return activity
}
