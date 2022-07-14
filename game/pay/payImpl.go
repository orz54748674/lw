package pay

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	"vn/game/pay/payWay"
	gate2 "vn/gate"
	"vn/storage"
	"vn/storage/activityStorage"
	"vn/storage/agentStorage"
	"vn/storage/gameStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

type Impl struct {
	App            module.App
	Settings       *conf.ModuleSettings
	methodInstance map[string]interface{}
}

func (s *Impl) payInfo(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	payConfList := payStorage.QueryPayConfList()
	//payActivityConfList := payStorage.QueryPayActivityConfList()
	receiveBankList := *payStorage.QueryCompanyBankList()
	res := make(map[string]interface{}, 2)
	res["payList"] = payConfList
	//res["activityInfo"] = payActivityConfList
	res["receiveBankList"] = receiveBankList

	for k, _ := range receiveBankList {
		receiveBankList[k].Phone = ""
	}
	user := userStorage.QueryUserId(utils.ConvertOID(session.GetUserID()))
	code := strconv.FormatInt(user.ShowId%10000000, 10) //utils.Base58encode(int(user.ShowId))
	for len(code) < 7 {
		code = "0" + code
	}
	res["code"] = code
	phoneChargeConf := payStorage.QueryAllPhoneChargeConf()
	res["phoneChargeConf"] = phoneChargeConf
	douDouBT := payStorage.QueryUserReceiveBt(utils.ConvertOID(uid))
	if douDouBT == nil {
		douDouBT = &payStorage.UserReceiveBt{}
	}
	res["myDouDouBT"] = douDouBT
	vgBankList := payStorage.QueryVGBankList()
	res["vgPayBankList"] = vgBankList
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) charge(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	if user.Type != userStorage.TypeNormal {
		return errCode.AccountNotAllow.GetI18nMap(), nil
	}
	if check, ok := utils.CheckParams2(params,
		[]string{"methodId"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), ok
	}
	methodId := params["methodId"].(string)
	payConf := payStorage.QueryPayConf(utils.ConvertOID(methodId))
	if payConf == nil {
		return errCode.ErrParams.SetKey("methodId").GetMap(), nil
	}
	amount, _ := utils.ConvertInt(params["amount"])
	if amount < int64(payConf.Mini) || amount > int64(payConf.Max) {
		allowRange := fmt.Sprintf("%d - %d", payConf.Mini, payConf.Max)
		return errCode.AmountNotAllow.SetKey(allowRange).GetI18nMap(), nil
	}
	if payConf.MethodType != "bank" && payConf.Merchant != "Official" {
		if check, ok := utils.CheckParams2(params,
			[]string{"amount"}); ok != nil {
			return errCode.ErrParams.SetKey(check).GetMap(), ok
		}
	}
	if payConf.Merchant == "Official" { //同一个用户未处理完，只能提交一次银行卡充值
		patternList := []string{`^[a-zA-Z ]+$`}
		accountName := params["accountName"].(string)
		for _, pattern := range patternList {
			match, _ := regexp.MatchString(pattern, accountName)
			if !match {
				return errCode.DouDouAccountNotAllow.GetI18nMap(), nil
			}
		}
		receiveId := params["receiveId"].(string)
		rId := utils.ConvertOID(receiveId)
		receive := payStorage.QueryCompanyBank(rId)
		if receive == nil {
			return errCode.ErrParams.SetKey("receiveId").GetMap(), nil
		}
		if receive.IsAuto == 0 {
			res := payStorage.QueryInitOrderByUid(utils.ConvertOID(uid), utils.ConvertOID(methodId))
			if res != nil && len(res) > 0 {
				return errCode.UncompleteOrder.GetI18nMap(), nil
			}
		} else if receive.IsAuto == 1 {
			res := payStorage.QueryInitOrderBy5Minute(utils.ConvertOID(uid), utils.ConvertOID(methodId))
			if res != nil && len(res) > 0 {
				return errCode.Order5MinuteOnce.GetI18nMap(), nil
			}
		}
	}
	params["env"] = s.App.GetSettings().Settings["env"].(string)

	ip := ""
	if tokenObj := userStorage.QueryTokenByUid(utils.ConvertOID(uid)); tokenObj != nil {
		ip = tokenObj.Ip
	}

	order := createOrder(uid, payConf.Oid, amount, ip)
	order.NotifyUrl = getNotifyUrl(payConf.Merchant)
	if instance, ok := s.methodInstance[payConf.Merchant]; ok {
		getValue := reflect.ValueOf(instance)
		methodValue := getValue.MethodByName("Charge")
		args := []reflect.Value{reflect.ValueOf(order), reflect.ValueOf(payConf), reflect.ValueOf(params)}
		res := methodValue.Call(args)
		if len(res) > 0 {
			err := res[0].Interface().(*common.Err)
			//if err.Code == 0 {
			payStorage.InsertOrder(order)
			//}
			return err.GetI18nMap(), nil
		}
	}
	return errCode.PayMethodIdErr.GetI18nMap(), nil
}
func (s *Impl) chargeLog(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"offset", "limit"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetI18nMap(), ok
	}
	uid := session.GetUserID()
	offset, _ := utils.ConvertInt(params["offset"])
	limit, _ := utils.ConvertInt(params["limit"])
	if limit > 101 {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	orderData, count := payStorage.GetOrderData(utils.ConvertOID(uid), int(offset), int(limit))
	res := make(map[string]interface{}, 1)
	res["orderData"] = orderData
	res["totalNum"] = count
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) doudouLog(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"offset", "limit"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetI18nMap(), ok
	}
	uid := session.GetUserID()
	offset, _ := utils.ConvertInt(params["offset"])
	limit, _ := utils.ConvertInt(params["limit"])
	if limit > 101 {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	douDouData, count := payStorage.GetDouDouData(utils.ConvertOID(uid), int(offset), int(limit))
	res := make(map[string]interface{}, 1)
	res["douDouData"] = douDouData
	res["totalNum"] = count
	return errCode.Success(res).GetI18nMap(), nil
}
func getNotifyUrl(merchant string) string {
	host := storage.QueryConf(storage.KApiHost).(string)
	//httpPort := int(common.App.GetSettings().Settings["httpPort"].(float64))
	return fmt.Sprintf("%s/charge/%s", host, strings.ToLower(merchant))
	//return fmt.Sprintf("http://%s:%d/charge/%s", host, httpPort, strings.ToLower(merchant))
}
func (s *Impl) getVGBankList(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	bankList := payStorage.QueryVGBankList()
	return errCode.Success(bankList).GetI18nMap(), nil
}

