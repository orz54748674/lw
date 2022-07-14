package data

import (
	"errors"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/data/bean"
	"vn/storage/activityStorage"
	"vn/storage/dataStorage"
	"vn/storage/gameStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"

	"gorm.io/gorm"
)

type RunOverView struct {
}

var platform = []string{"h5_iOS", "h5_Android", "Android", "iOS", "pc_Windows", "pc_OS X"}
var platformValue = map[string][]string{}

func init() {
	platformValue["h5_iOS"] = []string{"h5_iOS", "pc_iOS"}
	platformValue["h5_Android"] = []string{"h5_Android", "pc_Linux"}
	platformValue["Android"] = []string{"Android"}
	platformValue["iOS"] = []string{"iOS"}
	platformValue["pc_Windows"] = []string{"pc_Windows"}
	platformValue["pc_OS X"] = []string{"pc_OS X"}
}
func (s *RunOverView) Test() {
	dayStart, err := utils.StrFormatTime("yyyy-MM-dd", "2021-07-02")
	if err != nil {
		log.Error(err.Error())
	}
	raw := &RunOverViewRaw{
		dayStart: dayStart,
		dayEnd:   utils.GetDayEndTimeByStart(dayStart),
		game:     "all",
		channel:  "all",
		platform: "all",
	}
	gameStart := raw.getGameStartPeople()
	dau := raw.getDau()

	log.Info("gameStart: %v,dau:%v", gameStart, dau)
}

var running = false

func (s *RunOverView) Start() {
	if running {
		log.Warning("data overview still running...")
		return
	}
	running = true
	_ = common.GetMysql().AutoMigrate(&bean.Overview{})
	err, lastTime := s.queryLastTime()
	if err != nil {
		log.Error(err.Error())
		return
	}
	dayStart, _ := utils.GetDayStartTime(lastTime)
	for (utils.Now().Unix() - dayStart.Unix()) > 0 {
		dayEnd := utils.GetDayEndTimeByStart(dayStart)
		now := time.Now()
		s.updateOneDay(dayStart, dayEnd)
		log.Info("runDataOverview dayStart: %v , dayEnd: %v spent:%v", dayStart, dayEnd, time.Now().Sub(now))
		dayStart = time.Unix(dayStart.Unix()+86400, 0)
	}
	running = false
}
func (s *RunOverView) queryAllChannel(dayStart, dayEnd time.Time) []string {
	db := common.GetMysql().Model(&dataStorage.DataStartLog{})
	start := time.Unix(dayStart.Unix()-86400*30, 0)
	db.Where("create_at BETWEEN ? AND ?", start, utils.Now())
	var channels []uidStruct
	db.Select("DISTINCT channel as uid").Find(&channels)
	res := make([]string, len(channels))
	for i, c := range channels {
		res[i] = c.Uid
	}
	return res
}
func (s *RunOverView) updateOneDay(dayStart, dayEnd time.Time) {
	allGame := game.GameList
	allGame = append(allGame, game.Lobby)
	allGame = append(allGame, game.All)
	//allChannel := gameStorage.QueryChannels()
	allChannel := s.queryAllChannel(dayStart, dayEnd)
	allChannel = append(allChannel, "all")
	//allChannel = []string{"all"}
	for _, c := range allChannel {
		for _, g := range allGame {
			s.updateOneData(dayStart, dayEnd, g, c, "all")
		}
		for _, p := range platform {
			s.updateOneData(dayStart, dayEnd, "all", c, p)
		}
	}
}
func (s *RunOverView) updateOneData(dayStart, dayEnd time.Time, game game.Type, channel, platform string) {
	raw := &RunOverViewRaw{
		dayStart: dayStart,
		dayEnd:   dayEnd,
		game:     game,
		channel:  channel,
		platform: platform,
	}
	data := raw.getData()
	raw.data = &data
	data.DeviceNew = raw.getDevieceNew()
	data.Dnu = raw.getDnu()
	data.Dau = raw.getDau()
	sumAmount, sumPeople := raw.getCharge()
	data.Charge = sumAmount
	data.ChargePeople = sumPeople
	sumNewAmount, sumNewPeople := raw.getNewCharge()
	data.NewCharge = sumNewAmount
	data.NewChargePeople = sumNewPeople
	douDouAmount, douDouPeople := raw.getDouDou()
	data.DouDou = douDouAmount
	data.DouDouPeople = douDouPeople
	data.FirstDayChargePeople = raw.getFirstDayChangePeople()
	if data.Dau != 0 {
		data.Pur = int(data.ChargePeople * 100 / data.Dau)
		data.Arpu = data.Charge / data.Dau
	}
	if data.ChargePeople != 0 {
		data.Arpp = data.Charge / data.ChargePeople
	}
	data.Pcu, data.Acu = raw.getPcuAcu()
	raw.parserUrr()
	gameCount, betAmount, income, playerCount, winRate := raw.getBetAmount()
	data.BetAmount = betAmount
	data.Income = income
	data.WinRate = winRate
	data.GamePlayPeople = playerCount
	data.GameCount = gameCount
	data.ActivityGive = raw.getActivityGive()
	data.GameStartPeople = raw.getGameStartPeople()
	data.Save()
}

