package dataStorage

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"vn/common"
	"vn/common/utils"
)

type IpInfo struct {
	Ip          string    `gorm:"primary_key" json:"query"`
	Country     string    `json:"country"`
	CountryCode string    `json:"countryCode"`
	City        string    `json:"city"`
	Isp         string    `json:"isp"`
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	RegionName  string    `json:"regionName"`
	CreateAt    time.Time `json:"-"`
	UpdateAt    time.Time `json:"-"`
}

func (IpInfo) TableName() string {
	return "data_ip_info"
}

func (s *IpInfo) Save() {
	var ip IpInfo
	db := common.GetMysql().Model(&ip)
	db.Where("ip=?", s.Ip).First(&ip)
	if ip.Ip == "" {
		s.CreateAt = utils.Now()
		s.UpdateAt = utils.Now()
		common.GetMysql().Create(s)
	}else{
		s.UpdateAt = utils.Now()
		s.CreateAt = ip.CreateAt
		common.GetMysql().Updates(s)
	}
}
func (s *IpInfo) Create() {
	s.CreateAt = utils.Now()
	s.UpdateAt = utils.Now()
	common.GetMysql().Create(s)
}

func (s *IpInfo) Exists() bool {
	var ip IpInfo
	db := common.GetMysql().Model(&ip)
	db.Where("ip=?", s.Ip).First(&ip)
	if ip.Ip == "" {
		return false
	}else{
		return true
	}
}
func (s *IpInfo)RequestIpInfo() error {
	addr := fmt.Sprintf("https://pro.ip-api.com/json/%s?key=tITdaDWA7dqCBN7",s.Ip)
	//url := "http://pro.ip-api.com/json"
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if common.App.GetSettings().Settings["env"].(string) == "dev" {
		//proxy, _ := url.Parse("http://127.0.0.1:1080")
		//tr.Proxy = http.ProxyURL(proxy)
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 5, //超时时间
	}
	resp, err := client.Get(addr)
	if err != nil {
		//log.Error(err.Error())
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(body,s);err != nil{
		//log.Error(err.Error())
		return err
	}
	return nil
}
