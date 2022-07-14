package gameStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

type GameOverview struct {
	ID          int64     `bson:"-" json:"-"`
	Channel     string    `bson:"Channel"`
	ChannelName string    `bson:"ChannelName"`
	GameType    game.Type `bson:"GameType"`
	BetValue    int64     `bson:"BetValue"`
	BetUsers    int       `bson:"BetUsers"`    //下注人数
	SysProfit   int64     `bson:"SysProfit"`   //系统抽水
	BotProfit   int64     `bson:"BotProfit"`   //机器人抽水
	AgentProfit int64     `bson:"AgentProfit"` //代理佣金
	NetProfit   int64     `bson:"NetProfit"`   //纯利润
	UpdateAt    time.Time `bson:"UpdateAt"`
}
type UserOverview struct {
	ID               int64     `bson:"-" json:"-"`
	Channel          string    `bson:"Channel"`
	ChannelName      string    `bson:"ChannelName"`
	UserAdd          int       `bson:"UserAdd"`          //新增用户
	UserLogin        int       `bson:"UserLogin"`        //用户登录数
	ChargeAmount     int64     `bson:"ChargeAmount"`     //充值金额
	AddChargeUsers   int       `bson:"AddChargeUsers"`   //新增充值人数
	AddChargeAmount  int       `bson:"AddChargeAmount"`  //新增充值金额
	FirstChargeUsers int       `bson:"FirstChargeUsers"` //首次充值人数
	ChargeUsers      int       `bson:"ChargeUsers"`      //充值人数
	DouDouAmount     int64     `bson:"DouDouAmount"`     //换豆豆金额
	DouDouUsers      int       `bson:"DouDouUsers"`      //换豆豆人数
	UserTotal        int       `bson:"UserTotal"`        //注册总数
	VndBalance       int64     `bson:"VndBalance"`       //账户余额
	AgentBalance     int64     `bson:"AgentBalance"`     //佣金余额
	ActivityReceive  int64     `bson:"ActivityReceive"`  //活动领取金额
	UpdateAt         time.Time `bson:"UpdateAt"`
}

var (
	cGameOverview = "gameOverview"
	cUserOverview = "userOverview"
)

func InitGameOverview() {
	_ = common.GetMysql().AutoMigrate(&GameOverview{})
}
func InitUserOverview() {
	_ = common.GetMysql().AutoMigrate(&UserOverview{})
}
func UpsertGameOverview(gameOverview *GameOverview) {
	b := *gameOverview
	var queryB GameOverview
	common.GetMysql().First(&queryB,
		"channel=? and game_type=? and DATEDIFF(update_at,NOW())=0",
		b.Channel, b.GameType)
	b.ID = queryB.ID
	b.BetValue += queryB.BetValue
	b.SysProfit += queryB.SysProfit
	b.BotProfit += queryB.BotProfit
	b.AgentProfit += queryB.AgentProfit
	b.NetProfit = b.SysProfit + b.BotProfit - b.AgentProfit
	b.UpdateAt = time.Now()
	common.GetMysql().Save(&b)
}
func QueryGameOverview(channel string, gameType game.Type, time time.Time) *GameOverview {
	var gameOverview GameOverview
	common.GetMysql().First(&gameOverview,
		"channel=? and game_type=? and DATEDIFF(update_at,?)=0",
		channel, gameType, time)
	return &gameOverview
}

