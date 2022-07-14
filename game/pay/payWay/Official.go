package payWay

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/game/admin"
	gate2 "vn/gate"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
)

type Official struct {
}

var appID = "1395677221686874114"

type AutoCreate struct { //request params
	BizOrderId      string `json:"bizOrderId"` //业务单号
	PayMethod       string `json:"payMethod"`  //支付方式
	Amount          int64  `json:"amount"`     //金额
	AppId           string `json:"appId"`
	ProductBankId   string `json:"productBankId"`   //银行账号id
	BankId          string `json:"bankId"`          //收款银行id
	BankName        string `json:"bankName"`        //收款银行名称
	BankNo          string `json:"bankNo"`          //收款银行账号
	BankUserName    string `json:"bankUserName"`    //收款银行账户
	BankUserMobile  string `json:"bankUserMobile"`  //收款银行手机号
	ExtendValidCode string `json:"extendValidCode"` //扩展校验码
	ValidCode       string `json:"validCode"`       //
	CreateUserId    string `json:"createUserId"`    //接入系统的会员ID
	CreateUserName  string `json:"createUserName"`  //接入系统的会员名称
}

func (s *Official) Charge(order *payStorage.Order, payConf *payStorage.PayConf,
	params map[string]interface{}) *common.Err {
	if check, ok := utils.CheckParams2(params,
		[]string{"saveType", "accountName", "receiveId"}); ok != nil {
		return errCode.ErrParams.SetKey(check)
	}
	receiveId := params["receiveId"].(string)
	rId := utils.ConvertOID(receiveId)
	receive := payStorage.QueryCompanyBank(rId)
	if receive == nil {
		return errCode.ErrParams.SetKey("receiveId")
	}
	if receive.IsAuto == 0 {
		return s.ChargeOfficial(order, payConf, params)
	} else {
		return s.ChargeOfficialAuto(order, payConf, params)
	}
}
func (s *Official) ChargeOfficial(order *payStorage.Order, payConf *payStorage.PayConf,
	params map[string]interface{}) *common.Err {
	saveType := params["saveType"].(string)
	accountName := params["accountName"].(string)
	receiveId := params["receiveId"].(string)
	rId := utils.ConvertOID(receiveId)
	if receive := payStorage.QueryCompanyBank(rId); receive == nil {
		return errCode.ErrParams.SetKey("receiveId")
	}
	user := userStorage.QueryUserId(utils.ConvertOID(order.UserId.Hex()))
	code := strconv.FormatInt(user.ShowId%10000000, 10) //utils.Base58encode(int(user.ShowId))
	for len(code) < 7 {
		code = "0" + code
	}
	orderTransfer := &payStorage.OrderTransfer{
		OrderId:     order.Oid,
		ReceiveId:   rId,
		AccountName: accountName,
		SaveType:    saveType,
		Code:        code,
		CreateAt:    utils.Now(),
	}
	order.GotAmount = order.Amount
	payStorage.InsertOrderTransfer(orderTransfer)
	NotifyAdmin("order")
	return errCode.Success(nil)
}
func (s *Official) ChargeOfficialAuto(order *payStorage.Order, payConf *payStorage.PayConf,
	params map[string]interface{}) *common.Err {
	saveType := params["saveType"].(string)
	accountName := params["accountName"].(string)
	receiveId := params["receiveId"].(string)
	rId := utils.ConvertOID(receiveId)
	receive := payStorage.QueryCompanyBank(rId)
	if receive == nil {
		return errCode.ErrParams.SetKey("receiveId")
	}
	user := userStorage.QueryUserId(utils.ConvertOID(order.UserId.Hex()))
	code := strconv.FormatInt(user.ShowId%10000000, 10) //utils.Base58encode(int(user.ShowId))
	for len(code) < 7 {
		code = "0" + code
	}

	tr := &http.Transport{
		//Proxy:           http.ProxyURL(proxy),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 5, //超时时间
	}
	//获取银行列表
	getAll := "https://checkstandapi.prod.virtuous5.com/bld/api/pay/bank/getAll"
	resp, err := client.Get(getAll)
	bankId := ""
	if err == nil {
		body, _ := ioutil.ReadAll(resp.Body)
		res := make(map[string]interface{})
		if err := json.Unmarshal(body, &res); err != nil {
			log.Error(err.Error())
			return s.ChargeOfficial(order, payConf, params)
		}
		cc := res["result"].([]interface{})
		for _, v := range cc {
			dd := v.(map[string]interface{})
			if dd["bankName"].(string) == receive.BankName {
				bankId = dd["id"].(string)
				break
			}
		}
		if bankId == "" {
			return s.ChargeOfficial(order, payConf, params)
		}
	}

	//创建订单
	create := &AutoCreate{}
	create.BizOrderId = order.Oid.Hex()
	create.PayMethod = "BANK_TRANSFER"
	create.Amount = 100 * order.Amount
	create.AppId = appID
	create.BankId = bankId
	create.ValidCode = code
	create.BankName = receive.BankName
	create.BankNo = receive.CardNumber
	create.BankUserName = receive.AccountName
	create.BankUserMobile = receive.Phone
	requestParams, _ := json.Marshal(create)
	createUrl := "https://checkstandapi.prod.virtuous5.com/bld/api/pay/recharge/create"
	log.Info("request params : %v", string(requestParams))
	resp, err = client.Post(createUrl, "application/json", bytes.NewBuffer(requestParams))
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()
	body, _ := ioutil.ReadAll(resp.Body)
	var res map[string]interface{}
	if err := json.Unmarshal(body, &res); err != nil {
		log.Error(err.Error())
		return s.ChargeOfficial(order, payConf, params)
	}
	if res["code"].(float64) == 0 {
		orderTransfer := &payStorage.OrderTransfer{
			OrderId:     order.Oid,
			ReceiveId:   rId,
			AccountName: accountName,
			SaveType:    saveType,
			Code:        code,
			CreateAt:    utils.Now(),
		}

		cc := res["result"].(map[string]interface{})
		order.ThirdId = cc["id"].(string)
		order.GotAmount = order.Amount
		payStorage.InsertOrderTransfer(orderTransfer)

		return errCode.Success(nil)
	}
	log.Error(string(body))
	return s.ChargeOfficial(order, payConf, params)
}

