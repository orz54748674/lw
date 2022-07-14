package data

import (
	"time"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/storage/gameStorage"
)

type DataOverview struct {
}

func (this *DataOverview) dealGameOverview(channel string, gameType game.Type, channelName string) {
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	view := gameStorage.QueryGameOverview(channel, gameType, thatTime)
	if view.Channel != "" {
		thatTime = view.UpdateAt
	}
	betRecord := gameStorage.QueryBetRecordByChannel(channel, gameType, thatTime)
	newView := gameStorage.GameOverview{}
	if len(betRecord) != 0 {
		newView.BetValue = betRecord[0]["BetAmount"].(int64)
		newView.SysProfit = betRecord[0]["SysProfit"].(int64)
		newView.BotProfit = betRecord[0]["BotProfit"].(int64)
		newView.AgentProfit = betRecord[0]["AgentProfit"].(int64)
	}
	curTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	newView.BetUsers = gameStorage.QueryBetRecordByChannelUsers(channel, gameType, curTime)

	newView.Channel = channel
	newView.ChannelName = channelName
	newView.GameType = gameType
	gameStorage.UpsertGameOverview(&newView)
}
func (this *DataOverview) dealUserOverview(channel string, channelName string) {
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	uids, uidHexs := gameStorage.QueryUidsByChannel(channel)
	newUidHexs := gameStorage.QueryNewUidsByChannel(channel, thatTime)
	if uids == nil {
		uids = []string{}
	}
	if uidHexs == nil {
		uidHexs = []primitive.ObjectID{}
	}
	if newUidHexs == nil {
		newUidHexs = []primitive.ObjectID{}
	}
	view := gameStorage.UserOverview{}
	view.ChannelName = channelName
	view.Channel = channel
	view.UserAdd = gameStorage.QueryNewUsers(channel, thatTime)
	view.UserLogin = gameStorage.QueryUserLogin(uidHexs, thatTime)
	retCharge := gameStorage.QueryChargeData(uidHexs, newUidHexs, thatTime)
	view.AddChargeUsers = retCharge["AddChargeUsers"].(int)
	view.AddChargeAmount = retCharge["AddChargeAmount"].(int) - retCharge["AddFee"].(int)
	view.ChargeAmount = retCharge["ChargeAmount"].(int64) - retCharge["Fee"].(int64)
	view.ChargeUsers = retCharge["ChargeUsers"].(int)
	view.FirstChargeUsers = retCharge["FirstChargeUsers"].(int)
	retDouDou := gameStorage.QueryDouDouData(uids, thatTime)
	view.DouDouAmount = retDouDou["DouDouAmount"].(int64)
	view.DouDouUsers = retDouDou["DouDouUsers"].(int)
	view.UserTotal = len(uids)
	wallet := gameStorage.QueryWalletData(uidHexs)
	view.VndBalance = wallet["VndBalance"].(int64)
	view.AgentBalance = wallet["AgentBalance"].(int64)
	view.ActivityReceive = gameStorage.QueryActivityAmount(uids, thatTime)
	gameStorage.UpsertUserOverview(&view)
}
func (this *DataOverview) start() {
	gameStorage.InitGameOverview()
	gameStorage.InitUserOverview()
	start := time.Now()
	channels := gameStorage.QueryChannels()
	for _, channel := range channels {
		for _, gameType := range game.GameList {
			this.dealGameOverview(channel.Channel, gameType, channel.Name)
		}
		this.dealUserOverview(channel.Channel, channel.Name)
	}
	end := time.Now()
	log.Info("finished data overview work spent time: %v", end.Sub(start))
}
