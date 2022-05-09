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
	cLotteryPlay = "lotteryPlay"
)

func InitLotteryPlay() {
	log.Info("InitLotteryPlay start")
	play := &LotteryPlay{}
	count, _ := play.C().Find(bson.M{}).Count()
	if count > 0 {
		log.Info("lotteryPlay count:%d", count)
		return
	}
	plays, err := getLotteryPlayByFile()
	if err != nil {
		log.Error("Init lotteryPlay fail:%s", err.Error())
		return
	}
	err = play.AddBatch(plays)
	if err != nil {
		log.Error("Init lotteryPlay add Data err:%s", err.Error())
		return
	}
	log.Info("init lottery count:%d", len(plays))
}

func getLotteryPlayByFile() (plays []*LotteryPlay, err error) {
	path := utils.GetProjectAbsPath()
	f, err := ioutil.ReadFile(filepath.Join(path, "bin/lotteryPlay.json"))
	if err != nil {
		return
	}
	err = json.Unmarshal(f, &plays)
	return
}

func (m *LotteryPlay) AddBatch(plays []*LotteryPlay) error {
	data := make([]interface{}, len(plays))
	for i, play := range plays {
		data[i] = play
	}
	return m.C().InsertMany(data)
}
func (m *LotteryPlay) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *LotteryPlay) SetName() string {
	return cLotteryPlay
}

func (m *LotteryPlay) GetLotteryPlays() (plays []*LotteryPlay, err error) {
	plays = []*LotteryPlay{}
	err = m.C().Find(bson.M{}).All(&plays)
	return
}

func (m *LotteryPlay) GetPlayInfo(areaCode, playCode, subPlayCode string) (err error) {
	return m.C().Find(bson.M{
		"AreaCode":    areaCode,
		"PlayCode":    playCode,
		"SubPlayCode": subPlayCode,
	}).One(m)
}
