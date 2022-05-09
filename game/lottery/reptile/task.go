package reptile

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	mqrpc "vn/framework/mqant/rpc"
	"vn/storage"
	"vn/storage/lotteryStorage"

	"github.com/PuerkitoBio/goquery"
)

var (
// recordStartData = "2020-01-01"
)

type Task struct {
	Lottery *lotteryStorage.Lottery
	url     string
	date    string
	M       module.RPCModule
}

func (t *Task) initData() {
	t.date = utils.GetDate(time.Now())
	t.url = strings.Replace(t.Lottery.CollectUrl, "{city}", t.Lottery.CityCode, 1)
	t.url = strings.Replace(t.url, "{date}", t.date, 1)
}

func (t *Task) Run() {
	log.Debug("lottery Code:%s start", t.Lottery.LotteryCode)
	t.initData()
	record := &lotteryStorage.LotteryRecord{}
	nowDate := time.Now().Format("2006-01-02")
	if count, _ := record.IsExist(nowDate, t.Lottery.LotteryCode); count > 0 {
		log.Info("lottery Code: %s date：%s record is exist", nowDate, t.Lottery.LotteryCode)
		return
	}

	client, err := t.getHttpClient()
	if err != nil {
		log.Error("lottery Code: %s CollectUrl err:%s", t.Lottery.LotteryCode, err.Error())
		return
	}

	timeInterval, err := utils.ConvertInt(storage.QueryConf(storage.KLotteryCollectionInterval))
	if err != nil {
		log.Error("KLotteryCollectionInterval type err:", err.Error())
		timeInterval = 60
	}
	hours, err := utils.ConvertInt(storage.QueryConf(storage.KLotteryCollectionTime))
	if err != nil {
		log.Error("KLotteryCollectionTime type err:", err.Error())
		hours = 24
	}
	tk := time.NewTicker(time.Second * time.Duration(timeInterval))
	defer tk.Stop()
	var count int64 = 0
	second := hours * 3600
	for {
		if count > second {
			break
		}
		data, err := t.download(client)
		if err != nil {
			log.Error("lottery Code: %s download err:%s", t.Lottery.LotteryCode, err.Error())
		} else {
			record, err := t.parse(data)
			if err == nil {
				if record.CnNumber == nowDate {
					log.Debug("lottery Code: %s record.CnNumber:%s nowDate:%s get data success  ", t.Lottery.LotteryCode, record.CnNumber, nowDate)
					err = record.Add()
					if err == nil {
						go t.open(record)
						log.Debug("lottery Code: %s number:%s Data struggle, deposit success, stop collection", t.Lottery.LotteryCode, nowDate)
						break
					}
					log.Debug("lottery Code: %s add data err:%s", t.Lottery.LotteryCode, err.Error())
				} else {
					log.Debug("lottery Code: %s record.CnNumber:%s nowDate:%s number err  ", t.Lottery.LotteryCode, record.CnNumber, nowDate)
				}

			} else {
				log.Error("lottery Code: %s get data err:%s", t.Lottery.LotteryCode, err.Error())
			}
		}
		<-tk.C
		count += timeInterval
	}
	log.Debug("lottery Code:%s end", t.Lottery.LotteryCode)
}

func (t *Task) getHttpClient() (client *http.Client, err error) {
	if len(t.Lottery.CollectUrl) == 0 {
		return nil, fmt.Errorf("collect url is empty")
	}
	proxy, _ := url.Parse("http://wjlyf3000:t5ok13yVMH5jwhPg@proxy.packetstream.io:31112")
	if strings.HasPrefix(t.Lottery.CollectUrl, "http://") {
		tr := &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
		return &http.Client{
			Transport: tr,
		}, nil
	} else if strings.HasPrefix(t.Lottery.CollectUrl, "https://") {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyURL(proxy),
		}
		return &http.Client{Transport: tr}, nil
	} else {
		return nil, fmt.Errorf("protocol error")
	}
}

