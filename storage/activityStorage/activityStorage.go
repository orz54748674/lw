package activityStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/storage/userStorage"
)

var (
	cBetRecord        = "BetRecord"
	cGameDataInBet    = "GameDataInBet"
	cActivityRecord   = "ActivityRecord"
	cActivityConf     = "ActivityConf"
	cActivityUserInfo = "ActivityUserInfo"

	cActivityFirstChargeConf = "ActivityFirstChargeConf"
	cActivityFirstCharge     = "ActivityFirstCharge"

	cActivityTotalChargeConf = "ActivityTotalChargeConf"
	cActivityTotalCharge     = "ActivityTotalCharge"

	cActivitySignInConf = "ActivitySignInConf"
	cActivitySignIn     = "ActivitySignIn"

	cActivityEncouragementConf = "ActivityEncouragementConf"
	cActivityEncouragement     = "ActivityEncouragement"

	cActivityDayChargeConf = "ActivityDayChargeConf"
	cActivityDayCharge     = "ActivityDayCharge"

	cActivityDayGameConf = "ActivityDayGameConf"
	cActivityDayGame     = "ActivityDayGame"

	cActivityDayInviteConf = "ActivityDayInviteConf"
	cActivityDayInvite     = "ActivityDayInvite"

	cActivityVipConf = "ActivityVipConf"
	cActivityVip     = "ActivityVip"
	cActivityVipWeek = "ActivityVipWeek"

	cActivityTurnTableConf    = "ActivityTurnTableConf"
	cActivityTurnTableList    = "ActivityTurnTableList"
	cActivityTurnTableInfo    = "ActivityTurnTableInfo"
	cActivityTurnTableRecord  = "ActivityTurnTableRecord"
	cActivityTurnTableControl = "ActivityTurnTableControl"
)

//----------------------未结算的游戏数据-----------------------
func InitGameDataInBet() {
	c := common.GetMongoDB().C(cGameDataInBet)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create GameDataInBet Index: %s", err)
	}
}

/**
//每次下注和结算时调用 用来统计玩家是否结算了所有游戏   判断结算了所有游戏的条件betCnt为0
betCnt: 下注次数 该值为1 每次下注事件调用     该值为0 结算时清掉下注次数(彩票除外，彩票种类的结算时间不一致，需要开奖的时候，对应的每条下注记录，将该值 -1)
*/
func UpsertGameDataInBet(uid string, gameType game.Type, betInc int) {
	if !QueryActivityIsOpen(Encouragement) { //没开启该活动
		return
	}
	c := common.GetMongoDB().C(cGameDataInBet)
	if betInc == 0 {
		query := bson.M{"Uid": uid, "GameType": gameType}
		update := bson.M{"$set": bson.M{"BetCnt": betInc}}
		if _, err := c.Upsert(query, update); err != nil {
			log.Error(err.Error())
		}
	} else {
		query := bson.M{"Uid": uid, "GameType": gameType}
		update := bson.M{"$inc": bson.M{"BetCnt": betInc}}
		if _, err := c.Upsert(query, update); err != nil {
			log.Error(err.Error())
		}
	}
}
func RemoveAllGameDataInBet() {
	c := common.GetMongoDB().C(cGameDataInBet)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}
func QueryGameDataInBetAll(uid string) []GameDataInBet {
	c := common.GetMongoDB().C(cGameDataInBet)
	var data []GameDataInBet
	query := bson.M{"Uid": uid, "BetCnt": bson.M{"$gt": 0}}
	if err := c.Find(query).All(&data); err != nil {
		log.Error(err.Error())
	}
	return data
}

//----------------------领取记录-----------------------
func InitActivityReceiveRecord(incDataExpireDay time.Duration) {
	c := common.GetMongoDB().C(cActivityRecord)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create ActivityReceiveRecord Index: %s", err)
	}
	_ = common.GetMysql().AutoMigrate(&ActivityRecord{})
}
func InsertActivityReceiveRecord(activityRecord *ActivityRecord) {
	activityRecord.Oid = primitive.NewObjectID()
	c := common.GetMongoDB().C(cActivityRecord)
	err := c.Insert(activityRecord)
	if err != nil {
		log.Error("insert activityReceiveRecord error: %s", err)
	} else {
		//common.ExecQueueFunc(func() {
		userStorage.IncUserActivityTotal(utils.ConvertOID(activityRecord.Uid), activityRecord.Get)
		common.GetMysql().Create(activityRecord)
		//})
	}
}

