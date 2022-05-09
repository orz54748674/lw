package dxStorage

import (
	"context"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/storage"
)

func InsertDx(dx *Dx) {
	c := common.GetMongoDB().C(cGameDx)
	if err := c.Insert(dx); err != nil {
		log.Error(err.Error())
	}
}
func QueryLast() *Dx {
	c := common.GetMongoDB().C(cGameDx)
	var dx Dx
	if err := c.Find(bson.M{}).Sort("-_id").One(&dx); err != nil {
		log.Error(err.Error())
	}
	return &dx
}
func QueryDx(showId int64) *Dx {
	c := common.GetMongoDB().C(cGameDx)
	var dx Dx
	if err := c.Find(bson.M{"notify.ShowId": showId}).One(&dx); err != nil {
		log.Error(err.Error())
	}
	return &dx
}
func UpdateDx(dx *Dx) {
	//d := *dx
	c := common.GetMongoDB().C(cGameDx)
	if err := c.Update(bson.M{"_id": dx.Oid}, dx); err != nil {
		log.Error(err.Error())
	}
	//if d.RealBetBig > 0 || d.RealBetSmall > 0 || d.RealRefundBig > 0 || dx.RealRefundSmall > 0 {
	//	common.ExecQueueFunc(func() {
	//		var q Dx
	//		common.GetMysql().First(&q, "oid=?", dx.Oid.Hex())
	//		d.ID = q.ID
	//		common.GetMysql().Save(&d)
	//	})
	//}
}
func NewDxGame() *Dx {
	query := queryGame(0)
	if query.ShowId != 0 && query.Result == 0 {
		return query
	}
	newId := storage.NewGlobalId(storage.KeyGameDx)
	dx := &Dx{
		Notify: Notify{ShowId: newId, CreateAt: utils.Now()},
	}
	dx.Jackpot = GetJackpot().Amount
	InsertDx(dx)
	newDx := QueryLast()
	//common.ExecQueueFunc(func() {
	//	common.GetMysql().Create(newDx)
	//})
	return newDx
}
func IncDxBetLog(dxBetLog *DxBetLog) {
	c := common.GetMongoDB().C(cGameDxBetLog)
	selector := bson.M{
		"Uid":    dxBetLog.Uid,
		"GameId": dxBetLog.GameId,
	}
	if _, err := QueryBetLog(selector); err != nil {
		if err := c.Insert(dxBetLog); err != nil {
			log.Error(err.Error())
		}
	} else {
		//log.Info("DxBetLog Upsert: %v", query)
		var update bson.M
		if dxBetLog.Small > 0 {
			update = bson.M{"$inc": bson.M{"Small": dxBetLog.Small}}
		} else {
			update = bson.M{"$inc": bson.M{"Big": dxBetLog.Big}}
		}
		if _, err := c.Upsert(selector, update); err != nil {
			log.Error(err.Error())
		}
	}
}
func InsertBetLog(dxBetLog *DxBetLog) {
	c := common.GetMongoDB().C(cGameDxBetLog)
	if err := c.Insert(dxBetLog); err != nil {
		log.Error(err.Error())
	}
	//common.ExecQueueFunc(func() {
	//	common.GetMysql().Create(dxBetLog)
	//})
}
func InsertBetLogMany(dxBetLog []DxBetLog)  {
	c := common.GetMongoDB().Collection(cGameDxBetLog)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	var data = make([]interface{},len(dxBetLog))
	for i,d := range dxBetLog{
		data[i] = d
	}
	if _,err := c.InsertMany(ctx,data);err!= nil{
		log.Error(err.Error())
	}
}
func UpdateDxBetLog(dxBetLog *DxBetLog) {
	//save := *dxBetLog
	c := common.GetMongoDB().C(cGameDxBetLog)
	selector := bson.M{
		"_id": dxBetLog.Oid,
	}
	if err := c.Update(selector, dxBetLog); err != nil {
		log.Error(err.Error())
	}
	//if dxBetLog.UserType == UserTypeNormal {
	//	common.ExecQueueFunc(func() {
	//		var q DxBetLog
	//		common.GetMysql().First(&q, "oid=?",
	//			dxBetLog.Oid.Hex())
	//		save.ID = q.ID
	//		common.GetMysql().Save(&save)
	//	})
	//}
}
func QueryBetUidCount(gameId int64, BigOrSmall string) int {
	c := common.GetMongoDB().C(cGameDxBetLog)
	query := bson.M{
		"GameId":   gameId,
		BigOrSmall: bson.M{"$gt": 0},
	}
	count, err := c.Find(query).Count()
	if err != nil {
		log.Error(err.Error())
	}
	return int(count)
}
func QueryBetLog(query bson.M) (*DxBetLog, error) {
	c := common.GetMongoDB().C(cGameDxBetLog)
	var dxBetLog DxBetLog
	err := c.Find(query).One(&dxBetLog)
	return &dxBetLog, err
}
func QueryAllBetLog(gameId int64) *[]DxBetLog {
	c := common.GetMongoDB().C(cGameDxBetLog)
	query := bson.M{"GameId": gameId,
		"HasCheckout": 0,
		//"UserType":userType,
	}
	var dxBetLog []DxBetLog
	if err := c.Find(query).All(&dxBetLog); err != nil {
		return nil
	}
	return &dxBetLog
}
func QueryMyBet(gameId int64, uid string) (int64, int64, int64) {
	c := common.GetMongoDB().C(cGameDxBetLog)
	find := bson.M{"GameId": gameId, "Uid": uid}
	var dxBetLog []DxBetLog
	if err := c.Find(find).All(&dxBetLog); err != nil {
		return 0, 0, 0
	}
	var big int64
	var small int64
	var amount int64
	for _, log := range dxBetLog {
		big += log.Big
		small += log.Small
		amount += log.Result
	}
	return big, small, amount
}
func GetDxConf() *Conf {
	dxConf := QueryDxConf()
	if dxConf == nil {
		dxConf = newDxConf()
		insertDxConf(dxConf)
	}
	return dxConf
}
func QueryDxConf() *Conf {
	c := common.GetMongoDB().C(cGameDxConf)
	var dxConf Conf
	if err := c.Find(bson.M{}).One(&dxConf); err != nil {
		log.Error(err.Error())
		return nil
	}
	return &dxConf
}
func insertDxConf(dxConf *Conf) {
	c := common.GetMongoDB().C(cGameDxConf)
	if err := c.Insert(dxConf); err != nil {
		log.Error(err.Error())
	}
}
func QueryAllRefund(gameId int64, amount int64, refundType string) *[]DxBetLog {
	c := common.GetMongoDB().C(cGameDxBetLog)
	var query bson.M
	if refundType == "small" {
		query = bson.M{
			"GameId":      gameId,
			"HasCheckout": 0,
			"CurSmall":    bson.M{"$gt": amount},
			"Small":       bson.M{"$gt": 0},
		}
	} else {
		query = bson.M{
			"GameId":      gameId,
			"HasCheckout": 0,
			"CurBig":      bson.M{"$gt": amount},
			"Big":         bson.M{"$gt": 0},
		}
	}
	var dxBetLog []DxBetLog
	if err := c.Find(query).All(&dxBetLog); err != nil {
		log.Error(err.Error())
	}
	return &dxBetLog
}
func QueryRealRefundAmount(gameId int64, amount int64, refundType string) int64 {
	c := common.GetMongoDB().C(cGameDxBetLog)
	var query mongo.Pipeline
	if refundType == "small" {
		query = mongo.Pipeline{
			{{
				"$match", bson.M{
					"GameId": gameId,
					//"HasCheckout":0,
					"UserType": UserTypeNormal,
					"CurSmall": bson.M{"$gt": amount},
				},
			}},
			{{"$group",
				bson.M{"_id": "$GameId",
					"sum": bson.M{"$sum": "$Small"}}}},
		}
	} else {
		query = mongo.Pipeline{
			{{
				"$match", bson.M{
					"GameId": gameId,
					//"HasCheckout":0,
					"UserType": UserTypeNormal,
					"CurBig": bson.M{"$gt": amount},
				},
			}},
			{{"$group",
				bson.M{"_id": "$GameId",
				"sum": bson.M{"$sum": "$Big"}}}},
		}
	}
	var q map[string]interface{}
	if err := c.Pipe(query).One(&q); err != nil {
		return 0
	}
	sum, err := utils.ConvertInt(q["sum"])
	if err != nil {
		log.Error(err.Error())
	}
	return sum
}

