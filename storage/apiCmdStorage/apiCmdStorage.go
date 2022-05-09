package apiCmdStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var (
	cApiUser         = "apiUser"
	cApiCmdUserToken = "apiCmdUserToken"
	cApiCmdUpdateMsg = "apiCmdUpdateMsg"
	cApiCmdConf      = "apiCmdConf"
	cApiCmdReference = "apiCmdReferenceRecord"
	cApiCmdCashOutRecord = "apiCmdCashOutRecord"
	cApiCmdParlayRecord = "apiCmdParlayRecord"
	cApiCmdBetRecord = "apiCmdBetRecord"
	cApiCmdTeamLeagueInfo = "apiCmdTeamLeagueInfo"
)

/**
 *  @title	InitLotteryRecord
 *	@description	初始化记录集合并建(number和lotteryCode)复合唯一索引
 */

func InitCmdStorage() {
	c := common.GetMongoDB().C(cApiCmdReference)
	key1 := bsonx.Doc{{Key: "CreateAt",Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key1,options.Index().
		SetExpireAfterSeconds(30*24*3600));err != nil{
		log.Error("create cApiCmdReferenceRecord Index: %s",err)
	}

	c = common.GetMongoDB().C(cApiCmdUserToken)
	key2 := bsonx.Doc{{Key: "updateAt",Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key2,options.Index().
		SetExpireAfterSeconds(60));err != nil{
		log.Error("create cApiCmdUserToken Index: %s",err)
	}
}

func UpsertCmdUserToken(account, token string) error {
	c := common.GetMongoDB().C(cApiCmdUserToken)
	query := bson.M{"account": account}

	msg := UserToken{
		Account:  account,
		Token:    token,
		UpdateAt: utils.GetMillisecond(),
	}

	if _, err := c.Upsert(query, &msg); err != nil {
		log.Error(err.Error())
		return err
	}

	return nil
}

func CheckToken(token string) (bool, string) {
	c := common.GetMongoDB().C(cApiCmdUserToken)
	query := bson.M{"token": token}

	var tmpToken UserToken
	if err := c.Find(query).One(&tmpToken); err != nil {
		return false, ""
	}
	return true, tmpToken.Account
}

func GetUidByAccount(account string) string {
	var tmp ApiUser
	query := bson.M{"Account": account, "Type": 3}
	c := common.GetMongoDB().C(cApiUser)
	if err := c.Find(query).One(&tmp); err != nil {
		return ""
	}
	return tmp.Uid
}

func SaveUpdateBalanceMsg(data string) error {
	c := common.GetMongoDB().C(cApiCmdUpdateMsg)

	var msg RecordUpdateBalance
	msg.Data = data
	msg.UpdateTime = time.Now().Unix()
	if err := c.Insert(&msg); err != nil {
		return err
	}

	return nil
}

func GetApiCmdConf() (ApiCmdConf, error) {
	c := common.GetMongoDB().C(cApiCmdConf)

	var conf ApiCmdConf
	if err := c.Find(nil).One(&conf); err != nil {
		conf.VersionID = 0
		conf.PartnerKey = "6267807782793079"
		if err = c.Insert(&conf); err != nil {
			log.Error("insert api cmd conf err:", err.Error())
			return conf, err
		}
	}
	return conf, nil
}

func UpdateApiCmdConf(versionID int) error {
	c := common.GetMongoDB().C(cApiCmdConf)
	update := bson.M{
		"$set": bson.M{"versionID": versionID,
		},
	}
	if _, err := c.Upsert(nil, update); err != nil {
		log.Error("UpdateApiCmdConf err:", err.Error())
		return err
	}
	return nil
}

func UpsertApiCmdBetRecord(data map[string]interface{}) {
	c := common.GetMongoDB().C(cApiCmdBetRecord)
	query := bson.M{"ReferenceNo":data["ReferenceNo"].(string)}
	update := bson.M{"$set":data}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error("UpsertApiCmdBetRecord err:", err.Error())
	}
}

func GetSelectBetRecordId(referenceNo string) int {
	c := common.GetMongoDB().C(cApiCmdBetRecord)
	query := bson.M{"ReferenceNo":referenceNo}
	data := make(map[string]interface{})
	if err := c.Find(query).One(&data); err != nil {
		return 0
	}
	return int(data["Id"].(float64))
}

func InsertReferenceRecord(uid, referenceNo string, betAmount int64) {
	var referenceMsg ReferenceMsg
	referenceMsg.ReferenceNo = referenceNo
	referenceMsg.BetAmount = betAmount
	referenceMsg.Uid = uid
	referenceMsg.CreateAt = time.Now()
	c := common.GetMongoDB().C(cApiCmdReference)
	if err := c.Insert(&referenceMsg); err != nil {
		log.Error("InsertReferenceRecord err:", err.Error())
	}
}

func GetReferenceBetAmount(uid, referenceNo string) int64{
	c := common.GetMongoDB().C(cApiCmdReference)
	query := bson.M{"Uid":uid, "ReferenceNo":referenceNo}
	var tmp ReferenceMsg
	if err := c.Find(query).One(&tmp); err != nil {
		return 0
	}
	return tmp.BetAmount
}

func GetCmdTeamLeagueInfo(infoType, infoID int) (CmdTeamLeagueInfo, error) {
	c := common.GetMongoDB().C(cApiCmdTeamLeagueInfo)
	query := bson.M{"InfoType":infoType, "InfoID":infoID}
	var tmp CmdTeamLeagueInfo
	if err := c.Find(query).One(&tmp); err != nil {
		return tmp, err
	}
	return tmp, nil
}


func InsertCmdTeamLeagueInfo(infoType, infoID int, infoName string) error {
	c := common.GetMongoDB().C(cApiCmdTeamLeagueInfo)
	var tmp CmdTeamLeagueInfo
	tmp.InfoID = infoID
	tmp.InfoType = infoType
	tmp.InfoName = infoName
	if err := c.Insert(&tmp); err != nil {
		log.Error("InsertReferenceRecord err:", err.Error())
		return err
	}
	return nil
}