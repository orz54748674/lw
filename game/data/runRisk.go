package data

import (
	"errors"
	"strings"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/game/data/bean"
	"vn/storage/gameStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"

	"github.com/jinzhu/gorm"
)

type RunRisk struct {
	dayStart      time.Time
	dayEnd        time.Time
	recentlyStart time.Time
	recentlyEnd   time.Time
	recentlyDays  int64
	uuids         []string
	ips           []string
	allAccount    []string
}

var runningRisk = false

func (s *RunRisk) Start() {
	if runningRisk {
		log.Warning("data risk still running...")
		return
	}
	runningRisk = true

	_ = common.GetMysql().AutoMigrate(&bean.Risk{})
	err, lastTime := s.queryLastTime()
	if err != nil {
		log.Error(err.Error())
		return
	}
	s.recentlyDays = 3
	s.dayStart, _ = utils.GetDayStartTime(lastTime)
	for (utils.Now().Unix() - s.dayStart.Unix()) > 0 {
		s.dayEnd = utils.GetDayEndTimeByStart(s.dayStart)
		now := time.Now()
		s.updateOneDay()
		log.Info("runRisk dayStart: %v , dayEnd: %v ,spent:%v", s.dayStart, s.dayEnd, time.Now().Sub(now))
		s.dayStart = time.Unix(s.dayStart.Unix()+86400, 0)
	}

	runningRisk = false
}

func (s *RunRisk) queryLastTime() (error, time.Time) {
	db := common.GetMysql().Model(&bean.Risk{})
	var data bean.Risk
	db.Order("id desc").First(&data)
	if data.ID != 0 {
		return nil, data.UpdateAt
	}
	db2 := common.GetMysql().Model(&userStorage.LoginLog{})
	var record userStorage.LoginLog
	db2.First(&record)
	if record.ID != 0 {
		return nil, record.CreateAt
	}
	return errors.New("no data"), time.Now()
}
func (s *RunRisk) Test() {
	_ = common.GetMysql().AutoMigrate(&bean.Risk{})
	lastTime, _ := utils.StrFormatTime("yyyy-MM-dd", "2021-06-21")
	s.dayStart, _ = utils.GetDayStartTime(lastTime)
	s.dayEnd = utils.GetDayEndTimeByStart(s.dayStart)
	s.recentlyDays = 3
	s.updateOneDay()
}

type strResult struct {
	Str   string
	Count int64
}

func (s *RunRisk) updateOneDay() {
	/* 1，查询这天登陆的设备，这天登陆的ip
	 */
	uuids, ips := s.getIPsWithUuid()
	s.recentlyEnd = time.Unix(s.dayStart.Unix()+86400, 0)
	s.recentlyStart = time.Unix(s.recentlyEnd.Unix()-86400*s.recentlyDays, 0)
	db := common.GetMysql().Model(&userStorage.Login{})
	db.Where("last_time BETWEEN ? AND ? AND uuid in ?", s.dayStart, s.dayEnd, uuids)
	db.Select("uuid str,count(oid) count").Group("uuid").Having("count>1")
	var result []strResult
	if err := db.Find(&result).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err.Error())
	}
	for _, r := range result {
		s.uuids = append(s.uuids, r.Str)
	}
	s.parserUuid()

	db2 := common.GetMysql().Model(&userStorage.Login{})
	db2.Where("last_time BETWEEN ? AND ? AND last_ip in ?", s.dayStart, s.dayEnd, ips)
	db2.Select("last_ip str,count(oid) count").Group("last_ip").Having("count>1")
	var result2 []strResult
	if err := db2.Find(&result2).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err.Error())
	}
	for _, r := range result2 {
		s.ips = append(s.ips, r.Str)
	}
	s.parserIps()
}

