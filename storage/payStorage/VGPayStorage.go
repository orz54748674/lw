package payStorage

import (
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/log"
)

type VGBank struct {
	MethodType string `bson:"MethodType"`
	Identifier string `bson:"Identifier"`
	Name       string `bson:"Name"`
}

var (
	cVGPayConf = "vgPayBankConf"
)

func InitVGPay() {
	c := common.GetMongoDB().C(cVGPayConf)
	count, err := c.Find(bson.M{}).Count()
	if err != nil {
		log.Error(err.Error())
	}
	if count == 0 {
		initVGBankList()
	}
}
func initVGBankList() {
	bankList := []VGBank{
		{MethodType: "bankQr", Identifier: "ACB", Name: "ACB"},
		{MethodType: "bankQr", Identifier: "VCB", Name: "Vietcombank"},
		{MethodType: "bankQr", Identifier: "BIDV", Name: "BIDV BANK"},
		{MethodType: "bankQr", Identifier: "CTG", Name: "VIETINBANK"},
		{MethodType: "bankQr", Identifier: "TPB", Name: "TIENPHONG BANK"},
		{MethodType: "bankQr", Identifier: "VPB", Name: "VP BANK"},
		{MethodType: "bankQr", Identifier: "AGR", Name: "AGRIBANK"},
		{MethodType: "bankQr", Identifier: "NAB", Name: "NAMA BANK"},

		{MethodType: "direct", Identifier: "TCB", Name: "Techcombank"},
		{MethodType: "direct", Identifier: "ACB", Name: "ACB"},
		{MethodType: "direct", Identifier: "VCB", Name: "Vietcombank"},
		{MethodType: "direct", Identifier: "MBB", Name: "MB BANK"},
		{MethodType: "direct", Identifier: "VAB", Name: "VIETA BANK"},
		{MethodType: "direct", Identifier: "HD", Name: "HD BANK"},
		{MethodType: "direct", Identifier: "BIDV", Name: "BIDV"},
		{MethodType: "direct", Identifier: "STB", Name: "Sacombank"},
		{MethodType: "direct", Identifier: "EXIM", Name: "Eximbank"},
		{MethodType: "direct", Identifier: "CTG", Name: "Vietin BANK"},
		{MethodType: "direct", Identifier: "VIB", Name: "VIB BANK"},
	}
	for _, bank := range bankList {
		insertVGBank(bank)
	}
}
func insertVGBank(bank VGBank) {
	c := common.GetMongoDB().C(cVGPayConf)
	if err := c.Insert(&bank); err != nil {
		log.Error(err.Error())
	}
}

func QueryVGBankList() []map[string]interface{} {
	c := common.GetMongoDB().C(cVGPayConf)
	pipe := mongo.Pipeline{
		{{"$group", bson.M{"_id": "$MethodType", "bankList": bson.M{"$push": "$$ROOT"}}}},
	}
	var bankList []map[string]interface{}
	if err := c.Pipe(pipe).All(&bankList); err != nil {
		log.Error(err.Error())
	}
	return bankList
}
