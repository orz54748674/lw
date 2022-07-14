package userStorage

import (
	"fmt"
	"math/rand"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
)

type User struct {
	ID                uint64             `bson:"-" json:"-"`
	Oid               primitive.ObjectID `bson:"_id,omitempty" json:"Oid" gorm:"unique"`
	ShowId            int64              `bson:"ShowId"`
	Account           string             `bson:"Account"`
	NickName          string             `bson:"NickName"`
	Avatar            string             `bson:"Avatar"`
	Area              int64              `bson:"Area"`
	Phone             int64              `bson:"Phone"`
	Type              int8               `bson:"Type" gorm:"index:,type:hash"`
	Channel           string             `bson:"Channel" gorm:"type:varchar(32);index:,type:hash"`
	Platform          string             `bson:"Platform" gorm:"type:varchar(32);index:,type:hash"`
	Remark            string             `bson:"Remark"`
	ProfitPerThousand int                `bson:"ProfitPerThousand"`
	ProfitType        int                `bson:"ProfitType"`   //返佣类型 0默认 1管理员编辑
	Status            int                `bson:"Status"`       //0 正常，1 禁止登录
	RegisterIp        string             `bson:"RegisterIp"`   //
	RegisterUuid      string             `bson:"RegisterUuid"` //
	CreateAt          time.Time          `bson:"CreateAt" gorm:"index:,type:btree"`
	UpdateAt          time.Time          `bson:"UpdateAt"`
}

var (
	StatusNormal = 0
	StatusBlack  = 1
)

func (User) TableName() string {
	return "user"
}

type LoginLog struct {
	//common.MysqlBean
	ID       int64     `bson:"-" json:"-"`
	Uid      string    `bson:"Uid"`
	Platform string    `bson:"Platform"`
	Ip       string    `bson:"Ip"`
	Uuid     string    `bson:"Uuid"`
	UuidWeb  string    `bson:"UuidWeb"`
	Ua       string    `bson:"Ua"`
	CreateAt time.Time `bson:"CreateAt" gorm:"index"`
}

func (LoginLog) TableName() string {
	return "user_login_log"
}

type Login struct {
	//common.MysqlBean
	ID           int64              `bson:"-" json:"-"`
	Oid          primitive.ObjectID `bson:"_id,omitempty" json:"Oid" gorm:"unique"`
	Password     string             `bson:"Password" json:"Password"`
	LastPlatform string             `bson:"LastPlatform" json:"LastPlatform"`
	LastTime     time.Time          `bson:"LastTime" json:"LastTime"`
	LastIp       string             `bson:"LastIp" json:"LastIp"`
	Uuid         string             `bson:"Uuid" json:"Uuid"`
	UuidWeb      string             `bson:"UuidWeb" json:"UuidWeb"`
}

func (Login) TableName() string {
	return "user_login"
}

var (
	TypeNormal        int8 = 1
	TypeCompanyPlay   int8 = 2 //陪玩，在后台设置
	TypeAgent         int8 = 3 //推广
	PlatformAndroid        = "android"
	PlatformIos            = "ios"
	PlatformAndroidH5      = "h5_android"
	PlatformIosH5          = "h5_ios"
	PlatformWeb            = "web"
	ProfitNormal           = 0 //按收益配置表计算收益
)

func NewUser(account string, password string, platform string,
	channel string, uuid string, uuidWeb string, ip string) (*User, *Login) {
	user := &User{
		Oid:          primitive.NewObjectID(),
		Account:      account,
		Type:         TypeNormal,
		CreateAt:     utils.Now(),
		UpdateAt:     utils.Now(),
		Channel:      channel,
		Platform:     platform,
		Avatar:       GetSystemAvatar(),
		RegisterIp:   ip,
		RegisterUuid: uuid,
	}
	login := &Login{
		Password:     password,
		LastPlatform: platform,
		LastTime:     time.Unix(0, 0),
		LastIp:       ip,
		Uuid:         uuid,
		UuidWeb:      uuidWeb,
	}
	return user, login
}
func GetSystemAvatar() string {
	//rand.Seed(time.Now().UnixNano())
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := utils.RandInt64(1, 53, r)
	return fmt.Sprintf("system_%d", id)
}

//func NewLogin(id primitive.ObjectID, password string, platform string, ip string) *Login {
//	login := &Login{
//		Oid:           id,
//		Password:     password,
//		LastPlatform: platform,
//		LastTime:     utils.Now(),
//		LastIp:       ip,
//	}
//	return login

//}

type Token struct {
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	AccessToken string             `bson:"AccessToken"`
	Ip          string             `bson:"Ip"`
	SessionId   string             `bson:"SessionId"`
	CreateAt    time.Time          `bson:"CreateAt"`
	UpdateTime  time.Time          `bson:"UpdateAt"`
}
type Test struct {
	AccessToken string
	Expire      time.Time
	CreateAt    time.Time
}

func NewToken(id primitive.ObjectID, ip string) *Token {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	token := &Token{
		Oid:         id,
		AccessToken: utils.RandomString(32, r),
		CreateAt:    utils.Now(),
		Ip:          ip,
	}
	return token
}

type UserInfo struct {
	ID              int64              `bson:"-" json:"-"`
	Oid             primitive.ObjectID `bson:"_id,omitempty" json:"Oid" gorm:"unique"`
	SumBet          int64              `bson:"SumBet"`
	SumBetCount     int64              `bson:"SumBetCount"`
	SumCharge       int64              `bson:"SumCharge"`
	SumDouDou       int64              `bson:"SumDouDou"`
	SumAgentBalance int64              `bson:"SumAgentBalance"`
	WinAndLost      int64              `bson:"WinAndLost"` //累计输赢
	DouDouBet       int64              `bson:"DouDouBet"`  //换豆豆需要的流水
	HaveCharge      int                `bson:"HaveCharge"`
	FistChargeTime  time.Time          `bson:"FistChargeTime"`
	SumOnlineSec    int64              `bson:"SumOnlineSec"`
	ActivityTotal   int64              `bson:"ActivityTotal"`
	GiftCode        int64              `bson:"GiftCode"`
	VipLevel        int                `bson:"VipLevel"`
	SafeStatus      int                `bson:"SafeStatus"` //保险箱状态 0未激活 1加锁 2解锁
}

func (UserInfo) TableName() string {
	return "user_info"
}
