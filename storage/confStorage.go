package storage

import (
	"vn/common"
	"vn/common/errCode"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/log"
)

var (
	cConf                      = "conf"
	KCustomerOnline            = "customerOnline"
	KCustomerTel               = "customerTel"
	KCustomerTelegram          = "customerTelegram"
	KCustomerLiveChat          = "customerLiveChat"
	KCustomerMessenger         = "customerMessenger"
	KCustomerZalo              = "customerZalo"
	KTcpHost                   = "tcpHost"
	KApiHost                   = "apiHost"
	KCdnHost                   = "cdnHost"
	KCdnHost2                  = "cdnHost2"
	KAgentVipExpire            = "agentVipExpire"
	KLobbyNotice               = "lobbyNotice"
	KLobbyBannerUnUseExpire    = "lobbyBannerUnUseExpire"
	KLobbyBannerLowestAmount   = "lobbyBannerLowestAmount"
	KTestJackpot               = "testJackpot"
	KNotAllowBot               = "NotAllowBot"
	KSmsSuperCode              = "SmsSuperCode"
	KCurLanguage               = "CurLanguage"
	KIsCloseSmsGate            = "IsCloseSmsGate"
	KCanDouDouCount            = "CanDouDouCount"
	KActivityRewardBetTimes    = "activityRewardBetTimes"
	KBindPhoneReward           = "activityBindPhoneReward"
	KPhoneChargeNapTuDong      = "PhoneChargeNapTuDong"
	KAwcBetLimit               = "AwcBetLimit"
	KXgBetLimit                = "XgBetLimit"
	KWmBetLimit                = "WmBetLimit"
	KChargeIncProfitUserPer    = "ChargeIncProfitUserPer"
	KLotteryCollectionTime     = "LotteryCollectionTime"
	KLotteryCollectionInterval = "LotteryCollectionInterval"
	KPhoneChargeNapTuDongMM1S  = "PhoneChargeNapTuDongMM1S"
	KRegisterLimit             = "RegisterLimit"
	KCombineHttpHost             = "CombineHttpHost"
)

func InitCustomerConf() {
	//c := common.GetMongoDB().C(cConf)
	//count, err := c.Find(bson.M{}).Count()
	//if err != nil {
	//	log.Error(err.Error())
	//}
	//if count > 0 {
	//	return
	//}
	customerOnlineValue := "http://google.com"
	UpsertConf(KCustomerOnline, customerOnlineValue, "在线客服")
	UpsertConf(KCustomerTel, []string{"+855 110110", "+855 110112"}, "客服电话")
	UpsertConf(KCustomerTelegram, "@vn_poker", "客服TG账号")
	UpsertConf(KCustomerMessenger, "messenger", "客服messenger")
	UpsertConf(KCustomerLiveChat, "liveChat", "客服liveChat")
	UpsertConf(KCustomerZalo, "zalo", "客服zalo")
	UpsertConf(KTcpHost, "192.168.8.111", "长连接地址")
	UpsertConf(KApiHost, "192.168.8.111:8080", "API地址")
	UpsertConf(KCdnHost, "http://127.0.0.1:8001", "CDN地址")
	UpsertConf(KCdnHost2, "http://127.0.0.1:8001", "CDN地址2,为了跳转")
	UpsertConf(KAgentVipExpire, "7", "代理下属几天活跃值")
	UpsertConf(KLobbyNotice, "欢迎光临Luckwin", "大厅右上角公告")
	UpsertConf(KLobbyBannerUnUseExpire, "60", "未使用的大厅轮播过期时间")
	UpsertConf(KLobbyBannerLowestAmount, "100000", "能进入大厅轮播的最低金额")
	UpsertConf(KTestJackpot, "1", "dx jackpot测试模式")
	UpsertConf(KNotAllowBot, "0", "不允许机器人下注设置1")
	UpsertConf(KSmsSuperCode, "888666", "超级验证码，为空则不允许")
	UpsertConf(KCurLanguage, "zh_CN", "当前语言,枚举：VN，zh_CN")
	UpsertConf(KIsCloseSmsGate, "1", "是否关闭短信网关，1:关闭")
	UpsertConf(KCanDouDouCount, "3", "每天可换豆豆次数")
	UpsertConf(KActivityRewardBetTimes, "10", "获取奖励领取的押注流水要求，倍数")
	UpsertConf(KBindPhoneReward, "10,25", "最小，最大范围，随机值会乘以100")
	UpsertConf(KPhoneChargeNapTuDong, "9180139061,fb6016f21f4398ddb72cc71b86fb779a", "NapTuDong: partnerId,partnerKey")
	UpsertConf(KAwcBetLimit, `{"SEXYBCRT":{"LIVE":{"limitId":[261106,261107,261120]}}}`, "awc API 限红")
	UpsertConf(KXgBetLimit, "131,132,133", "xg API 限红")
	UpsertConf(KWmBetLimit, "124", "wm API 限红")
	UpsertConf(KChargeIncProfitUserPer, "100", "充值增加个人机器人余额百分比 100就是一倍")
	UpsertConf(KLotteryCollectionTime, 36, "lottery采集单期多少个小时没采集到数据停止采集")
	UpsertConf(KLotteryCollectionInterval, 8, "lottery同一个任务采集时间间隔")
	UpsertConf(KPhoneChargeNapTuDongMM1S, "aWc5K0pQOXhkNWwzazJ3YnJUU1AxZz09,2815579d7a177c1f3f5db4cb2f6ce494", "NapTuDongMM1S: PartnerId,AccessKey")
	UpsertConf(KRegisterLimit, "1,5,10", "同IP或设备限制注册 第一个参数间隔天数, 第二个参数 设备数量上限, 第三个参数 同IP数量上限(1,5,10表示1天同设备上限5个，同IP10个) 注意 设置成<=0 是无限制")
	UpsertConf(KCombineHttpHost, "http://127.0.0.1:8081", "综合盘http地址")
}

func QueryCustomerInfo() map[string]interface{} {
	info := make(map[string]interface{}, 6)
	info[KCustomerOnline] = QueryConf(KCustomerOnline)
	info[KCustomerTel] = QueryConf(KCustomerTel)
	info[KCustomerTelegram] = QueryConf(KCustomerTelegram)
	info[KCustomerLiveChat] = QueryConf(KCustomerLiveChat)
	info[KCustomerMessenger] = QueryConf(KCustomerMessenger)
	info[KCustomerZalo] = QueryConf(KCustomerZalo)
	return info
}
func QueryConf(key string) interface{} {
	c := common.GetMongoDB().C(cConf)
	conf := make(map[string]interface{}, 4)
	if err := c.Find(bson.M{"key": key}).One(&conf); err != nil {
		log.Info("not found conf key: %s ", key)
		return ""
	}
	return conf["value"]
}
func UpsertConf(key string, value interface{}, remark string) *common.Err {
	c := common.GetMongoDB().C(cConf)
	selector := bson.M{"key": key}
	conf := make(map[string]interface{}, 4)
	if err := c.Find(selector).One(&conf); err != nil { //不存在则创建
		update := make(map[string]interface{}, 4)
		update["value"] = value
		update["key"] = key
		update["remark"] = remark
		_, err := c.Upsert(selector, update)
		if err != nil {
			log.Info("Upsert user conf error: %s", err)
			return errCode.ServerError.SetErr(err.Error())
		}
	}
	return nil
}
func QueryLobbyNotice() interface{} {
	return QueryConf(KLobbyNotice)
}
