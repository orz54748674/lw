package payWay

//import (
//	"crypto/md5"
//	"crypto/tls"
//	"encoding/json"
//	"fmt"
//	"io/ioutil"
//	"math/rand"
//	"net/http"
//	"sort"
//	"strconv"
//	"strings"
//	"time"
//	"vn/common"
//	"vn/common/errCode"
//	"vn/common/utils"
//	"vn/framework/mqant/log"
//	"vn/storage"
//	"vn/storage/payStorage"
//)
//
//type NapTuDong struct {
//}
//
//var partnerId = "9180139061"
//var partnerKey = "fb6016f21f4398ddb72cc71b86fb779a"
//var telcoArray = []string{"VIETTEL", "VINAPHONE", "MOBIFONE"}
//
//func (s *NapTuDong) initKey() {
//	conf := strings.Split(storage.QueryConf(storage.KPhoneChargeNapTuDong).(string), ",")
//	partnerId = conf[0]
//	partnerKey = conf[1]
//}
//func (s *NapTuDong) Charge(order *payStorage.Order, payConf *payStorage.PayConf,
//	params map[string]interface{}) *common.Err {
//	if check, ok := utils.CheckParams2(params,
//		[]string{"code", "serial", "telco"}); ok != nil {
//		return errCode.ErrParams.SetKey(check)
//	}
//	s.initKey()
//	code := params["code"].(string)     //mathe 刮刮卡代码
//	serial := params["serial"].(string) //seri 刮刮卡序列号
//	telco := strings.ToUpper(params["telco"].(string))
//	//callback := order.NotifyUrl
//	if !utils.IsContainStr(telcoArray, telco) {
//		return errCode.ErrParams.SetKey("telco")
//	}
//	phoneChargeConf := payStorage.QueryPhoneChargeConf(telco, int(order.Amount))
//	if phoneChargeConf == nil {
//		return errCode.ErrParams.SetKey("amount")
//	}
//	order.Remark = telco
//	r := rand.New(rand.NewSource(time.Now().UnixNano()))
//	requestId := utils.RandInt64(100000000, 999999999,r)
//	order.ThirdId = strconv.Itoa(int(requestId))
//	order.Fee = order.Amount * int64(phoneChargeConf.FeePerThousand) / 1000
//	order.GotAmount = order.Amount - order.Fee
//	requestParams := map[string]interface{}{
//		"telco":      telco,
//		"code":       code,
//		"serial":     serial,
//		"request_id": requestId,
//		"partner_id": partnerId,
//		"command":    "charging",
//		//"command": "check",
//	}
//	requestParams["sign"] = s.getSign(requestParams)
//	requestParams["amount"] = order.Amount
//	url := "http://api.naptudong.com/chargingws/v2"
//	newStr := ""
//	for k, v := range requestParams {
//		n := ""
//		switch v.(type) {
//		case int64:
//			n = strconv.Itoa(int(v.(int64)))
//		default:
//			n = v.(string)
//		}
//		newStr += fmt.Sprintf("%s=%s&", k, n)
//	}
//	newStr = strings.TrimRight(newStr, "&")
//	tr := &http.Transport{
//		//Proxy:           http.ProxyURL(proxy),
//		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
//	}
//	client := &http.Client{
//		Transport: tr,
//		Timeout:   time.Second * 5, //超时时间
//	}
//	resp, err := client.Post(url, "application/x-www-form-urlencoded",
//		strings.NewReader(newStr))
//	log.Info("request params : %v", newStr)
//	if err != nil {
//		log.Error(err.Error())
//		return errCode.ChargeProtectError.SetKey()
//	}
//	defer func() {
//		if resp != nil {
//			resp.Body.Close()
//		}
//	}()
//	body, _ := ioutil.ReadAll(resp.Body)
//	var res map[string]interface{}
//	if err := json.Unmarshal(body, &res); err != nil {
//		log.Error(err.Error())
//		return errCode.ChargeProtectError.SetKey()
//	}
//	log.Info("napTuDong uid:%v response: %v", order.UserId.Hex(), string(body))
//	phoneCharge := &payStorage.PhoneCharge{
//		Oid:      order.Oid,
//		Seri:     serial,
//		Password: code,
//		Amount:   order.Amount,
//		CreateAt: utils.Now(),
//	}
//	//errCode := errCode.Success(&Response{})
//	if res["status"].(float64) == 99 {
//		payStorage.InsertPhoneCharge(phoneCharge)
//		order.Status = payStorage.StatusProcess
//	} else if res["status"].(float64) == 1 { //success
//		payStorage.InsertPhoneCharge(phoneCharge)
//		o := *order
//		go func() {
//			time.Sleep(100 * time.Millisecond)
//			SuccessOrder(&o)
//		}()
//	} else {
//		order.Status = payStorage.StatusFailed
//		order.Remark = res["message"].(string)
//		//errCode.Code = -1
//		//errCode.ErrMsg = "Thẻ đã sử dụng hoặc điền sai vị trí seri và mã thẻ !"//fmt.Sprintf("code: %d,message:%s",int(res["status"].(float64)),res["message"].(string))
//		return errCode.NapTuDongError.SetKey()
//	}
//	return errCode.Success(nil)
//}
//
//func (s *NapTuDong) NotifyCharge(w http.ResponseWriter, r *http.Request) {
//	_ = r.ParseForm()
//	p := r.PostForm
//	log.Info("receive params: %v", p)
//	if check, ok := utils.CheckParams(p, []string{"status", "request_id", "amount", "value"}); ok != nil {
//		ToResponse(w, fmt.Sprintf("params is error: %s", check))
//		return
//	}
//	b, _ := json.Marshal(p)
//	payStorage.NewCallBack("charge", string(b), "napTuDong", "")
//	s.initKey()
//	status, _ := utils.ConvertInt(p["status"][0])
//	thirdId, _ := utils.ConvertInt(p["request_id"][0])
//	amount, _ := utils.ConvertInt(p["amount"][0]) //您收到的金额（VND）
//	value, _ := utils.ConvertInt(p["value"][0])
//	message := p["message"][0]
//	code := p["code"][0]
//	serial := p["serial"][0]
//	signStr := fmt.Sprintf("%s%s%s", partnerKey, code, serial)
//	sum := md5.Sum([]byte(signStr))
//	sign := fmt.Sprintf("%x", sum)
//	if sign != p["callback_sign"][0] {
//		log.Error("callback_sign is error")
//		ToResponse(w, "callback_sign is error")
//		return
//	}
//	order := payStorage.QueryOrderThirdId(strconv.Itoa(int(thirdId)))
//	if order == nil {
//		log.Error("order is not found")
//		ToResponse(w, "order is not found")
//		return
//	}
//	if order.Status == payStorage.StatusFailed || order.Status == payStorage.StatusSuccess {
//		ToResponse(w, "already processed")
//		return
//	}
//	phoneCharge := payStorage.QueryPhoneCharge(order.Oid)
//	phoneCharge.RealAmount = value
//	payStorage.UpdatePhoneCharge(phoneCharge)
//	if status == 1 || status == 2 {
//		order.GotAmount = amount
//		order.Fee = value - amount
//		order.Amount = value
//		if status == 2 {
//			order.Remark = fmt.Sprintf("%d-%s", status, message)
//		}
//		SuccessOrder(order)
//		ToResponse(w, "success")
//		return
//	} else {
//		order.Status = payStorage.StatusCallBack
//		order.Remark = fmt.Sprintf("%d-%s", status, message)
//		order.UpdateAt = utils.Now()
//		payStorage.UpdateOrder(order)
//		log.Info("status is err :%v,msg:%v", status, message)
//		ToResponse(w, "failed")
//		return
//	}
//}
//func (s *NapTuDong) getSign(params map[string]interface{}) string {
//	var keys []string
//	for k := range params {
//		keys = append(keys, k)
//	}
//	sort.Strings(keys)
//	str := partnerKey
//	for _, k := range keys {
//		v := ""
//		switch params[k].(type) {
//		case float64:
//			v = fmt.Sprintf("%.2f", params[k].(float64))
//		case int64:
//			v = strconv.FormatInt(params[k].(int64), 10)
//		case int:
//			v = strconv.Itoa(params[k].(int))
//		default:
//			v = params[k].(string)
//		}
//		str = fmt.Sprintf("%s%s", str, v)
//	}
//	sum := md5.Sum([]byte(str))
//	md5str1 := fmt.Sprintf("%x", sum)
//	return md5str1
//}
//
////func (s *NapTuDong)response(writer http.ResponseWriter,response *common.Err)  {
////	byte,err := response.Json()
////	if err != nil {
////		log.Error("json format is err in your response")
////	}
////	if _,err := writer.Write(byte);err != nil{
////		log.Error(err.Error())
////	}
////}
