package apiCmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"vn/common/utils"
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

func HttpBuildQuery(params []Param) (param_str string) {
	params_arr := make([]string, 0, len(params))
	for _, v := range params {
		params_arr = append(params_arr, fmt.Sprintf("%s=%s", v.Key, utils.ConvertStr(v.Value)))
	}
	param_str = strings.Join(params_arr, "&")
	return param_str
}

func httpGet(url string) (res *Rep, err error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	requestUrl := ApiUrl + url
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
	err = json.Unmarshal(body, &res)
	if err != nil {
		return
	}
	return
}
