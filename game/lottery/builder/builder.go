package builder

import (
	"context"
	"fmt"
	"math/rand"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	mqrpc "vn/framework/mqant/rpc"
	"vn/game"
	"vn/storage/gameStorage"
	"vn/storage/lotteryStorage"

	"github.com/mitchellh/mapstructure"
)

type Builder struct {
	basemodule.BaseModule
}

var Module = func() module.Module {
	return new(Builder)
}

func (m *Builder) GetType() string {
	return "builder"
}

func (m *Builder) Version() string {
	return "1.0.0"
}

func (m *Builder) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)
}
func (m *Builder) Run(closeSig chan bool) {
	log.Info("%v 模块运行中...", m.GetType())
	<-closeSig
	log.Info("%v 模块已停止...", m.GetType())
}

func (m *Builder) OnDestroy() {
	//一定别忘了继承
	m.BaseModule.OnDestroy()
	log.Info("%v 模块已回收...", m.GetType())
}

func (m *Builder) start(lottery *lotteryStorage.Lottery) {
	sTime, err := utils.StrToCnTime(fmt.Sprintf("%s %s", utils.GetCnDate(time.Now()), lottery.StartTime))
	if err != nil {
		log.Error("lottery startTime parse err:%s", err.Error())
		return
	}
	eTime, err := utils.StrToCnTime(fmt.Sprintf("%s %s", utils.GetCnDate(time.Now()), lottery.EndTime))
	if err != nil {
		log.Error("lottery endTime parse err:%s", err.Error())
		return
	}
	numberCount := (eTime.Unix() - sTime.Unix()) / lottery.Intervals
	numberLen := len(fmt.Sprintf("%d", numberCount+1))
	intervals := time.Duration(lottery.Intervals)
	ticker := time.NewTicker(intervals * time.Second)
	defer ticker.Stop()

	for {
		sTimeUnix := sTime.Unix()
		eTimeUnix := eTime.Unix()
		if time.Now().Unix() <= sTimeUnix {
			if sleepTime := sTimeUnix - time.Now().Unix(); sleepTime > 0 {
				time.Sleep(time.Duration(sleepTime) * time.Second)
			}
			ticker.Reset(intervals * time.Second)
			count := 1
			for {
				go m.task(count, numberLen, lottery)
				count++
				<-ticker.C
				if time.Now().Unix() <= eTimeUnix {
					continue
				}
			}
		} else if sTimeUnix > time.Now().Unix() && time.Now().Unix() <= eTimeUnix {
			count := 1
			for {
				sleepTime := sTimeUnix - time.Now().Unix()
				if sleepTime < lottery.Intervals {
					time.Sleep(time.Duration(sleepTime) * time.Second)
					ticker.Reset(intervals * time.Second)
					break
				}
				count++
				sTimeUnix += lottery.Intervals
			}
			for {
				go m.task(count, numberLen, lottery)
				count++
				if sleepTime := sTimeUnix - time.Now().Unix(); sleepTime > lottery.Intervals {
					sTimeUnix += lottery.Intervals
					continue
				} else if sleepTime == lottery.Intervals {
					go m.task(count, numberLen, lottery)
					ticker.Reset(intervals * time.Second)
				}
				<-ticker.C
				if time.Now().Unix() <= eTimeUnix {
					continue
				}
			}

		}
		time.Sleep(time.Duration(sTime.Add(24*time.Hour).Unix()-eTimeUnix) * time.Second)
	}
}

func (m *Builder) task(issue, numberLen int, lottery *lotteryStorage.Lottery) {
	strIssue := fmt.Sprintf("%0*d", numberLen, issue)
	number := fmt.Sprintf("%s%s", time.Now().Format("20060102"), strIssue)
	mRecord := &lotteryStorage.LotteryRecord{Number: number, AreaCode: lottery.AreaCode}
	record, err := mRecord.GetRecord(number, lottery.LotteryCode)
	if err == mongo.ErrNoDocuments {
		m.createOpenCode(mRecord)
		ok, err := m.check(mRecord)
		if err != nil {
			return
		}
		if ok {
			go m.open(mRecord)
		}
	} else if err != nil {
		log.Error("system lottery read db lottery err:%s", err.Error())
		return
	} else {
		if err := mapstructure.Decode(record, mRecord); err != nil {
			log.Error("system lottery paser open code err:%s", err.Error())
			return
		}
		mBetRecord := &lotteryStorage.LotteryBetRecord{}
		res, err := mBetRecord.SetOpenCode(number, lottery.LotteryCode, mRecord.OpenCode)
		if err != nil {
			log.Error("system lottery SetOpenCode err:%s", err.Error())
			return
		}
		if res.ModifiedCount > 0 {
			go m.open(mRecord)
		}
	}
}

func (m *Builder) open(record *lotteryStorage.LotteryRecord) {
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
		m.App.Call(
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
	return
}

func (m *Builder) createOpenCode(record *lotteryStorage.LotteryRecord) {
	codeRule := lotteryStorage.CodeRuleMap[record.AreaCode]
	rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, rule := range codeRule {
		for i := 0; i < rule.Count; i++ {
			record.OpenCode[rule.Level] = append(record.OpenCode[rule.Level], fmt.Sprintf("%0*d", rule.CodeLen, rand.Intn(rule.Max+1)))
		}
	}
}

func (m *Builder) check(record *lotteryStorage.LotteryRecord) (bool, error) {
	mBetRecord := &lotteryStorage.LotteryBetRecord{}
	offset := 0
	limit := 1000
	var payoutAmount, maxPayoutAmount int64
	for {
		bets, err := mBetRecord.GetNumberBets(record.Number, record.AreaCode, offset, limit)
		if err == mongo.ErrNoDocuments {
			break
		}
		if err != nil {
			log.Error("GetNumberBets err:%s", err.Error())
			return false, err
		}
		for _, bet := range bets {
			funcKey := fmt.Sprintf("%s_%s", bet.AreaCode, bet.SubPlayCode)
			//type playFunc func(bet string, betAmount, odds int64, OpenCode map[PrizeLevel][]string) int64
			sProfit := lotteryStorage.PlayMap[funcKey](bet.Code, bet.UnitBetAmount, bet.Odds, bet.OpenCode) / 100
			payoutAmount += sProfit
		}
		if len(bets) < limit {
			break
		}
		offset = (offset + 1) * limit
	}
	if maxPayoutAmount <= 0 {
		gameProfit := gameStorage.QueryProfit(game.Lottery)
		return gameProfit.BotBalance >= payoutAmount, nil
	} else {
		return maxPayoutAmount >= payoutAmount, nil
	}
}

func (m *Builder) stats(record *lotteryStorage.LotteryRecord) {
	betStats := &lotteryStorage.BetStats{}
	totalPatAmount, err := betStats.NumberTotalPatAmount(record.Number, record.LotteryCode)
	if err != nil {
		log.Error("stats TotalPatAmount err:%s", err.Error())
		return
	}
	gameProfit := gameStorage.QueryProfit(game.Lottery)
	if gameProfit.BotBalance >= totalPatAmount {
		m.createOpenCode(record)
		go m.open(record)
		return
	}
	res, err := betStats.GetTotalPatAmounts(record.Number, record.LotteryCode)
	if err != nil {
		log.Error("stats GetTotalPatAmounts err:%s", err.Error())
		return
	}
	var playCodes []string
	for _, item := range res {
		totalPatAmount -= int64(item["TotalPatAmount"].(float64))
		if gameProfit.BotBalance >= totalPatAmount {
			break
		}
		playCodes = append(playCodes, item["_id"].(string))
	}
}
