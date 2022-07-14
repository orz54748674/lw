package activityStorage

import (
	"github.com/liangdas/mqant/log"
	"github.com/liangdas/mqant/utils/fatih/structs"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
)

//Vip
type VipType string

var (
	ChargeNeed           VipType = "ChargeNeed"           //充值需求
	PointsExtraThousand  VipType = "PointsExtraThousand"  //积分加成
	WeekGet              VipType = "WeekGet"              //每周彩金
	ChargeGetPerThousand VipType = "ChargeGetPerThousand" //充值赠送千分数
	KeepGradeNeed        VipType = "KeepGradeNeed"        //保级需求
	VipGetGold           VipType = "VipGetGold"           //VIP获取的金币
	VipGetPoints         VipType = "VipGetPoints"         //VIP获取的积分
)

type ActivityVipConf struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Type     VipType            `bson:"Type",json:"Type"`
	Status   int                `bson:"Status"` //1 开启 0关闭
	BetTimes int64              `bson:"BetTimes"`
	Vip0     int64              `bson:"Vip0"`
	Vip1     int64              `bson:"Vip1"`
	Vip2     int64              `bson:"Vip2"`
	Vip3     int64              `bson:"Vip3"`
	Vip4     int64              `bson:"Vip4"`
	Vip5     int64              `bson:"Vip5"`
	Vip6     int64              `bson:"Vip6"`
	Vip7     int64              `bson:"Vip7"`
	Vip8     int64              `bson:"Vip8"`
	Vip9     int64              `bson:"Vip9"`
	CreateAt time.Time          `bson:"CreateAt"`
	UpdateAt time.Time          `bson:"UpdateAt"`
}

type ActivityVip struct {
	ActivityID string         `bson:"ActivityID",json:"ActivityID"`
	Uid        string         `bson:"Uid",json:"Uid"`
	Level      int            `bson:"Level"` //
	GetGold    int64          `bson:"GetGold",json:"GetGold"`
	GetPoints  float64        `bson:"GetPoints",json:"GetPoints"`
	BetTimes   int64          `bson:"BetTimes",json:"BetTimes"`
	Status     ActivityStatus `bson:"Status",json:"Status"` //
	UpdateAt   time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time      `bson:"CreateAt",json:"CreateAt"`
}
type ActivityVipWeek struct {
	ActivityID string         `bson:"ActivityID",json:"ActivityID"`
	Uid        string         `bson:"Uid",json:"Uid"`
	Level      int            `bson:"Level"` //
	GetGold    int64          `bson:"GetGold",json:"GetGold"`
	BetTimes   int64          `bson:"BetTimes",json:"BetTimes"`
	Status     ActivityStatus `bson:"Status",json:"Status"` //
	UpdateAt   time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time      `bson:"CreateAt",json:"CreateAt"`
}