type DxUserBet struct {
	Uid    string `bson:"_id"`
	Small  int64  `bson:"Small"`
	Big    int64  `bson:"Big"`
	Result int64  `bson:"Result"`
	Refund int64  `bson:"Refund"`
	UserType []string  `bson:"UserType"`
}

func (s *DxUserBet) GetRealBet() int64 {
	return s.Small + s.Big - s.Refund
}
//func QueryDxBetLogUnCheckOut(gameId int64) []DxUserBet {
//	c := common.GetMongoDB().C(cGameDxBetLog)
//	query := mongo.Pipeline{
//		{
//			{"$match", bson.D{
//				{"GameId", gameId},
//				//"HasCheckout":0,
//				{"UserType",    UserTypeNormal},
//				{"HasCheckout", 0},
//			}},
//		},
//		{{"$group", bson.D{{"_id", "$Uid"},
//			{"Small",  bson.D{{"$sum", "$Small"}}},
//			{"Big",    bson.D{{"$sum", "$Big"}}},
//			{"Result", bson.D{{"$sum", "$Result"}}},
//		}}},
//	}
//	var dxBetLogs []DxUserBet
//	if err := c.Pipe(query).All(&dxBetLogs); err != nil {
//	}
//	return dxBetLogs
//}
func QueryDxBetLogNeedNotify(gameId int64) []DxUserBet {
	c := common.GetMongoDB().C(cGameDxBetLog)
	query := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"GameId", gameId},
				//"HasCheckout":0,
				{"UserType",    bson.M{"$nin": []string{UserTypeBot}}},
				{"HasCheckout", 1},
			}},
		},
		{{"$group", bson.D{
			{"_id", "$Uid"},
			{"Small",  bson.D{{"$sum", "$Small"}}},
			{"Big",    bson.D{{"$sum", "$Big"}}},
			{"Result", bson.D{{"$sum", "$Result"}}},
			{"Refund", bson.D{{"$sum", "$Refund"}}},
			{"UserType", bson.D{{"$push", "$UserType"}}},
		}}},
	}
	var dxBetLogs []DxUserBet
	if err := c.Pipe(query).All(&dxBetLogs); err != nil {
		//log.Error(err.Error())
	}
	return dxBetLogs
}

