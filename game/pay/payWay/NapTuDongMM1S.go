package payWay

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/storage"
	"vn/storage/payStorage"
)

type NapTuDong struct {
}

var partnerId = "aWc5K0pQOXhkNWwzazJ3YnJUU1AxZz09"
var accessKey = "2815579d7a177c1f3f5db4cb2f6ce494"
var telcoArray = []string{"VIETTEL", "VINAPHONE", "MOBIFONE"}

func (s *NapTuDong) initKey() {
	conf := strings.Split(storage.QueryConf(storage.KPhoneChargeNapTuDongMM1S).(string), ",")
	partnerId = conf[0]
	accessKey = conf[1]
}
func (s *NapTuDong) Charge(order *payStorage.Order, payConf *payStorage.PayConf,
	params map[string]interface{}) *common.Err {
	if check, ok := utils.CheckParams2(params,
		[]string{"code", "serial", "telco"}); ok != nil {
		return errCode.ErrParams.SetKey(check)
	}
	s.initKey()
	code := params["code"].(string)     //mathe 刮刮卡代码
	serial := params["serial"].(string) //seri 刮刮卡序列号
	telco := strings.ToUpper(params["telco"].(string))
	//callback := order.NotifyUrl
	if !utils.IsContainStr(telcoArray, telco) {
		return errCode.ErrParams.SetKey("telco")
	}
	phoneChargeConf := payStorage.QueryPhoneChargeConf(telco, int(order.Amount))
	if phoneChargeConf == nil {
		return errCode.ErrParams.SetKey("amount")
	}
	order.Remark = telco
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	requestId := utils.RandInt64(100000, 999999, r)
	order.ThirdId = strconv.FormatInt(requestId, 10) + strconv.FormatInt(time.Now().Unix(), 10)
	order.Fee = order.Amount * int64(phoneChargeConf.FeePerThousand) / 1000
	order.GotAmount = order.Amount - order.Fee
	telcoNumber := int64(0)
	for k, v := range telcoArray {
		if v == telco {
			telcoNumber = int64(k) + 1
			break
		}
	}
	requestParams := map[string]interface{}{
		"ref_id":       order.ThirdId,
		"card_code":    code,
		"card_serial":  serial,
		"card_value":   order.Amount,
		"card_telco":   telcoNumber,
		"partner_id":   partnerId,
		"callback_url": order.NotifyUrl,
	}
	requestParams["signature"] = s.getSign(requestParams)

	url := "http://api.mm1s.com/api/v2/add?"
	newStr := ""
	for k, v := range requestParams {
		n := ""
		switch v.(type) {
		case int64:
			n = strconv.Itoa(int(v.(int64)))
		default:
			n = v.(string)
		}
		newStr += fmt.Sprintf("%s=%s&", k, n)
	}
	newStr = strings.TrimRight(newStr, "&")
	tr := &http.Transport{
		//Proxy:           http.ProxyURL(proxy),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 5, //超时时间
	}
	url += newStr
	resp, err := client.Get(url)
	log.Info("request params : %v", newStr)
	if err != nil {
		log.Error(err.Error())
		return errCode.ChargeProtectError.SetKey()
	}
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()
	body, _ := ioutil.ReadAll(resp.Body)
	var res map[string]interface{}
	if err := json.Unmarshal(body, &res); err != nil {
		log.Error(err.Error())
		return errCode.ChargeProtectError.SetKey()
	}
	log.Info("napTuDong uid:%v response: %v", order.UserId.Hex(), string(body))
	phoneCharge := &payStorage.PhoneCharge{
		Oid:      order.Oid,
		Seri:     serial,
		Password: code,
		Amount:   order.Amount,
		CreateAt: utils.Now(),
	}
	//errCode := errCode.Success(&Response{})
	if res["status"].(float64) == 1 {
		payStorage.InsertPhoneCharge(phoneCharge)
		order.Status = payStorage.StatusProcess
	} else {
		order.Status = payStorage.StatusFailed
		order.Remark = res["message"].(string)
		//errCode.Code = -1
		//errCode.ErrMsg = "Thẻ đã sử dụng hoặc điền sai vị trí seri và mã thẻ !"//fmt.Sprintf("code: %d,message:%s",int(res["status"].(float64)),res["message"].(string))
		return errCode.NapTuDongError.SetKey()
	}
	return errCode.Success(nil)
}

