package payStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

func QueryPayConf(methodId primitive.ObjectID) *PayConf {
	c := common.GetMongoDB().C(cPayConf)
	query := bson.M{"_id": methodId}
	var payConf PayConf
	if err := c.Find(query).One(&payConf); err != nil {
		return nil
	}
	return &payConf
}
func QueryPayConfByMethodType(methodType string) *PayConf {
	c := common.GetMongoDB().C(cPayConf)
	query := bson.M{"MethodType": methodType}
	var payConf PayConf
	if err := c.Find(query).One(&payConf); err != nil {
		log.Error("not found methodType:%s ,err:%s", methodType, err.Error())
		return nil
	}
	return &payConf
}

func QueryPayConfList() []PayConf {
	c := common.GetMongoDB().C(cPayConf)
	query := bson.M{"IsClose": 0}
	var payConfList []PayConf
	if err := c.Find(query).Sort("Priority").All(&payConfList); err != nil {
		log.Error(err.Error())
		return []PayConf{}
	}
	return payConfList
}

//func QueryPayActivityConfList() []PayActivityConf {
//	c := common.GetMgo().C(cPayActivityConf)
//	query := bson.M{}
//	var payConfList []PayActivityConf
//	if err := c.Find(query).Sort("-Charge").All(&payConfList);err != nil{
//		log.Error(err.Error())
//	}
//	return payConfList
//}
func QueryCompanyBankList() *[]CompanyBank {
	c := common.GetMongoDB().C(cCompanyBankConf)
	var myBankList []CompanyBank
	if err := c.Find(bson.M{"IsClosed": 0}).All(&myBankList); err != nil {
		log.Error(err.Error())
	}
	if len(myBankList) == 0 {
		_ = c.Find(bson.M{"IsClosed": bson.M{"$exists": false}}).All(&myBankList)
	}
	return &myBankList
}
func QueryCompanyBank(id primitive.ObjectID) *CompanyBank {
	c := common.GetMongoDB().C(cCompanyBankConf)
	var bank CompanyBank
	if err := c.FindId(id).One(&bank); err != nil {
		log.Info(err.Error())
		return nil
	}
	return &bank
}

func initPayConf() {
	insertPayConf(newPayConf("gift code", "giftCode", "giftCode", 50000000, 10000, 6, "giftCode", 0))
	insertPayConf(newPayConf("银行转账", "Official", "bank", 50000000, 10000, 1, "银行卡转账", 0))
	//insertPayConf(newPayConf("自动银行转账", "AutoOfficial", "autoBank", 50000000, 10000, 1, "自动银行卡转账",0))
	insertPayConf(newPayConf("Momo", "VgPay", "MomoPay", 50000000, 10000, 2, "Momo", 35))
	insertPayConf(newPayConf("Zalo", "VgPay", "ZaloPay", 50000000, 10000, 3, "Zalo", 35))
	insertPayConf(newPayConf("bankQr", "VgPay", "bankQr", 50000000, 10000, 4, "银行扫码", 25))
	insertPayConf(newPayConf("direct", "VgPay", "direct", 50000000, 10000, 5, "直連", 25))
	insertPayConf(newPayConf("NapTuDong", "NapTuDong", "naptudong", 50000000, 1000, 6, "话费卡", 0))
	insertPayConf(newPayConf("customerService", "CS", "customerService", 50000000, 1000, 7, "后台客服", 0))
	//insertPayConf(newPayConf("NapTuDong", "doiCard", "vinaphone", 50000000, 1000, 7, "vinaphone",25))
	//insertPayConf(newPayConf("NapTuDong", "doiCard", "mobifone", 50000000, 1000, 8, "mobifone",25))
	//insertPayConf(newPayConf("NapTuDong", "doiCard", "vietnamobile", 50000000, 1000, 9, "vietnamobile",25))
	//insertPayConf(newPayConf("gate", "VgPay", "gate", 50000000, 10000, 6, "网关",25))
	//insertPayConf(newPayConf("VTPay", "VgPay", "VTPay", 50000000, 10000, 7, "VTPay",25))
}

