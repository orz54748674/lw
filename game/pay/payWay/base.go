package payWay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"vn/common"
	"vn/common/utils"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/activity"
	"vn/gate"
	"vn/storage/agentStorage"
	"vn/storage/gameStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

type Pay interface {
}

func structToMap(i interface{}) (values url.Values) {
	values = url.Values{}
	iVal := reflect.ValueOf(i).Elem()
	typ := iVal.Type()
	for i := 0; i < iVal.NumField(); i++ {
		values.Set(strings.ToLower(typ.Field(i).Name), fmt.Sprint(iVal.Field(i)))
	}
	return
}

type Response struct {
	OrderNo   string
	TargetUrl string
	QrCode    string
}

func ToResponse(w http.ResponseWriter,content string) {
	if content != "success"{
		log.Error(content)
	}
	if _,err := w.Write([]byte(content));err != nil{
		log.Error(err.Error())
	}
}
func DealInfoFormat() map[string]interface{} {
	res := make(map[string]interface{})
	res["Code"] = 0
	res["Action"] = "HD_info"
	res["ErrMsg"] = "操作成功"
	res["GameType"] = "lobby"
	return res
}
//func ActivityDeal(order *payStorage.Order){
//	activityList := gameStorage.QueryActivityConfListAsc()
//	for _,activity := range activityList {
//		if order.Amount >= activity.Charge{
//			activityRecord := gameStorage.QueryActivityRecord(order.UserId.Hex(),activity.ActivityID.Hex())
//			if activityRecord == nil || activityRecord.Status == gameStorage.Undo{
//				record := gameStorage.ActivityRecord{
//					ActivityID: activity.ActivityID.Hex(),
//					Type: gameStorage.RechargeGift,
//					Uid:order.UserId.Hex(),
//					Charge: activity.Charge,
//					Get: activity.Get,
//					Status: gameStorage.Done,
//					UpdateAt: time.Now(),
//				}
//				if activityRecord == nil{
//					record.CreateAt = time.Now()
//					gameStorage.InsertActivityRecord(&record)
//				}else{
//					record.Oid = activityRecord.Oid
//					gameStorage.UpdateActivityRecord(&record)
//				}
//				break
//			}
//		}
//	}
//
//	uInfo := userStorage.QueryUserInfo(order.UserId)
//	activityFirstCharge := storage.QueryConf(storage.KActivityFirstCharge)
//	activityFirstChargePer,_ := utils.ConvertInt(storage.QueryConf(storage.KActivityFirstChargePer))
//	if activityFirstCharge == "1" && uInfo.HaveCharge == 0{ //首冲  冲多少送多少
//		bill := walletStorage.NewBill(order.UserId.Hex(),walletStorage.TypeIncome,
//			walletStorage.EventFirstCharge,order.Oid.Hex(),order.GotAmount * activityFirstChargePer / 100)
//		if err := walletStorage.OperateVndBalance(bill);err != nil{
//			return
//		}
//		activityRecord := &gameStorage.ActivityRecord{
//			Type: gameStorage.FirstRecharge,
//			ActivityID: "",
//			Uid: order.UserId.Hex(),
//			Charge: order.Amount,
//			Get: order.GotAmount * activityFirstChargePer / 100,
//			Status: gameStorage.Received,
//			UpdateAt: time.Now(),
//			CreateAt: time.Now(),
//		}
//		activityRewardBetTimes,_ := utils.ConvertInt(storage.QueryConf(storage.KActivityRewardBetTimes))
//		douDouBet := order.GotAmount * activityRewardBetTimes * (100 + activityFirstChargePer) / 100 - order.GotAmount //后面已经加了一倍流水
//		userStorage.IncUserDouDouBet(order.UserId, douDouBet)
//		agentStorage.OnActivityData(order.UserId.Hex(),activityRecord.Get)
//		gameStorage.InsertActivityRecord(activityRecord)
//		sessionBean := gate.QuerySessionBean(order.UserId.Hex())
//		if sessionBean !=nil{
//			session,err := basegate.NewSession(common.App, sessionBean.Session)
//			if err != nil{
//				log.Error(err.Error())
//			}else{
//				res := DealInfoFormat()
//				haveCharge := make(map[string]interface{},1)
//				haveCharge["haveCharge"] = true
//				res["Data"] = haveCharge
//				ret,_ := json.Marshal(res)
//				if err := session.SendNR(game.Push, ret);err != ""{
//					log.Error(err)
//				}
//			}
//		}
//	}
//}
func SuccessOrder(order *payStorage.Order)  {
	order.Status = payStorage.StatusSuccess
	order.UpdateAt = utils.Now()
	//ActivityDeal(order)
	go activity.NotifyDealChargeActivity(order)

	bill := walletStorage.NewBill(order.UserId.Hex(),walletStorage.TypeIncome,
		walletStorage.EventCharge,order.Oid.Hex(),order.GotAmount)
	if err := walletStorage.OperateVndBalance(bill);err != nil{
		return
	}
	payStorage.UpdateOrder(order)
	NotifyUserWallet(order.UserId.Hex())
	agentStorage.OnPayData(order.UserId.Hex(),order.GotAmount,0)
	userStorage.IncUserDouDouBet(order.UserId,order.GotAmount)
	userStorage.IncUserCharge(order.UserId, order.GotAmount)
	gameStorage.ChargeCalcProfitByUser(order.UserId.Hex(),order.GotAmount)

	uInfo := userStorage.QueryUserInfo(order.UserId)
	if uInfo.HaveCharge == 0{
		uInfo.HaveCharge = 1
		uInfo.FistChargeTime = utils.Now()
		userStorage.UpsertUserInfo(order.UserId,uInfo)
	}
}

func NotifyUserWallet(uid string)  {
	go func() {
		topic := game.Push
		wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
		msg := make(map[string]interface{})
		msg["Wallet"] = wallet
		msg["Action"] = "wallet"
		msg["GameType"] = game.All
		b, _ := json.Marshal(msg)
		sessionBean := gate.QuerySessionBean(uid)
		if sessionBean !=nil{
			session,err := basegate.NewSession(common.App, sessionBean.Session)
			if err != nil{
				log.Error(err.Error())
			}else{
				if err := session.SendNR(topic, b);err != ""{
					log.Error(err)
				}
			}
		}
	}()
}

