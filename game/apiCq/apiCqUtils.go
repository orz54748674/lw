package apiCq

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vn/common/utils"
	"vn/framework/mqant/log"
)

type Param struct {
	Key   string
	Value interface{}
}

var (
	token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyaWQiOiI2MDgyOTg3ZTlmNGNlMjAwMDE5MmUxMTAiLCJhY2NvdW50IjoibGt3X3N3Iiwib3duZXIiOiI2MDgyOTg3ZTlmNGNlMjAwMDE5MmUxMTAiLCJwYXJlbnQiOiJzZWxmIiwiY3VycmVuY3kiOiJWTkQiLCJqdGkiOiI4NDU2MDc1NDAiLCJpYXQiOjE2MTkxNzE0NTQsImlzcyI6IkN5cHJlc3MiLCJzdWIiOiJTU1Rva2VuIn0.oVnwgROsZWcNMoudtJ2sa0I2YdV0YKI2NmBb8DeOJ_Q"
)

func (p *Param) String() string {
	return fmt.Sprintf("%s:%s", p.Key, utils.ConvertStr(p.Value))
}

func NewParam(key string, value string) Param {
	return Param{Key: key, Value: value}
}

func httpGet(url string) (res *Rep, err error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	requestUrl := LoginHost + url
	log.Info("requestUrl:%s", requestUrl)
	request, err := http.NewRequest("GET", requestUrl, nil)
	//增加header选项
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Authorization", token)
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

func httpGet2(url string, requestBody []Param) (res *Rep, err error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	requestUrl := LoginHost + url
	reader := makeJson2(requestBody)
	log.Info("requestUrl:%s body: %s", requestUrl, reader)
	request, err := http.NewRequest("GET", requestUrl, strings.NewReader(reader))
	//增加header选项
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Authorization", token)
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
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	requestUrl := LoginHost + urlStr
	reader := makeJson2(requestBody)
	log.Info("requestUrl:%s body: %s", requestUrl, reader)
	request, err := http.NewRequest("POST", requestUrl, strings.NewReader(reader))

	//增加header选项
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Authorization", token)

	resp, err := client.Do(request)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	log.Info("res: %s", string(body))
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
