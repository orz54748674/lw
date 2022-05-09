package agentStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/log"
	"vn/storage/userStorage"
)

type Invite struct {
	ID         int64              `bson:"-" json:"-"`
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"Oid" gorm:"unique"`
	ParentOid  primitive.ObjectID `bson:"ParentOid"`
	ParentOid2 primitive.ObjectID `bson:"ParentOid2"`
	ParentOid3 primitive.ObjectID `bson:"ParentOid3"`
	AgentOid   primitive.ObjectID `bson:"AgentOid"`
	CreateAt   time.Time          `bson:"CreateAt"`
}

func (Invite) TableName() string {
	return "user_invite"
}

func QueryInvite(uid primitive.ObjectID) Invite {
	c := common.GetMongoDB().C(cUserInvite)
	var invite Invite
	if err := c.Find(bson.M{"_id": uid}).One(&invite); err != nil {
		//log.Error(err.Error())
		// return nil
	}
	return invite
}

var (
	cUserInvite = "userInvite"
)

func queryBelongAgent(parentUid primitive.ObjectID) *userStorage.User {
	if user := userStorage.QueryUser(bson.M{"_id": parentUid}); user != nil { //查爸爸
		if user.Type == userStorage.TypeAgent {
			return user
		} else {
			var agentInvite Invite
			if agentInvite = QueryInvite(parentUid); !agentInvite.Oid.IsZero() {
				if user := userStorage.QueryUser(bson.M{"_id": agentInvite.ParentOid}); user != nil { //查爷爷
					if user.Type == userStorage.TypeAgent {
						return user
					}
				}
			}
		}
	}
	return nil
}
func InsertInvite(uid primitive.ObjectID, parentUid primitive.ObjectID) {
	c := common.GetMongoDB().C(cUserInvite)
	invite := Invite{
		Oid:       uid,
		ParentOid: parentUid,
		CreateAt:  utils.Now(),
	}
	if !parentUid.IsZero() {
		if agentUser := queryBelongAgent(parentUid); agentUser != nil {
			invite.AgentOid = agentUser.Oid
		}
		parentUid2 := QueryInvite(parentUid).ParentOid
		if !parentUid2.IsZero() {
			invite.ParentOid2 = parentUid2
			parentUid3 := QueryInvite(parentUid2).ParentOid
			if !parentUid3.IsZero() {
				invite.ParentOid3 = parentUid3
			}
		}
	}
	if err := c.Insert(&invite); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			db := common.GetMysql()
			db.Create(&invite)
			if !invite.AgentOid.IsZero() {
				now := invite.CreateAt
				agentUid := invite.AgentOid.Hex()
				var agentVipData AgentVipData
				db.FirstOrInit(&agentVipData,
					&AgentVipData{Date: utils.GetDateStr(now), AgentUid: agentUid})
				agentVipData.UpdateAt = now
				agentVipData.TodayNewVip += 1
				agentVipData.SumVip = QuerySumVip(agentUid)
				db.Save(&agentVipData)
			}

		})
	}
}
func QuerySumVip(parentUid string) int64 {
	var count int64
	common.GetMysql().Model(&Invite{}).
		Where("parent_oid=?", parentUid).
		Count(&count)
	return count
}

func QueryInviteData() *[]InviteData {
	c := common.GetMongoDB().C(cUserInvite)
	pipe := mongo.Pipeline{
		{{"$group", bson.M{"_id": "$ParentOid", "Count": bson.M{"$sum": 1},
			"Users": bson.M{"$push": "$_id"}}}},
	}
	var inviteData []InviteData
	if err := c.Pipe(pipe).All(&inviteData); err != nil {
		log.Error(err.Error())
	}
	return &inviteData
}
func QueryAgentInviteData(parentOid primitive.ObjectID) *InviteData {
	c := common.GetMongoDB().C(cUserInvite)
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"ParentOid": parentOid}}},
		{{"$group", bson.M{"_id": "$ParentOid", "Count": bson.M{"$sum": 1},
			"Users": bson.M{"$push": "$_id"}}}},
	}
	var inviteData InviteData
	if err := c.Pipe(pipe).One(&inviteData); err != nil {
		//log.Error(err.Error())
	}
	return &inviteData
}
func QueryAgentInviteDataByDate(parentOid primitive.ObjectID, thatTime time.Time) *InviteData {
	c := common.GetMongoDB().C(cUserInvite)
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"ParentOid": parentOid, "CreateAt": bson.M{"$gt": thatTime}}}},
		{{"$group", bson.M{"_id": "$ParentOid", "Count": bson.M{"$sum": 1},
			"Users": bson.M{"$push": "$_id"}}}},
	}
	var inviteData InviteData
	if err := c.Pipe(pipe).One(&inviteData); err != nil {
		//log.Error(err.Error())
	}
	return &inviteData
}