func Init(incDataExpireDay time.Duration) {
	c := common.GetMongoDB().C(cGameDxBetLog)
	key := bsonx.Doc{{Key: "CreateAt",Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key,options.Index().
		SetExpireAfterSeconds(int32(3*24*time.Hour/time.Second)));err != nil{
		log.Error("create GameDxBetLog Index: %s",err)
	}
	c = common.GetMongoDB().C(cGameDxBetLog)
	key2 := bsonx.Doc{{Key: "Uid",Value: bsonx.Int32(1)},{Key: "GameId",Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key2,options.Index());err != nil{
		log.Error("create GameDxBetLog Index: %s",err)
	}
	c = common.GetMongoDB().C(cGameDxBetLog)
	key3 := bsonx.Doc{{Key: "GameId",Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key3,options.Index());err != nil{
		log.Error("create GameDxBetLog Index: %s",err)
	}
	c2 := common.GetMongoDB().C(cGameDx)
	k2 := bsonx.Doc{{Key: "CreateAt",Value: bsonx.Int32(1)}}
	if err := c2.CreateIndex(k2,options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second)));err != nil{
		log.Error("create dx Index: %s",err)
	}
	c3 := common.GetMongoDB().C(cJackpotDetails)
	k3 := bsonx.Doc{{Key: "CreateAt",Value: bsonx.Int32(1)}}
	if err := c3.CreateIndex(k3,options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second)));err != nil{
		log.Error("create JackpotDetails Index: %s",err)
	}
	c3 = common.GetMongoDB().C(cJackpotDetails)
	k31 := bsonx.Doc{{Key: "GameId",Value: bsonx.Int32(1)}}
	if err := c3.CreateIndex(k31,options.Index());err != nil{
		log.Error("create JackpotDetails Index: %s",err)
	}
	//_ = common.GetMysql().AutoMigrate(&Dx{})
	//_ = common.GetMysql().AutoMigrate(&DxBetLog{})
	//_ = common.GetMysql().AutoMigrate(&Jackpot{})
	//_ = common.GetMysql().AutoMigrate(&DxJackpotDetails{})
}