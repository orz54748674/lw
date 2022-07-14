package utils

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
)

func ConvertUidToOid(uid []string) []primitive.ObjectID {
	userIds := make([]primitive.ObjectID, 0)
	for _, id := range uid {
		userIds = append(userIds, ConvertOID(id))
	}
	return userIds
}

func RmStrStartZero(str string) string {
	newStr := str
	for i := 0; i < len(str); i++ {
		ch := str[i]
		if ch == '0' {
			newStr = strings.TrimPrefix(newStr, "0")
		} else {
			return newStr
		}
	}
	return newStr
}

func ParsePhone(inputPhone interface{}) (int64, int) {
	var phone int64
	switch inputPhone.(type) {
	case string:
		phone, _ = ConvertInt(RmStrStartZero(inputPhone.(string)))
	default:
		phone, _ = ConvertInt(inputPhone)
	}
	length := len(strconv.Itoa(int(phone)))
	return phone, length
}

func ConvertOID(id string) primitive.ObjectID {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Error("ConvertOID err:%v, id: %v", err.Error(), id)
	}
	return oid
}

func Now() time.Time {
	return time.Unix(0, time.Now().UnixNano()/1e6*1e6)
}

func Goid() int {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("panic recover:panic info:%v", err)
		}
	}()

	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}
