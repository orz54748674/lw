package slotCsStorage

import (
	"github.com/fatih/structs"
	"strconv"
	"vn/common"
	"vn/common/errCode"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
)

var(
	cRoomData = "SlotCsRoomData"
	cRoomConf = "SlotCsRoomConf"
)
func GetRoomData() *RoomData {
	c := common.GetMongoDB().C(cRoomData)
	var roomData RoomData
	if err := c.Find(nil).One(&roomData); err != nil{
		log.Info("not found room data ",err)
		return nil
	}
	return &roomData
}
func InsertRoomData(roomData *RoomData)*common.Err{
	c := common.GetMongoDB().C(cRoomData)
	if error := c.Find(nil). One(&roomData); error == nil{
		log.Info("found room data,no need insert")
		return nil //errCode.ServerError.SetErr(error.Error())
	}
	if error := c.Insert(roomData); error != nil{
		log.Info("Insert room error: %s", error)
		return errCode.ServerError.SetErr(error.Error())
	}
	return nil
}
func UpsertRoomData(roomData *RoomData) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	selector := bson.M{"_id":roomData.Oid}
	update := structs.Map(roomData)
	_,err :=c.Upsert(selector,update)
	if err != nil{
		log.Error("Upsert room data error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}

func IncJackpot(pos int,jackpot int64){
	c := common.GetMongoDB().C(cRoomData)
	update := bson.M{"$inc":bson.M{
		"Jackpot."+strconv.Itoa(pos):jackpot,
	}}
	if _,err := c.Upsert(nil, update);err !=nil{
		log.Error(err.Error())
	}
}
func GetJackpot() (jackpot []int64){
	c := common.GetMongoDB().C(cRoomData)
	var roomData RoomData
	if err := c.Find(nil).One(&roomData); err != nil{
		log.Info("GetTableInfo  ",err)
		return []int64{0}
	}
	return roomData.Jackpot
}
func InsertRoomConf(conf *Conf)*common.Err{
	c := common.GetMongoDB().C(cRoomConf)
	if error := c.Find(nil). One(&conf); error == nil{
		log.Info("found room conf,no need insert")
		return nil //errCode.ServerError.SetErr(error.Error())
	}
	if error := c.Insert(conf); error != nil{
		log.Info("Insert room conf error: %s", error)
		return errCode.ServerError.SetErr(error.Error())
	}
	return nil
}

func GetRoomConf() *Conf {
	c := common.GetMongoDB().C(cRoomConf)
	var conf Conf
	if err := c.Find(bson.M{}).One(&conf); err != nil{
		log.Info("not found room conf ",err)
		return nil
	}
	return &conf
}
func RemoveTableInfo(tableID string) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	selector := bson.M{"nil":nil}
	update := bson.M{"$unset":bson.M{"TablesInfo."+ tableID:""}}
	err :=c.Update(selector,update)
	if err != nil{
		log.Error("RemoveTableInfo error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}
func GetTableInfo(tableID string) TableInfo {
	c := common.GetMongoDB().C(cRoomData)
	var roomData RoomData
	if err := c.Find(nil).One(&roomData); err != nil{
		log.Info("not found room data ",err)
		return TableInfo{}
	}
	return roomData.TablesInfo[tableID]
}
func UpsertTableInfo(tableInfo TableInfo,tableID string) *common.Err {
	c := common.GetMongoDB().C(cRoomData)
	selector := bson.M{"nil":nil}
	update := bson.M{"$set":bson.M{"TablesInfo":bson.M{tableID:tableInfo}}}

	_,err :=c.Upsert(selector,update)
	if err != nil{
		log.Error("Upsert room data error: %s", err)
		return nil //errCode.ServerError.SetErr(err.Error())
	}
	return nil
}