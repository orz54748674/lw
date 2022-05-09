package apiXg

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/log"
)

type Param struct {
	Key   string
	Value interface{}
}

func (p *Param) String() string {
	return fmt.Sprintf("%s:%s", p.Key, utils.ConvertStr(p.Value))
}
func NewParam(key string, value string) Param {
	return Param{Key: key, Value: value}
}

func getSign(queryStr string) string {
	var cstSh, err = time.LoadLocation("America/New_York")
	t := time.Now()
	if err != nil {
		log.Error("xg getSign time.LoadLocation err:%v", err.Error())
	} else {
		t.In(cstSh)
	}

	date := t.Format("06012")
	keyG := doMd5(fmt.Sprintf("%s%s%s", date, agentId, agentKey))
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	key := utils.RandomString(6,r) + doMd5(queryStr+keyG) + utils.RandomStringV1(6)
	return key
}
func doMd5(str string) string {
	data := []byte(str)
	has := md5.Sum(data)
	md5str1 := fmt.Sprintf("%x", has)
	return md5str1
}
func HttpBuildQuery(params []Param) (param_str string) {
	params_arr := make([]string, 0, len(params))
	for _, v := range params {
		params_arr = append(params_arr, fmt.Sprintf("%s=%s", v.Key, utils.ConvertStr(v.Value)))
	}
	//fmt.Println(params_arr)
	param_str = strings.Join(params_arr, "&")
	return param_str
}
func httpGet(urlStr string) (res *Rep, err error) {
	client := utils.GetHttpClient(defaultEnv, common.App.GetSettings().Settings["devProxy"].(string))
	requestUrl := host + urlStr
	log.Info("requestUrl:%s", requestUrl)
	request, err := http.NewRequest("GET", requestUrl, nil)
	//增加header选项
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	//log.Info("res: %s", string(body))
	err = json.Unmarshal(body, &res)
	if err != nil {
		//log.Error(err.Error())
		return
	}
	return
}
func httpPost(urlStr string, requestBody []Param) (res *Rep, err error) {
	client := utils.GetHttpClient(defaultEnv, common.App.GetSettings().Settings["devProxy"].(string))
	requestUrl := host + urlStr
	//reader,err := json.Marshal(params)
	reader := makeJson2(requestBody)
	log.Info("requestUrl:%s \r\nbody: %s", requestUrl, string(reader))
	//tmp := `{"AgentId":"8630e392-a347-11eb-8856-06b6fe12fbc4","Account":"test1","Key":"an9wgve67736ddfbfce0ecafbf5f67d3bb2697qo087k"}`
	request, err := http.NewRequest("POST", requestUrl, strings.NewReader(string(reader)))
	//增加header选项
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	// log.Info("res: %s", string(body))
	err = json.Unmarshal(body, &res)
	if err != nil {
		//log.Error(err.Error())
		return
	}
	return
}
func makeJson2(params []Param) string {
	jsonInfo := []byte(`{`)
	size := len(params)
	for i, p := range params {
		value := convert(p.Value)
		if i < size-1 {
			value = value + ","
		}
		jsonInfo = append(jsonInfo, fmt.Sprintf("\"%s\":%s", p.Key, value)...)
	}
	jsonInfo = append(jsonInfo, `}`...)
	return string(jsonInfo)
}

func convert(any interface{}) string {
	res := ""
	switch any.(type) {
	case float64:
		res = fmt.Sprintf("%.2f", any.(float64))
	case int64:
		res = strconv.FormatInt(any.(int64), 10)
	case int:
		res = strconv.Itoa(any.(int))
	case string:
		res = fmt.Sprintf("\"%s\"", any.(string))
	}
	return res
}