func (RunOverView) queryLastTime() (error, time.Time) {
	db := common.GetMysql().Model(&bean.Overview{})
	var data bean.Overview
	db.Order("id desc").First(&data)
	if data.ID != 0 {
		return nil, data.UpdateAt
	}
	db2 := common.GetMysql().Model(&userStorage.User{})
	var user userStorage.User
	db2.First(&user)
	if user.ID != 0 {
		return nil, user.CreateAt
	}
	return errors.New("no data"), time.Now()
}

type RunOverViewRaw struct {
	dayStart time.Time
	dayEnd   time.Time
	game     game.Type
	channel  string
	platform string
	data     *bean.Overview
}
type uidStruct struct {
	Uid string
}

func (s *RunOverViewRaw) getDevieceNew() (deviceNew int64) {
	if s.game == "all" {
		db := common.GetMysql().Model(&dataStorage.DataStart{})
		if s.channel != "all" {
			db.Where("channel = ? ", s.channel)
		}
		if s.platform != "all" {
			db.Where("platform in ?", platformValue[s.platform])
		}
		db.Where("create_at BETWEEN ? AND ?", s.dayStart, s.dayEnd)
		var uids []uidStruct
		if err := db.Select("DISTINCT uuid").Find(&uids).Error; err != nil {
			log.Error(err.Error())
		}
		deviceNew = int64(len(uids))
	}
	return
}