func (t *Task) download(client *http.Client) (data io.ReadCloser, err error) {

	resp, err := client.Get(t.url)
	if err != nil {
		return
	}
	data = resp.Body
	return
}

func (t *Task) parse(data io.ReadCloser) (lotteryRecord *lotteryStorage.LotteryRecord, err error) {

	defer data.Close()
	html, err := goquery.NewDocumentFromReader(data)
	if err != nil {
		return
	}

	table := html.Find(".box_kqxs").First()
	if len(table.Text()) == 0 {
		err = fmt.Errorf("not find data")
		return
	}
	dateInfo := strings.TrimSpace(table.Find("div.ngay").Text())
	number := dateInfo[len(dateInfo)-10:]
	numbers := strings.Split(number, "/")
	if len(numbers) != 3 {
		err = fmt.Errorf("number err:%v;numbers:%v", number, numbers)
		return
	}
	number = strings.Join(numbers, "-")
	codes := make(map[lotteryStorage.PrizeLevel][]string)
	levels := []string{"db", "1", "2", "3", "4", "5", "6", "7", "8"}
	count := 0
	for i := 0; i < len(levels); i++ {
		st := fmt.Sprintf(".box_kqxs_content td.giai%s div", levels[i])
		table.Find(st).Each(func(_ int, item *goquery.Selection) {
			key := lotteryStorage.PrizeLevel(fmt.Sprintf("L%d", i))
			codes[key] = append(codes[key], strings.TrimSpace(item.Text()))
		})
		if t.Lottery.AreaCode == "North" && i == 8 {
			break
		}
		count++
	}
	now := time.Now()
	openTimeStr := fmt.Sprintf("%s %s", now.Format("2006-01-02"), t.Lottery.OpenTime)

	loc, _ := time.LoadLocation("Local")
	openTime, err := time.ParseInLocation("2006-01-02 15:04:05", openTimeStr, loc)
	if err != nil {
		log.Debug("parse open time Err:%s", err.Error())
		openTime = now
		err = nil
	}
	lotteryRecord = &lotteryStorage.LotteryRecord{
		Number:      number,
		CnNumber:    fmt.Sprintf("%s-%s-%s", numbers[2], numbers[1], numbers[0]),
		Date:        number,
		WeekNumber:  t.Lottery.WeekNumber,
		OpenCode:    codes,
		LotteryCode: t.Lottery.LotteryCode,
		AreaCode:    t.Lottery.AreaCode,
		CityCode:    t.Lottery.CityCode,
		CollectTime: now,
		CollectUrl:  t.url,
		OpenTime:    openTime,
	}
	return
}

func (t *Task) open(record *lotteryStorage.LotteryRecord) {
	log.Debug("lottery Code: %s open Number:%s start", record.LotteryCode, record.Number)
	betRecord := &lotteryStorage.LotteryBetRecord{}
	_, err := betRecord.SetOpenCode(record.Number, record.LotteryCode, record.OpenCode)
	if err != nil {
		log.Error("lottery Code: %s set Number:%s err:%s", record.LotteryCode, record.Number, err.Error())
		return
	}
	params := map[string]interface{}{}
	params["Number"] = record.Number
	params["LotteryCode"] = record.LotteryCode
	ctx, _ := context.WithTimeout(context.TODO(), time.Second*3) //3s后超时
	res, err := mqrpc.String(
		t.M.Call(
			ctx,
			"settle",        //要访问的moduleType
			"/lottery/open", //访问模块中handler路径
			mqrpc.Param(params),
		),
	)
	if err != nil {
		log.Debug("lottery Code: %s  rpc res:%v, err：%v", record.LotteryCode, res, err.Error())
		return
	}
	log.Debug("lottery Code: %s open Number:%s end  rpc res:%v", record.LotteryCode, record.Number, res)
}
