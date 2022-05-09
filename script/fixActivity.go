package main

import (
	"time"
	"vn/common"
	"vn/storage/activityStorage"
	"vn/storage/userStorage"
)
type sumTmp struct {
	SumGet int64
}
func fixActivity() {
	users := userStorage.QueryUsers()
	for _,v := range users{
		mysql := common.GetMysql()
		var sum sumTmp
		mysql.Model(activityStorage.ActivityRecord{}).Select("sum(`get`) sum_get").Where("uid=? and update_at<?",v.Oid.Hex(),"2021-07-10 14:15:00").Find(&sum)
		if sum.SumGet > 0{
			userStorage.IncUserActivityTotal(v.Oid,sum.SumGet)
		}
	}
	time.Sleep(time.Second * 300)
}