func insertPayConf(conf *PayConf) {
	c := common.GetMongoDB().C(cPayConf)
	if err := c.Insert(conf); err != nil {
		log.Error(err.Error())
	}
}

//func insertPayActivity(activityConf *PayActivityConf) {
//	c := common.GetMgo().C(cPayActivityConf)
//	if err := c.Insert(activityConf); err != nil {
//		log.Error(err.Error())
//	}
//}

func insertCompanyBank(bank *CompanyBank) {
	c := common.GetMongoDB().C(cCompanyBankConf)
	if err := c.Insert(bank); err != nil {
		log.Error(err.Error())
	}
}
func initCompanyBankConf() {
	insertCompanyBank(&CompanyBank{BankName: "测试银行1", CardNumber: "110110110110", BankBranch: "测试支行", AccountName: "name"})
	insertCompanyBank(&CompanyBank{BankName: "测试银行2", CardNumber: "220220220220", BankBranch: "测试支行2", AccountName: "name"})
}
func InitPay(incDataExpireDay time.Duration) {
	c := common.GetMongoDB().C(cPayConf)
	count, err := c.Find(bson.M{}).Count()
	if err == nil && count == 0 {
		initPayConf()
	}

	c3 := common.GetMongoDB().C(cCompanyBankConf)
	count3, err := c3.Find(bson.M{}).Count()
	if err == nil && count3 == 0 {
		initCompanyBankConf()
	}
	initDouDouBtList()
	initPhoneChargeConf()
	_ = common.GetMysql().AutoMigrate(&Order{})
	_ = common.GetMysql().AutoMigrate(&OrderTransfer{})
	_ = common.GetMysql().AutoMigrate(&CallBackLog{})
	_ = common.GetMysql().AutoMigrate(&DouDou{})
	_ = common.GetMysql().AutoMigrate(&UserReceiveBt{})
	_ = common.GetMysql().AutoMigrate(&PhoneCharge{})
	createIndex(incDataExpireDay * 6)
}
func createIndex(incDataExpireDay time.Duration) {
	c := common.GetMongoDB().C(cCallBackLog)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create CallBackLog Index: %s", err)
	}
	c2 := common.GetMongoDB().C(cOrder)
	key2 := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c2.CreateIndex(key2, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create Order Index: %s", err)
	}
	c3 := common.GetMongoDB().C(cOrderTransfer)
	key3 := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c3.CreateIndex(key3, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create OrderTransfer Index: %s", err)
	}
	c4 := common.GetMongoDB().C(cDouDou)
	key4 := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c4.CreateIndex(key4, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create DouDou Index: %s", err)
	}
	c5 := common.GetMongoDB().C(cChargeCode)
	key5 := bsonx.Doc{{Key: "Code", Value: bsonx.Int32(1)}}
	if err := c5.CreateIndex(key5, options.Index().SetUnique(true)); err != nil {
		log.Error("create ChargeCode Index: %s", err)
	}
	c51 := common.GetMongoDB().C(cChargeCode)
	key51 := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c51.CreateIndex(key51, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second)*2)); err != nil {
		log.Error("create DouDou Index: %s", err)
	}
	//c7 := common.GetMgo().C(cChargeFromPhone)
	//index7 := mgo.Index{
	//	Key:         []string{"CreateAt"},
	//	Unique:      false,
	//	DropDups:    false,
	//	Background:  false, // See notes.
	//	Sparse:      false,
	//	ExpireAfter: incDataExpireDay,
	//}
	//if err := c7.EnsureIndex(index7); err != nil {
	//	log.Error("create OrderTransfer Index: %s",err)
	//}
}

