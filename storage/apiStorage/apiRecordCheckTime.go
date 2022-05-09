package apiStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
)

var (
	cApiRecordCheckTime = "ApiRecordCheckTime"
)

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *ApiRecordCheckTime) SetName() string {
	return cApiRecordCheckTime
}

func (m *ApiRecordCheckTime) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}
func (m *ApiRecordCheckTime) GetLastTime(apiType int8) (t *ApiRecordCheckTime, err error) {
	t = &ApiRecordCheckTime{}
	err = m.C().Find(bson.M{"Type": apiType}).One(t)
	return
}

func (m *ApiRecordCheckTime) UpdateTime(apiType int8) (err error) {
	_, err = m.C().Upsert(bson.M{"Type": apiType}, bson.M{"$set": bson.M{"Time": m.Time, "UpdateAt": time.Now()}})
	return
}
