package activityStorage

import (
	"github.com/liangdas/mqant/log"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
)

type ActivityTurnTableConf struct {
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	ChargeSwapPoints float64          `bson:"ChargeSwapPoints"` //充值兑换成积分
	BetsSwapPoints float64          	`bson:"BetsSwapPoints"` //下注流水兑换成积分
	TurnNeedPoints float64          	`bson:"TurnNeedPoints"` //转一次需要的积分
	GetAward   int64          	`bson:"GetAward"` //集齐获得奖励
	BetTimes   int64          `bson:"BetTimes"`
	UpdateAt   time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time      `bson:"CreateAt",json:"CreateAt"`
}
const maxPageNum = 200
type WordType string
var (
	L 			WordType = "L" //
	U 			WordType = "U" //
	C 			WordType = "C" //
	K 			WordType = "K" //
	Y 			WordType = "Y" //
	TRUOT       WordType = "TRUOT" //
)
type ActivityTurnTableList struct {
	No     	   int          `bson:"No"` //序号 从12点顺时针
	WordType   WordType     `bson:"WordType"` //字的类型
	GetVnd	   int64		`bson:"GetVnd"`
	GetPoints	   float64		`bson:"GetPoints"`
	UpdateAt   time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time      `bson:"CreateAt",json:"CreateAt"`
}
type ActivityTurnTableInfo struct {
	ID           int64              `bson:"-" json:"-"`
	Oid          primitive.ObjectID `bson:"_id,omitempty",json:"Oid"`
	Uid          string             `bson:"Uid",json:"Uid"`
	NickName     string             `bson:"NickName",json:"NickName"`
	UserType     string             `bson:"UserType",json:"UserType"`//区分机器人
	Points	   float64            `bson:"Points"`//积分
	TurnTableCnt   int            `bson:"TurnTableCnt"`//转了多少次
	SumGetGold	   int64            `bson:"SumGetGold"`//总共获得金币
	L      int              `bson:"L"`//
	U       int             `bson:"U"`//
	C      int             `bson:"C"`//
	K       int             `bson:"K"`//
	Y       int             `bson:"Y"`//
	UpdateAt   time.Time      	  `bson:"UpdateAt",json:"UpdateAt"`
}
type ActivityTurnTableRecord struct {
	ID         int64              `bson:"-" json:"-"`
	Oid        primitive.ObjectID `bson:"_id,omitempty",json:"Oid"`
	Uid        string             `bson:"Uid",json:"Uid"`
	No		   string             `bson:"No",json:"No"`//期号
	WordType   WordType           `bson:"WordType",json:"WordType"`
	GetVnd     int64           	  `bson:"GetVnd",json:"GetVnd"`
	GetPoints     int64           	  `bson:"GetPoints",json:"GetPoints"`
	IsJackPot  bool           	  `bson:"IsJackPot",json:"IsJackPot"`
	BetTimes   int64              `bson:"BetTimes",json:"BetTimes"`
	UpdateAt   time.Time          `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time          `bson:"CreateAt",json:"CreateAt" gorm:"index"`
}
type ActivityTurnTableControl struct {//占比
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	L     	   int64          `bson:"L"` //
	U     	   int64          `bson:"U"` //
	C     	   int64          `bson:"C"` //
	K     	   int64          `bson:"K"` //
	Y     	   int64          `bson:"Y"` //
	TRUOT	   int64          `bson:"TRUOT"` //
	UpdateAt   time.Time      `bson:"UpdateAt",json:"UpdateAt"`
	CreateAt   time.Time      `bson:"CreateAt",json:"CreateAt"`
}
func InitActivityTurnTableInfo() {
	c := common.GetMongoDB().C(cActivityTurnTableInfo)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create InitActivityTurnTableInfo Index: %s", err)
	}
}
func InitActivityTurnTableConf() {
	log.Info("init ActivityTurnTable of mongo db")
	c := common.GetMongoDB().C(cActivityTurnTableConf)
	count, err := c.Find(bson.M{}).Count()
	if err == nil && count == 0 {
		conf := ActivityTurnTableConf{
			ChargeSwapPoints: 1000,
			BetsSwapPoints: 10000,
			TurnNeedPoints: 1000,
			GetAward:10000000,
			BetTimes: 10,
			UpdateAt: utils.Now(),
			CreateAt: utils.Now(),
		}
		if err := c.Insert(conf); err != nil {
			log.Error(err.Error())
		}
	}
}
func InitActivityTurnTableList() { //转盘表
	log.Info("init ActivityTurnTableList of mongo db")
	initActivityTurnTableList()
}
func initActivityTurnTableList() {
	insertActivityTurnTableList(&ActivityTurnTableList{No:1,WordType:L,GetVnd:1000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:2,WordType:TRUOT,GetVnd:0,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:3,WordType:K,GetVnd:20000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:4,WordType:L,GetVnd:1000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:5,WordType:U,GetVnd:2000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:6,WordType:L,GetVnd:1000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:7,WordType:TRUOT,GetVnd:0,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:8,WordType:C,GetVnd:5000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:9,WordType:L,GetVnd:1000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:10,WordType:TRUOT,GetVnd:0,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:11,WordType:K,GetVnd:20000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:12,WordType:L,GetVnd:1000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:13,WordType:U,GetVnd:2000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:14,WordType:L,GetVnd:1000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:15,WordType:Y,GetVnd:200000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
	insertActivityTurnTableList(&ActivityTurnTableList{No:16,WordType:C,GetVnd:5000,CreateAt: utils.Now(),UpdateAt: utils.Now()})
}
func insertActivityTurnTableList(activityConf *ActivityTurnTableList) {
	c := common.GetMongoDB().C(cActivityTurnTableList)
	selector := bson.M{"No": activityConf.No}
	var conf ActivityTurnTableList
	if err := c.Find(selector).One(&conf); err != nil { //不存在则创建
		if err := c.Insert(activityConf); err != nil {
			log.Error(err.Error())
		}
	}
}
func QueryActivityTurnTableConf() ActivityTurnTableConf {
	c := common.GetMongoDB().C(cActivityTurnTableConf)
	query := bson.M{}
	var activityConf ActivityTurnTableConf
	if err := c.Find(query).One(&activityConf);err != nil{
		log.Error(err.Error())
	}
	return activityConf
}
func QueryActivityTurnTableList() []ActivityTurnTableList {
	c := common.GetMongoDB().C(cActivityTurnTableList)
	query := bson.M{}
	var activityConf []ActivityTurnTableList
	if err := c.Find(query).Sort("No").All(&activityConf);err != nil{
		log.Error(err.Error())
	}
	return activityConf
}
func QueryActivityTurnTableListByNo(no int) ActivityTurnTableList {
	c := common.GetMongoDB().C(cActivityTurnTableList)
	query := bson.M{"No":no}
	var activityConf ActivityTurnTableList
	if err := c.Find(query).One(&activityConf);err != nil{
		log.Error(err.Error())
	}
	return activityConf
}
func QueryActivityTurnTableListByWordType(wordType WordType) []ActivityTurnTableList {
	c := common.GetMongoDB().C(cActivityTurnTableList)
	query := bson.M{"WordType":wordType}
	activityConf := make([]ActivityTurnTableList,0)
	if err := c.Find(query).Sort("No").All(&activityConf);err != nil{
		log.Error(err.Error())
	}
	return activityConf
}
func IncActivityWordNum(uid string,wordType WordType, amount int64) {
	c := common.GetMongoDB().C(cActivityTurnTableInfo)
	query := bson.M{"Uid":uid}
	update := bson.M{"$inc":bson.M{string(wordType):amount}}
	if _,err := c.Upsert(query, update);err !=nil{
		log.Error(err.Error())
	}
}
func ResetActivityWordNum(uid string) {
	c := common.GetMongoDB().C(cActivityTurnTableInfo)
	query := bson.M{"Uid":uid}
	update := bson.M{"$set":bson.M{string(L):0,string(U):0,string(C):0,string(K):0,string(Y):0}}
	if _,err := c.Upsert(query, update);err !=nil{
		log.Error(err.Error())
	}
}
func IncActivityTurnTableCnt(uid string,nickName string,userType string, amount int64) {
	c := common.GetMongoDB().C(cActivityTurnTableInfo)
	query := bson.M{"Uid":uid}
	update := bson.M{"$inc":bson.M{"TurnTableCnt":amount},"$set":bson.M{"NickName":nickName,"UserType":userType,"UpdateAt":utils.Now()}}
	if _,err := c.Upsert(query, update);err !=nil{
		log.Error(err.Error())
	}
}
func IncTurnTableInfoPoints(uid string, amount float64) {
	c := common.GetMongoDB().C(cActivityTurnTableInfo)
	query := bson.M{"Uid":uid}
	update := bson.M{"$inc":bson.M{"Points":amount}}
	if _,err := c.Upsert(query, update);err !=nil{
		log.Error(err.Error())
	}
}
func IncTurnTableSumGetGold(uid string, amount int64) {
	c := common.GetMongoDB().C(cActivityTurnTableInfo)
	query := bson.M{"Uid":uid}
	update := bson.M{"$inc":bson.M{"SumGetGold":amount}}
	if _,err := c.Upsert(query, update);err !=nil{
		log.Error(err.Error())
	}
}
func QueryTurnTableInfo(uid string) ActivityTurnTableInfo {
	c := common.GetMongoDB().C(cActivityTurnTableInfo)
	var userInfo ActivityTurnTableInfo
	selector := bson.M{"Uid":uid}
	if err := c.Find(selector).One(&userInfo);err != nil{
	}
	return userInfo
}
func QueryTurnTableInfoAll(size int) []ActivityTurnTableInfo {
	c := common.GetMongoDB().C(cActivityTurnTableInfo)
	var info []ActivityTurnTableInfo
	err := c.Find(nil).Sort("-SumGetGold").Limit(size).All(&info)
	if err != nil {
		return []ActivityTurnTableInfo{}
	}
	if info == nil {
		return []ActivityTurnTableInfo{}
	}
	for k,v := range info{
		info[k].UpdateAt = v.UpdateAt.Local()
	}
	return info
}
//----------------------转盘记录-----------------------
func InitActivityTurnTableRecord() {
	c := common.GetMongoDB().C(cActivityTurnTableRecord)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create ActivityReceiveRecord Index: %s", err)
	}
}
func InsertActivityTurnTableRecord(activityRecord *ActivityTurnTableRecord) {
	activityRecord.Oid = primitive.NewObjectID()
	c := common.GetMongoDB().C(cActivityTurnTableRecord)
	err :=c.Insert(activityRecord)
	if err != nil{
		log.Error("insert activityReceiveRecord error: %s", err)
	}
}
func QueryTurnTableRecordByUid(uid string,offset int, pageSize int) []ActivityTurnTableRecord {
	c := common.GetMongoDB().C(cActivityTurnTableRecord)
	if pageSize != 0 {
		if offset/pageSize > maxPageNum {
			return []ActivityTurnTableRecord{}
		}
	}
	var userInfo []ActivityTurnTableRecord
	selector := bson.M{"Uid":uid}
	if err := c.Find(selector).Sort("-UpdateAt").Skip(offset).Limit(pageSize).All(&userInfo);err != nil{
		return []ActivityTurnTableRecord{}
	}
	if userInfo == nil {
		return []ActivityTurnTableRecord{}
	}
	for k,v := range userInfo{
		userInfo[k].CreateAt = v.CreateAt.Local()
		userInfo[k].UpdateAt = v.UpdateAt.Local()
	}
	return userInfo
}
func QueryTurnTableRecordByUidTotal(uid string) int64{
	c := common.GetMongoDB().C(cActivityTurnTableRecord)
	selector := bson.M{"Uid":uid}
	count,err := c.Find(selector).Count()
	if err != nil {
		return 0
	}
	return count
}
func InitActivityTurnTableControl() {
	c := common.GetMongoDB().C(cActivityTurnTableControl)
	var conf ActivityTurnTableControl
	if err := c.Find(nil).One(&conf); err != nil { //不存在则创建
		conf = ActivityTurnTableControl{
			L: 250,
			U: 200,
			C: 75,
			K: 20,
			Y: 5,
			TRUOT: 450,
			CreateAt: utils.Now(),
			UpdateAt: utils.Now(),
		}
		if err := c.Insert(conf); err != nil {
			log.Error(err.Error())
		}
	}
}
func QueryActivityTurnTableControl() *ActivityTurnTableControl{
	c := common.GetMongoDB().C(cActivityTurnTableControl)
	var activityConf *ActivityTurnTableControl
	if err := c.Find(nil).One(&activityConf);err != nil{
		log.Error(err.Error())
	}
	return activityConf
}