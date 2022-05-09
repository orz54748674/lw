package activityStorage

import (
	"vn/common"
	"vn/common/errCode"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)
type ActivityUserInfo struct {
	ID           int64              `bson:"-" json:"-"`
	Oid          primitive.ObjectID `bson:"_id,omitempty",json:"Oid"`
	Uid          string             `bson:"Uid",json:"Uid"`
	ActivityType ActivityType       `bson:"ActivityType"`
	SumCharge    int64              `bson:"SumCharge"`
	WeekBets   int64              `bson:"WeekBets"`//周流水
	WeekHaveUpGrade   bool        `bson:"WeekHaveUpGrade"`//本周是否升过级
}
//----------------------活动用户信息-----------------------
func InitActivityUserInfo() {
	c := common.GetMongoDB().C(cActivityUserInfo)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)},{Key: "ActivityType", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create ActivityUserInfo Index: %s", err)
	}
	//_ = common.GetMysql().AutoMigrate(&ActivityUserInfo{})
}
//func updateActivityUserInfo2mysql(userInfo ActivityUserInfo)  {
//	common.ExecQueueFunc(func() {
//		var u ActivityUserInfo
//		common.GetMysql().Where("uid=? and activity_type=?", userInfo.Uid,userInfo.ActivityType).First(&u)
//		common.GetMysql().Save(&userInfo)
//	})
//}
func UpsertActivityUserInfo(userInfo ActivityUserInfo) *common.Err{
	c := common.GetMongoDB().C(cActivityUserInfo)
	selector := bson.M{"Uid": userInfo.Uid,"ActivityType":userInfo.ActivityType}
	_, err := c.Upsert(selector, userInfo)
	if err != nil {
		log.Error("Upsert user userInfo error: %s", err)
		return errCode.ServerError.SetErr(err.Error())
	}
	//updateActivityUserInfo2mysql(userInfo)
	return nil
}
func IncActivityUserCharge(uid string,activityType ActivityType, amount int64) {
	c := common.GetMongoDB().C(cActivityUserInfo)
	query := bson.M{"Uid":uid,"ActivityType":activityType}
	update := bson.M{"$inc":bson.M{"SumCharge":amount}}
	if _,err := c.Upsert(query, update);err !=nil{
		log.Error(err.Error())
	}
}
func SetActivityUserCharge(uid string,activityType ActivityType, amount int64) {
	c := common.GetMongoDB().C(cActivityUserInfo)
	query := bson.M{"Uid":uid,"ActivityType":activityType}
	update := bson.M{"$set":bson.M{"SumCharge":amount}}
	if _,err := c.Upsert(query, update);err !=nil{
		log.Error(err.Error())
	}
}

func IncActivityWeekBets(uid string,activityType ActivityType, amount int64) {
	c := common.GetMongoDB().C(cActivityUserInfo)
	query := bson.M{"Uid":uid,"ActivityType":activityType}
	update := bson.M{"$inc":bson.M{"WeekBets":amount}}
	if _,err := c.Upsert(query, update);err !=nil{
		log.Error(err.Error())
	}
}
func ResetActivityWeekBets(uid string,activityType ActivityType) {
	c := common.GetMongoDB().C(cActivityUserInfo)
	query := bson.M{"Uid":uid,"ActivityType":activityType}
	update := bson.M{"$set":bson.M{"WeekBets":0}}
	if _,err := c.Upsert(query, update);err !=nil{
		log.Error(err.Error())
	}
}
func SetActivityWeekHaveUpGrade(uid string,activityType ActivityType, flag bool) {
	c := common.GetMongoDB().C(cActivityUserInfo)
	query := bson.M{"Uid":uid,"ActivityType":activityType}
	update := bson.M{"$set":bson.M{"WeekHaveUpGrade":flag}}
	if _,err := c.Upsert(query, update);err !=nil{
		log.Error(err.Error())
	}
}
func QueryActivityUserInfo(uid string,activityType ActivityType) ActivityUserInfo {
	c := common.GetMongoDB().C(cActivityUserInfo)
	var userInfo ActivityUserInfo
	selector := bson.M{"Uid":uid,"ActivityType":activityType}
	if err := c.Find(selector).One(&userInfo);err != nil{
	}
	return userInfo
}