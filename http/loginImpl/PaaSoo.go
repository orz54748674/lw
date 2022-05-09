package loginImpl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"vn/framework/mqant/log"
)

type PaaSoo struct {

}
const (
	area = 84
	key = "xyxycqyd"
	secret = "VsPQnXqV"
)

func (PaaSoo)Send(phone int64,code string) error {
	sendPhone := fmt.Sprintf("%d%d",area,phone)
	requestUrl := "https://api.paasoo.cn/json?key="+ key +
		"&secret="+ secret +"&from=SMS&to="+ sendPhone +"&text=" + url.QueryEscape(code)
	resp, err := http.Get(requestUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var res map[string]string
	if err := json.Unmarshal(body, &res);err != nil{
		log.Error(err.Error())
		return err
	}else{
		if res["status"] == "0"{
			fmt.Println(string(body))
			return nil
		}
	}
	return errors.New(string(body))
}