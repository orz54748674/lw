package settle

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	mqrpc "vn/framework/mqant/rpc"
	"vn/game"
	"vn/game/activity"
	"vn/storage/activityStorage"
	"vn/storage/gameStorage"
	"vn/storage/lotteryStorage"
	"vn/storage/walletStorage"

	"github.com/mitchellh/mapstructure"
)

const (
	settleFailed  = 2
	settleSuccess = 8
)

type Settle struct {
	basemodule.BaseModule
}

var Module = func() module.Module {
	return new(Settle)
}

func (m *Settle) Version() string {
	return "1.0.0"
}
func (m *Settle) GetType() string {
	return "settle"
}
func (m *Settle) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)
	hook := game.NewHook(m.GetType())
	hook.RegisterAndCheckLogin(m.GetServer(), "HD_open", m.lotteryOpen)
	m.GetServer().RegisterGO("/lottery/open", m.open)
}

func (m *Settle) Run(closeSig chan bool) {
	log.Info("%v 模块运行中...", m.GetType())
	<-closeSig
	log.Info("%v 模块已停止...", m.GetType())
}

func (m *Settle) OnDestroy() {
	//一定别忘了继承
	m.BaseModule.OnDestroy()
	log.Info("%v 模块已回收...", m.GetType())
}

// lotteryOpen 测试
func (m *Settle) lotteryOpen(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	record := &lotteryStorage.LotteryRecord{}
	if err := mapstructure.Decode(params, record); err != nil {
		return errCode.ErrParams.GetI18nMap(), err
	}
	betRecord := &lotteryStorage.LotteryBetRecord{}
	_, err := betRecord.SetOpenCode(record.Number, record.LotteryCode, record.OpenCode)
	if err != nil {
		log.Error("set %s:%s err:%s", record.LotteryCode, record.Number, err.Error())
	}
	m.open(params)
	data := map[string]interface{}{"success": true}
	return errCode.Success(data).GetI18nMap(), nil
}

func (m *Settle) open(data map[string]interface{}) (r string, err error) {
	number := data["Number"].(string)
	lotteryCode := data["LotteryCode"].(string)
	log.Debug("number:%s,lotteryCode:%s", number, lotteryCode)
	betRecord := &lotteryStorage.LotteryBetRecord{}
	limit := 1000
	offset := 0
	// 单用户单期最大赔10亿
	var MaxPayAmount int64 = 1000000000
	userPayMap := make(map[string]int64)
	for {
		betRecords, err := betRecord.GetNumberBets(number, lotteryCode, offset, limit)
		if err != nil {
			log.Error("GetBets err:%s", err.Error())
			break
		}
		for _, bRecord := range betRecords {
			funcKey := fmt.Sprintf("%s_%s", bRecord.AreaCode, bRecord.SubPlayCode)
			//type playFunc func(bet string, betAmount, odds int64, OpenCode map[PrizeLevel][]string) int64
			Func, ok := lotteryStorage.PlayMap[funcKey]
			if !ok {
				log.Error("settle func is not find(funcKey:%s)", funcKey)
				continue
			}
			sProfit := Func(bRecord.Code, bRecord.UnitBetAmount, bRecord.Odds, bRecord.OpenCode) / 100
			wallet := walletStorage.QueryWallet(utils.ConvertOID(bRecord.Uid))
			gameRes, _ := json.Marshal(map[string]interface{}{"OpenCode": bRecord.OpenCode, "SettleTime": time.Now()})
			income := sProfit - bRecord.TotalAmount
			if sProfit == 0 {
				bRecord.SProfit = -bRecord.TotalAmount
				log.Debug("lottery open %v  bRecord.TotalAmount:%v", bRecord.SProfit, bRecord.TotalAmount)
				changeSettleStatus(bRecord, settleSuccess)
				gameStorage.UpdateLotteryBetRecord(bRecord.Uid, bRecord.Oid.Hex(), bRecord.Number, string(gameRes), game.Lottery, income, bRecord.TotalAmount, wallet.VndBalance+wallet.SafeBalance, 0, 0)
				activityStorage.UpsertGameDataInBet(bRecord.Uid, game.Lottery, -1)
				activity.CalcEncouragementFunc(bRecord.Uid)
				continue
			}
			if (userPayMap[bRecord.Uid] + sProfit) > MaxPayAmount {
				sProfit = MaxPayAmount - userPayMap[bRecord.Uid]
			}
			bRecord.SetTransactionUnits(lotteryStorage.ChangeSettleStatus)
			bRecord.SettleStatus = settleSuccess
			bRecord.SProfit = sProfit
			bill := walletStorage.NewBill(bRecord.Uid, walletStorage.TypeIncome, walletStorage.EventGameLottery, bRecord.Oid.Hex(), sProfit)
			if err := walletStorage.OperateVndBalanceV1(bill, bRecord); err != nil {
				log.Error("Oid:%v settle err:%s", bRecord.Oid, err.Error())
				changeSettleStatus(bRecord, settleFailed)
				continue
			}
			activityStorage.UpsertGameDataInBet(bRecord.Uid, game.Lottery, -1)
			activity.CalcEncouragementFunc(bRecord.Uid)
			gameStorage.UpdateLotteryBetRecord(bRecord.Uid, bRecord.Oid.Hex(), bRecord.Number, string(gameRes), game.Lottery, income, bRecord.TotalAmount, wallet.VndBalance+wallet.SafeBalance, 0, 0)
			userPayMap[bRecord.Uid] += sProfit
		}
		if len(betRecords) < limit {
			break
		}
		offset = (offset + 1) * limit
	}
	ctx, _ := context.WithTimeout(context.TODO(), time.Second*3) //3s后超时
	res, err := mqrpc.String(
		m.App.Call(
			ctx,
			"lottery",             //要访问的moduleType
			"/lottery/noticeOpen", //访问模块中handler路径
			mqrpc.Param(data),
		),
	)
	if err != nil {
		log.Debug("lottery Code: %s  rpc res:%v, err：%v", lotteryCode, res, err.Error())
		return
	}
	log.Debug("lottery Code: %s open Number:%s end  rpc res:%v", lotteryCode, number, res)
	return fmt.Sprintf("hi %v", data), nil
}

func changeSettleStatus(bRecord *lotteryStorage.LotteryBetRecord, settleStatus int) {
	if err := bRecord.ChangeSettleStatus(settleStatus); err != nil {
		log.Error("change Oid:%v settelStatus err:%s", bRecord.Oid, err.Error())
	}
}
