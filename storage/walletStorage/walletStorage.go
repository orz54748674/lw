package walletStorage

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

var (
	CWallet = "wallet"
	CBill   = "bill"
)

/**
不存在会创建
*/
func GetWallet(uid primitive.ObjectID) *Wallet {
	wallet := QueryWallet(uid)
	if wallet == nil {
		wallet = newWallet(uid)
		UpsertWallet(wallet)
		//w := *wallet
		//common.ExecQueueFunc(func() {
		//common.GetMysql().Create(&w)
		//})
	}
	return wallet
}

/**
仅查询
*/
func QueryWallet(uid primitive.ObjectID) *Wallet {
	c := common.GetMongoDB().C(CWallet)
	var wallet Wallet
	if err := c.Find(bson.M{"_id": uid}).One(&wallet); err != nil {
		return nil
	}
	return &wallet
}
func UpsertWallet(wallet *Wallet) {
	wallet.UpdateAt = utils.Now()
	c := common.GetMongoDB().C(CWallet)

	selector := bson.M{"_id": wallet.Oid}
	//update := structs.Map(wallet)
	if _, err := c.Upsert(selector, wallet); err != nil {
		log.Error(err.Error())
	}
	w := *wallet
	var q Wallet
	common.GetMysql().First(&q, "oid=?", w.Oid.Hex())
	w.ID = q.ID
	common.GetMysql().Save(&w)
	//common.ExecQueueFunc(func() {
	//
	//})
}

func NewBill(uid string, Type string, event string, eventId string, amount int64) *Bill {
	bill := &Bill{
		Uid:      uid,
		Type:     Type,
		Event:    event,
		EventId:  eventId,
		Amount:   amount,
		Status:   StatusInit,
		CreateAt: utils.Now(),
		UpdateAt: utils.Now(),
	}
	return bill
}

func Init(incDataExpireDay time.Duration) {
	c := common.GetMongoDB().C(CBill)
	key := bsonx.Doc{{Key: "UpdateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create Bill Index: %s", err)
	}
	_ = common.GetMysql().AutoMigrate(&Wallet{})
	_ = common.GetMysql().AutoMigrate(&Bill{})
}
func QueryWalletByUids(uid []primitive.ObjectID) []Wallet {
	c := common.GetMongoDB().C(CWallet)
	query := bson.M{"_id": bson.M{"$in": uid}}
	var wallet []Wallet
	if err := c.Find(query).All(&wallet); err != nil {
		log.Error(err.Error())
		return []Wallet{}
	}
	return wallet
}

//func QueryBillByBet(uids []string, time time.Time)int {
//	c := common.GetMongoDB().C(CBill)
//	find := bson.M{
//		"Uid":bson.M{"$in":uids},
//		"Type":TypeExpenses,
//		"CreateAt": bson.M{"$gt":time},
//	}
//	pipe := []bson.M{
//		{"$match":find},
//		{"$group":bson.M{"_id":"$Uid","Count":bson.M{"$sum":1}}},
//	}
//	//count,err := c.Find(find).Count()
//	var res []map[string]interface{}
//	err := c.Pipe(pipe).All(&res)
//	if err != nil {
//		log.Error(err.Error())
//	}
//	return len(res)
//}