type ReceiveData struct { //receive params
	AppId         string `json:"appId"`         //
	BizOrderId    string `json:"bizOrderId"`    //
	TotalAmount   int64  `json:"totalAmount"`   //
	ReceiveAmount int64  `json:"receiveAmount"` //
}

func (s *Official) NotifyCharge(w http.ResponseWriter, r *http.Request) {
	ip := utils.GetIP(r)

	body, _ := ioutil.ReadAll(r.Body)

	log.Info("ip: %v,receive params: %s", ip, body)
	payStorage.NewCallBack("charge", string(body), "autoOfficial", "")

	//if !utils.IsContainStr(ipWhiteList, ip){
	//	log.Info("ip is not allow: %s", ip)
	//	//ToResponse(w,"ip is not allowed");return
	//}
	var res map[string]interface{}
	if err := json.Unmarshal(body, &res); err != nil {
		log.Error(err.Error())
		return
	}
	if res["appId"].(string) == appID {
		cc := res["content"].(string)
		receiveData := ReceiveData{}
		json.Unmarshal([]byte(cc), &receiveData)
		orderNo := receiveData.BizOrderId
		order := payStorage.QueryOrder(utils.ConvertOID(orderNo))
		if order == nil {
			ToResponse(w, "order is not found")
			return
		}
		if order.Status != payStorage.StatusInit {
			ToResponse(w, "already processed")
			return
		}
		orderAmount, _ := utils.ConvertInt(receiveData.TotalAmount)
		if orderAmount != order.Amount*100 {
			ToResponse(w, "Amount error")
			return
		}
		receiveAmount, _ := utils.ConvertInt(receiveData.ReceiveAmount)
		order.Fee = (orderAmount - receiveAmount) / 100
		order.GotAmount = receiveAmount / 100
		SuccessOrder(order)
		NotifyAdmin("order")
		ToResponse(w, "SUCCESS")
	} else {
		ToResponse(w, "appId error")
		return
	}
}
func NotifyAdmin(Type string) {
	go func() {
		topic := "adminSystem/pay"
		msg := make(map[string]interface{})
		msg["type"] = Type
		b, _ := json.Marshal(msg)
		sessionBean := gate2.QuerySessionBean(admin.AdminUid)
		if sessionBean != nil {
			session, err := basegate.NewSession(common.App, sessionBean.Session)
			if err != nil {
				log.Error(err.Error())
			} else {
				if err := session.SendNR(topic, b); err != "" {
					log.Error(err)
				}
			}
		}
	}()
}
