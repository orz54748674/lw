package main

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	"vn/game/pay/payWay"
	"vn/storage/agentStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"

	"gorm.io/gorm"
)

func successVgPayOrder(orderId string) {
	order := payStorage.QueryOrder(utils.ConvertOID(orderId))
	if order != nil && order.Status != payStorage.StatusSuccess {
		if payConf := payStorage.QueryPayConf(order.MethodId); payConf != nil {
			order.Fee = order.Amount * int64(payConf.FeePerThousand) / 1000
		}
		payWay.SuccessOrder(order)
		log.Info("SuccessOrder: %s", orderId)
	}
	select {}
}

func checkUserReceiveBt() {
	c := common.GetMongoDB().C("userReceiveBt")
	var userReceiveBts []payStorage.UserReceiveBt
	if err := c.Find(nil).All(&userReceiveBts); err != nil {
		log.Error(err.Error())
	}
	db := common.GetMysql().Model(&payStorage.UserReceiveBt{})
	for _, bt := range userReceiveBts {
		if bt.CreateAt.IsZero() {
			bt.CreateAt = time.Unix(0, 0)
		}
		var query payStorage.UserReceiveBt
		db.Where("oid=?", bt.Oid.Hex()).First(&query)
		bt.ID = query.ID
		db.Save(&bt)
	}

}
func checkAgentMember() {
	db := common.GetMysql().Model(&agentStorage.AgentMemberData{})
	var datas []agentStorage.AgentMemberData
	db.Debug().Select("distinct account").Find(&datas)
	for _, d := range datas {
		user := queryUserAccount(d.Account)
		invite := queryInvite(user.Oid.Hex())
		parentUser := queryUserId(invite.ParentOid.Hex())
		belongAgent := queryUserId(invite.AgentOid.Hex())
		d.SuperiorAccount1 = parentUser.Account
		d.BelongAgent = belongAgent.Account
		if d.CreateAt.IsZero() {
			d.CreateAt = time.Unix(0, 0)
		}
		if d.UpdateAt.IsZero() {
			d.CreateAt = time.Unix(0, 0)
		}
		//db.Where("id=?",d.ID).Save(&d)
		update := map[string]interface{}{
			"superior_account1": d.SuperiorAccount1,
			"belong_agent":      d.BelongAgent,
		}
		db.Debug().Where("account=?", d.Account).Updates(update)
		log.Info("save id:%v", d.ID)
	}
}
func queryInvite(uid string) agentStorage.Invite {
	db := common.GetMysql().Model(&agentStorage.Invite{})
	var invite agentStorage.Invite
	if err := db.Where("oid=?", uid).First(&invite).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error("queryInvite err:", err.Error())
	}
	return invite
}
func queryUserId(oid string) userStorage.User {
	db := common.GetMysql().Model(&userStorage.User{})
	var user userStorage.User
	if err := db.Where("oid=?", oid).First(&user).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error("err:", err.Error())
	}
	return user
}

func queryUserAccount(account string) userStorage.User {
	db := common.GetMysql().Model(&userStorage.User{})
	var user userStorage.User
	if err := db.Where("account=?", account).First(&user).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error("err:", err.Error())
	}
	return user
}

func syncUserInfo() {
	if err := common.GetMysql().AutoMigrate(&userStorage.UserInfo{}); err != nil {
		log.Error("err: %v", err.Error())
		return
	}
	c := common.GetMongoDB().C("userInfo")
	var userInfos []userStorage.UserInfo
	c.Find(nil).All(&userInfos)
	for _, info := range userInfos {
		updateUserInfo2mysql(info)
	}
}

func updateUserInfo2mysql(userInfo userStorage.UserInfo) {
	var u userStorage.UserInfo
	common.GetMysql().Where("oid=?", userInfo.Oid.Hex()).First(&u)
	userInfo.ID = u.ID
	log.Info("id:%v,sumAgentIncome:%v", u.ID, u.SumAgentBalance)
	if userInfo.FistChargeTime.IsZero() {
		userInfo.FistChargeTime = time.Unix(0, 0)
	}
	common.GetMysql().Save(&userInfo)
}

func fixUserReceiveBt() {
	c := common.GetMongoDB().C("userReceiveBt")
	var receiveBt []payStorage.UserReceiveBt
	if err := c.Find(nil).All(&receiveBt); err != nil {
		log.Error(err.Error())
		return
	}
	for _, bt := range receiveBt {
		common.GetMysql().Create(&bt)
	}
}
func fixAgentMemberData() {
	db := common.GetMysql().Model(&agentStorage.AgentMemberData{})
	var datas []agentStorage.AgentMemberData
	db.Find(&datas)
	for _, data := range datas {
		if data.CreateAt.IsZero() {

		}
	}
}

func fixUserInvite0602() {
	mgo := common.GetMongoDB()
	cInvite := mgo.C("userInvite")
	var invites []agentStorage.Invite
	if err := cInvite.Find(nil).All(&invites); err != nil {
		log.Error(err.Error())
		return
	}
	for _, invite := range invites {
		if invite.AgentOid.IsZero() {
			//if user := queryBelongAgent(invite.ParentOid);user != nil{
			//	invite.AgentOid = user.Oid
			//	updateInvite(invite)
			//}
			updateInvite(invite)
		}
	}
}
func updateInvite(invite agentStorage.Invite) {
	mgo := common.GetMongoDB()
	cInvite := mgo.C("userInvite")
	if err := cInvite.Update(bson.M{"_id": invite.Oid}, invite); err != nil {
		log.Error(err.Error())
	} else {
		mysql := common.GetMysql()
		var in agentStorage.Invite
		mysql.Model(agentStorage.Invite{}).Where("oid=?", invite.Oid.Hex()).First(&in)
		invite.ID = in.ID
		mysql.Save(invite)
	}
}

func queryBelongAgent(parentUid primitive.ObjectID) *userStorage.User {
	if user := userStorage.QueryUser(bson.M{"_id": parentUid}); user != nil { //查爸爸
		if user.Type == userStorage.TypeAgent {
			return user
		} else {
			var agentInvite *agentStorage.Invite
			if agentInvite = agentStorage.QueryInvite(parentUid); !agentInvite.Oid.IsZero() {
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
