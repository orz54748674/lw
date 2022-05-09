package main

import (
	"time"
	"vn/common"
	"vn/storage/activityStorage"
	"vn/storage/userStorage"
)
type sumGiftCode struct {
	SumGet int64
}
func fixGiftCode() {
	users := userStorage.QueryUsers()
	for _,v := range users{
		mysql := common.GetMysql()
		var sum sumGiftCode
		mysql.Model(activityStorage.ActivityRecord{}).Select("sum(`get`) sum_get").Where("uid=? and type=? and update_at<?",v.Oid.Hex(),"GiftCode","2021-07-17 21:09:00").Find(&sum)
		if sum.SumGet > 0{
			userStorage.IncUserGiftCode(v.Oid,sum.SumGet)
		}
	}
	time.Sleep(time.Second * 300)
}