func UpsertUserOverview(userOverview *UserOverview) {
	b := *userOverview
	var queryB UserOverview
	common.GetMysql().First(&queryB,
		"channel=? and DATEDIFF(update_at,NOW())=0",
		b.Channel)
	b.ID = queryB.ID
	b.UpdateAt = time.Now()
	common.GetMysql().Save(&b)
}
func QueryUidsByChannel(channel string) ([]string, []primitive.ObjectID) {
	c := common.GetMongoDB().C("user")
	selector := bson.M{"Channel": channel, "Type": userStorage.TypeNormal}
	var user []userStorage.User
	if err := c.Find(selector).All(&user); err != nil {
		log.Info("not found users: ,err: %v", err)
		return nil, nil
	}
	var Uids []string
	var UidHexs []primitive.ObjectID
	for _, v := range user {
		Uids = append(Uids, v.Oid.Hex())
		UidHexs = append(UidHexs, v.Oid)
	}
	return Uids, UidHexs
}
func QueryNewUidsByChannel(channel string, time time.Time) []primitive.ObjectID {
	c := common.GetMongoDB().C("user")
	selector := bson.M{"Channel": channel, "CreateAt": bson.M{"$gte": time}, "Type": userStorage.TypeNormal}
	var user []userStorage.User
	if err := c.Find(selector).All(&user); err != nil {
		log.Info("not found users: ,err: %v", err)
		return nil
	}
	//	var Uids []string
	var UidHexs []primitive.ObjectID
	for _, v := range user {
		//		Uids = append(Uids,v.Oid.Hex())
		UidHexs = append(UidHexs, v.Oid)
	}
	return UidHexs
}
func QueryNewUsers(channel string, time time.Time) int {
	c := common.GetMongoDB().C("user")
	selector := bson.M{"Channel": channel, "CreateAt": bson.M{"$gte": time}, "Type": userStorage.TypeNormal}
	num, _ := c.Find(selector).Count()
	return int(num)
}
func QueryUserLogin(uidHexs []primitive.ObjectID, time time.Time) int {
	c := common.GetMongoDB().C("userLogin")
	selector := bson.M{"Oid": bson.M{"$in": uidHexs}, "LastTime": bson.M{"$gte": time}}
	num, _ := c.Find(selector).Count()
	return int(num)
}
func QueryChargeData(uidHexs []primitive.ObjectID, newUidsHex []primitive.ObjectID, time time.Time) map[string]interface{} {
	c := common.GetMongoDB().C("order")
	payConf := payStorage.QueryPayConfByMethodType("giftCode")
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"UpdateAt": bson.M{"$gte": time},
			"UserId":   bson.M{"$in": uidHexs},
			"Status":   payStorage.StatusSuccess,
			"MethodId": bson.M{"$ne": payConf.Oid},
		},
		}},
		{{"$group", bson.M{"_id": "$UserId",
			"ChargeAmount": bson.M{"$sum": "$Amount"},
			"Fee":          bson.M{"$sum": "$Fee"},
		},
		}},
	}
	var res []map[string]interface{}
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	ret := make(map[string]interface{})
	var ChargeAmount int64 = 0
	var Fee int64 = 0
	var chargeUids []primitive.ObjectID
	chargeUids = []primitive.ObjectID{}
	for _, v := range res {
		ChargeAmount += v["ChargeAmount"].(int64)
		Fee += v["Fee"].(int64)
		chargeUids = append(chargeUids, v["_id"].(primitive.ObjectID))
	}
	ret["Fee"] = Fee
	ret["ChargeAmount"] = ChargeAmount
	ret["ChargeUsers"] = len(res)

	pipe = mongo.Pipeline{
		{{"$match", bson.M{"UpdateAt": bson.M{"$gte": time},
			"UserId":   bson.M{"$in": newUidsHex},
			"Status":   payStorage.StatusSuccess,
			"MethodId": bson.M{"$ne": payConf.Oid},
		}},
		},
		{{"$group", bson.M{"_id": "$UserId",
			"ChargeAmount": bson.M{"$sum": "$Amount"},
			"Fee":          bson.M{"$sum": "$Fee"},
		},
		}},
	}
	res = append(res[:0], res[len(res):]...)
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	ChargeAmount = 0
	Fee = 0
	for _, v := range res {
		ChargeAmount += v["ChargeAmount"].(int64)
		Fee += v["Fee"].(int64)
	}
	ret["AddFee"] = Fee
	ret["AddChargeAmount"] = ChargeAmount
	ret["AddChargeUsers"] = len(res)

	pipe = mongo.Pipeline{
		{{"$match", bson.M{"UpdateAt": bson.M{"$lt": time},
			"UserId":   bson.M{"$in": chargeUids},
			"Status":   payStorage.StatusSuccess,
			"MethodId": bson.M{"$ne": payConf.Oid},
		}},
		},
		{{"$group", bson.M{"_id": "$UserId"}}},
	}
	res = append(res[:0], res[len(res):]...)
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	ret["FirstChargeUsers"] = ret["ChargeUsers"].(int) - len(res)
	return ret
}
func QueryDouDouData(uids []string, time time.Time) map[string]interface{} {
	c := common.GetMongoDB().C("doudou")
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"UpdateAt": bson.M{"$gte": time},
			"UserId": bson.M{"$in": uids},
			"Status": payStorage.StatusSuccess,
		},
		}},
		{{"$group", bson.M{"_id": "$UserId",
			"DouDouAmount": bson.M{"$sum": "$Amount"},
		},
		}},
	}
	var res []map[string]interface{}
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	ret := make(map[string]interface{})
	var DouDouAmount int64 = 0
	for _, v := range res {
		DouDouAmount += v["DouDouAmount"].(int64)
	}
	ret["DouDouAmount"] = DouDouAmount
	ret["DouDouUsers"] = len(res)
	return ret
}
func QueryWalletData(uidHexs []primitive.ObjectID) map[string]interface{} {
	wallet := walletStorage.QueryWalletByUids(uidHexs)
	ret := make(map[string]interface{})
	var VndBalance int64 = 0
	var AgentBalance int64 = 0
	var SafeBalance int64 = 0
	for _, v := range wallet {
		VndBalance += v.VndBalance
		AgentBalance += v.AgentBalance
		SafeBalance += v.SafeBalance
	}
	ret["VndBalance"] = VndBalance
	ret["AgentBalance"] = AgentBalance
	ret["SafeBalance"] = SafeBalance
	return ret
}
func QueryActivityAmount(uids []string, time time.Time) int64 {
	c := common.GetMongoDB().C("bill")
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"UpdateAt": bson.M{"$gte": time},
			"Uid": bson.M{"$in": uids},
			//"$or": []bson.M{{"Event": walletStorage.EventBindPhone}, {"Event": walletStorage.EventActivityAward}, {"Event": walletStorage.EventGiftCode}},
			"$or": []bson.M{{"Event": bson.M{"$in": walletStorage.ActivityEvent}}},
		},
		}},
		{{"$group", bson.M{"_id": "$Uid",
			"Amount": bson.M{"$sum": "$Amount"},
		},
		}},
	}
	var res []map[string]interface{}
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	var Amount int64 = 0
	for _, v := range res {
		Amount += v["Amount"].(int64)
	}
	return Amount
}
