package fishStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/log"
)

var (
	cRoomData           = "FishRoomData"
	cFishConf           = "FishConf"           //捕鱼配置表
	cFishPlayerFireInfo = "FishPlayerFireInfo" //玩家每日发射子弹表
	cFishPlayerConf     = "FishPlayerConf"     //玩家信息
	cFishSysBalance     = "FishSysBalance"     //捕鱼三个房间的系统余额
)

func GetTablesInfo() map[string]TableInfo {
	c := common.GetMongoDB().C(cRoomData)
	var roomData RoomData
	if err := c.Find(nil).One(&roomData); err != nil {
		log.Error("GetTablesInfo err:%s", err.Error())
		return nil
	}
	return roomData.TablesInfo
}

func UpsertTablesInfo(tablesInfo map[string]TableInfo) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	update := bson.M{"$set": bson.M{"TablesInfo": tablesInfo}}

	_, err := c.Upsert(nil, update)
	if err != nil {
		log.Error("UpsertTablesInfo error: %s", err.Error())
		return nil
	}
	return nil
}

func GetFishSysBalance(tableType int) (int64, int64) {
	c := common.GetMongoDB().C(cFishSysBalance)
	var tmp FishSysBalance
	date := time.Now().Format("2006-01-02")
	query := bson.M{"Date": date}
	if err := c.Find(query).One(&tmp); err != nil {
		log.Error("GetFishSysBalance err:%s", err.Error())
		tmp.Date = date
		if err = c.Insert(&tmp); err != nil {
			log.Error("GetFishSysBalance insert err:%s", err.Error())
		}
		return 0, 0
	}
	if tableType == 1 {
		return tmp.Room1, tmp.EffectBetRoom1
	}
	if tableType == 2 {
		return tmp.Room2, tmp.EffectBetRoom2
	}
	if tableType == 3 {
		return tmp.Room3, tmp.EffectBetRoom3
	}
	return 0, 0
}

func UpsertFishSysBalance(tableType int, balance int64, effectBet int64) {
	c := common.GetMongoDB().C(cFishSysBalance)
	roomStr := ""
	effectBetStr := ""
	if tableType == 1 {
		roomStr = "Room1"
		effectBetStr = "EffectBetRoom1"
	}
	if tableType == 2 {
		roomStr = "Room2"
		effectBetStr = "EffectBetRoom2"
	}
	if tableType == 3 {
		roomStr = "Room3"
		effectBetStr = "EffectBetRoom3"
	}
	update := bson.M{"$inc": bson.M{roomStr: balance, effectBetStr: effectBet}}
	query := bson.M{"Date": time.Now().Format("2006-01-02")}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error("UpsertFishSysBalance err:%s", err.Error())
	}
}

func GetFishConf() FishConf {
	c := common.GetMongoDB().C(cFishConf)
	var fishConf FishConf
	if err := c.Find(nil).One(&fishConf); err != nil {
		log.Error("GetFishConf err:%s", err.Error())
		fishConf.CannonConf = map[int][]int64{
			1: {100, 200, 300, 500, 800, 1000},
			2: {1000, 2000, 3000, 5000, 8000, 10000},
			3: {10000, 20000, 30000, 60000, 80000, 100000},
		}
		fishConf.LunZhouRewardArr = []int{30, 30, 40, 50, 60, 70, 80, 90, 100, 200, 300}
		fishConf.RateByFireArr = []RateByFireInfo{{Rate: 0.005, FireMin: 0, FireMax: 300}, {Rate: 0.01, FireMin: 301, FireMax: 500}, {Rate: 0.02, FireMin: 501, FireMax: 650}, {Rate: -0.07, FireMin: 651, FireMax: 800}, {Rate: 0, FireMin: 801, FireMax: 1000000000}}
		fishConf.SysRoom1 = SysBalanceRateInfo{BalanceMin: 100000000, BalanceMax: 1000000000, Rate: 0.05}
		fishConf.SysRoom2 = SysBalanceRateInfo{BalanceMin: 100000000, BalanceMax: 1000000000, Rate: 0.05}
		fishConf.SysRoom3 = SysBalanceRateInfo{BalanceMin: 100000000, BalanceMax: 1000000000, Rate: 0.05}
		fishConf.RateRoom1 = 0.03
		fishConf.RateRoom2 = 0
		fishConf.RateRoom3 = -0.03
		fishConf.BlockRate = -0.3
		fishConf.EffectBetRoom1 = EffectBetRate{MinValue: 0.01, MinRate: -0.05, MaxValue: 0.05, MaxRate: 0.03}
		fishConf.EffectBetRoom2 = EffectBetRate{MinValue: 0.01, MinRate: -0.05, MaxValue: 0.05, MaxRate: 0.03}
		fishConf.EffectBetRoom3 = EffectBetRate{MinValue: 0.01, MinRate: -0.05, MaxValue: 0.05, MaxRate: 0.03}
		c.Insert(&fishConf)
	}
	return fishConf
}

func GetFishPlayerConf(uid, account string) FishPlayerConf {
	var playerConf FishPlayerConf
	c := common.GetMongoDB().C(cFishPlayerConf)
	query := bson.M{"uid": uid}
	if err := c.Find(query).One(&playerConf); err != nil {
		log.Error("GetFishPlayerConf err:%s, uid:%s", err.Error(), uid)
		if err == mongo.ErrNoDocuments {
			playerConf.Uid = uid
			playerConf.Account = account
			playerConf.UpdateTime = time.Now()
			if err = c.Insert(&playerConf); err != nil {
				log.Error("GetFishPlayerConf insert err:%s", err.Error())
			}
		}
	}
	return playerConf
}

func RemoveRoomData() *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	_, err := c.RemoveAll(bson.M{})
	if err != nil {
		log.Error("RemoveTableInfo error: %s", err.Error())
		return nil
	}
	return nil
}

func GetFireTimes(uid string, tableType int) int {
	c := common.GetMongoDB().C(cFishPlayerFireInfo)
	date := time.Now().Format("2006-01-02")
	query := bson.M{"Uid": uid, "Date": date}
	var info FishPlayerFireInfo
	if err := c.Find(query).One(&info); err != nil {
		info.Uid = uid
		info.Date = date
		info.UpdateAt = time.Now()
		c.Insert(&info)
	}
	if tableType == 1 {
		return info.Room1
	}
	if tableType == 2 {
		return info.Room2
	}
	if tableType == 3 {
		return info.Room3
	}
	return 0
}

func UpsertFireTimes(uid string, tableType, fireTimes int) {
	c := common.GetMongoDB().C(cFishPlayerFireInfo)
	date := time.Now().Format("2006-01-02")
	query := bson.M{"Uid": uid, "Date": date}

	tableStr := ""
	if tableType == 1 {
		tableStr = "Room1"
	}
	if tableType == 2 {
		tableStr = "Room2"
	}
	if tableType == 3 {
		tableStr = "Room3"
	}

	update := bson.M{"$set": bson.M{tableStr: fireTimes, "UpdateAt": time.Now()}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
}
