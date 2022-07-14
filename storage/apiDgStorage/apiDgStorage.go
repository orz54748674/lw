package apiDgStorage

import (
	"fmt"
	"strings"
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
)

var (
	cApiUser              = "apiUser"
	cApiCmdUserToken      = "apiCmdUserToken"
	cApiCmdUpdateMsg      = "apiCmdUpdateMsg"
	cApiCmdConf           = "apiCmdConf"
	cApiCmdReference      = "apiCmdReferenceRecord"
	cApiCmdCashOutRecord  = "apiCmdCashOutRecord"
	cApiCmdParlayRecord   = "apiCmdParlayRecord"
	cApiCmdBetRecord      = "apiCmdBetRecord"
	cApiCmdTeamLeagueInfo = "apiCmdTeamLeagueInfo"
	cApiDgTransferRecord  = "apiDgTransferRecord"
	apiType               = 3
	cDgApiUser            = "dgApiUser"
)

/**
 *  @title	InitLotteryRecord
 *	@description	初始化记录集合并建(number和lotteryCode)复合唯一索引
 */

func InitDgStorage() {
	c := common.GetMongoDB().C(cApiCmdReference)
	key1 := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key1, options.Index().
		SetExpireAfterSeconds(30*24*3600)); err != nil {
		log.Error("create cApiCmdReferenceRecord Index: %s", err)
	}

	c = common.GetMongoDB().C(cApiCmdUserToken)
	key2 := bsonx.Doc{{Key: "updateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key2, options.Index().
		SetExpireAfterSeconds(60)); err != nil {
		log.Error("create cApiCmdUserToken Index: %s", err)
	}
}

func GetDgUserInfoByUid(uid string) (DgUserInfo, error) {
	var userInfo DgUserInfo
	query := bson.M{"Uid": uid}
	c := common.GetMongoDB().C(cDgApiUser)
	if err := c.Find(query).One(&userInfo); err != nil {
		return userInfo, err
	}
	return userInfo, nil
}

func UpsertDgUserInfo(dgUserInfo DgUserInfo) error {
	c := common.GetMongoDB().C(cDgApiUser)
	query := bson.M{"Uid": dgUserInfo.Uid}
	if _, err := c.Upsert(query, dgUserInfo); err != nil {
		log.Error("InsertReferenceRecord err:", err.Error())
		return err
	}
	return nil
}

func GetTransferRecordBySerialNo(serialNo string) (DgTransferInfo, error) {
	var transferRecord DgTransferInfo
	query := bson.M{"SerialNo": serialNo}
	c := common.GetMongoDB().C(cApiDgTransferRecord)
	if err := c.Find(query).One(&transferRecord); err != nil {
		return transferRecord, err
	}
	return transferRecord, nil
}

func GetTransferRecordByTicketId(ticketId string) ([]DgTransferInfo, error) {
	var transferRecords []DgTransferInfo
	query := bson.M{"TicketId": ticketId}
	c := common.GetMongoDB().C(cApiDgTransferRecord)
	if err := c.Find(query).All(&transferRecords); err != nil {
		return transferRecords, err
	}
	return transferRecords, nil
}

func GetTransferRecordByToken(token string) ([]DgTransferInfo, error) {
	var transferRecords []DgTransferInfo
	query := bson.M{"Token": token}
	c := common.GetMongoDB().C(cApiDgTransferRecord)
	if err := c.Find(query).All(&transferRecords); err != nil {
		return transferRecords, err
	}
	return transferRecords, nil
}

func RemoveTransferRecordBySerialNo(serialNo string) error {
	query := bson.M{"SerialNo": serialNo}
	c := common.GetMongoDB().C(cApiDgTransferRecord)
	return c.Remove(query)
}

func InsertTransferRecord(serialNo, ticketId, username, token string, balance, amount int64) error {
	var transferRecord DgTransferInfo
	transferRecord.Username = username
	transferRecord.SerialNo = serialNo
	transferRecord.TicketId = ticketId
	transferRecord.Token = token
	transferRecord.Amount = amount
	transferRecord.Balance = balance
	transferRecord.CreateAt = time.Now()
	c := common.GetMongoDB().C(cApiDgTransferRecord)
	if err := c.Insert(&transferRecord); err != nil {
		log.Error("InsertReferenceRecord err:", err.Error())
		return err
	}
	return nil
}

func GetDgUserInfoByUsername(username string) (DgUserInfo, error) {
	var userInfo DgUserInfo
	fmt.Println("bogdgds.........", username)
	query := bson.M{"Username": strings.ToLower(username)}
	c := common.GetMongoDB().C(cDgApiUser)
	if err := c.Find(query).One(&userInfo); err != nil {
		fmt.Println("GetDgUserInfoByUsername err:", err.Error())
		return userInfo, err
	}
	return userInfo, nil
}