func (s *NapTuDong) NotifyCharge(w http.ResponseWriter, r *http.Request) {
	res := r.URL.Query()
	log.Info("receive params: %s", res)

	if _, ok := utils.CheckParams(res,
		[]string{"status", "amount_add", "amount_real", "amount", "ref_id", "message", "signature"}); ok != nil {
		log.Error("---- error params---")
		ToResponse(w, "error params")
		return
	}
	payStorage.NewCallBack("charge", r.RequestURI, "napTuDong", "")
	s.initKey()
	status, _ := utils.ConvertInt(res["status"][0])
	thirdId := res["ref_id"][0]
	amount, _ := utils.ConvertInt(res["amount"][0]) //您收到的金额（VND）
	value, _ := utils.ConvertInt(res["amount_real"][0])
	message := res["message"][0]
	signStr := fmt.Sprintf("%s|%s|%s|%s", res["status"][0], res["amount"][0], res["ref_id"][0], accessKey)
	sum := md5.Sum([]byte(signStr))
	sign := fmt.Sprintf("%x", sum)
	if sign != res["signature"][0] {
		log.Error("callback_sign is error")
		ToResponse(w, "callback_sign is error")
		return
	}
	order := payStorage.QueryOrderThirdId(thirdId)
	if order == nil {
		log.Error("order is not found")
		ToResponse(w, "order is not found")
		return
	}
	if order.Status == payStorage.StatusFailed || order.Status == payStorage.StatusSuccess {
		ToResponse(w, "already processed")
		return
	}
	phoneCharge := payStorage.QueryPhoneCharge(order.Oid)
	phoneCharge.RealAmount = value
	payStorage.UpdatePhoneCharge(phoneCharge)
	if status == 1 {
		phoneChargeConf := payStorage.QueryPhoneChargeConf(order.Remark, int(amount))
		if phoneChargeConf == nil {
			return
		}
		order.Amount = amount
		order.Fee = amount * int64(phoneChargeConf.FeePerThousand) / 1000
		order.GotAmount = amount - order.Fee
		if status == 2 {
			order.Remark = fmt.Sprintf("%d-%s", status, message)
		}
		SuccessOrder(order)
		ToResponse(w, "success")
		return
	} else {
		order.Status = payStorage.StatusCallBack
		order.Remark = fmt.Sprintf("%d-%s", status, message)
		order.UpdateAt = utils.Now()
		payStorage.UpdateOrder(order)
		log.Info("status is err :%v,msg:%v", status, message)
		ToResponse(w, "failed")
		return
	}
}
func (s *NapTuDong) getSign(params map[string]interface{}) string {
	keys := []string{
		"ref_id",
		"card_code",
		"card_serial",
		"card_value",
		"card_telco",
	}
	str := ""
	for _, v := range keys {
		t := ""
		switch params[v].(type) {
		case float64:
			t = fmt.Sprintf("%.2f", params[v].(float64))
		case int64:
			t = strconv.FormatInt(params[v].(int64), 10)
		case int:
			t = strconv.Itoa(params[v].(int))
		default:
			t = params[v].(string)
		}
		t = t + "|"
		str = fmt.Sprintf("%s%s", str, t)
	}
	str = fmt.Sprintf("%s%s", str, accessKey)
	sum := md5.Sum([]byte(str))
	md5str1 := fmt.Sprintf("%x", sum)
	return md5str1
}
