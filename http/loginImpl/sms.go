package loginImpl

import (
	"vn/framework/mqant/log"
	"vn/storage"
	"vn/storage/lobbyStorage"
)

var smsTemplate = "[lucky win]Your verification code is %s"
var smsTemplateNoSign = "Your verification code is %s"
var smsFiboTemplate = "Fibo-PTTT. Ma su dung dich vu (%s) Moi thac mac vui long lien he CSKH Xin cam on!"
var NoSignPhoneHead = []string{
	"90", "93", "89", "70", "76", "77", "78", "79",
}

func SendSmsCode(sms *lobbyStorage.Sms) {
	log.Info("SendSmsCode: %v", *sms)
	lobbyStorage.InsertSms(sms)
	isClose := storage.QueryConf(storage.KIsCloseSmsGate)
	if isClose == "1" {
		log.Info("sms gate was closed.")
		return
	}
	//paaSoo := &PaaSoo{}
	//var smsContent string
	//phoneHead := utils.Substr(strconv.FormatInt(sms.Phone,10),0,2)
	//if utils.IsContainStr(NoSignPhoneHead,phoneHead){
	//	smsContent = fmt.Sprintf(smsTemplateNoSign,sms.Code)
	//}else{
	//	smsContent = fmt.Sprintf(smsTemplate,sms.Code)
	//}
	//err := paaSoo.Send(sms.Phone,smsContent)
	//if err != nil{
	//	log.Error(err.Error())
	//}

	//

	voiceOtp := &VoiceOtp{}
	err := voiceOtp.Send(sms.Phone, sms.Code)
	if err != nil {
		log.Error(err.Error())
	}
}

func CheckoutCode(area int64, phone int64, event string, code string) bool {
	superCode := storage.QueryConf(storage.KSmsSuperCode)
	if superCode != "" && superCode == code {
		return true
	}
	sms := lobbyStorage.QuerySms(area, phone, event)
	if sms != nil && sms.Code == code {
		return true
	}
	return false
}
