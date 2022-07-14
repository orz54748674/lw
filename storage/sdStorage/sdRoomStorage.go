package sdStorage

import (
	"vn/common"
	"vn/common/errCode"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
)

var (
	cRoomData   = "SdRoomData"
	cRoomRecord = "SdRoomRecord"
	cRoomConf   = "SdRoomConf"
	cRobotConf  = "SdRobotConf"
)

func GetRoomData() *RoomData {
	c := common.GetMongoDB().C(cRoomData)
	var roomData RoomData
	if err := c.Find(nil).One(&roomData); err != nil {
		log.Info("not found room data ", err)
		return nil
	}
	return &roomData
}
func GetTablesInfo() map[string]TableInfo {
	c := common.GetMongoDB().C(cRoomData)
	var roomData RoomData
	if err := c.Find(nil).One(&roomData); err != nil {
		log.Info("GetTablesInfo ", err)
		return nil
	}
	return roomData.TablesInfo
}
func UpsertTablesInfo(tablesInfo map[string]TableInfo) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	update := bson.M{"$set": bson.M{"TablesInfo": tablesInfo}}

	_, err := c.Upsert(nil, update)
	if err != nil {
		log.Error("UpsertTablesInfo error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}
func GetTableInfo(tableID string) TableInfo {
	c := common.GetMongoDB().C(cRoomData)
	//var tableInfo TableInfo
	var roomData RoomData
	if err := c.Find(nil).One(&roomData); err != nil {
		log.Info("GetTableInfo  ", err)
		return TableInfo{}
	}
	return roomData.TablesInfo[tableID]
}
func UpsertTableInfo(tableInfo TableInfo, tableID string) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	selector := bson.M{"nil": nil}
	update := bson.M{"$set": bson.M{"TablesInfo." + tableID: tableInfo}}
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("UpsertTableInfo error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}
func RemoveTableInfo(tableID string) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	selector := bson.M{"nil": nil}
	update := bson.M{"$unset": bson.M{"TablesInfo." + tableID: ""}}
	err := c.Update(selector, update)
	if err != nil {
		log.Error("RemoveTableInfo error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}

//---------------------------RoomRecord------
func InsertRoomRecord(roomRecord *RoomRecord) *common.Err {
	c := common.GetMongoDB().C(cRoomRecord)
	//id := "000000"
	if error := c.Find(nil).One(&roomRecord); error == nil {
		log.Info("found room record,no need insert")
		return nil //errCode.ServerError.SetErr(error.Error())
	}
	//InitRoomRecord()
	if error := c.Insert(roomRecord); error != nil {
		log.Info("Insert room record error: %s", error)
		return errCode.ServerError.SetErr(error.Error())
	}
	return nil
}

func GetRoomRecord() *RoomRecord {
	c := common.GetMongoDB().C(cRoomRecord)
	var roomRecord RoomRecord
	if err := c.Find(nil).One(&roomRecord); err != nil {
		log.Info("not found room record ", err)
		return nil
	}
	return &roomRecord
}

func GetResultsRecord(tableId string) ResultsRecord {
	c := common.GetMongoDB().C(cRoomRecord)
	var roomRecord RoomRecord
	if err := c.Find(nil).One(&roomRecord); err != nil {
		log.Info("not found result record ", err)
		return ResultsRecord{}
	}
	return roomRecord.ResultsRecord[tableId]
}
func UpsertResultsRecord(resultsRecord ResultsRecord, tableId string) *common.Err {
	c := common.GetMongoDB().C(cRoomRecord)
	selector := bson.M{"nil": nil}
	update := bson.M{"$set": bson.M{"ResultsRecord." + tableId: resultsRecord}}

	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("Upsert result record error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}
func RemoveResultsRecord(tableID string) *common.Err {
	c := common.GetMongoDB().C(cRoomRecord)
	selector := bson.M{"nil": nil}
	update := bson.M{"$unset": bson.M{"ResultsRecord." + tableID: ""}}
	err := c.Update(selector, update)
	if err != nil {
		log.Error("RemoveResultsRecord error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}

//----------------------Conf
func InsertRoomConf(conf *Conf) *common.Err {
	c := common.GetMongoDB().C(cRoomConf)
	//id := "000000"
	if error := c.Find(nil).One(&conf); error == nil {
		log.Info("found room conf,no need insert")
		return nil //errCode.ServerError.SetErr(error.Error())
	}
	//	InitRoomData()
	if error := c.Insert(conf); error != nil {
		log.Info("Insert room conf error: %s", error)
		return errCode.ServerError.SetErr(error.Error())
	}
	return nil
}

func GetRoomConf() *Conf {
	c := common.GetMongoDB().C(cRoomConf)
	var conf Conf
	if err := c.Find(nil).One(&conf); err != nil {
		log.Info("not found room conf ", err)
		return nil
	}
	return &conf
}
func UpsertRoomConf(serverID string, flag bool) *common.Err {
	c := common.GetMongoDB().C(cRoomConf)
	selector := bson.M{"nil": nil}
	update := bson.M{"$set": bson.M{"ServerRebot." + serverID: flag}}
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("UpsertRoomConf error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}

//---------------------------Robot----------------
func UpsertRobotConf(robotConf RobotConf) *common.Err {
	c := common.GetMongoDB().C(cRobotConf)
	query := bson.M{"TableID": robotConf.TableID, "StartHour": robotConf.StartHour}
	update := bson.M{"$set": robotConf}
	_, err := c.Upsert(query, update)
	if err != nil {
		log.Error("Upsert Robot error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}
func GetTableRobotConf(tableID string) []RobotConf {
	c := common.GetMongoDB().C(cRobotConf)
	var robotConf []RobotConf
	query := bson.M{"TableID": tableID}
	if err := c.Find(query).All(&robotConf); err != nil {
		log.Info("not found robot conf ", err)
		return nil
	}
	return robotConf
}
func GetTableRobotConfByHour(tableID string, startHour int) RobotConf {
	c := common.GetMongoDB().C(cRobotConf)
	var robotConf RobotConf
	query := bson.M{"TableID": tableID, "StartHour": startHour}
	if err := c.Find(query).One(&robotConf); err != nil {
		log.Info("not found robot conf ", err)
		return RobotConf{}
	}
	return robotConf
}
