package slotLs

import (
	"math/rand"
	"time"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/module"
	vGate "vn/gate"
	"vn/storage/slotStorage/slotLsStorage"
)

func (this *MyTable) ClearTable() { //
	if !this.IsInCheckout{
		slotLsStorage.RemoveTableInfo(this.tableID)

		myRoom := (this.module).(*Room)
		myRoom.DestroyTable(this.tableID)
	}
}
func (this *MyTable) TableInit(module module.RPCModule,app module.App,tableID string){
	this.Players = map[string]room.BasePlayer{}
	this.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	this.GameConf = slotLsStorage.GetRoomConf()
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
	this.ReelsListFree = ReelsListFree2  //ReelsNormalList
	this.ReelsListTrial = ReelsListTrial //ReelsNormalList
	this.ReelsListTrialFree = ReelsListTrialFree //ReelsNormalList

	this.CoinValue = CoinValue[0]
	this.CoinNum = CoinNum[0]
	this.XiaZhuV = this.CoinValue * this.CoinNum
	this.TrialModeConf = TrialModeConf{
		VndBalance:    200000000,
		GoldJackpot:   []int64{100000000,200000000,400000000,600000000,1000000000},
		SilverJackpot: []int64{10000000,20000000,40000000,60000000,100000000},
	}

	this.onlinePush = &vGate.OnlinePush{
		App:       app,
		TraceSpan: this.Trace(),
	}
	this.onlinePush.OnlinePushInit(this, 512)
}