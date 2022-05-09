package utils

import (
	"time"
	"vn/framework/mqant/log"
)

func GetDayStartTime(time time.Time) (time.Time,error) {
	t,err := StrFormatTime("yyyy-MM-dd", GetCnDate(time))
	if err != nil{
		log.Error(err.Error())
	}
	return t,err
}
func GetDayEndTimeByStart(t time.Time) (time.Time) {
	ti := time.Unix(t.Unix()+86400, 0)
	return ti
}

func IsToday(t time.Time) bool {
	now := GetCnDate(time.Now())
	tmp := GetCnDate(t)
	return now == tmp
}