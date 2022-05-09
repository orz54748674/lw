package loginImpl

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"vn/framework/mqant/log"
)

type Fibo struct {

}
const (
	FiboArea = 84
	FiboNo = "CL2105290001"
	FiboPwd = "KpPOQG0KOzutjJx"
	FiboSenderName = "FIBO"
)
//type FiboRes struct {
//	string string `xml:"xmlns"`
//	SMS []FiboSms `xml:"SMS"`
//}
//type FiboSms struct {
//	Code    string      `xml:"Code"`
//	Message    string      `xml:"Message"`
//	Time    string      `xml:"Time"`
//}

func (Fibo)Send(phone int64,code string) error {
	sendPhone := fmt.Sprintf("0%d",phone)//fmt.Sprintf("%d%d",FiboArea,phone)
	requestUrl := "https://ha-api.fibosms.com/SendMT/service.asmx/SendMaskedSMS?clientNo="+FiboNo+"&clientPass="+FiboPwd+
		"&senderName="+ FiboSenderName+"&phoneNumber="+sendPhone+"&smsMessage="+url.QueryEscape(code)+"&smsGUID=0&serviceType=0"
	resp, err := http.Get(requestUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	log.Info("%s",body)
	return nil
}