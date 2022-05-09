package gameStorage

import (
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var(
	cMail = "Mail"
	cMailRecord = "MailRecord"
)
func InitMail() {
	//c := common.GetMongoDB().C(cMail)
	//index := mgo.Index{
	//	Key:        []string{"Oid"},
	//	Unique:     true,
	//	DropDups:   true,
	//	Background: true, // See notes.
	//	Sparse:     true,
	//}
	//if err := c.EnsureIndex(index); err != nil {
	//	log.Error("create Mail err: %s", err)
	//}
	//log.Info("init Mail of mongo db")
	//_ = common.GetMysql().AutoMigrate(&Mail{})
	c := common.GetMongoDB().C(cMail)
	key := bsonx.Doc{{Key: "SendTime", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create Mail Index: %s", err)
	}
}
func InitMailRecord(incDataExpireDay time.Duration) {
	//c := common.GetMgo().C(cMailRecord)
	//index := mgo.Index{
	//	Key:        []string{"Oid"},
	//	Unique:     true,
	//	DropDups:   true,
	//	Background: true, // See notes.
	//	Sparse:     true,
	//}
	//if err := c.EnsureIndex(index); err != nil {
	//	log.Error("create MailRecord err: %s", err)
	//}
	//log.Info("init MailRecord of mongo db")
	//_ = common.GetMysql().AutoMigrate(&MailRecord{})
	c := common.GetMongoDB().C(cMailRecord)
	key := bsonx.Doc{{Key: "SendTime", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create MailRecord Index: %s", err)
	}
	key = bsonx.Doc{{Key: "Account", Value: bsonx.Int32(1)},{Key: "ReadState", Value: bsonx.Int32(1)},{Key: "Type", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create MailRecord Index: %s", err)
	}
	key = bsonx.Doc{{Key: "Account", Value: bsonx.Int32(1)},{Key: "Type", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create MailRecord Index: %s", err)
	}
}

func InsertMail(mail *Mail)*common.Err{
	c := common.GetMongoDB().C(cMail)
	if error := c.Insert(mail); error != nil{
		log.Info("Insert mail error: %s", error)
		return errCode.ServerError.SetErr(error.Error())
	}
	return nil
}
func InsertMailRecord(mailRecord *MailRecord)*common.Err{
	c := common.GetMongoDB().C(cMailRecord)
	if error := c.Insert(mailRecord); error != nil{
		log.Info("Insert mail Record error: %s", error)
		return errCode.ServerError.SetErr(error.Error())
	}
	return nil
}
func QueryMailUnreadNum(account string,mailType MailType) int{
	c := common.GetMongoDB().C(cMailRecord)
	var query map[string]interface{}
	if mailType == MailAll{
		query = bson.M{"Account":account,"ReadState":"unread"}
	}else{
		query = bson.M{"Account":account,"ReadState":"unread","Type":mailType}
	}
	num,err := c.Find(query).Count()
	if err != nil{
		log.Info("QueryMailUnreadNum error: %s", err)
		return 0
	}
	return int(num)
}
func UpdateMailRecordReadState(oid primitive.ObjectID,readState ReadStatus){
	c := common.GetMongoDB().C(cMailRecord)
	selector := bson.M{"_id":oid}
	update := bson.M{"$set":bson.M{"ReadState":readState}}
	c.Update(selector,update)
}
func DeleteMail(oid primitive.ObjectID){
	c := common.GetMongoDB().C(cMailRecord)
	selector := bson.M{"_id":oid}
	c.Remove(selector)
}
func QueryMailRecord(account string,mailType MailType) []MailRecord {
	c := common.GetMongoDB().C(cMailRecord)
	var mailRecord []MailRecord
	query := bson.M{"Account":account,"Type":mailType}
	if err := c.Find(query).Sort("-SendTime").All(&mailRecord); err != nil{
		log.Info("not found mailRecord ",err)
		return []MailRecord{}
	}
	if mailRecord == nil{
		return []MailRecord{}
	}
	return mailRecord
}