func createOrder(uid string, methodId primitive.ObjectID, amount int64, ip string) *payStorage.Order {
	order := payStorage.NewOrder(utils.ConvertOID(uid), methodId, amount, ip)
	return order
}
func (s *Impl) initPayMethod() {
	s.methodInstance = make(map[string]interface{}, 4)
	s.methodInstance["Official"] = &payWay.Official{}
	s.methodInstance["VgPay"] = &payWay.VgPay{}
	s.methodInstance["NapTuDong"] = &payWay.NapTuDong{}
}
func (s *Impl) adminAddOrder(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	adminId, _ := utils.ConvertInt(params["admin_id"])
	amount, _ := utils.ConvertInt(params["amount"])
	account := params["account"].(string)
	remark := params["remark"].(string)
	user := userStorage.QueryUser(bson.M{"Account": account})
	if user == nil {
		return errCode.AccountNotExist.GetI18nMap(), nil
	}
	payConf := payStorage.QueryPayConfByMethodType("customerService")
	order := payStorage.NewOrder(user.Oid, payConf.Oid, amount, "")
	order.GotAmount = amount
	order.Remark = remark
	order.AdminId = uint(adminId)
	payStorage.InsertOrder(order)
	payWay.SuccessOrder(order)
	//order.Status = payStorage.StatusSuccess
	//
	//bill := walletStorage.NewBill(order.UserId.Hex(),walletStorage.TypeIncome,
	//	walletStorage.EventCharge,order.Oid.Hex(),order.GotAmount)
	//if err := walletStorage.OperateVndBalance(bill);err != nil{
	//	return errCode.ServerError.GetI18nMap(),nil
	//}
	//payStorage.UpdateOrder(order)
	//payWay.NotifyUserWallet(order.UserId.Hex())
	return errCode.Success(nil).GetI18nMap(), nil
}
func (s *Impl) adminOrder(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	adminId, _ := utils.ConvertInt(params["adminId"])
	status, _ := utils.ConvertInt(params["status"])
	remark := params["remark"].(string)
	order := payStorage.QueryOrder(utils.ConvertOID(params["oid"].(string)))
	if order == nil {
		err := &common.Err{Code: -1, ErrMsg: "order is not found."}
		return err.GetMap(), nil
	}
	if order.Status == payStorage.StatusSuccess {
		return errCode.Success(nil).GetI18nMap(), nil
	}
	uid := order.UserId
	order.Remark = remark
	user := userStorage.QueryUserId(uid)
	if user.Type != userStorage.TypeNormal {
		return errCode.AccountNotAllow.GetI18nMap(), nil
	}
	order.AdminId = uint(adminId)
	if status == payStorage.StatusSuccess && order.Status != payStorage.StatusSuccess {
		payWay.SuccessOrder(order)
	} else {
		order.Status = int(status)
		order.UpdateAt = utils.Now()
		payStorage.UpdateOrder(order)
	}
	return errCode.Success(nil).GetI18nMap(), nil
}
func (s *Impl) douDou(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"amount"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetI18nMap(), ok
	}
	uid := session.GetUserID()
	amount, _ := utils.ConvertInt(params["amount"])
	// btName := params["btName"].(string)
	haveBt := payStorage.QueryUserReceiveBt(utils.ConvertOID(uid))
	if haveBt == nil { //没有绑定银行卡
		return errCode.NotBindBtCard.GetI18nMap(), nil
	}
	bank := payStorage.QueryDouDouBtByName(haveBt.BtName)
	if bank != nil {
		//return errCode.BindBankNotExist.GetI18nMap(), nil
		if amount > int64(bank.Max) || amount < int64(bank.Mini) {
			return errCode.DouDouMinAmount.GetI18nMap(), nil
		}
	}

	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	if user.Type != userStorage.TypeNormal {
		return errCode.AccountNotAllow.GetI18nMap(), nil
	}
	wallet := walletStorage.QueryWallet(user.Oid)
	if wallet.VndBalance < amount {
		return errCode.BalanceNotEnough.GetI18nMap(), nil
	}
	uInfo := userStorage.QueryUserInfo(user.Oid)
	if uInfo.DouDouBet > 0 {
		needBet := utils.ConvertThousandsSeparate(uInfo.DouDouBet)
		return errCode.BetNotEnough.SetKey(needBet).GetMap(), nil
	}
	countLimit, _ := utils.ConvertInt(storage.QueryConf(storage.KCanDouDouCount))
	todayCount := payStorage.QueryTodayCount(uid)
	if int64(todayCount) >= countLimit {
		return errCode.DouDouCountOverLimit.GetI18nMap(), nil
	}
	//同一个用户未处理完，只能提交一次兑换
	res := payStorage.QueryInitDouDouByUid(uid)
	if res != nil && len(res) > 0 {
		return errCode.UncompleteOrder.GetI18nMap(), nil
	}

	//accountName := params["accountName"].(string)
	//cardNum := params["cardNum"].(string)
	ip := ""
	if tokenObj := userStorage.QueryTokenByUid(utils.ConvertOID(uid)); tokenObj != nil {
		ip = tokenObj.Ip
	}
	douDou := payStorage.NewDouDou(uid, haveBt.BtName, haveBt.AccountName, haveBt.CardNum, ip, amount)
	bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses,
		walletStorage.EventDouDou, douDou.Oid.Hex(), -1*amount)
	if err := walletStorage.OperateVndBalance(bill); err == nil {
		payStorage.InsertDouDou(douDou)
		walletStorage.NotifyUserWallet(uid)
		agentStorage.OnWalletChange(uid)
		payWay.NotifyAdmin("douDou")
		return errCode.Success(nil).GetI18nMap(), nil
	}
	return errCode.ServerBusy.GetI18nMap(), nil
}

