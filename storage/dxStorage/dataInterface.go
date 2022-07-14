package dxStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/log"
	"vn/storage/botStorage"
	"vn/storage/userStorage"
)

type History struct {
	GameId   int64     `bson:"GameId"`
	Result   uint8     `bson:"Result"`
	Dice1    uint8     `bson:"Dice1"`
	Dice2    uint8     `bson:"Dice2"`
	Dice3    uint8     `bson:"Dice3"`
	CreateAt time.Time `bson:"CreateAt"`
}

func GetHistory(size int) *[]History {
	c := common.GetMongoDB().C(cGameDx)
	query := bson.M{"notify.Result": bson.M{"$gt": 0}}
	var res []Dx
	err := c.Find(query).Sort("-_id").Limit(size).All(&res)
	if err != nil {
		log.Error(err.Error())
	}
	var historys []History
	for _, dx := range res {
		history := History{
			GameId:   dx.ShowId,
			Result:   dx.Result,
			Dice1:    dx.Dice1,
			Dice2:    dx.Dice2,
			Dice3:    dx.Dice3,
			CreateAt: dx.CreateAt,
		}
		historys = append(historys, history)
	}
	return &historys
}

type Details struct {
	Dx           Notify
	BetBigList   []OneBet
	BetSmallList []OneBet
}
type OneBet struct {
	CreateAt  time.Time
	Name      string
	Avatar    string
	BetAmount int64
	Refund    int64
	Uid       string
	Type      int
}

type JackpotData struct {
	GameId       int64 `bson:"_id,omitempty"`
	Result       uint8
	CreateAt     time.Time
	ResultAmount int64
	Jackpot      int64
	LogCount     int
	JackpotLog   []DxJackpotDetails `bson:"JackpotLog"`
}

func GetGameDetails(gameId int64) *Details {
	dx := queryGame(gameId)
	allBet := QueryDetails(gameId)
	details := &Details{}
	var smallList []OneBet
	var bigList []OneBet
	details.Dx = dx.Notify
	uids := []int64{}
	for _, bet := range allBet {
		oneBet := OneBet{
			Uid:       bet["_id"].(string),
			CreateAt:  bet["CreateAt"].(primitive.A)[0].(primitive.DateTime).Time(),
			BetAmount: bet["Small"].(int64) + bet["Big"].(int64),
			Refund:    bet["Refund"].(int64),
		}
		if bet["UserType"].(primitive.A)[0].(string) == UserTypeNormal {
			uOid, _ := primitive.ObjectIDFromHex(oneBet.Uid)
			user := userStorage.QueryUserId(uOid)
			oneBet.Name = user.NickName
			oneBet.Avatar = user.Avatar
			oneBet.Type = 1
		} else {
			uid, _ := utils.ConvertInt(oneBet.Uid)
			uids = append(uids, uid)
			oneBet.Type = 2
		}
		if bet["Small"].(int64) > 0 {
			smallList = append(smallList, oneBet)
		} else {
			bigList = append(bigList, oneBet)
		}
	}
	botMap := botStorage.QueryBotByUid(uids)
	for i, one := range smallList {
		if one.Type != 1 {
			uid, _ := utils.ConvertInt(one.Uid)
			bot := botMap[uid]
			smallList[i].Name = bot.NickName
			smallList[i].Avatar = bot.Avatar
		}
	}
	for i, one := range bigList {
		if one.Type != 1 {
			uid, _ := utils.ConvertInt(one.Uid)
			bot := botMap[uid]
			bigList[i].Name = bot.NickName
			bigList[i].Avatar = bot.Avatar
		}
	}
	details.BetSmallList = smallList
	details.BetBigList = bigList
	return details
}

type UserBet struct {
	Uid      string    `bson:"_id"`
	Big      int64     `bson:"Big"`
	Small    int64     `bson:"Small"`
	UserType string    `bson:"UserType"`
	Refund   int64     `bson:"Refund"`
	CreateAt time.Time `bson:"CreateAt"`
}

func QueryDetails(gameId int64) []map[string]interface{} {
	c := common.GetMongoDB().C(cGameDxBetLog)
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"GameId": gameId}}},
		{{"$group", bson.M{
			"_id":      "$Uid",
			"Small":    bson.M{"$sum": "$Small"},
			"Big":      bson.M{"$sum": "$Big"},
			"Refund":   bson.M{"$sum": "$Refund"},
			"UserType": bson.M{"$push": "$UserType"},
			"CreateAt": bson.M{"$push": "$CreateAt"},
		}}},
		{{"$sort", bson.M{"CreateAt": -1}}},
	}
	var data []map[string]interface{}
	if err := c.Pipe(pipe).All(&data); err != nil {
		log.Info(err.Error())
	}
	return data
}
func queryGame(gameId int64) *Dx {
	c := common.GetMongoDB().C(cGameDx)
	var dx Dx
	if gameId == 0 {
		if err := c.Find(bson.M{}).Sort("-_id").Skip(1).One(&dx); err != nil {
			log.Error(err.Error())
		}
	} else {
		query := bson.M{"notify.ShowId": gameId}
		if err := c.Find(query).One(&dx); err != nil {
			log.Error(err.Error())
		}
	}
	dx.CreateAt = dx.CreateAt.Local()
	return &dx
}