func InsertOrder(order *Order) {
	c := common.GetMongoDB().C(cOrder)
	if err := c.Insert(order); err != nil {
		log.Error(err.Error())
	}
	o := *order
	common.ExecQueueFunc(func() {
		common.GetMysql().Create(&o)
	})
}

func QueryOrder(orderId primitive.ObjectID) *Order {
	c := common.GetMongoDB().C(cOrder)
	var order Order
	if err := c.Find(bson.M{"_id": orderId}).One(&order); err != nil {
		log.Error(err.Error())
		return nil
	}
	return &order
}
func QueryPayConfByMerchant(merchant string) []primitive.ObjectID {
	var payConf []PayConf
	c := common.GetMongoDB().C(cPayConf)
	if err := c.Find(bson.M{"Merchant": merchant}).All(&payConf); err != nil {
		log.Error(err.Error())
	}
	methodIds := make([]primitive.ObjectID, 0)
	for _, conf := range payConf {
		methodIds = append(methodIds, conf.Oid)
	}
	return methodIds
}
func QueryAllVGPayWaitOrder() []Order {
	c := common.GetMongoDB().C(cOrder)
	var orders []Order
	methodIds := QueryPayConfByMerchant("VgPay")
	query := bson.M{"MethodId": bson.M{"$in": methodIds}, "Status": StatusInit}
	if err := c.Find(query).All(&orders); err != nil {
		log.Error(err.Error())
		return nil
	}
	return orders
}

//func QuerySuccessOrderByUid(userID primitive.ObjectID) []Order {
//	c := common.GetMongoDB().C(cOrder)
//	var order []Order
//	selector := bson.M{"UserId":userID,"Status":StatusSuccess}
//	if err := c.Find(selector).All(&order);err != nil{
//		log.Error(err.Error())
//		return []Order{}
//	}
//	return order
//}
func QueryOrderThirdId(thirdId string) *Order {
	c := common.GetMongoDB().C(cOrder)
	var order Order
	if err := c.Find(bson.M{"ThirdId": thirdId}).One(&order); err != nil {
		log.Error(err.Error())
		return nil
	}
	return &order
}
func QueryOrderLog(uid primitive.ObjectID, offset int, limit int) ([]Order, int64) {
	c := common.GetMongoDB().C(cOrder)
	query := bson.M{"UserId": uid}
	var orderList []Order
	if err := c.Find(query).Sort("-_id").Skip(offset).Limit(limit).
		All(&orderList); err != nil {
	}
	count, _ := c.Find(query).Count()
	return orderList, count
}

func UpdateOrder(order *Order) {
	c := common.GetMongoDB().C(cOrder)
	if err := c.Update(bson.M{"_id": order.Oid}, order); err != nil {
		log.Error(err.Error())
	}
	o := *order
	common.ExecQueueFunc(func() {
		var q Order
		common.GetMysql().First(&q, "oid=?", o.Oid.Hex())
		o.ID = q.ID
		common.GetMysql().Save(&o)
	})
}
func QueryInitOrderByUid(useID primitive.ObjectID, methodID primitive.ObjectID) []Order {
	c := common.GetMongoDB().C(cOrder)
	var order []Order
	selector := bson.M{"UserId": useID, "MethodId": methodID, "Status": StatusInit}
	if err := c.Find(selector).All(&order); err != nil {
		log.Error(err.Error())
		return []Order{}
	}
	return order
}

func QueryInitOrderBy5Minute(useID primitive.ObjectID, methodID primitive.ObjectID) []Order {
	c := common.GetMongoDB().C(cOrder)
	var order []Order
	calcTime := time.Now().Add(-time.Minute * 5)
	selector := bson.M{"UserId": useID, "MethodId": methodID, "Status": StatusInit, "CreateAt": bson.M{"$gt": calcTime}}
	if err := c.Find(selector).All(&order); err != nil {
		log.Error(err.Error())
		return []Order{}
	}
	return order
}
func InsertOrderTransfer(transfer *OrderTransfer) {
	c := common.GetMongoDB().C(cOrderTransfer)
	if err := c.Insert(transfer); err != nil {
		log.Error(err.Error())
	}
	t := *transfer
	common.ExecQueueFunc(func() {
		common.GetMysql().Create(&t)
	})
}
func QueryOrderTransfer(orderId primitive.ObjectID) OrderTransfer {
	c := common.GetMongoDB().C(cOrderTransfer)
	var transfer OrderTransfer
	if err := c.FindId(orderId).One(&transfer); err != nil {
		log.Error(err.Error())
	}
	return transfer
}