func (s *Impl) bindBtCard(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"btName", "accountName", "cardNum"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetI18nMap(), ok
	}
	uid := session.GetUserID()
	haveBt := payStorage.QueryUserReceiveBt(utils.ConvertOID(uid))
	if haveBt != nil { //已经绑定过doudouBT
		return errCode.AlreadyBindBankCard.GetI18nMap(), nil
	}

	btName := params["btName"].(string)
	accountName := params["accountName"].(string)
	cardNum := params["cardNum"].(string)

	bank := payStorage.QueryDouDouBtByName(btName)
	if bank == nil {
		return errCode.ErrParams.SetKey("btName").GetI18nMap(), nil
	}
	cardNum = strings.Replace(cardNum, " ", "", -1)
	patternList := []string{`^[0-9]+$`}
	for _, pattern := range patternList {
		match, _ := regexp.MatchString(pattern, cardNum)
		if !match {
			return errCode.DouDouNumNotAllow.GetI18nMap(), nil
		}
	}

	patternList = []string{`^[a-zA-Z ]+$`}
	for _, pattern := range patternList {
		match, _ := regexp.MatchString(pattern, accountName)
		if !match {
			return errCode.DouDouAccountNotAllow.GetI18nMap(), nil
		}
	}

	allBtCard := payStorage.QueryAllReceiveBt()
	for _, v := range allBtCard { //判断有没有被绑过
		if v.CardNum == cardNum {
			return errCode.DouDouSameBt.GetI18nMap(), nil
		}
	}
	bt := &payStorage.UserReceiveBt{
		Oid:         utils.ConvertOID(uid),
		BtName:      btName,
		AccountName: accountName,
		CardNum:     cardNum,
		CreateAt:    utils.Now(),
	}
	payStorage.InsertUserReceiveBt(bt)
	return errCode.Success(nil).GetMap(), nil
}
func (s *Impl) douDouBtList(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	douDouBtList := payStorage.QueryAllDouDouBt()
	res := make(map[string]interface{}, 1)
	res["douDouBtList"] = douDouBtList
	return errCode.Success(res).GetI18nMap(), nil
}
func (s *Impl) adminDouDou(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	log.Info("params: %v", params)
	status, _ := utils.ConvertInt(params["status"])
	adminId, _ := utils.ConvertInt(params["adminId"])
	remark := params["remark"].(string)
	doudouId := params["doudouId"].(string)
	doudou := payStorage.QueryDouDou(utils.ConvertOID(doudouId))
	doudou.UpdateAt = utils.Now()
	doudou.AdminId = uint(adminId)
	doudou.Remark = remark
	if status == 2 || status == payStorage.DouDouStatusReject { //拒绝
		doudou.Status = payStorage.DouDouStatusReject
		bill := walletStorage.NewBill(doudou.UserId, walletStorage.TypeIncome,
			walletStorage.EventDouDouRefund, doudouId, doudou.Amount)
		if err := walletStorage.OperateVndBalance(bill); err == nil {
			walletStorage.NotifyUserWallet(doudou.UserId)
			agentStorage.OnWalletChange(doudou.UserId)
		}
	} else {
		doudou.Status = payStorage.StatusSuccess
		agentStorage.OnPayData(doudou.UserId, 0, doudou.Amount)
		userStorage.IncUserDoudou(utils.ConvertOID(doudou.UserId), doudou.Amount)
	}
	doudou.UpdateAt = utils.Now()
	payStorage.UpdateDouDou(&doudou)
	unDouDous := payStorage.QueryDouDouByUser(doudou.UserId)
	for _, w := range unDouDous {
		if w.BtName != doudou.BtName ||
			w.AccountName != doudou.AccountName ||
			w.CardNum != doudou.CardNum {
			w.Status = payStorage.DouDouStatusReject
			w.Remark = common.I18str("DouDouBtNotSame")
			w.UpdateAt = utils.Now()
			payStorage.UpdateDouDou(&w)
		}
	}
	return errCode.Success(nil).GetMap(), nil
}
func (s *Impl) agentIncome2wallet(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"amount"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetI18nMap(), ok
	}
	uid := session.GetUserID()
	amount, _ := utils.ConvertInt(params["amount"])
	if amount < 1 {
		return errCode.ErrParams.SetKey("amount").GetI18nMap(), nil
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	if wallet.AgentBalance < amount {
		return errCode.AgentBalanceNotEnough.GetI18nMap(), nil
	}
	bill := walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventAgentDouDou,
		"", -1*amount)
	bill.Oid = primitive.NewObjectID()
	if err := walletStorage.AgentBalance2vnd(*bill); err != nil {
		return errCode.ServerBusy.GetI18nMap(), nil
	} else {
		walletStorage.NotifyUserWallet(uid)
		agentStorage.OnWalletChange(uid)
		betDetails := map[string]string{
			"amount": strconv.Itoa(int(amount)),
			"event":  "agentIncome2wallet",
		}
		bet, _ := json.Marshal(betDetails)
		wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
		params := gameStorage.BetRecordParam{
			Uid:        uid,
			GameType:   game.Lobby,
			Income:     0,
			BetAmount:  0,
			BetDetails: string(bet),
			CurBalance: wallet.VndBalance + wallet.SafeBalance,
			GameId:     bill.Oid.Hex(),
			GameNo:     "-",
		}
		gameStorage.InsertBetRecord(params)
	}
	return errCode.Success(nil).GetI18nMap(), nil
}
func (s *Impl) safe2wallet(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"amount", "safeType"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetI18nMap(), ok
	}
	uid := session.GetUserID()
	safeType := params["safeType"].(string)
	if safeType != "Store" && safeType != "Pick" {
		return errCode.ErrParams.SetKey().GetI18nMap(), nil
	}
	amount, _ := utils.ConvertInt(params["amount"])
	if amount < 1 {
		return errCode.ErrParams.SetKey("amount").GetI18nMap(), nil
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	var bill *walletStorage.Bill
	if safeType == "Store" {
		if wallet.VndBalance < amount {
			return errCode.AmountNotAllow.GetI18nMap(), nil
		}
		bill = walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventSafeChange, "", amount)
	} else if safeType == "Pick" {
		userInfo := userStorage.QueryUserInfo(utils.ConvertOID(uid))
		if userInfo.SafeStatus == 1 {
			return errCode.PleaseUnlockSafe.GetI18nMap(), nil
		}
		if wallet.SafeBalance < amount {
			return errCode.AmountNotAllow.GetI18nMap(), nil
		}
		bill = walletStorage.NewBill(uid, walletStorage.TypeExpenses, walletStorage.EventSafeChange, "", -amount)
	}
	if bill == nil {
		return errCode.ServerBusy.GetI18nMap(), nil
	}
	bill.Oid = primitive.NewObjectID()
	if err := walletStorage.SafeBalance2vnd(*bill); err != nil {
		return errCode.ServerBusy.GetI18nMap(), nil
	} else {
		walletStorage.NotifyUserWallet(uid)
		agentStorage.OnWalletChange(uid)
	}
	return errCode.Success(nil).GetI18nMap(), nil
}
func (s *Pay) DealProtocolFormat(in interface{}, action string, gameType string, error *common.Err) []byte {
	info := struct {
		Data     interface{}
		GameType string
		Action   string
		ErrMsg   string
		Code     int
	}{
		Data:     in,
		GameType: gameType,
		Action:   action,
	}
	if error == nil {
		info.Code = 0
		info.ErrMsg = "操作成功"
	} else {
		info.Code = error.Code
		info.ErrMsg = error.SetKey().ErrMsg
	}
	ret, _ := json.Marshal(info)
	return ret
}
func (s *Pay) giftCode(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"code"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetI18nMap(), ok
	}
	uid := session.GetUserID()
	code := params["code"].(string)
	if code == "" {
		return errCode.ErrParams.SetKey("code").GetI18nMap(), nil
	}
	common.AddQueueByTag(uid, func() {
		chargeCode := payStorage.QueryChargeCode(code)
		sb := gate2.QuerySessionBean(uid)
		if chargeCode == nil {
			if sb == nil {
				return
			}
			ret := s.DealProtocolFormat(nil, actionGiftCode, "pay", errCode.GiftCodeErr)
			s.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
			return
		}
		if chargeCode.Status != 0 {
			if sb == nil {
				return
			}
			ret := s.DealProtocolFormat(nil, actionGiftCode, "pay", errCode.GiftCodeUsed)
			s.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
			return
		}
		if chargeCode.Belong != "mline" {
			invite := agentStorage.QueryInvite(utils.ConvertOID(uid))
			if invite.Oid.IsZero() || invite.ParentOid.IsZero() {
				ret := s.DealProtocolFormat(nil, actionGiftCode, "pay", errCode.GiftCodeErr)
				s.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
				return
			} else {
				baba := userStorage.QueryUserId(invite.ParentOid)
				if chargeCode.Belong != baba.Account {
					ret := s.DealProtocolFormat(nil, actionGiftCode, "pay", errCode.GiftCodeErr)
					s.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
					return
				}
			}
		}
		chargeCode.Uid = uid
		chargeCode.Status = payStorage.StatusUsed
		if err := payStorage.UpdateChargeCode(chargeCode); err != nil {
			if sb == nil {
				return
			}
			ret := s.DealProtocolFormat(nil, actionGiftCode, "pay", errCode.ServerBusy)
			s.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
			return
		}
		bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventGiftCode,
			chargeCode.Code, int64(chargeCode.Amount))
		if err := walletStorage.OperateVndBalance(bill); err == nil {
			walletStorage.NotifyUserWallet(uid)
			agentStorage.OnActivityData(uid, int64(chargeCode.Amount))
			activityRewardBetTimes, _ := utils.ConvertInt(storage.QueryConf(storage.KActivityRewardBetTimes))
			douDouBet := int64(chargeCode.Amount) * activityRewardBetTimes
			userStorage.IncUserDouDouBet(utils.ConvertOID(uid), douDouBet)
			activityStorage.InsertActivityReceiveRecord(&activityStorage.ActivityRecord{
				Type:       activityStorage.GiftCode,
				ActivityID: "",
				Uid:        uid,
				Charge:     0,
				Get:        int64(chargeCode.Amount),
				BetTimes:   activityRewardBetTimes,
				UpdateAt:   time.Now(),
				CreateAt:   time.Now(),
			})
			userStorage.IncUserGiftCode(utils.ConvertOID(uid), int64(chargeCode.Amount))
		}
		if sb == nil {
			return
		}
		ret := s.DealProtocolFormat(nil, actionGiftCode, "pay", nil)
		s.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
		return
	})

	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
