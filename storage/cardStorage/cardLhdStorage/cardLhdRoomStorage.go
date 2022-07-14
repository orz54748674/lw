package cardLhdStorage

import (
	"github.com/fatih/structs"
	"vn/common"
	"vn/common/errCode"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
)

var (
	cRoomData   = "cardLhdRoomData"
	cRoomRecord = "cardLhdRoomRecord"
	cRoomConf   = "cardLhdRoomConf"
	cRobotConf  = "cardLhdRobotConf"
)

//func InitRoomData(){
//	c := common.GetMongoDB().C(cRoomData)
//	index := mgo.Index{
//		Key: []string{"Oid"},
//		Unique: true,
//		DropDups: true,
//		Background: true, // See notes.
//		Sparse: true,
//	}
//	if err := c.EnsureIndex(index);err != nil {
//		log.Error("GlobalIdIndex: %s",err)
//	}
//	log.Info("init roomDataGlobal of mongo db")
//}
func InsertRoomData(roomData *RoomData) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	//id := "000000"
	if error := c.Find(nil).One(&roomData); error == nil {
		log.Info("found room data,no need insert")
		return nil //errCode.ServerError.SetErr(error.Error())
	}
	//	InitRoomData()
	if error := c.Insert(roomData); error != nil {
		log.Info("Insert room error: %s", error)
		return errCode.ServerError.SetErr(error.Error())
	}
	return nil
}

func GetRoomData() *RoomData {
	c := common.GetMongoDB().C(cRoomData)
	var roomData RoomData
	if err := c.Find(nil).One(&roomData); err != nil {
		log.Info("not found room data ", err)
		return nil
	}
	return &roomData
}
func UpsertRoomData(roomData *RoomData) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	selector := bson.M{"_id": roomData.Oid}
	update := structs.Map(roomData)
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("Upsert room data error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}
func GetTablesInfo() map[string]TableInfo {
	c := common.GetMongoDB().C(cRoomData)
	var roomData RoomData
	if err := c.Find(nil).One(&roomData); err != nil {
		log.Info("not found room data ", err)
		return nil
	}
	return roomData.TablesInfo
}
func UpsertTablesInfo(tablesInfo map[string]TableInfo) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	selector := bson.M{"nil": nil}
	update := bson.M{"$set": bson.M{"TablesInfo": tablesInfo}}

	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("Upsert room data error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}
func GetTableInfo(tableID string) TableInfo {
	c := common.GetMongoDB().C(cRoomData)
	var roomData RoomData
	if err := c.Find(nil).One(&roomData); err != nil {
		log.Info("not found room data ", err)
		return TableInfo{}
	}
	return roomData.TablesInfo[tableID]
}
func UpsertTableInfo(tableInfo TableInfo, tableID string) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	selector := bson.M{"nil": nil}
	update := bson.M{"$set": bson.M{"TablesInfo": bson.M{tableID: tableInfo}}}

	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("Upsert room data error: %s", err)
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
//func InitRoomRecord(){
//	c := common.GetMongoDB().C(cRoomRecord)
//	index := mgo.Index{
//		Key: []string{"Oid"},
//		Unique: true,
//		DropDups: true,
//		Background: true, // See notes.
//		Sparse: true,
//	}
//	if err := c.EnsureIndex(index);err != nil {
//		log.Error("GlobalIdIndex: %s",err)
//	}
//	log.Info("init roomRecordGlobal of mongo db")
//}
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
	update := bson.M{"$set": bson.M{"ResultsRecord": bson.M{tableId: resultsRecord}}}

	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("Upsert result record error: %s", err)
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
	if err := c.Find(bson.M{}).One(&conf); err != nil {
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