func (s *RunRisk) parserUuid() {
	for _, u := range s.uuids {
		db := common.GetMysql().Model(&userStorage.Login{})
		db.Joins("LEFT JOIN user ON user.oid=user_login.oid")
		db.Where("user_login.last_time BETWEEN ? AND ? AND user_login.uuid=? AND user.type=1",
			s.dayStart, s.dayEnd, u)
		var result []strResult
		db.Select("user_login.oid str").Find(&result)
		for i := 0; i < len(result); i++ {
			data := s.getData(result[i].Str)
			s.parserUuidData(&data)
			s.parserData(&data)
			s.queryRecentlyAccount(&data)
			data.Save()
		}
	}
}
func (s *RunRisk) parserIps() {
	for _, ip := range s.ips {
		db := common.GetMysql().Model(&userStorage.Login{})
		db.Joins("LEFT JOIN user ON user.oid=user_login.oid")
		db.Where("user_login.last_time BETWEEN ? AND ? AND user_login.last_ip=? AND user.type=1",
			s.dayStart, s.dayEnd, ip)
		var result []strResult
		db.Select("user_login.oid str").Find(&result)
		for i := 0; i < len(result); i++ {
			data := s.getData(result[i].Str)
			s.parserIpsData(&data)
			s.parserData(&data)
			s.queryRecentlyAccount(&data)
			data.Save()
		}
	}
}
func (s *RunRisk) queryRecentlyAccount(data *bean.Risk) {
	deviceAccount := strings.Split(data.DeviceAccount, ",")
	ipsAccount := strings.Split(data.IpAccount, ",")
	allAccount := s.allAccount
	for _, s := range deviceAccount {
		if !utils.IsContainStr(allAccount, s) {
			allAccount = append(allAccount, s)
		}
	}
	for _, s := range ipsAccount {
		if !utils.IsContainStr(allAccount, s) {
			allAccount = append(allAccount, s)
		}
	}
	if len(allAccount) == 0 {
		return
	}
	uids, uMap := queryAllUidByAccount(allAccount)
	db := common.GetMysql().Model(&userStorage.Login{})
	db.Where("last_time BETWEEN ? AND ? AND oid IN ?",
		s.recentlyStart, s.recentlyEnd, uids)
	var result []strResult
	db.Select("DISTINCT oid str").Find(&result)
	loginAccount := make([]string, 0)
	loginUid := make([]string, 0)
	for _, r := range result {
		loginAccount = append(loginAccount, uMap[r.Str])
		loginUid = append(loginUid, r.Str)
	}
	data.RecentlyLoginAccount = strings.Join(loginAccount, ",")
	douDb := common.GetMysql().Model(&payStorage.DouDou{})
	douDb.Where("update_at BETWEEN ? AND ? AND status=9 AND user_id IN ?",
		s.recentlyStart, s.recentlyEnd, loginUid)
	var dou []strResult
	douDb.Select("DISTINCT user_id str").Find(&dou)
	douAccount := make([]string, 0)
	for _, d := range dou {
		douAccount = append(douAccount, uMap[d.Str])
	}
	data.RecentlyDouDouAccount = strings.Join(douAccount, ",")
}

