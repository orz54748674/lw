package slotCs

import (
	"math/rand"
	"strconv"
	"time"
	"vn/common/protocol"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/module"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/slotStorage/slotCsStorage"
	"vn/storage/walletStorage"
)

func (this *MyTable) ClearTable() { //
	if !this.IsInCheckout{
		if this.ModeType == NORMAL && this.JieSuanData.BonusGame{
			if this.BonusGameData.TotalSymbolScore > 0{
				this.EventID = string(game.SlotCs) + "_" + "Bonus" + "_" + strconv.FormatInt(time.Now().Unix(), 10)
				bill := walletStorage.NewBill(this.UserID, walletStorage.TypeIncome, walletStorage.EventGameSlotCs, this.EventID, this.BonusGameData.TotalSymbolScore)
				walletStorage.OperateVndBalance(bill)

				this.notifyWallet(this.UserID)
			}
		}
		slotCsStorage.RemoveTableInfo(this.tableID)

		myRoom := (this.module).(*Room)
		myRoom.DestroyTable(this.tableID)
	}
}
func (this *MyTable) TableInit(module module.RPCModule,app module.App,tableID string){
	this.Players = map[string]room.BasePlayer{}
	this.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	this.GameConf = slotCsStorage.GetRoomConf()
	//
	//tableInfo := yxxStorage.GetTableInfo(tableID)
	//tableInfo.TableID = tableID
	//tableInfo.ServerID = module.GetServerID()
	//yxxStorage.UpsertTableInfo(tableInfo,tableID)

	this.onlinePush = &vGate.OnlinePush{
		App:       app,
		TraceSpan: this.Trace(),
	}
	this.onlinePush.OnlinePushInit(this, 512)

	this.ReelsList = ReelsListNormal2      //ReelsNormalList
	this.ReelsListTrial = ReelsListTrial //ReelsNormalList
	this.BonusSymbolList = BonusSymbolList
	this.BonusGameData = BonusGameData{}
	this.MiniGameData = MiniGameData{}

	this.CoinValue = CoinValue[0]
	this.CoinNum = CoinNum[0]
	this.XiaZhuV = this.CoinValue * this.CoinNum
	this.TrialModeConf = TrialModeConf{
		VndBalance:    200000000,
	}

	this.onlinePush = &vGate.OnlinePush{
		App:       app,
		TraceSpan: this.Trace(),
	}
	this.onlinePush.OnlinePushInit(this, 512)

	//go func() {
	//	c := cron.New()
	//	c.AddFunc("*/1 * * * * ?",this.OnTimer)
	//	c.Start()
	//}()
}
func (this *MyTable) OnTimer() { //定时任务
	if this.JieSuanData.BonusGame{
		if this.CountDown <= 0 && this.BonusGameData.State > 0{
			this.PutQueue(protocol.BonusTimeOut,this.Players[this.UserID].Session())
		}
	}
	this.CountDown--
}