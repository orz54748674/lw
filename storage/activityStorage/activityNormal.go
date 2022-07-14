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
)

//累计充值
type ActivityTotalChargeConf struct {
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	TotalCharge int64              `bson:"TotalCharge"`
	Get         int64              `bson:"Get"`
	BetTimes    int64              `bson:"BetTimes"`
	CreateAt    time.Time          `bson:"CreateAt"`
	UpdateAt    time.Time          `bson:"UpdateAt"`
}
type ActivityTotalCharge struct {
	ActivityID string         `bson:"ActivityID",json:"ActivityID"`
	Uid        string         `bson:"Uid",json:"Uid"`
	Charge     int64          `bson:"Charge",json:"Charge"`
	Get        int64          `bson:"Get",json:"Get"`
	BetTimes   int64          `bson:"BetTimes",json:"BetTimes"`
	Status     ActivityStatus `bson:"Status",json:"Status"` //
	UpdateAt   time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time      `bson:"CreateAt",json:"CreateAt"`
}

//七日签到
type ActivitySignInConf struct {
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Type        ActivityGetType    `bson:"Type"`
	Day         int                `bson:"Day"` //第几天
	TotalCharge int64              `bson:"TotalCharge"`
	Get         int64              `bson:"Get"`
	BetTimes    int64              `bson:"BetTimes"`
	CreateAt    time.Time          `bson:"CreateAt"`
	UpdateAt    time.Time          `bson:"UpdateAt"`
}
type ActivitySignIn struct {
	ActivityID string          `bson:"ActivityID",json:"ActivityID"`
	Uid        string          `bson:"Uid",json:"Uid"`
	Type       ActivityGetType `bson:"Type"`
	Day        int             `bson:"Day"` //第几天
	Charge     int64           `bson:"Charge",json:"Charge"`
	Get        int64           `bson:"Get",json:"Get"`
	BetTimes   int64           `bson:"BetTimes",json:"BetTimes"`
	Status     ActivityStatus  `bson:"Status",json:"Status"` //
	UpdateAt   time.Time       `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time       `bson:"CreateAt",json:"CreateAt"`
}

//鼓励金
type ActivityEncouragementConf struct {
	Oid            primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	TotalCharge    int64              `bson:"TotalCharge"`
	Get            int64              `bson:"Get"`
	UnChargeGetCnt int                `bson:"UnChargeGetCnt"` //未充值领取次数
	ChargeGetCnt   int                `bson:"ChargeGetCnt"`   //充值领取次数
	BetTimes       int64              `bson:"BetTimes"`
	MinVnd         int64              `bson:"MinVnd"` //小于该余额，开始发放鼓励金
	CreateAt       time.Time          `bson:"CreateAt"`
	UpdateAt       time.Time          `bson:"UpdateAt"`
}
type ActivityEncouragement struct {
	ActivityID string         `bson:"ActivityID",json:"ActivityID"`
	Uid        string         `bson:"Uid",json:"Uid"`
	Get        int64          `bson:"Get",json:"Get"`
	BetTimes   int64          `bson:"BetTimes",json:"BetTimes"`
	Status     ActivityStatus `bson:"Status",json:"Status"` //
	UpdateAt   time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time      `bson:"CreateAt",json:"CreateAt"`
}

//----------------------累计充值-----------------------
func InitActivityTotalChargeConf() {
	log.Info("init ActivityTotalChargeConf of mongo db")
	c2 := common.GetMongoDB().C(cActivityTotalChargeConf)
	count2, err := c2.Find(bson.M{}).Count()
	if err == nil && count2 == 0 {
		initActivityTotalChargeConf()
	}
}
func initActivityTotalChargeConf() {
	insertActivityTotalChargeConf(&ActivityTotalChargeConf{TotalCharge: 16888, Get: 1000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityTotalChargeConf(&ActivityTotalChargeConf{TotalCharge: 58888, Get: 3500, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityTotalChargeConf(&ActivityTotalChargeConf{TotalCharge: 168888, Get: 10000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityTotalChargeConf(&ActivityTotalChargeConf{TotalCharge: 688888, Get: 55000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityTotalChargeConf(&ActivityTotalChargeConf{TotalCharge: 5888888, Get: 580000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityTotalChargeConf(&ActivityTotalChargeConf{TotalCharge: 16888888, Get: 1500000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
}
func insertActivityTotalChargeConf(conf *ActivityTotalChargeConf) {
	c := common.GetMongoDB().C(cActivityTotalChargeConf)
	if err := c.Insert(conf); err != nil {
		log.Error(err.Error())
	}
}
func QueryActivityTotalChargeConf() []ActivityTotalChargeConf {
	c := common.GetMongoDB().C(cActivityTotalChargeConf)
	query := bson.M{}
	var activityConf []ActivityTotalChargeConf
	if err := c.Find(query).All(&activityConf); err != nil {
		log.Error(err.Error())
	}
	return activityConf
}

func InitActivityTotalCharge() {
	c := common.GetMongoDB().C(cActivityTotalCharge)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}, {Key: "ActivityID", Value: bsonx.Int32(1)}, {Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create ActivityTotalCharge Index: %s", err)
	}
}
func UpsertActivityTotalCharge(activity *ActivityTotalCharge) {
	c := common.GetMongoDB().C(cActivityTotalCharge)
	selector := bson.M{"Uid": activity.Uid, "ActivityID": activity.ActivityID}
	update := structs.Map(activity)
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("insert activityRecord error: %s", err)
	}
}
func RemoveAllTotalCharge() {
	c := common.GetMongoDB().C(cActivityTotalCharge)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}

//首次充值
type ActivityFirstChargeConf struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	GetPer   int64              `bson:"GetPer"`
	GetMax   int64              `bson:"GetMax"`
	BetTimes int64              `bson:"BetTimes"`
	CreateAt time.Time          `bson:"CreateAt"`
	UpdateAt time.Time          `bson:"UpdateAt"`
}
type ActivityFirstCharge struct {
	ActivityID string         `bson:"ActivityID",json:"ActivityID"`
	Uid        string         `bson:"Uid",json:"Uid"`
	GetPer     int64          `bson:"GetPer"`
	GetMax     int64          `bson:"GetMax"`
	BetTimes   int64          `bson:"BetTimes"`
	Status     ActivityStatus `bson:"Status",json:"Status"` //""
	UpdateAt   time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time      `bson:"CreateAt",json:"CreateAt"`
}

//----------------------七日签到-----------------------

func InitActivitySignInConf() {
	log.Info("init ActivitySignInConf of mongo db")
	c2 := common.GetMongoDB().C(cActivitySignInConf)
	count2, err := c2.Find(bson.M{}).Count()
	if err == nil && count2 == 0 {
		initActivitySignInConf()
	}
}
func initActivitySignInConf() {
	insertActivitySignInConf(&ActivitySignInConf{Type: Gold, Day: 1, Get: 1000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivitySignInConf(&ActivitySignInConf{Type: Gold, Day: 2, Get: 1000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivitySignInConf(&ActivitySignInConf{Type: Gold, Day: 3, Get: 1000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivitySignInConf(&ActivitySignInConf{Type: Gold, Day: 4, Get: 2000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivitySignInConf(&ActivitySignInConf{Type: Gold, Day: 5, Get: 2000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivitySignInConf(&ActivitySignInConf{Type: Gold, Day: 6, Get: 2000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivitySignInConf(&ActivitySignInConf{Type: Gold, Day: 7, Get: 3000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
}
func insertActivitySignInConf(conf *ActivitySignInConf) {
	c := common.GetMongoDB().C(cActivitySignInConf)
	if err := c.Insert(conf); err != nil {
		log.Error(err.Error())
	}
}
func QueryActivitySignInConf() []ActivitySignInConf {
	c := common.GetMongoDB().C(cActivitySignInConf)
	query := bson.M{}
	var activityConf []ActivitySignInConf
	if err := c.Find(query).All(&activityConf); err != nil {
		log.Error(err.Error())
	}
	return activityConf
}
func InitActivitySignIn() {
	c := common.GetMongoDB().C(cActivitySignIn)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}, {Key: "ActivityID", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create ActivitySignIn Index: %s", err)
	}
}
func UpsertActivitySignIn(activity *ActivitySignIn) {
	c := common.GetMongoDB().C(cActivitySignIn)
	selector := bson.M{"Uid": activity.Uid, "ActivityID": activity.ActivityID}
	update := structs.Map(activity)
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("insert ActivitySignIn error: %s", err)
	}
}
func RemoveAllSignInByUid(uid string) {
	c := common.GetMongoDB().C(cActivitySignIn)
	if _, e := c.RemoveAll(bson.M{"Uid": uid}); e != nil {
		log.Error(e.Error())
	}
}
func QuerySignInById(uid string, activityID string) *ActivitySignIn {
	c := common.GetMongoDB().C(cActivitySignIn)
	var activity *ActivitySignIn
	query := bson.M{"Uid": uid, "ActivityID": activityID}
	if err := c.Find(query).One(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return activity
}
func QuerySignInAll(uid string) []ActivitySignIn {
	c := common.GetMongoDB().C(cActivitySignIn)
	var activity []ActivitySignIn
	query := bson.M{"Uid": uid}
	if err := c.Find(query).Sort("Day").All(&activity); err != nil {
		log.Error(err.Error())
	}
	return activity
}

//----------------------鼓励金-----------------------
func InitActivityEncouragementConf() {
	log.Info("init ActivityEncouragement of mongo db")
	c2 := common.GetMongoDB().C(cActivityEncouragementConf)
	count2, err := c2.Find(bson.M{}).Count()
	if err == nil && count2 == 0 {
		initActivityEncouragementConf()
	}
}
func initActivityEncouragementConf() {
	insertActivityEncouragementConf(&ActivityEncouragementConf{TotalCharge: 200000, Get: 2000, BetTimes: 10, UnChargeGetCnt: 1, ChargeGetCnt: 3, MinVnd: 500, CreateAt: utils.Now(), UpdateAt: utils.Now()})
}
func insertActivityEncouragementConf(conf *ActivityEncouragementConf) {
	c := common.GetMongoDB().C(cActivityEncouragementConf)
	if err := c.Insert(conf); err != nil {
		log.Error(err.Error())
	}
}
func QueryActivityEncouragementConf() ActivityEncouragementConf {
	c := common.GetMongoDB().C(cActivityEncouragementConf)
	query := bson.M{}
	var activityConf ActivityEncouragementConf
	if err := c.Find(query).One(&activityConf); err != nil {
		log.Error(err.Error())
	}
	return activityConf
}
func InitActivityEncouragement(incDataExpireDay time.Duration) {
	c := common.GetMongoDB().C(cActivityEncouragement)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create ActivityEncouragement Index: %s", err)
	}
}
func InsertActivityEncouragement(activity *ActivityEncouragement) {
	c := common.GetMongoDB().C(cActivityEncouragement)
	err := c.Insert(activity)
	if err != nil {
		log.Error("insert ActivityEncouragement error: %s", err)
	}
}
func QueryEncouragementAllByUid(uid string) []ActivityEncouragement {
	c := common.GetMongoDB().C(cActivityEncouragement)
	var activity []ActivityEncouragement
	query := bson.M{"Uid": uid}
	if err := c.Find(query).All(&activity); err != nil {
		log.Error(err.Error())
	}
	return activity
}
func RemoveAllEncouragement() {
	c := common.GetMongoDB().C(cActivityEncouragement)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}

//----------------------首充-----------------------
func InitActivityFirstChargeConf() {
	log.Info("init ActivityIFirstCharge of mongo db")
	c2 := common.GetMongoDB().C(cActivityFirstChargeConf)
	count2, err := c2.Find(bson.M{}).Count()
	if err == nil && count2 == 0 {
		conf := &ActivityFirstChargeConf{
			GetPer:   20,
			GetMax:   2000000,
			BetTimes: 10,
			CreateAt: utils.Now(),
			UpdateAt: utils.Now(),
		}
		if err := c2.Insert(conf); err != nil {
			log.Error(err.Error())
		}
	}
}

func QueryActivityFistChargeConf() []ActivityFirstChargeConf {
	c := common.GetMongoDB().C(cActivityFirstChargeConf)
	query := bson.M{}
	var activityConf []ActivityFirstChargeConf
	if err := c.Find(query).All(&activityConf); err != nil {
		log.Error(err.Error())
	}
	return activityConf
}

func InitActivityFirstCharge() {
	c := common.GetMongoDB().C(cActivityFirstCharge)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create ActivityFirstCharge Index: %s", err)
	}
}
func UpsertActivityFirstCharge(activity *ActivityFirstCharge) {
	c := common.GetMongoDB().C(cActivityFirstCharge)
	selector := bson.M{"Uid": activity.Uid}
	update := structs.Map(activity)
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("insert ActivityFirstCharge error: %s", err)
	}
}
func QueryFirstChargeByUid(uid string) []ActivityFirstCharge {
	c := common.GetMongoDB().C(cActivityFirstCharge)
	var activity []ActivityFirstCharge
	query := bson.M{"Uid": uid}
	if err := c.Find(query).All(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return activity
}
