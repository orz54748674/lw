package slotDance

import (
	"math/rand"
	"time"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/module"
	vGate "vn/gate"
	"vn/storage/slotStorage/slotDanceStorage"
)

func (this *MyTable) ClearTable() { //
	if !this.IsInCheckout {
		slotDanceStorage.RemoveTableInfo(this.tableID)

		myRoom := (this.module).(*Room)
		myRoom.DestroyTable(this.tableID)
	}
}
func (this *MyTable) TableInit(module module.RPCModule, app module.App, tableID string) {
	this.Players = map[string]room.BasePlayer{}
	this.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	this.GameConf = slotDanceStorage.GetRoomConf()
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

	this.ReelsList = ReelsList87 //ReelsNormalList
	this.ReelsListTrial = ReelsList110
	this.JieSuanData = JieSuanData{}
	this.JieSuanData.FreeData = FreeData{}
	this.JieSuanData.FreeData.FreeStepTimes = make([]int64, 0)

	this.CoinValue = CoinValue[0]
	this.CoinNum = CoinNum[0]
	this.XiaZhuV = this.CoinValue * this.CoinNum
	this.TrialModeConf = TrialModeConf{
		VndBalance: 200000000,
	}

	this.onlinePush = &vGate.OnlinePush{
		App:       app,
		TraceSpan: this.Trace(),
	}
	this.onlinePush.OnlinePushInit(this, 512)
}