//----------------------活动配置-----------------------
func InitActivityConf() {
	log.Info("init ActivityConf of mongo db")
	insertActivityConf(&ActivityConf{ActivityType: FirstCharge, Status: 1, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityConf(&ActivityConf{ActivityType: BindPhone, Status: 1, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityConf(&ActivityConf{ActivityType: TotalCharge, Status: 1, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityConf(&ActivityConf{ActivityType: SignIn, Status: 1, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityConf(&ActivityConf{ActivityType: Encouragement, Status: 1, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityConf(&ActivityConf{ActivityType: DayCharge, Status: 1, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityConf(&ActivityConf{ActivityType: DayGame, Status: 1, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityConf(&ActivityConf{ActivityType: DayInvite, Status: 1, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityConf(&ActivityConf{ActivityType: Vip, Status: 0, CreateAt: utils.Now(), UpdateAt: utils.Now()})
	insertActivityConf(&ActivityConf{ActivityType: TurnTable, Status: 0, CreateAt: utils.Now(), UpdateAt: utils.Now()})
}
func insertActivityConf(activityConf *ActivityConf) {
	c := common.GetMongoDB().C(cActivityConf)
	selector := bson.M{"ActivityType": activityConf.ActivityType}
	var conf ActivityConf
	if err := c.Find(selector).One(&conf); err != nil { //不存在则创建
		if err := c.Insert(activityConf); err != nil {
			log.Error(err.Error())
		}
	}
}
func QueryActivityConf() []ActivityConf {
	c := common.GetMongoDB().C(cActivityConf)
	query := bson.M{}
	var activityConf []ActivityConf
	if err := c.Find(query).All(&activityConf); err != nil {
		log.Error(err.Error())
	}
	return activityConf
}
func QueryActivityConfByType(tp ActivityType) *ActivityConf {
	c := common.GetMongoDB().C(cActivityConf)
	selector := bson.M{"ActivityType": tp}
	var activityConf *ActivityConf
	if err := c.Find(selector).One(&activityConf); err != nil {
		return nil
	}
	return activityConf
}
func QueryActivityIsOpen(tp ActivityType) bool {
	conf := QueryActivityConfByType(tp)
	if conf == nil || conf.Status == 0 {
		return false
	}
	return true
}

func QueryTodayTotalCharge(uid string, activityID string) *ActivityTotalCharge {
	c := common.GetMongoDB().C(cActivityTotalCharge)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	var activity ActivityTotalCharge
	query := bson.M{"Uid": uid, "ActivityID": activityID, "CreateAt": bson.M{"$gt": thatTime}}
	if err := c.Find(query).One(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return &activity
}
func QueryTodayTotalChargeByUid(uid string) []ActivityTotalCharge {
	c := common.GetMongoDB().C(cActivityTotalCharge)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	var activity []ActivityTotalCharge
	query := bson.M{"Uid": uid, "CreateAt": bson.M{"$gt": thatTime}}
	if err := c.Find(query).All(&activity); err != nil {
		//log.Info("not found PayActivityRecord ",err)
		return nil
	}
	return activity
}

func QueryBetRecordByUsers(uids []string, time time.Time) int {
	c := common.GetMongoDB().C(cBetRecord)
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"Uid": bson.M{"$in": uids},
			"CreateAt": bson.M{"$gt": time}},
		}},
		{{"$group", bson.M{"_id": "$Uid", "Count": bson.M{"$sum": 1}}}},
	}
	var res []map[string]interface{}
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	return len(res)
}
func QueryTodayBetRecordTotal(uid string, gameType game.Type) int64 {
	c := common.GetMongoDB().C(cBetRecord)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"Uid": uid, "GameType": gameType, "CreateAt": bson.M{"$gt": thatTime}}}},
		{{"$group", bson.M{"_id": "$Uid", "BetAmount": bson.M{"$sum": "$BetAmount"}}}},
	}
	var res []map[string]interface{}
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	if len(res) != 0 {
		return res[0]["BetAmount"].(int64)
	}
	return 0
}