func (s *RunOverViewRaw) getGameStartPeople() (GameStartPeople int64) {
	if s.game == "all" {
		db := common.GetMysql().Model(&dataStorage.DataStartLog{})
		if s.channel != "all" {
			db.Where("channel = ? ", s.channel)
		}
		if s.platform != "all" {
			db.Where("platform in ?", platformValue[s.platform])
		}
		db.Where("create_at BETWEEN ? AND ?", s.dayStart, s.dayEnd)
		var uids []uidStruct
		if err := db.Select("DISTINCT uuid").Find(&uids).Error; err != nil {
			log.Error(err.Error())
		}
		GameStartPeople = int64(len(uids))
	}
	return
}
func toConditionStr(arr []string) string {
	str := ""
	for _, s := range arr {
		if str != "" {
			str += ","
		}
		str += "'" + s + "'"
	}
	return str
}
func (s *RunOverViewRaw) getActivityGive() (activityGive int64) {
	if s.game == "all" {
		activityDb := common.GetMysql().Model(&activityStorage.ActivityRecord{})
		activityDb.Joins("left join user ON user.oid=activity_records.uid")
		if s.channel != "all" {
			activityDb.Where("user.channel = ?", s.channel)
		}
		if s.platform != "all" {
			activityDb.Where("user.platform IN ?", platformValue[s.platform])
		}
		activityDb.Where("user.type=1 AND activity_records.create_at BETWEEN ? AND ?",
			s.dayStart, s.dayEnd)
		var sum sumTmp
		if err := activityDb.Select("sum(activity_records.get) sum_amount").Find(&sum).Error; err != nil {
			log.Error(err.Error())
		}
		activityGive = sum.SumAmount
	}
	return
}
func (s *RunOverViewRaw) getGameStartCount() (gameStartCount int64) {
	if s.game == "all" {
		db := common.GetMysql().Model(&dataStorage.DataStartLog{})
		db.Where("create_at BETWEEN ? AND ?", s.dayStart, s.dayEnd)
		if s.channel != "all" {
			db.Where("channel=?", s.channel)
		}
		if s.platform != "all" {
			db.Where("platform in ?", platformValue[s.platform])
		}
		if err := db.Select("DISTINCT uuid").Count(&gameStartCount).Error; err != nil {
			log.Error(err.Error())
		}
	} else {
		gameDb := common.GetMysql().Model(&dataStorage.UserOnlinePage{})
		gameDb.Joins("left join user ON user.oid=data_game_start_log.uid")
		if s.channel != "all" {
			gameDb.Where("user.channel = ?", s.channel)
		}
		gameDb.Where("user.type=1 AND data_game_start_log.game_type=? AND data_game_start_log.create_at BETWEEN ? AND ?",
			s.game, s.dayStart, s.dayEnd)
		if err := gameDb.Select("DISTINCT uid").Count(&gameStartCount).Error; err != nil {
			log.Error(err.Error())
		}
	}
	return
}
func (s *RunOverViewRaw) getBetRecordDb() (tx *gorm.DB) {
	db := common.GetMysql().Model(&gameStorage.BetRecord{})
	if s.platform != "all" {
		db.Joins("LEFT JOIN user ON bet_records.uid=user.oid ")
		db.Where("user.platform in ?", platformValue[s.platform])
	}
	if s.channel != "all" {
		db.Where("bet_records.channel=?", s.channel)
	}
	if s.game != "all" {
		db.Where("bet_records.game_type=?", s.game)
	}
	db.Where("bet_records.create_at BETWEEN ? AND ? AND bet_records.user_type=1", s.dayStart, s.dayEnd)
	return db
}
func (s *RunOverViewRaw) getBetAmount() (gameCount int64, betAmount, income int64, playerCount, winRate int64) {
	db := s.getBetRecordDb()
	if err := db.Count(&gameCount).Error; err != nil {
		log.Error(err.Error())
	}
	winDb := s.getBetRecordDb()
	var winCount int64
	winDb.Where("income <0").Count(&winCount)
	if gameCount > 0 {
		winRate = winCount * 100 / gameCount
	}
	var sumBet sumTmp
	if err := db.Select("sum(bet_amount) sum_amount").First(&sumBet).Error; err != nil {
		log.Error(err.Error())
	}
	betAmount = sumBet.SumAmount
	var sumIncome sumTmp
	if err := db.Select("sum(income) sum_amount").First(&sumIncome).Error; err != nil {
		log.Error(err.Error())
	}
	income = sumIncome.SumAmount
	var uids []uidStruct
	db2 := s.getBetRecordDb()
	if err := db2.Select("DISTINCT uid").Find(&uids).Error; err != nil {
		log.Error(err.Error())
	}
	playerCount = int64(len(uids))
	//log.Info("gameCount:%v,winCount:%v,sumBetAmount:%v,playerCount:%v,income:%v",
	//	gameCount,winCount,betAmount,playerCount,income)
	return
}

