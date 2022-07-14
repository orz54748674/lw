package payWay

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/storage/payStorage"

	"github.com/fatih/structs"
)

type VgPay struct {
}

const (
	//代收付
	//appSecret: WfjMNLPqfUGhbdGRtw1vPw
	// key 密钥：cJoHVbFAYkmMrzYwmMDDiQ
	merchantNo = "32858"
	appSecret  = "WfjMNLPqfUGhbdGRtw1vPw"
	appKey     = "cJoHVbFAYkmMrzYwmMDDiQ"
)

var channelNos = map[string]int{
	"MomoPay": 0,
	"ZaloPay": 1,
	"bankQr":  2, //bankName
	"direct":  3, //bankName
	"gate":    4,
	"VTPay":   5,
}

func (s *VgPay) Charge(order *payStorage.Order, payConf *payStorage.PayConf,
	params map[string]interface{}) *common.Err {
	bankName := ""
	if payConf.MethodType == "bankQr" ||
		payConf.MethodType == "direct" {
		if check, ok := utils.CheckParams2(params,
			[]string{"bankName"}); ok != nil {
			return errCode.ErrParams.SetKey(check)
		}
		bankName = params["bankName"].(string)
	}

	now := time.Now()
	//user := userStorage.QueryUserId(order.UserId)
	create := &Create{}
	create.MerchantNo = merchantNo
	create.AppSecret = appSecret
	create.Datetime = now.Format("2006-01-02 15:04:05")
	create.Time = time.Now().UnixNano() / 1e6
	create.OrderNo = order.Oid.Hex()
	create.UserName = order.UserId.Hex()
	create.ChannelNo = channelNos[payConf.MethodType]
	create.Amount = fmt.Sprintf("%.2f", float64(order.Amount))
	create.Discount = "0.00"
	create.BankName = bankName
	create.NotifyUrl = order.NotifyUrl

	create.Sign = s.getSign(*create)
	requestParams, _ := json.Marshal(create)
	_url := "https://lf.thepay.co.nz/order/create"

	tr := &http.Transport{
		//Proxy:           http.ProxyURL(proxy),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if params["env"].(string) == "dev" {
		proxy, _ := url.Parse("http://127.0.0.1:1080")
		tr.Proxy = http.ProxyURL(proxy)
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 5, //超时时间
	}

	resp, err := client.Post(_url, "application/json", bytes.NewBuffer(requestParams))
	log.Info("request params : %v", string(requestParams))
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
	if res["code"].(float64) == 0 {
		order.ThirdId = res["tradeNo"].(string)
		order.GotAmount = order.Amount
		response := &Response{
			OrderNo:   order.Oid.Hex(),
			TargetUrl: res["targetUrl"].(string),
			QrCode:    res["qrcode"].(string),
		}
		return errCode.Success(response)
	}
	log.Error(string(body))
	return errCode.ChargeProtectError.SetErr(string(body))
}
func (VgPay) getSign(request Create) string {
	params := structs.Map(request)
	return sign(params)
}
func sign(params map[string]interface{}) string {
	delete(params, "UserName")
	delete(params, "userName")
	delete(params, "AmountBeforeFixed")
	delete(params, "amountBeforeFixed")
	delete(params, "ChannelNo")
	delete(params, "channelNo")
	delete(params, "PayeeName")
	delete(params, "payeeName")
	delete(params, "AppSecret")
	delete(params, "appSecret")
	delete(params, "BtName")
	delete(params, "bankName")
	delete(params, "BankName")
	delete(params, "Sign")
	delete(params, "sign")
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	str := ""
	for _, k := range keys {
		v := ""
		switch params[k].(type) {
		case float64:
			v = fmt.Sprintf("%.2f", params[k].(float64))
		case int64:
			v = strconv.FormatInt(params[k].(int64), 10)
		case int:
			v = strconv.Itoa(params[k].(int))
		default:
			v = params[k].(string)
		}
		str = fmt.Sprintf("%s%s=%s&", str, toLowFirstChar(k), v)
	}
	str = strings.TrimRight(str, "&")
	str = fmt.Sprintf("%s%s", str, appKey)
	log.Info("sign params str: %v", str)
	sum := sha256.Sum256([]byte(str))
	slice := sum[:]
	hex := hex.EncodeToString(slice)
	has := md5.Sum([]byte(hex))
	md5str1 := fmt.Sprintf("%x", has)
	return strings.ToUpper(md5str1)
}