func initDouDouBtList() {
	c := common.GetMongoDB().C(cDouDouBt)
	count, err := c.Find(bson.M{}).Count()
	if err != nil {
		log.Error(err.Error())
	}
	if count > 0 {
		return
	}
	insertDouDouBt(&DouDouBt{BtName: "test1", Max: 1000000, Mini: 10000, Priority: 10})
	insertDouDouBt(&DouDouBt{BtName: "test2", Max: 1000000, Mini: 10000, Priority: 12})
}
func insertDouDouBt(bt *DouDouBt) {
	c := common.GetMongoDB().C(cDouDouBt)
	if err := c.Insert(bt); err != nil {
		log.Error(err.Error())
	}
}
func QueryAllDouDouBt() *[]DouDouBt {
	c := common.GetMongoDB().C(cDouDouBt)
	var douDouBts []DouDouBt
	if err := c.Find(bson.M{"IsClosed": 0}).Sort("Priority").All(&douDouBts); err != nil {
		log.Error(err.Error())
	}
	if len(douDouBts) == 0 {
		_ = c.Find(bson.M{"IsClosed": bson.M{"$exists": false}}).All(&douDouBts)
	}
	return &douDouBts
}

func QueryDouDouBtByName(bankName string) *DouDouBt {
	c := common.GetMongoDB().C(cDouDouBt)
	query := bson.M{"BtName": bson.M{"$regex": primitive.Regex{Pattern: bankName, Options: "i"}}}
	var bank DouDouBt
	if err := c.Find(query).One(&bank); err != nil {
		return nil
	}
	return &bank
}

func InsertDouDou(doudou *DouDou) {
	c := common.GetMongoDB().C(cDouDou)
	if err := c.Insert(doudou); err != nil {
		log.Error(err.Error())
	}
	w := *doudou
	common.ExecQueueFunc(func() {
		common.GetMysql().Create(&w)
	})
}
func QueryInitDouDouByUid(userID string) []DouDou {
	c := common.GetMongoDB().C(cDouDou)
	var douDous []DouDou
	selector := bson.M{"UserId": userID, "Status": StatusInit}
	if err := c.Find(selector).All(&douDous); err != nil {
		log.Error(err.Error())
		return []DouDou{}
	}
	return douDous
}
func QueryTodayCount(uid string) int {
	c := common.GetMongoDB().C(cDouDou)
	today := utils.GetTodayTime()
	query := bson.M{"UserId": uid, "CreateAt": bson.M{"$gt": today}, "Status": StatusSuccess}
	count, err := c.Find(query).Count()
	if err != nil {
		log.Error(err.Error())
	}
	return int(count)
}

