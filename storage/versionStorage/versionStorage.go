package versionStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
)

type Version struct {
	VersionName string    `bson:"VersionName"`
	VersionCode int       `bson:"VersionCode"`
	Msg         string    `bson:"Msg"`
	Platform    string    `bson:"Platform"`
	UrlPath     string    `bson:"UrlPath"`
	AppKey      string    `bson:"AppKey"`
	CreateAt    time.Time `bson:"CreateAt"`
	UpdateAt    time.Time `bson:"UpdateAt"`
}

var (
	PlatformAndroid = "android"
	PlatformIos     = "ios"

	AppKey               = "vnFirstPPdwckd"
	cVersion             = "version"
	cOfficialJackpotConf = "officialJackpotConf"
)

func InitVersion() {
	if check := Query(AppKey); len(*check) == 0 {
		insert(newVersion(PlatformAndroid))
		insert(newVersion(PlatformIos))
	}
	log.Info("InitVersion")
}
func newVersion(platform string) *Version {
	version := &Version{
		VersionCode: 1,
		VersionName: "1.0",
		Msg:         "发现新版本",
		Platform:    platform,
		UrlPath:     "https://google.com",
		AppKey:      AppKey,
		CreateAt:    utils.Now(),
		UpdateAt:    utils.Now(),
	}
	return version
}

func Query(appKey string) *[]Version {
	c := common.GetMongoDB().C(cVersion)
	var version []Version
	if err := c.Find(bson.M{"AppKey": appKey}).All(&version); err != nil {
		log.Error(err.Error())
		return nil
	}
	return &version
}
func insert(version *Version) {
	c := common.GetMongoDB().C(cVersion)
	if err := c.Insert(version); err != nil {
		log.Error(err.Error())
	}
}

type OfficialJackpotConf struct {
	Range        int64     `bson:"Range"`
	Start        int64     `bson:"Start"`
	IncPerSecond int       `bson:"IncPerSecond"`
	CreateAt     time.Time `bson:"CreateAt"`
}

func initOfficialJackpotConf() OfficialJackpotConf {
	conf := OfficialJackpotConf{
		Range:        450000000,
		Start:        432668922,
		IncPerSecond: 12,
		CreateAt:     utils.Now(),
	}
	c := common.GetMongoDB().C(cOfficialJackpotConf)
	if err := c.Insert(conf); err != nil {
		log.Error(err.Error())
	}
	return conf
}
func QueryOfficialJackpotConf() OfficialJackpotConf {
	c := common.GetMongoDB().C(cOfficialJackpotConf)
	var conf OfficialJackpotConf
	if err := c.Find(bson.M{}).One(&conf); err != nil {
		conf = initOfficialJackpotConf()
	}
	return conf
}

func QueryNewVersionByPlatform(platform string) (vSion *Version, err error) {
	c := common.GetMongoDB().C(cVersion)
	vSion = &Version{}
	find := bson.M{"Platform": platform}
	err = c.Find(find).Sort("-VersionCode").One(vSion)
	return
}
