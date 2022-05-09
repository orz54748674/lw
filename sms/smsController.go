package sms

import (
	"encoding/json"
	"time"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
)

type smsController struct {
}

func (c *smsController) getSmsInfo(session gate.Session, data map[string]interface{}) (res map[string]interface{}, err error) {
	return
}

func (c *smsController) smsBind(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("smsBind time:%v  data:%v", time.Now().Unix(), data)
	params := &struct {
		Phone string `json:"phone"`
		Code  string `json:"text"`
	}{}

	resp = map[string]interface{}{}
	if err = c.getData(data, params); err != nil {
		return
	}
	log.Debug("smsBind params:%v", params)
	return
}

func (c *smsController) getData(reqData map[string]interface{}, params interface{}) (err error) {
	body := reqData["Body"].(string)
	err = json.Unmarshal([]byte(body), params)
	if err != nil {
		log.Error("SaBaRpc auth json.Unmarshal err:%v", err.Error())
		return
	}
	return
}