func UpdateDouDou(douDou *DouDou) {
	c := common.GetMongoDB().C(cDouDou)
	if err := c.Update(bson.M{"_id": douDou.Oid}, douDou); err != nil {
		log.Error(err.Error())
	}
	w := *douDou
	common.ExecQueueFunc(func() {
		var q DouDou
		common.GetMysql().First(&q, "oid=?", w.Oid.Hex())
		w.ID = q.ID
		common.GetMysql().Save(&w)
	})
}
func QueryDouDou(oid primitive.ObjectID) DouDou {
	c := common.GetMongoDB().C(cDouDou)
	var douDou DouDou
	if err := c.FindId(oid).One(&douDou); err != nil {
		log.Error(err.Error())
	}
	return douDou
}
func QueryDouDouByUser(uid string) []DouDou {
	c := common.GetMongoDB().C(cDouDou)
	var douDou []DouDou
	if err := c.Find(bson.M{"UserId": uid, "Status": 0}).All(&douDou); err != nil {

	}
	return douDou
}
func QueryDouDouLog(uid primitive.ObjectID, offset int, limit int) ([]DouDou, int64) {
	c := common.GetMongoDB().C(cDouDou)
	query := bson.M{"UserId": uid.Hex()}
	var douDouList []DouDou
	if err := c.Find(query).Sort("-_id").Skip(offset).Limit(limit).
		All(&douDouList); err != nil {
	}
	count, _ := c.Find(query).Count()
	return douDouList, count
}
func QueryChargeCode(code string) *ChargeCode {
	c := common.GetMongoDB().C(cChargeCode)
	var chargeCode ChargeCode
	if err := c.Find(bson.M{"Code": code}).One(&chargeCode); err != nil {
		return nil
	}
	return &chargeCode
}
func UpdateChargeCode(chargeCode *ChargeCode) error {
	chargeCode.UpdateAt = utils.Now()
	c := common.GetMongoDB().C(cChargeCode)
	query := bson.M{"_id": chargeCode.Oid}
	if err := c.Update(query, chargeCode); err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}
func UpdatePhoneCharge(charge *PhoneCharge) {
	c := common.GetMongoDB().C(cChargeFromPhone)
	if err := c.Update(bson.M{"_id": charge.Oid}, charge); err != nil {
		log.Error(err.Error())
	}
	db := common.GetMysql().Model(charge)
	db.Where("oid=?", charge.Oid.Hex()).Updates(charge)
}
func QueryPhoneCharge(oid primitive.ObjectID) *PhoneCharge {
	c := common.GetMongoDB().C(cChargeFromPhone)
	var phoneCharge PhoneCharge
	if err := c.FindId(oid).One(&phoneCharge); err != nil {
		return nil
	}
	return &phoneCharge
}
func InsertPhoneCharge(phoneCharge *PhoneCharge) {
	c := common.GetMongoDB().C(cChargeFromPhone)
	if err := c.Insert(phoneCharge); err != nil {
		log.Error(err.Error())
	}
	common.GetMysql().Create(phoneCharge)
}

func QueryUserReceiveBt(uid primitive.ObjectID) *UserReceiveBt {
	c := common.GetMongoDB().C(cUserReceiveBt)
	var userReceiveBt UserReceiveBt
	if err := c.FindId(uid).One(&userReceiveBt); err != nil {
		return nil
	}
	return &userReceiveBt
}

func InsertUserReceiveBt(userReceiveBt *UserReceiveBt) {
	c := common.GetMongoDB().C(cUserReceiveBt)
	if err := c.Insert(userReceiveBt); err != nil {
		log.Error(err.Error())
	}
	bt := *userReceiveBt
	common.ExecQueueFunc(func() {
		common.GetMysql().Create(&bt)
	})
}
func QueryAllReceiveBt() []UserReceiveBt {
	c := common.GetMongoDB().C(cUserReceiveBt)
	var userReceiveBt []UserReceiveBt
	if err := c.Find(nil).All(&userReceiveBt); err != nil {
		return []UserReceiveBt{}
	}
	return userReceiveBt
}
func QueryTodayChargeByUid(uid primitive.ObjectID) int64 {
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	c := common.GetMongoDB().C(cOrder)
	payConf := QueryPayConfByMethodType("giftCode")
	var order []Order
	query := bson.M{"UserId": uid,
		"UpdateAt": bson.M{"$gt": thatTime},
		"Status":   StatusSuccess,
		"MethodId": bson.M{"$ne": payConf.Oid},
	}
	if err := c.Find(query).All(&order); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return 0
	}
	res := int64(0)
	for _, v := range order {
		res += v.GotAmount
	}
	return res
}
