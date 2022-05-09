package dxStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
)

type Jackpot struct {
	ID         uint64        `bson:"-" json:"-"`
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Amount     int64         `bson:"Amount"`
	RealAmount int64         `bson:"RealAmount"`
	UpdateAt   time.Time     `bson:"UpdateAt"`
}

func (Jackpot) TableName() string {
	return "dx_jackpot"
}

var (
	cJackpot        = "dxJackpot"
	cJackpotDetails = "dxJackpotDetails"
	cJackpotGameId  = "dxJackpotGameId"
)

type DxJackpotDetails struct {
	ID       uint64    `bson:"-" json:"-"`
	GameId   int64     `bson:"GameId"`
	Uid      string    `bson:"Uid"`
	NickName string    `bson:"NickName"`
	Amount   int64     `bson:"Amount"`
	UserType string    `bson:"UserType"`
	Result   int       `bson:"Result"`
	CreateAt time.Time `bson:"CreateAt"`
}
type DxJackpotGameId struct {
	GameId int64 `bson:"_id,omitempty" json:"GameId"`
}

func insertJackpotGameId(id int64) {
	c := common.GetMongoDB().C(cJackpotGameId)
	if err := c.Insert(&DxJackpotGameId{GameId: id}); err != nil {
		log.Error(err.Error())
	}
}
func queryJackpotGameId(size int) []DxJackpotGameId {
	c := common.GetMongoDB().C(cJackpotGameId)
	var dxJackpotGameIds []DxJackpotGameId
	if err := c.Find(bson.M{}).Sort("-_id").Limit(size).
		All(&dxJackpotGameIds); err != nil {

	}
	return dxJackpotGameIds
}
func (DxJackpotDetails) TableName() string {
	return "dx_jackpot_details"
}

func NewJackpotLog(gameId int64, uid string,nickName string, amount int64,
	userType string, result int) *DxJackpotDetails {
	return &DxJackpotDetails{
		GameId:   gameId,
		Uid:      uid,
		NickName: nickName,
		Amount:   amount,
		UserType: userType,
		Result: result,
		CreateAt: utils.Now(),
	}
}
func InsertJackpotLog(dxJackpotDetails *DxJackpotDetails) {
	//save := *dxJackpotDetails
	c := common.GetMongoDB().C(cJackpotDetails)
	find := bson.M{"GameId" : dxJackpotDetails.GameId,
			"Uid":dxJackpotDetails.Uid,
		}
	var query DxJackpotDetails
	if err := c.Find(find).One(&query);err != nil{
		if err := c.Insert(dxJackpotDetails);err != nil{
			log.Error(err.Error())
		}
	}else{
		update := bson.M{"$inc":bson.M{"Amount":dxJackpotDetails.Amount}}
		if err := c.Update(find, update);err != nil{
			log.Error(err.Error())
		}
	}

	//if dxJackpotDetails.UserType == UserTypeNormal {
	//	common.ExecQueueFunc(func() {
	//		common.GetMysql().Create(&save)
	//	})
	//}
	c2 := common.GetMongoDB().C(cJackpotGameId)
	var jackpotGameId DxJackpotGameId
	if err := c2.FindId(dxJackpotDetails.GameId).One(&jackpotGameId);err!=nil{
		insertJackpotGameId(dxJackpotDetails.GameId)
	}
}

//func QueryJackpotLog(size int64) []JackpotData {
//	c := common.GetMgo().C(cJackpotDetails)
//	pip := []bson.M{
//		{"$group": bson.M{"_id": "$GameId",
//			"JackpotLog1": bson.M{"$push": "$$ROOT"}}},
//		{"$sort": bson.M{"_id": -1}},
//		{"$limit": size},
//	}
//	var res []JackpotData
//	if err := c.Pipe(pip).AllowDiskUse().All(&res); err != nil {
//		log.Error(err.Error())
//	}
//	return res
//}
func QueryJackpotCount() (int,int) {
	c := common.GetMongoDB().C(cGameDx)
	var dxs []Dx
	_ = c.Find(bson.M{"notify.ResultJackpot":1}).Sort("-_id").Limit(100).
		All(&dxs)
	small := 0
	big := 0
	for _,dx := range dxs{
		if dx.Result == ResultBig{
			big++
		}else{
			small ++
		}
	}
	return small, big
}
func QueryJackpotDetails(uid string, gameId int64) *DxJackpotDetails {
	var details DxJackpotDetails
	c := common.GetMongoDB().C(cJackpotDetails)
	query := bson.M{"GameId":gameId,"Uid":uid}
	if err := c.Find(query).One(&details);err != nil{
		return nil
	}
	return &details
}
func QueryJackpotLog2(offset int,limit int) []JackpotData {
	c := common.GetMongoDB().C(cJackpotGameId)
	var jackpotIds []DxJackpotGameId
	if err := c.Find(bson.M{}).Sort("-_id").Skip(offset).Limit(limit).
		All(&jackpotIds); err != nil {

	}
	log.Info("len %v", len(jackpotIds))
    res := make([]JackpotData,0)
	for _, id := range jackpotIds {
		c2 := common.GetMongoDB().C(cJackpotDetails)
		var details []DxJackpotDetails
		_ = c2.Find(bson.M{"GameId": id.GameId}).Limit(20).All(&details)
		jackpotData := JackpotData{}
		dx := QueryDx(id.GameId)
		jackpotData.Jackpot = dx.Jackpot
		jackpotData.Result = dx.Result
		jackpotData.CreateAt = dx.CreateAt
		jackpotData.ResultAmount = dx.BetSmall
		jackpotData.GameId = id.GameId
		jackpotData.JackpotLog = details
		count,_ := c2.Find(bson.M{"GameId": id.GameId}).Count()
		jackpotData.LogCount = int(count)
		res = append(res, jackpotData)
	}
	return res
}
func GetJackpot() *Jackpot {
	c := common.GetMongoDB().C(cJackpot)
	var jackpot Jackpot
	if err := c.Find(bson.M{}).One(&jackpot); err != nil {
		jackpot = Jackpot{UpdateAt: utils.Now()}
		if err := c.Insert(&jackpot); err != nil {
			log.Error(err.Error())
		}
	}
	return &jackpot
}
func UpdateJackpot(jackpot *Jackpot) {
	jackpot.UpdateAt = utils.Now()
	c := common.GetMongoDB().C(cJackpot)
	query := bson.M{"_id": jackpot.Oid}
	if err := c.Update(query, jackpot); err != nil {
		log.Error(err.Error())
	}
	//save := *jackpot
	//common.ExecQueueFunc(func() {
	//	var q Jackpot
	//	common.GetMysql().First(&q, "oid=?",
	//		save.Oid.Hex())
	//	save.ID = q.ID
	//	common.GetMysql().Save(&save)
	//})
}
