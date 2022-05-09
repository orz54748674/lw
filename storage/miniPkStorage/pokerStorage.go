package miniPkStorage

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
	cPkPlay = "PokerPlay"
)

func InitPkPlay() {
	pk := &PkPlay{}
	count, _ := pk.C().Find(bson.M{}).Count()
	if count > 0 {
		log.Info("PkPlay count:%d", count)
		return
	}
	plays, err := getPkPlayByFile()
	if err != nil {
		log.Error("Init PkPlay fail:%s", err.Error())
	}

	data := make([]interface{}, len(plays))
	for i, p := range plays {
		data[i] = p
	}
	if err := pk.C().InsertMany(data); err != nil {
		log.Error("add PkPlay err:%v", err.Error())
		return
	}
	log.Info("init PkPlay count:%d", len(plays))
}

func getPkPlayByFile() (lotteries []*PkPlay, err error) {
	path := utils.GetProjectAbsPath()
	f, err := ioutil.ReadFile(filepath.Join(path, "bin/game/miniPoker.json"))
	if err != nil {
		log.Error("read fail: %v", err)
		return
	}

	if err = json.Unmarshal(f, &lotteries); err != nil {
		log.Error("parse miniPoker conf err: %s", err.Error())
		return
	}
	return
}

func (m *PkPlay) SetName() string {
	return cPkPlay
}
func (m *PkPlay) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *PkPlay) GetPlays() (res []*PkPlay, err error) {
	err = m.C().Find(bson.M{}).All(&res)
	return
}
