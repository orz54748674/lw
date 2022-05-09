package lotteryStorage

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
)

var (
	cLottery = "lottery"
)

func getCLottery() *common.Collect {
	return common.GetMongoDB().C(cLottery)
}

func InitLottery() {
	count, _ := getCLottery().Find(bson.M{}).Count()
	if count > 0 {
		log.Info("lottery count:%d", count)
		return
	}
	lotteries, err := getLotteryByFile()
	if err != nil {
		log.Error("Init Lottery fail:%s", err.Error())
	}
	BatchAddLottery(lotteries)
	log.Info("init lottery count:%d", len(lotteries))
}

func getLotteryByFile() (lotteries []*Lottery, err error) {
	path := utils.GetProjectAbsPath()
	f, err := ioutil.ReadFile(filepath.Join(path, "bin/lottery.json"))
	if err != nil {
		log.Error("read fail: %v", err)
		return
	}

	if err = json.Unmarshal(f, &lotteries); err != nil {
		log.Error("parse lottery conf err: %s", err.Error())
		return
	}
	return
}

func BatchAddLottery(lotteries []*Lottery) {
	data := make([]interface{}, len(lotteries))
	for i, lottery := range lotteries {
		data[i] = lottery
	}
	if err := getCLottery().InsertMany(data); err != nil {
		log.Error("InsertMany lottery conf err:%s", err.Error())
	}
}

func GetOfficialLotteries() (lotteries []*Lottery, err error) {
	lotteries = []*Lottery{}
	query := bson.M{"LotteryType": OfficialLottery}
	if err := getCLottery().Find(query).All(&lotteries); err != nil {
		log.Error("GetOfficialLotteries fail:%s", err.Error())
	}
	return
}

func (m *Lottery) SetName() string {
	return cLottery
}
func (m *Lottery) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *Lottery) GetLotteries() (lotteries []*Lottery, err error) {
	lotteries = []*Lottery{}
	err = getCLottery().Find(bson.M{"Status": 0}).All(&lotteries)
	return
}

func (m *Lottery) GetLotteryInfo(lotteryCode string) (err error) {
	return m.C().Find(bson.M{"LotteryCode": lotteryCode}).One(m)
}
