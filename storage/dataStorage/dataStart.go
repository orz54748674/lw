package dataStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/log"
)

type DataStart struct {
	ID               uint64
	Uuid             string `gorm:"unique"`
	UuidWeb          string
	Channel          string
	UserAgent        string
	Platform         string
	Brand            string
	Model            string
	SystemVersion    string
	Language         string
	CellularProvider string
	Ip               string
	IsRoot           int64
	CreateAt         time.Time
}

func (DataStart) TableName() string {
	return "data_start"
}

func (d *DataStart) Save() {
	db := common.GetMysql().Model(&DataStart{})
	if err := db.FirstOrCreate(d, DataStart{Uuid: d.Uuid}).Error; err != nil {
		log.Error("insert db err: %v", err.Error())
	}
	dataLog := &DataStartLog{
		Uuid:     d.Uuid,
		UuidWeb:  d.UuidWeb,
		Channel:  d.Channel,
		Platform: d.Platform,
		CreateAt: utils.Now(),
	}
	common.GetMysql().Create(dataLog)
}

type DataStartLog struct {
	ID       uint64
	Uuid     string
	UuidWeb  string
	Channel  string
	Platform string
	CreateAt time.Time
}

func (DataStartLog) TableName() string {
	return "data_start_log"
}