//----------------------VIP-----------------------
func InitActivityVipConf() {
	log.Info("init ActivityVip of mongo db")
	initActivityVipConf()
}
func initActivityVipConf() {
	insertActivityVipConf(&ActivityVipConf{Type: ChargeNeed, Status: 1, Vip0: 0, Vip1: 1000000, Vip2: 3000000, Vip3: 10000000, Vip4: 30000000, Vip5: 100000000, Vip6: 300000000, Vip7: 1000000000, Vip8: 3000000000, Vip9: 10000000000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityVipConf(&ActivityVipConf{Type: PointsExtraThousand, Status: 1, Vip0: 1000, Vip1: 1100, Vip2: 1200, Vip3: 1300, Vip4: 1400, Vip5: 1500, Vip6: 1600, Vip7: 1700, Vip8: 1800, Vip9: 2000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityVipConf(&ActivityVipConf{Type: WeekGet, Status: 1, Vip0: 0, Vip1: 0, Vip2: 0, Vip3: 0, Vip4: 30000, Vip5: 100000, Vip6: 300000, Vip7: 1000000, Vip8: 3000000, Vip9: 10000000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityVipConf(&ActivityVipConf{Type: ChargeGetPerThousand, Status: 1, Vip0: 0, Vip1: 0, Vip2: 0, Vip3: 10, Vip4: 15, Vip5: 5, Vip6: 25, Vip7: 30, Vip8: 35, Vip9: 40, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityVipConf(&ActivityVipConf{Type: KeepGradeNeed, Status: 1, Vip0: 0, Vip1: 3000000, Vip2: 9000000, Vip3: 30000000, Vip4: 90000000, Vip5: 300000000, Vip6: 900000000, Vip7: 3000000000, Vip8: 9000000000, Vip9: 30000000000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityVipConf(&ActivityVipConf{Type: VipGetGold, Status: 1, Vip0: 0, Vip1: 0, Vip2: 0, Vip3: 2000, Vip4: 5000, Vip5: 10000, Vip6: 30000, Vip7: 80000, Vip8: 200000, Vip9: 500000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityVipConf(&ActivityVipConf{Type: VipGetPoints, Status: 1, Vip0: 1001, Vip1: 1000, Vip2: 2000, Vip3: 2000, Vip4: 3000, Vip5: 3000, Vip6: 3000, Vip7: 5000, Vip8: 5000, Vip9: 10000, BetTimes: 10, CreateAt: utils.Now(), UpdateAt: utils.Now()})
}
func insertActivityVipConf(activityConf *ActivityVipConf) {
	c := common.GetMongoDB().C(cActivityVipConf)
	selector := bson.M{"Type": activityConf.Type}
	var conf ActivityConf
	if err := c.Find(selector).One(&conf); err != nil { //不存在则创建
		if err := c.Insert(activityConf); err != nil {
			log.Error(err.Error())
		}
	}
}
func QueryActivityVipConf() []ActivityVipConf {
	c := common.GetMongoDB().C(cActivityVipConf)
	query := bson.M{}
	var activityConf []ActivityVipConf
	if err := c.Find(query).All(&activityConf); err != nil {
		log.Error(err.Error())
	}
	return activityConf
}
func QueryActivityVipConfByType(tp VipType) ActivityVipConf {
	c := common.GetMongoDB().C(cActivityVipConf)
	query := bson.M{"Type": tp}
	var activityConf ActivityVipConf
	if err := c.Find(query).One(&activityConf); err != nil {
		log.Error(err.Error())
	}
	return activityConf
}
func InitActivityVip() {
	c := common.GetMongoDB().C(cActivityVip)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}, {Key: "ActivityID", Value: bsonx.Int32(1)}, {Key: "Type", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create ActivityVip Index: %s", err)
	}
}
func UpsertActivityVip(activity *ActivityVip) {
	c := common.GetMongoDB().C(cActivityVip)
	selector := bson.M{"Uid": activity.Uid, "Level": activity.Level}
	update := structs.Map(activity)
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("insert ActivityVip error: %s", err)
	}
}

//func QueryVipById(uid string,activityID string) *ActivityVip{
//	c := common.GetMongoDB().C(cActivityVip)
//	var activity *ActivityVip
//	query := bson.M{"Uid":uid,"ActivityID":activityID}
//	if err := c.Find(query).One(&activity); err != nil{
//		//log.Info("not found PayActivityRecord ",err)
//		return nil
//	}
//	return activity
//}
func QueryVipByLevel(uid string, level int) *ActivityVip {
	c := common.GetMongoDB().C(cActivityVip)
	var activity *ActivityVip
	query := bson.M{"Uid": uid, "Level": level}
	if err := c.Find(query).One(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return activity
}
func QueryVipAll(uid string) []ActivityVip {
	c := common.GetMongoDB().C(cActivityVip)
	var activity []ActivityVip
	query := bson.M{"Uid": uid}
	if err := c.Find(query).Sort("Level").All(&activity); err != nil {
		log.Error(err.Error())
	}
	return activity
}
func QueryVipAllDone(uid string) []ActivityVip {
	c := common.GetMongoDB().C(cActivityVip)
	var activity []ActivityVip
	query := bson.M{"Uid": uid, "Status": Done}
	if err := c.Find(query).Sort("Level").All(&activity); err != nil {
		log.Error(err.Error())
	}
	return activity
}
func InitActivityVipWeek() {
	c := common.GetMongoDB().C(cActivityVipWeek)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}, {Key: "ActivityID", Value: bsonx.Int32(1)}, {Key: "Type", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create ActivityVip Index: %s", err)
	}
}
func UpsertActivityVipWeek(activity *ActivityVipWeek) {
	c := common.GetMongoDB().C(cActivityVipWeek)
	selector := bson.M{"Uid": activity.Uid, "ActivityID": activity.ActivityID}
	update := structs.Map(activity)
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("insert ActivityVipWeek error: %s", err)
	}
}

func QueryVipWeekById(uid string, level int) *ActivityVipWeek {
	c := common.GetMongoDB().C(cActivityVipWeek)
	var activity *ActivityVipWeek
	query := bson.M{"Uid": uid, "Level": level}
	if err := c.Find(query).One(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return activity
}
func QueryVipWeekByLevel(uid string, level int) *ActivityVipWeek {
	c := common.GetMongoDB().C(cActivityVipWeek)
	var activity *ActivityVipWeek
	query := bson.M{"Uid": uid, "Level": level}
	if err := c.Find(query).One(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return activity
}
func QueryVipWeekAll(uid string) []ActivityVipWeek {
	c := common.GetMongoDB().C(cActivityVipWeek)
	var activity []ActivityVipWeek
	query := bson.M{"Uid": uid}
	if err := c.Find(query).Sort("Level").All(&activity); err != nil {
		log.Error(err.Error())
	}
	return activity
}
func RemoveAllVipWeek() {
	c := common.GetMongoDB().C(cActivityVipWeek)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}
