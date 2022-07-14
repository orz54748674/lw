package payStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson/primitive"
)

type OrderInterface struct {
	CreateAt time.Time
	Type     string
	Method   string
	Amount   int64
	Status   string
	Remark   string
}

func GetOrderData(uid primitive.ObjectID, offset int, limit int) ([]OrderInterface, int64) {
	orderList, count := QueryOrderLog(uid, offset, limit)
	var orderData []OrderInterface
	for _, order := range orderList {
		sType := "Gift Code"
		payConf := QueryPayConf(order.MethodId)
		method := payConf.Name
		if payConf.Merchant == "Official" {
			sType = common.I18str("Bank")
			transfer := QueryOrderTransfer(order.Oid)
			companyBank := QueryCompanyBank(transfer.ReceiveId)
			if companyBank == nil {
				method = ""
			} else {
				method = companyBank.BankName
			}

		} else if payConf.Merchant == "AutoOfficial" {
			sType = common.I18str("NẠP NHANH")
		} else if payConf.Merchant == "VgPay" {
			sType = common.I18str("PayOnline")
		} else if payConf.Merchant == "NapTuDong" {
			sType = common.I18str("PhoneCharge")
		} else if payConf.Merchant == "customerService" {
			sType = common.I18str("CustomerService")
		}
		status := common.I18str("StrSuccess")
		if order.Status == DouDouStatusReject {
			status = common.I18str("Reject")
		} else if order.Status != StatusSuccess {
			status = common.I18str("Unpaid")
		}
		data := OrderInterface{
			CreateAt: order.CreateAt.Local(),
			Type:     sType,
			Method:   method,
			Amount:   order.Amount,
			Status:   status,
			Remark:   order.Remark,
		}
		orderData = append(orderData, data)
	}
	return orderData, count
}

func GetDouDouData(uid primitive.ObjectID, offset int, limit int) ([]OrderInterface, int64) {
	douDouList, count := QueryDouDouLog(uid, offset, limit)
	var douDouData []OrderInterface
	for _, doudou := range douDouList {
		sType := common.I18str("Bank")
		method := doudou.BtName
		status := common.I18str("StrSuccess")
		if doudou.Status == StatusInit {
			status = common.I18str("WaiteReview")
		} else if doudou.Status == DouDouStatusReject {
			status = common.I18str("Reject")
		}
		data := OrderInterface{
			CreateAt: doudou.CreateAt.Local(),
			Type:     sType,
			Method:   method,
			Amount:   doudou.Amount,
			Status:   status,
			Remark:   doudou.Remark,
		}
		douDouData = append(douDouData, data)
	}
	return douDouData, count
}
