package reptile

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
	"vn/common/utils"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/storage/lotteryStorage"

	cron "github.com/robfig/cron/v3"
)

const (
	lotteryStartDate = "2020-01-01"
)

type Reptile struct {
	basemodule.BaseModule
	cron           *cron.Cron
	EntryMap       map[string]cron.EntryID
	ConcurrencyMap map[string]chan bool
	Concurrency    chan bool
}

var Module = func() module.Module {
	return new(Reptile)
}

func (m *Reptile) Version() string {
	return "1.0.0"
}

func (m *Reptile) GetType() string {
	return "reptile"
}
func (m *Reptile) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)
	m.EntryMap = make(map[string]cron.EntryID)
	m.ConcurrencyMap = make(map[string]chan bool)
	m.Concurrency = make(chan bool, 1)
	m.runTasks()
}

func (m *Reptile) runTasks() {

	lotteries, _ := lotteryStorage.GetOfficialLotteries()

	if len(lotteries) == 0 {
		return
	}

	m.cron = cron.New(cron.WithSeconds())
	for _, lottery := range lotteries {
		t, err := utils.StrToCnTime("2006-01-02 " + lottery.OpenTime)
		if err != nil {
			log.Error("%s open time format err:%s", lottery.LotteryName, err.Error())
			continue
		}
		hour := t.Hour()
		minute := t.Minute()
		second := t.Second()
		spec := fmt.Sprintf("%v %v %v %v %v %v", second, minute, hour, "*", "*", int(lottery.WeekNumber))
		// spec = "0/5 * * * * *"
		//spec = fmt.Sprintf("%v %v %v %v %v %v", 0, 45, 9, "*", "*", 4)
		//log.Debug("crontab %v  =>%s", lottery.LotteryCode, spec)
		m.ConcurrencyMap[lottery.LotteryCode] = make(chan bool, 1)
		go m.checkHistoryRecord(lottery)
		task := &Task{Lottery: lottery, M: m}
		entryID, err := m.cron.AddJob(spec, task)
		if err != nil {
			log.Error("add collect job:%s err:%s", lottery.LotteryName, err.Error())
			continue
		}
		m.EntryMap[lottery.Oid.String()] = entryID
	}
	m.cron.Start()
}

func (m *Reptile) Run(closeSig chan bool) {
	log.Info("%v 模块运行中...", m.GetType())
	<-closeSig
	log.Info("%v 模块已停止...", m.GetType())
}

func (m *Reptile) OnDestroy() {
	//一定别忘了继承
	m.BaseModule.OnDestroy()
	log.Info("%v 模块已回收...", m.GetType())
	m.cron.Stop()
}

func (m *Reptile) checkHistoryRecord(lottery *lotteryStorage.Lottery) {
	dateMap := make(map[string]string)
	startTime, err := utils.StrDateToTime(lotteryStartDate)
	if err != nil {
		log.Error("checkHistoryRecord StrToTime err:%v", err.Error())
		return
	}
	nowCnDate := utils.GetCnDate(time.Now())
	for {
		startCnDate := utils.GetCnDate(startTime)
		if nowCnDate <= startCnDate {
			break
		}
		if startTime.Weekday() == lottery.WeekNumber {
			dateMap[startCnDate] = utils.GetDate(startTime)
		}
		startTime = startTime.Add(time.Hour * 24)
	}

	record := &lotteryStorage.LotteryRecord{}
	res, err := record.GetRecords(lotteryStartDate, utils.NowDate(), lottery.LotteryCode)
	if err != nil {
		log.Error("checkHistoryRecord  %v get DB data err:%s", err.Error())
		return
	}
	for _, v := range res {
		delete(dateMap, v.CnNumber)
	}
	d, err := NewDownLoader(lottery.CollectUrl, lottery.LotteryName, lottery.AreaCode)
	if err != nil {
		log.Error("%v create Downloader err:%s", lottery.LotteryName, err.Error())
		return
	}
	dateUrl := strings.Replace(lottery.CollectUrl, "{city}", lottery.CityCode, 1)
	for cnNumber, dateKey := range dateMap {
		now := time.Now()
		openTime, _ := utils.StrToCnTime(fmt.Sprintf("%s %s", cnNumber, lottery.OpenTime))
		r := rand.New(rand.NewSource(time.Now().Unix()))
		randMillisecond := time.Duration(r.Intn(500)+1000) * time.Millisecond
		url := strings.Replace(dateUrl, "{date}", dateKey, 1)

		m.Concurrency <- true
		codes, number, err := d.Download(url)
		if err != nil {
			//log.Error("checkHistoryRecord  Download err:%v", err.Error())
			time.Sleep(randMillisecond)
			<-m.Concurrency
			continue
		}
		if number != dateKey {
			time.Sleep(randMillisecond)
			<-m.Concurrency
			continue
		}
		record.Number = number
		record.CnNumber = cnNumber
		record.Date = number
		record.OpenCode = codes
		record.LotteryCode = lottery.LotteryCode
		record.AreaCode = lottery.AreaCode
		record.CityCode = lottery.CityCode
		record.WeekNumber = lottery.WeekNumber
		record.CollectTime = now
		record.CollectUrl = url
		record.OpenTime = openTime

		if err := record.Add(); err != nil {
			log.Error("checkHistoryRecord  Add data err:%v", err.Error())
			time.Sleep(randMillisecond)
			<-m.Concurrency
			continue
		}
		time.Sleep(randMillisecond)
		<-m.Concurrency
	}
}