func (s *RunOverViewRaw) urrGameNewUser(dayStart time.Time) []string {
	dayEnd := utils.GetDayEndTimeByStart(dayStart)
	db := common.GetMysql().Model(&dataStorage.DataGameStart{})
	if s.channel != "all" {
		db.Where("channel =?", s.channel)
	}
	db.Where("create_at BETWEEN ? AND ? AND user_type=1 AND is_new=1 AND game=?", dayStart, dayEnd, s.game)
	var gameStart []dataStorage.DataGameStart
	if err := db.Find(&gameStart).Error; err != nil {
		log.Error(err.Error())
	}
	uids := make([]string, len(gameStart))
	for i, g := range gameStart {
		uids[i] = g.Uid
	}
	return uids
}
func (s *RunOverViewRaw) urrRegisterUser(dayStart time.Time) []string {
	userDb := common.GetMysql().Model(&userStorage.User{})
	if s.channel != "all" {
		userDb.Where("channel =?", s.channel)
	}
	dayEnd := utils.GetDayEndTimeByStart(dayStart)
	userDb.Where("type=1 AND create_at BETWEEN ? AND ?", dayStart, dayEnd)
	var uids []uidStruct
	userDb.Select("DISTINCT oid as uid").Find(&uids)
	userIds := make([]string, len(uids))
	for i, id := range uids {
		userIds[i] = id.Uid
	}
	return userIds
}
func (s *RunOverViewRaw) getOldData(dayStart time.Time) bean.Overview {
	db := common.GetMysql().Model(&bean.Overview{})
	date := utils.GetCnDate(dayStart)

	var data bean.Overview
	db.Where("date = ? AND game = ? AND channel=? AND platform=?",
		date, s.game, s.channel, s.platform).First(&data)

	if !utils.IsToday(s.dayEnd) {
		data.UpdateAt = s.dayEnd
	} else {
		data.UpdateAt = utils.Now()
	}
	return data
}
func (s *RunOverViewRaw) doParserLoginUrr(day int64) (bean.Overview, int) {
	dayStart := time.Unix(s.dayStart.Unix()-86400*day, 0)
	//log.Info("doParserLoginUrr day: %v,dayStart:%v",day,dayStart)
	userIds := s.urrRegisterUser(dayStart)
	urr := s.urrLoginLog(userIds)
	data := s.getOldData(dayStart)
	//log.Info("day: %v ,urr:%v",day,urr)
	return data, urr
}
func (s *RunOverViewRaw) doParserGameUrr(day int64) (bean.Overview, int) {
	dayStart := time.Unix(s.dayStart.Unix()-86400*day, 0)
	userIds := s.urrGameNewUser(dayStart)
	urr := s.urrGameStartLog(userIds)
	data := s.getOldData(dayStart)
	return data, urr
}
func (s *RunOverViewRaw) parserUrr() {
	if s.game == "all" {
		data2, urr := s.doParserLoginUrr(1)
		if data2.ID != 0 {
			data2.Urr = urr
			data2.Save()
			// if s.platform == "all"{
			// 	log.Info("date:%v,channel:%v,game:all,platform:all,urr2:%v",
			// 		s.dayStart,s.channel,urr)
			// }
		}
		data3, urr := s.doParserLoginUrr(3)
		if data3.ID != 0 {
			data3.Urr3 = urr
			data3.Save()
		}
		data7, urr := s.doParserLoginUrr(7)
		if data7.ID != 0 {
			data7.Urr7 = urr
			data7.Save()
		}
	} else {
		data2, urr := s.doParserGameUrr(1)
		if data2.ID != 0 {
			data2.Urr = urr
			data2.Save()
		}
		data3, urr := s.doParserGameUrr(3)
		if data3.ID != 0 {
			data3.Urr3 = urr
			data3.Save()
		}
		data7, urr := s.doParserGameUrr(7)
		if data7.ID != 0 {
			data7.Urr7 = urr
			data7.Save()
		}
	}
}
func (s *RunOverViewRaw) urrGameStartLog(userIds []string) int {
	if len(userIds) == 0 {
		return 0
	}
	loginDb := common.GetMysql().Model(&dataStorage.UserOnlinePage{})
	loginDb.Joins("JOIN user ON user.oid=data_game_start_log.uid")
	if s.channel != "all" {
		loginDb.Where("user.channel=?", s.channel)
	}
	loginDb.Where("user.type=1 AND data_game_start_log.create_at BETWEEN ? AND ? AND uid in ?", s.dayStart, s.dayEnd, userIds)
	var uids []uidStruct
	loginDb.Select("DISTINCT uid").Find(&uids)
	return len(uids) * 100 / len(userIds)
}
func (s *RunOverViewRaw) urrLoginLog(userIds []string) int {
	if len(userIds) == 0 {
		return 0
	}
	loginDb := common.GetMysql().Model(&userStorage.LoginLog{})
	loginDb.Joins("JOIN user ON user.oid=user_login_log.uid")
	if s.channel != "all" {
		loginDb.Where("user.channel = ?", s.channel)
	}
	if s.platform != "all" {
		loginDb.Where("user_login_log.platform in ?", platformValue[s.platform])
	}
	loginDb.Where("user.type=1 AND user_login_log.create_at BETWEEN ? AND ? AND uid in ?", s.dayStart, s.dayEnd, userIds)
	var uids []uidStruct
	loginDb.Select("DISTINCT uid").Find(&uids)
	return len(uids) * 100 / len(userIds)
}
func (s *RunOverViewRaw) getPcuAcu() (pcu int64, acu int64) {
	if s.channel == "all" {
		db := common.GetMysql().Model(&GameOnlineLog{})
		var onlineLog GameOnlineLog
		db.Where("game=? AND create_at BETWEEN ? AND ?",
			s.game, s.dayStart, s.dayEnd).Order("online_people desc").
			First(&onlineLog)
		pcu = onlineLog.OnlinePeople
		db2 := common.GetMysql().Model(&GameOnlineLog{})
		db2.Where("game=? AND create_at BETWEEN ? AND ?",
			s.game, s.dayStart, s.dayEnd)
		var sum sumTmp
		db2.Select("sum(online_people) sum_amount").First(&sum)
		acu = sum.SumAmount / 288
	}
	return
}
func (s *RunOverViewRaw) getDouDou() (amount int64, people int64) {
	douDb := common.GetMysql().Model(&payStorage.DouDou{})
	if s.game == "all" {
		douDb.Joins("JOIN user ON user.oid=doudou.user_id")
		if s.channel != "all" {
			douDb.Where("user.channel = ?", s.channel)
		}
		if s.platform != "all" {
			douDb.Where("user.platform in ?", platformValue[s.platform])
		}
		douDb.Where("user.type=1 AND doudou.update_at between ? AND ? AND doudou.status=9", s.dayStart, s.dayEnd)
		var sum sumTmp
		if err := douDb.Select("sum(doudou.amount) sum_amount").Find(&sum).Error; err != nil {
			log.Error(err.Error())
		}
		amount = sum.SumAmount
		var orders []payStorage.DouDou
		if err := douDb.Select("DISTINCT doudou.user_id as user_id").Find(&orders).Error; err != nil {
			log.Error(err.Error())
		}
		people = int64(len(orders))
	}
	return
}
func (s *RunOverViewRaw) getFirstDayChangePeople() (people int64) {
	infoDb := common.GetMysql().Model(&userStorage.UserInfo{})
	if s.game == "all" {
		infoDb.Joins("JOIN user ON user.oid=user_info.oid")
		if s.channel != "all" {
			infoDb.Where("user.channel = ?", s.channel)
		}
		if s.platform != "all" {
			infoDb.Where("user.platform in ?", platformValue[s.platform])
		}
		infoDb.Where("user.type=1 AND user_info.fist_charge_time between ? AND ?", s.dayStart, s.dayEnd)
		//if s.data.Charge>0 {
		//	infoDb = infoDb.Debug()
		//	log.Info("channel:%v,date:%v,charge:%v",s.channel,s.dayStart,s.data.Charge)
		//}
		if err := infoDb.Count(&people).Error; err != nil {
			log.Error(err.Error())
		}
	} else {
	}
	return
}
func (s *RunOverViewRaw) getDnu() int64 {
	var count int64
	if s.game == "all" {
		userDb := common.GetMysql().Model(&userStorage.User{})
		userDb.Where("create_at BETWEEN ? AND ? AND type=1", s.dayStart, s.dayEnd)
		if s.channel != "all" {
			userDb.Where("channel = ?", s.channel)
		}
		if s.platform != "all" {
			userDb.Where("platform in ?", platformValue[s.platform])
		}
		if err := userDb.Count(&count).Error; err != nil {
			log.Error(err.Error())
		}
	} else {
		db := common.GetMysql().Model(&dataStorage.DataGameStart{})
		if s.channel != "all" {
			db.Where("channel =?", s.channel)
		}
		db.Where("create_at BETWEEN ? AND ? AND user_type=1 AND is_new=1 AND game=?", s.dayStart, s.dayEnd, s.game)
		if err := db.Count(&count).Error; err != nil {
			log.Error(err.Error())
		}
	}
	return count
}

