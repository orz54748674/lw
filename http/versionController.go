package http

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
	"vn/storage"
	"vn/storage/versionStorage"
)

type VersionController struct {
	BaseController
}

func (s *VersionController) jackpot(w http.ResponseWriter, r *http.Request) {
	conf := versionStorage.QueryOfficialJackpotConf()
	res := map[string]interface{}{
		"conf":   conf,
		"create": conf.CreateAt.Unix(),
		"now":    time.Now().Unix(),
	}
	s.response(w, errCode.Success(res))
}
func (s *VersionController) conf(w http.ResponseWriter, r *http.Request) {
	//_ = r.ParseForm()
	//p := r.Form
	//if check,ok := utils.CheckParams(p,
	//	[]string{"keys"});ok != nil{
	//	s.response(w, errCode.ErrParams.SetKey(check));return
	//}
	customerService := storage.QueryCustomerInfo()
	s.response(w, errCode.Success(customerService))
}
func (s *VersionController) Get(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	p := r.Form
	if check, ok := utils.CheckParams(p,
		[]string{"app_key"}); ok != nil {
		s.response(w, errCode.ErrParams.SetKey(check))
		return
	}
	data := versionStorage.Query(p["app_key"][0])
	res := make(map[string]interface{})
	for _, version := range *data {
		if version.Platform == versionStorage.PlatformAndroid {
			res[versionStorage.PlatformAndroid] = version
		} else {
			res[versionStorage.PlatformIos] = version
		}
	}
	s.response(w, errCode.Success(res))
}

func (s *VersionController) test(w http.ResponseWriter, r *http.Request) {
	c := common.GetMongoDB().C("user")
	count, _ := c.Find(bson.M{}).Count()
	res := make(map[string]interface{})
	res["c"] = count
	s.response(w, errCode.Success(res))
}

func (s *VersionController) ToDownload(w http.ResponseWriter, r *http.Request) {
	log.Debug("Request %v", r.UserAgent())
	params := r.URL.Query()
	userAgent := r.UserAgent()
	mobileRe, _ := regexp.Compile("(?i:iPod|iPhone)")
	deviceType := "android"
	if len(mobileRe.FindString(userAgent)) > 0 {
		deviceType = "ios"
	}
	version, err := versionStorage.QueryNewVersionByPlatform(deviceType)
	url := params.Get("url")
	if err != nil {
		log.Error("QueryNewVersionByPlatform err:%s", err.Error())
	} else {
		url = version.UrlPath
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *VersionController) GetAppDownloadUrl(w http.ResponseWriter, r *http.Request) {
	log.Debug("GetAppDownloadUrl Request %v", r.UserAgent())

	params := r.URL.Query()
	platform := params.Get("platform")
	deviceType := "android"
	if platform == "1" {
		deviceType = "ios"
	}
	log.Debug("deviceType:%v", deviceType)
	version, err := versionStorage.QueryNewVersionByPlatform(deviceType)
	repData := map[string]interface{}{"code": 0, "url": ""}
	if err != nil {
		repData["code"] = -1
		log.Error("GetAppDownloadUrl QueryNewVersionByPlatform err:%s", err.Error())
	} else {
		repData["url"] = version.UrlPath
	}
	btData, err := json.Marshal(repData)
	if err != nil {
		repData["code"] = -1
		log.Error("GetAppDownloadUrl  json.Marshal err:%s", err.Error())
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型
	w.Header().Set("content-type", "application/json")             //返回数据格式是json
	w.Write(btData)
}
