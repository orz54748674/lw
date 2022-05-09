package apiDg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"vn/common"
	"vn/common/utils"
)

type Rep struct {
	SerialKey string `json:"SerialKey"`
	Timestamp int64 `json:"Timestamp"`
	Code int `json:"Code"`
	Message string `json:"Message"`
	Data   interface{} `json:"data"`
}

type RepStatus struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	DateTime  string `json:"datatime"`
	TraceCode string `json:"traceCode"`
}

func UserRegister(msg map[string]interface{}) error {
	fmt.Println("msg.........", msg)
	path := "/user/signup/DGTE010525/"
	res, err := httpPost(path, msg)
	if err != nil {
		fmt.Println("err.er", err.Error())
		return err
	}
	fmt.Println("res.......", res)

	code := int(res["codeId"].(float64))
	if code != 0 {
		return fmt.Errorf("Dg Register fail, errcode:%d", code)
	}
	return nil
}

func Login(msg map[string]interface{}, deviceType int, lang string) (url string, err error) {
	res, err := httpPost("/user/login/DGTE010525/", msg)
	if err != nil {
		return "", err
	}

	code := int(res["codeId"].(float64))
	if code != 0 {
		return "", fmt.Errorf("Dg Login fail, errcode:%d", code)
	}

	tmpToken := res["token"].(string)
	tmpList := res["list"].([]interface{})

	return tmpList[deviceType].(string) + tmpToken + "&language=" + lang   , nil
}

func GetReport(param map[string]interface{}) (map[string]interface{}, error) {
	res, err := httpPost("/game/getReport/DGTE010525/", param)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func MarkReport(param map[string]interface{}) (map[string]interface{}, error) {
	res, err := httpPost("/game/markReport/DGTE010525/", param)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func httpPost(urlStr string, requestParam map[string]interface{}) (res map[string]interface{}, err error) {
	client := utils.GetHttpClient("dev", common.App.GetSettings().Settings["devProxy"].(string))
	//client := http.Client{
	//	Timeout: 5 * time.Second,
	//}

	requestUrl := ApiUrl + urlStr
	reader, _ := json.Marshal(requestParam)
	request, err := http.NewRequest("POST", requestUrl, strings.NewReader(string(reader)))
	if err != nil {
		return
	}
	//增加header选项
	request.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	res = make(map[string]interface{})
	err = json.Unmarshal(body, &res)
	if err != nil {
		//log.Error(err.Error())
		return
	}
	return
}
