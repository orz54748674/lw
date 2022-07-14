package activity

import (
	"vn/common/utils"
	"vn/storage/activityStorage"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
)

func dealTurnTableChargeActivity(order *payStorage.Order) {
	if activityStorage.QueryActivityIsOpen(activityStorage.TurnTable) { //开启该活动
		conf := activityStorage.QueryActivityTurnTableConf()
		vipLevel := userStorage.QueryUserInfo(order.UserId).VipLevel
		vipConf := activityStorage.QueryActivityVipConfByType(activityStorage.PointsExtraThousand) //积分加成
		levelSwitchVipTable := []int64{
			vipConf.Vip0, vipConf.Vip1, vipConf.Vip2, vipConf.Vip3, vipConf.Vip4, vipConf.Vip5, vipConf.Vip6, vipConf.Vip7, vipConf.Vip8, vipConf.Vip9,
		}
		points := float64(order.GotAmount) / conf.ChargeSwapPoints * float64(levelSwitchVipTable[vipLevel]) / 1000
		if points > 0 {
			activityStorage.IncTurnTableInfoPoints(order.UserId.Hex(), points)
		}
	}
}
func dealTurnTableBetsActivity(uid string, bets int64) {
	if activityStorage.QueryActivityIsOpen(activityStorage.TurnTable) { //开启该活动
		conf := activityStorage.QueryActivityTurnTableConf()
		vipLevel := userStorage.QueryUserInfo(utils.ConvertOID(uid)).VipLevel
		vipConf := activityStorage.QueryActivityVipConfByType(activityStorage.PointsExtraThousand) //积分加成
		levelSwitchVipTable := []int64{
			vipConf.Vip0, vipConf.Vip1, vipConf.Vip2, vipConf.Vip3, vipConf.Vip4, vipConf.Vip5, vipConf.Vip6, vipConf.Vip7, vipConf.Vip8, vipConf.Vip9,
		}
		points := float64(bets) / conf.BetsSwapPoints * float64(levelSwitchVipTable[vipLevel]) / 1000
		if points > 0 {
			activityStorage.IncTurnTableInfoPoints(uid, points)
		}
	}
}