type sumTmp struct {
	SumAmount int64
}

func (s *RunOverViewRaw) getCharge() (sumAmount int64, sumPeople int64) {
	if s.game == "all" {
		orderDb := common.GetMysql().Model(&payStorage.Order{})
		payConf := payStorage.QueryPayConfByMethodType("giftCode")
		if s.channel != "all" {
			orderDb.Where("user.channel=?", s.channel)
		}
		if s.platform != "all" {
			orderDb.Where("user.platform IN ?", platformValue[s.platform])
		}
		orderDb.Joins("JOIN user ON user.oid=order.user_id")
		orderDb.Where("user.type=1 AND order.status=9 AND order.method_id <>? AND order.update_at BETWEEN ? AND ?",
			payConf.Oid.Hex(), s.dayStart, s.dayEnd)
		var sum sumTmp
		if err := orderDb.Select("sum(order.got_amount) sum_amount").Find(&sum).Error; err != nil {
			log.Error(err.Error())
		}
		sumAmount = sum.SumAmount
		var orders []payStorage.Order
		if err := orderDb.Select("DISTINCT order.user_id as user_id").Find(&orders).Error; err != nil {
			log.Error(err.Error())
		}
		sumPeople = int64(len(orders))
	}
	return
}
func (s *RunOverViewRaw) getNewCharge() (sumAmount int64, sumPeople int64) {
	if s.game == "all" {
		orderDb := common.GetMysql().Model(&payStorage.Order{})
		payConf := payStorage.QueryPayConfByMethodType("giftCode")
		if s.channel != "all" {
			orderDb.Where("user.channel=?", s.channel)
		}
		if s.platform != "all" {
			orderDb.Where("user.platform IN ?", platformValue[s.platform])
		}
		orderDb.Joins("JOIN user ON user.oid=order.user_id")
		orderDb.Where("user.create_at BETWEEN ? AND ? AND user.type=1 AND order.status=9 AND order.method_id <>? AND order.update_at BETWEEN ? AND ?",
			s.dayStart, s.dayEnd, payConf.Oid.Hex(), s.dayStart, s.dayEnd)
		var sum sumTmp
		if err := orderDb.Select("sum(order.got_amount) sum_amount").Find(&sum).Error; err != nil {
			log.Error(err.Error())
		}
		sumAmount = sum.SumAmount
		var orders []payStorage.Order
		if err := orderDb.Select("DISTINCT order.user_id as user_id").Find(&orders).Error; err != nil {
			log.Error(err.Error())
		}
		sumPeople = int64(len(orders))
	}
	return
}
func (s *RunOverViewRaw) getDau() int64 {
	var count int64
	loginDb := common.GetMysql().Model(&userStorage.LoginLog{})
	if s.game == "all" {
		if s.channel != "all" {
			loginDb.Where("user.channel=?", s.channel)
		}
		if s.platform != "all" {
			loginDb.Where("user.platform IN ?", platformValue[s.platform])
		}
		loginDb.Joins("JOIN user ON user.oid=user_login_log.uid")
		loginDb.Where("user.type=1 AND user_login_log.create_at BETWEEN ? AND ?", s.dayStart, s.dayEnd)
		//if err := loginDb.Count(&count).Error;err != nil{
		//	log.Error(err.Error())
		//}
		var uids []uidStruct
		loginDb.Select("DISTINCT uid").Find(&uids)
		count = int64(len(uids))
	} else {
		gameDb := common.GetMysql().Model(&dataStorage.UserOnlinePage{})
		gameDb.Joins("left join user ON user.oid=data_game_start_log.uid")
		if s.channel != "all" {
			gameDb.Where("user.channel=?", s.channel)
		}
		gameDb.Where("user.type=1 AND data_game_start_log.game_type=? AND data_game_start_log.create_at BETWEEN ? AND ?",
			s.game, s.dayStart, s.dayEnd)
		var uids []uidStruct
		gameDb.Select("DISTINCT uid").Find(&uids)
		count = int64(len(uids))
	}
	return count
}
func (s *RunOverViewRaw) getData() bean.Overview {
	db := common.GetMysql().Model(&bean.Overview{})
	date := utils.GetCnDate(s.dayStart)

	var data bean.Overview
	db.Where("date = ? AND game = ? AND channel=? AND platform=?",
		date, s.game, s.channel, s.platform).First(&data)
	if data.ID == 0 {
		data.Date = date
		data.Game = s.game
		data.Channel = s.channel
		data.Platform = s.platform
		data.CreateAt = utils.Now()
		data.UpdateAt = data.CreateAt
	}
	//if !utils.IsToday(s.dayEnd) {
	if utils.Now().Unix()-s.dayEnd.Unix() > 0 {
		data.UpdateAt = s.dayEnd
	} else {
		data.UpdateAt = utils.Now()
	}
	return data
}
