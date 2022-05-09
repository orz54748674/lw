package apiDg

import (
	"fmt"
	"math/rand"
	"strings"
	"vn/common/utils"
)

type Param struct {
	Key   string
	Value interface{}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func (p *Param) String() string {
	return fmt.Sprintf("%s:%s", p.Key, utils.ConvertStr(p.Value))
}

func NewParam(key string, value string) Param {
	return Param{Key: key, Value: value}
}

func HttpBuildQuery(params []Param) (param_str string) {
	params_arr := make([]string, 0, len(params))
	for _, v := range params {
		params_arr = append(params_arr, fmt.Sprintf("%s=%s", v.Key, utils.ConvertStr(v.Value)))
	}
	param_str = strings.Join(params_arr, "&")
	return param_str
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func GetToken() (string, string) {
	var randStr string
	randStr = RandStringBytes(6)
	return utils.MD5(fmt.Sprintf("%s%s%s", AgentName, ApiKey, randStr)), randStr
}

func GetPassword() string {
	return utils.MD5(fmt.Sprintf("%s", RandStringBytes(6)))
}
