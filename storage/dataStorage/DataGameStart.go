package dataStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/storage/userStorage"
)

type UserOnlinePage struct {
	ID       uint64             `bson:"-" json:"-"`
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid" gorm:"-"`
	Uid      primitive.ObjectID `bson:"Uid"`
	GameType game.Type          `bson:"GameType"`
	CreateAt time.Time          `bson:"CreateAt"`
}

func (UserOnlinePage) TableName() string {
	return "data_game_start_log"
}

type DataGameStart struct {
	ID        uint64
	Date      string
	IsNew     int
	Channel   string
	Uid       string
	Account   string
	UserType  int8
	Game      game.Type
	OnlineSec int64
	CreateAt  time.Time
	UpdateAt  time.Time
}

func (DataGameStart) TableName() string {
	return "data_game_start"
}

func (d *DataGameStart) Save() {
	d.UpdateAt = utils.Now()
	if err := common.GetMysql().Save(d).Error; err != nil {
		log.Error(err.Error())
	}
}

func GetDataGameStart(user userStorage.User, game game.Type, date string) DataGameStart {
	db := common.GetMysql().Model(&DataGameStart{})
	uid := user.Oid.Hex()
	var data DataGameStart
	db.Where("date=? and uid =? and game=?", date, uid, game).
		First(&data)
	if data.ID == 0 {
		data = DataGameStart{
			Date:     date,
			Channel:  user.Channel,
			Uid:      user.Oid.Hex(),
			Game:     game,
			Account:  user.Account,
			UserType: user.Type,
			CreateAt: utils.Now(),
			UpdateAt: utils.Now(),
		}
		var d DataGameStart
		common.GetMysql().Model(&DataGameStart{}).
			Where("uid =? and game=?", uid, game).
			First(&d)
		if d.ID == 0 {
			data.IsNew = 1
		}
	}
	return data
}
