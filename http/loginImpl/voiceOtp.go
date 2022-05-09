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

type VoiceOtp struct {

}
const (
	VoiceOtpArea = 84
	VoiceOtpLoginName = "AB12811"
	VoiceOtpSign = "5a0bf969152e718aedc8881b1a818b48"
	VoiceOtpTypeId = "271"
)
func (VoiceOtp)Send(phone int64,code string) error {
	sendPhone := fmt.Sprintf("%d%d",VoiceOtpArea,phone)
	requestUrl := "https://api.abenla.com/api/SendSms?loginName="+VoiceOtpLoginName+"&sign="+VoiceOtpSign+"&serviceTypeId="+VoiceOtpTypeId+
		"&phoneNumber="+sendPhone+"&message="+url.QueryEscape(code)+"&callBack=false"
	resp, err := http.Get(requestUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var res map[string]interface{}
	if err := json.Unmarshal(body, &res);err != nil{
		log.Error(err.Error())
		return err
	}else{
		if res["Code"].(float64) == 106{
			fmt.Println(string(body))
			return nil
		}
	}
	return errors.New(string(body))
}