type Create struct { //request params
	MerchantNo        string `json:"merchantNo"`        //商户编号 (由聯發VGPAY平台提供,见:商户信息)
	OrderNo           string `json:"orderNo"`           //商户单号 (用于对账、查询; 不超过32个字符)
	UserNo            int    `json:"userNo"`            //商户客户号 (可选;用于查对:可以是 用户Id或编号)
	UserName          string `json:"userName"`          //商户客户名 (可选;用于查对:可以是 用户名、手机号等) (不参加加密)
	ChannelNo         int    `json:"channelNo"`         //支付通道编号 (纯数字格式; MomoPay:0 | ZaloPay:1 | 银行扫码:2 | 直連:3 | 网关:4 |VTPay:5 ) (不参加加密)
	Amount            string `json:"amount"`            //订单金额 (单位：VND； 最小充值金额10K，不超过50000K。此金额可能会变动,请与聯發VGPAY平台确认),
	AmountBeforeFixed int    `json:"amountBeforeFixed"` //修改前订单金额 (单位：VND，可选，不参与加密)
	Discount          string `json:"discount"`          //立减金额 (可选;单位:VND; 配合商户活动用),
	PayeeName         string `json:"payeeName"`         //付款人姓名 (可选;用于实名匹配订单:付款人的真实姓名。不确定此用途的话,请留空此项。有此项的话,网关通道客人无需再次填写付款人姓名) (不参加加密),
	BankName          string `json:"bankName"`          //银行名称 (用于银行扫码（通道2）,直連（通道3） 的收款账户分配) (不参加加密),
	Extra             string `json:"extra"`             //附加信息 (可选;回调时原样返回)
	Datetime          string `json:"datetime"`          //日期时间 (格式:2018-01-01 23:59:59)
	NotifyUrl         string `json:"notifyUrl"`         //异步通知地址 (当用户完成付款时,支付平台将向此URL地址,异步发送付款通知。建议使用 https)
	Time              int64  `json:"time"`              //时间戳, 1970-01-01开始的Linux timestamp
	AppSecret         string `json:"appSecret"`         //(由聯發VGPAY平台提供,见:商户信息/appSecret) (不参加加密)
	Sign              string `json:"sign"`              //数据签名 (参见【签名规则】中的说明)
}

func toLowFirstChar(str string) string {
	char := strings.ToLower(string(str[0]))
	newS := char + trimLeftChar(str)
	return newS
}

func trimLeftChar(s string) string {
	for i := range s {
		if i > 0 {
			return s[i:]
		}
	}
	return s[:0]
}

var ipWhiteList = []string{"13.94.46.176", "52.163.187.113", "52.175.72.66", "20.189.72.226", "127.0.0.1"}

func (s *VgPay) NotifyCharge(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	p := r.PostForm
	json, _ := json.Marshal(p)
	call := string(json)
	ip := utils.GetIP(r)
	log.Info("ip: %v,receive params: %v", ip, call)
	payStorage.NewCallBack("charge", call, "vgPay", "")

	if !utils.IsContainStr(ipWhiteList, ip) {
		log.Error("ip is not allow: %s", ip)
		ToResponse(w, "ip is not allowed")
		return
	}
	params := make(map[string]interface{}, len(p))
	for k, v := range p {
		params[k] = v[0]
	}
	getSign := sign(params)
	if getSign != p["sign"][0] {
		ToResponse(w, "sign is error")
		return
	}
	if p["status"][0] == "PAID" || p["status"][0] == "MANUAL PAID" {
		orderNo := p["orderNo"][0]
		order := payStorage.QueryOrder(utils.ConvertOID(orderNo))
		if order == nil {
			ToResponse(w, "order is not found")
			return
		}
		if order.Status == payStorage.StatusSuccess {
			ToResponse(w, "already processed")
			return
		}
		orderAmount, _ := utils.ConvertInt(p["amount"][0])
		if orderAmount < order.Amount {
			ToResponse(w, "Amount error")
			return
		}
		if payConf := payStorage.QueryPayConf(order.MethodId); payConf != nil {
			order.Fee = order.Amount * int64(payConf.FeePerThousand) / 1000
		}
		SuccessOrder(order)
		ToResponse(w, "success")
	}
}

func AutoCheck() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buff := make([]byte, 1024)
				runtime.Stack(buff, false)
				log.Error("panic(%v)\n info:%s", r, string(buff))
			}
		}()
		log.Info("StartCheckVgPayOrder")
		for true {
			StartCheckVgPayOrder()
			time.Sleep(5 * time.Minute)
		}
	}()
}
func StartCheckVgPayOrder() {
	orders := payStorage.QueryAllVGPayWaitOrder()
	now := time.Now()
	for _, order := range orders {
		diff := now.Sub(order.CreateAt)
		if diff > 10*time.Minute {
			order.Status = payStorage.StatusFailed
			order.UpdateAt = utils.Now()
			payStorage.UpdateOrder(&order)
		}
	}
}

//func parseSign(p url.Values) string {
//	params := make(map[string]interface{},len(p))
//	for k,v := range p{
//		params[k] = v[0]
//	}
//	return sign(params)
//}