func (s *RunRisk) parserIpsData(data *bean.Risk) {
	db := common.GetMysql().Model(&userStorage.Login{})
	db.Joins("LEFT JOIN user ON user.oid=user_login.oid").
		Where("user_login.last_ip=?", data.Ip)
	var result []strResult
	if err := db.Select("user.account str").Find(&result).Error; err != nil {
		log.Error(err.Error())
	}
	data.IpAccountCount = len(result)
	data.IpAccount = s.parserResult(data.IpAccount, result)
}
func (s *RunRisk) parserUuidData(data *bean.Risk) {
	db := common.GetMysql().Model(&userStorage.Login{})
	db.Joins("LEFT JOIN user ON user.oid=user_login.oid").
		Where("user_login.uuid=?", data.Uuid)
	var result []strResult
	if err := db.Select("user.account str").Find(&result).Error; err != nil {
		log.Error(err.Error())
	}
	data.DeviceAccountCount = len(result)
	data.DeviceAccount = s.parserResult(data.DeviceAccount, result)

}
func (s *RunRisk) parserData(data *bean.Risk) {
	if utils.IsContainStr(s.allAccount, data.Account) {
		return
	}
	db := common.GetMysql().Model(&gameStorage.BetRecord{})
	db.Where("uid=? AND create_at BETWEEN ? AND ?", data.Uid, s.recentlyStart, s.recentlyEnd)
	var betCount int64
	var count []strResult
	db.Select("DISTINCT game_type as str").Find(&count)
	db.Select("id").Count(&betCount)
	var result strResult
	db.Select("sum(`bet_amount`) count").Find(&result)
	data.RecentlyBetAmount = result.Count
	data.RecentlyBetCount = betCount
	data.RecentlyGameCount = int64(len(count))

	orderDb := common.GetMysql().Model(&payStorage.Order{})
	payConf := payStorage.QueryPayConfByMethodType("giftCode")
	orderDb.Where("method_id<>? AND update_at BETWEEN ? AND ? AND user_id=? AND status=9",
		payConf.Oid.Hex(), s.recentlyStart, s.recentlyEnd, data.Uid)
	var chargeCount int64
	orderDb.Count(&chargeCount)
	var ca strResult
	orderDb.Select("sum(`got_amount`) count").Find(&ca)
	data.RecentlyChargeAmount = ca.Count
	data.RecentlyChargeCount = chargeCount

	douDb := common.GetMysql().Model(&payStorage.DouDou{})
	douDb.Where("user_id=? AND status=9 AND update_at BETWEEN ? AND ?",
		data.Uid, s.recentlyStart, s.recentlyEnd)
	var douCount int64
	douDb.Count(&douCount)
	var dou strResult
	douDb.Select("sum(`amount`) count").Find(&dou)
	data.RecentlyDouDouCount = douCount
	data.RecentlyDouDouAmount = dou.Count
}

func (s *RunRisk) parserResult(account string, result []strResult) string {
	res := strings.Split(account, ",")
	for _, r := range result {
		if !utils.IsContainStr(res, r.Str) {
			res = append(res, r.Str)
		}
	}
	return strings.Join(res, ",")
}

func (s *RunRisk) getData(uid string) bean.Risk {
	date := utils.GetCnDate(s.dayStart)
	db := common.GetMysql().Model(&bean.Risk{})
	db.Where("date=? AND uid=?", date, uid)
	var data bean.Risk
	db.First(&data)
	if data.ID == 0 {
		user := queryUserOid(uid)
		login := queryUserLogin(uid)
		parent := queryParent(uid)
		data.Date = date
		data.RegisterTime = user.CreateAt
		data.Uid = uid
		data.Uuid = login.Uuid
		data.Ip = login.LastIp
		data.Account = user.Account
		data.ParentAccount = parent.Account
		data.CreateAt = utils.Now()
		data.UpdateAt = data.CreateAt
	}
	if utils.Now().Unix()-s.dayEnd.Unix() > 0 {
		data.UpdateAt = s.dayEnd
	} else {
		data.UpdateAt = utils.Now()
	}
	return data
}

func (s *RunRisk) getIPsWithUuid() ([]string, []string) {
	db := common.GetMysql().Model(&userStorage.Login{})
	var uids []uidStruct
	db.Select("DISTINCT uuid as uid").
		Where("last_time BETWEEN ? AND ?", s.dayStart, s.dayEnd)
	if err := db.Find(&uids).Error; err != nil {
		log.Error(err.Error())
	}
	uuids := make([]string, len(uids))
	for i, uid := range uids {
		uuids[i] = uid.Uid
	}
	db2 := common.GetMysql().Model(&userStorage.Login{})
	var ipResult []uidStruct
	db2.Select("DISTINCT last_ip as uid").
		Where("last_time BETWEEN ? AND ?", s.dayStart, s.dayEnd)
	if err := db2.Find(&ipResult).Error; err != nil {
		log.Error(err.Error())
	}
	ips := make([]string, len(ipResult))
	for i, r := range ipResult {
		ips[i] = r.Uid
	}
	return uuids, ips
}
