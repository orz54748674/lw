package payStorage

import (
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
)

func initPhoneChargeConf() {
	c := common.GetMongoDB().C(cPhoneChargeConf)
	if count, _ := c.Find(bson.M{}).Count(); count == 0 {
		insertPhoneChargeConf("VIETTEL", 260, 10000)
		insertPhoneChargeConf("VIETTEL", 260, 20000)
		insertPhoneChargeConf("VIETTEL", 260, 30000)
		insertPhoneChargeConf("VIETTEL", 240, 50000)
		insertPhoneChargeConf("VIETTEL", 240, 100000)
		insertPhoneChargeConf("VIETTEL", 240, 200000)
		insertPhoneChargeConf("VIETTEL", 240, 300000)
		insertPhoneChargeConf("VIETTEL", 270, 500000)
		insertPhoneChargeConf("VIETTEL", 270, 1000000)

		insertPhoneChargeConf("VINAPHONE", 220, 10000)
		insertPhoneChargeConf("VINAPHONE", 220, 20000)
		insertPhoneChargeConf("VINAPHONE", 220, 30000)
		insertPhoneChargeConf("VINAPHONE", 220, 50000)
		insertPhoneChargeConf("VINAPHONE", 220, 100000)
		insertPhoneChargeConf("VINAPHONE", 220, 200000)
		insertPhoneChargeConf("VINAPHONE", 220, 300000)
		insertPhoneChargeConf("VINAPHONE", 220, 500000)
		insertPhoneChargeConf("VINAPHONE", 220, 1000000)

		insertPhoneChargeConf("MOBIFONE", 250, 10000)
		insertPhoneChargeConf("MOBIFONE", 250, 20000)
		insertPhoneChargeConf("MOBIFONE", 250, 30000)
		insertPhoneChargeConf("MOBIFONE", 250, 50000)
		insertPhoneChargeConf("MOBIFONE", 250, 100000)
		insertPhoneChargeConf("MOBIFONE", 250, 200000)
		insertPhoneChargeConf("MOBIFONE", 250, 300000)
		insertPhoneChargeConf("MOBIFONE", 250, 500000)
		insertPhoneChargeConf("MOBIFONE", 250, 1000000)

		insertPhoneChargeConf("VINAPHONE", 220, 10000)
		insertPhoneChargeConf("VINAPHONE", 220, 20000)
		insertPhoneChargeConf("VINAPHONE", 220, 30000)
		insertPhoneChargeConf("VINAPHONE", 220, 50000)
		insertPhoneChargeConf("VINAPHONE", 220, 100000)
		insertPhoneChargeConf("VINAPHONE", 220, 200000)
		insertPhoneChargeConf("VINAPHONE", 220, 300000)
		insertPhoneChargeConf("VINAPHONE", 220, 500000)
		insertPhoneChargeConf("VINAPHONE", 220, 1000000)
	}
}

func insertPhoneChargeConf(name string, feePerThousand int, amount int) {
	c := common.GetMongoDB().C(cPhoneChargeConf)
	phoneChargeConf := &PhoneChargeConf{
		Name:           name,
		FeePerThousand: feePerThousand,
		Amount:         amount,
	}
	if err := c.Insert(phoneChargeConf); err != nil {
		log.Error(err.Error())
	}
}

func QueryPhoneChargeConf(name string, amount int) *PhoneChargeConf {
	c := common.GetMongoDB().C(cPhoneChargeConf)
	query := bson.M{"Name": name, "Amount": amount}
	var phoneChargeConf PhoneChargeConf
	if err := c.Find(query).One(&phoneChargeConf); err != nil {
		log.Error(err.Error())
		return nil
	}
	return &phoneChargeConf
}

func QueryAllPhoneChargeConf() []PhoneChargeConf {
	c := common.GetMongoDB().C(cPhoneChargeConf)
	var phoneChargeConf []PhoneChargeConf
	if err := c.Find(bson.M{}).All(&phoneChargeConf); err != nil {
		log.Error(err.Error())
	}
	return phoneChargeConf
}
