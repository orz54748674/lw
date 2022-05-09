package agentStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/storage/payStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

type AgentVipData struct {
	ID          int64
	Date        string
	TodayIncome int64 //game.BetRecord
	MonthIncome int64 //game.BetRecord
	SumIncome   int64 //game.BetRecord
	TodayNewVip int64 //agentStorage.Invite
	ActiveVip   int   //Agent
	SumVip      int64 //agentStorage.Invite
	Level       int   //Agent
	AgentUid    string
	UpdateAt    time.Time
}

func (AgentVipData) TableName() string {
	return "agent_vip_data"
}
func QueryAgentVipData(date string, agentUid string) AgentVipData {
	var myVipData AgentVipData
	common.GetMysql().FirstOrInit(&myVipData,
		&AgentVipData{AgentUid: agentUid, Date: date})
	return myVipData
}
func QueryAgentVipDataSumMonth(month string, agentUid string) int64 {
	res := map[string]interface{}{}
	common.GetMysql().Model(&AgentVipData{}).
		Select("sum(today_income) month_income").
		Where("date like ? and agent_uid=?", month+"%", agentUid).
		First(&res)
	monthIncome, _ := utils.ConvertInt(res["month_income"])
	return monthIncome
}
func QueryAgentVipDataSumIncome(agentUid string) int64 {
	res := map[string]interface{}{}
	common.GetMysql().Model(&AgentVipData{}).
		Select("sum(today_income) month_income").
		Where("agent_uid=?", agentUid).
		First(&res)
	monthIncome, _ := utils.ConvertInt(res["month_income"])
	return monthIncome
}
func UpsertAgentVipData(data AgentVipData) {
	if data.ID == 0 {
		common.GetMysql().Create(&data)
	} else {
		common.GetMysql().Updates(&data)
	}
}
func OnBetRecord(vipUid string, income int64, betAmount int64) {

	now := time.Now()
	common.AddQueueByTag(vipUid, func() {
		date := utils.GetDateStr(now)
		month := utils.GetMonthStr(now)
		vipOid, _ := primitive.ObjectIDFromHex(vipUid)

		userFinance := QueryUserFinance(date, vipUid)
		userFinance.UpdateAt = now
		userFinance.TodayIncome += income
		userFinance.MonthIncome = QueryUserFinanceSumMonth(month, vipUid) + income
		userFinance.TodayBet += betAmount

		UpsertUserFinance(userFinance)

		agentMemberData := QueryAgentMemberData(date, vipUid)
		agentMemberData.UpdateAt = now
		agentMemberData.TodayBet += betAmount
		agentMemberData.TodayIncome += income
		UpsertAgentMemberData(agentMemberData)

		invite := QueryInvite(vipOid)
		if invite.Oid.IsZero() {
			return
		}
		agent := QueryAgent(invite.ParentOid)
		if agent == nil {
			return
		}
		agentUid := agent.Oid.Hex()
		AgentVipData := QueryAgentVipData(date, agentUid)
		AgentVipData.UpdateAt = now
		AgentVipData.TodayIncome += income
		AgentVipData.MonthIncome = QueryAgentVipDataSumMonth(month, agentUid) + income
		AgentVipData.SumIncome = QueryAgentVipDataSumIncome(agentUid) + income
		AgentVipData.ActiveVip = agent.Count
		AgentVipData.Level = agent.Level
		AgentVipData.SumVip = QuerySumVip(agentUid)
		UpsertAgentVipData(AgentVipData)

	})
}
func OnRegister(uid string) {
	now := time.Now()
	common.AddQueueByTag(uid, func() {
		date := utils.GetDateStr(now)
		agentMemberData := QueryAgentMemberData(date, uid)
		UpsertAgentMemberData(agentMemberData)
	})
}
func OnPayData(uid string, charge int64, douDou int64) {
	now := time.Now()
	common.AddQueueByTag(uid, func() {
		date := utils.GetDateStr(now)
		userFinance := QueryUserFinance(date, uid)
		userFinance.TodayCharge += charge
		userFinance.TodayDouDou += douDou
		UpsertUserFinance(userFinance)

		agentMemberData := QueryAgentMemberData(date, uid)
		agentMemberData.UpdateAt = now
		agentMemberData.TodayCharge += charge
		agentMemberData.TodayDouDou += douDou
		UpsertAgentMemberData(agentMemberData)
	})
}
func OnAgentBalanceData(uid string, agentBalance int64, level int) { //返佣
	now := time.Now()
	common.AddQueueByTag(uid, func() {
		date := utils.GetDateStr(now)
		agentMemberData := QueryAgentMemberData(date, uid)
		agentMemberData.UpdateAt = now
		agentMemberData.TodayAgentBalance += agentBalance
		if level == 1 {
			agentMemberData.SuperiorProfit1 += agentBalance
		} else if level == 2 {
			agentMemberData.SuperiorProfit2 += agentBalance
		} else if level == 3 {
			agentMemberData.SuperiorProfit3 += agentBalance
		}
		UpsertAgentMemberData(agentMemberData)
	})
}
func OnWalletChange(uid string) { //钱包变动
	now := time.Now()
	common.AddQueueByTag(uid, func() {
		date := utils.GetDateStr(now)
		agentMemberData := QueryAgentMemberData(date, uid)
		agentMemberData.UpdateAt = now
		UpsertAgentMemberData(agentMemberData)
	})
}
func OnActivityData(uid string, activity int64) {
	now := time.Now()
	common.AddQueueByTag(uid, func() {
		date := utils.GetDateStr(now)
		agentMemberData := QueryAgentMemberData(date, uid)
		agentMemberData.UpdateAt = now
		agentMemberData.TodayActivity += activity
		UpsertAgentMemberData(agentMemberData)
	})
}
func OnUpdateAgentMemberData(all []AgentMemberData) { //刷新数据
	now := time.Now()
	common.ExecQueueFunc(func() {
		date := utils.GetDateStr(now)
		for _, v := range all {
			agentMemberData := QueryAgentMemberData(date, v.Uid)
			agentMemberData.UpdateAt = now
			UpsertAgentMemberData(agentMemberData)
		}
	})
}

type UserFinanceData struct {
	ID          int64
	Date        string `gorm:"index:search"`
	Uid         string `gorm:"index:search"`
	TodayIncome int64  //当日输赢
	MonthIncome int64  //本月输赢
	TodayBet    int64  //当日押注
	TodayCharge int64  //累计充值
	TodayDouDou int64  //当日换豆豆
	UpdateAt    time.Time
}

func (UserFinanceData) TableName() string {
	return "user_finance_data"
}
func QueryUserFinance(date string, uid string) UserFinanceData {
	var userFinance UserFinanceData
	common.GetMysql().FirstOrInit(&userFinance,
		&UserFinanceData{Uid: uid, Date: date})
	return userFinance
}
func UpsertUserFinance(userFinance UserFinanceData) {
	userFinance.UpdateAt = utils.Now()
	if userFinance.ID == 0 {
		common.GetMysql().Create(&userFinance)
	} else {
		common.GetMysql().Updates(&userFinance)
	}
}
func QueryUserFinanceSumMonth(month string, uid string) int64 {
	res := map[string]interface{}{}
	common.GetMysql().Model(&UserFinanceData{}).
		Select("sum(today_income) month_income").
		Where("date like ? and uid=?", month+"%", uid).
		First(&res)
	monthIncome, _ := utils.ConvertInt(res["month_income"])
	return monthIncome
}

type AgentMemberData struct {
	ID                int64
	Date              string `gorm:"index:idx_member"`
	Uid               string `gorm:"index:idx_member"`
	Account           string
	NickName		  string
	SuperiorAccount1  string //上一级代理
	SuperiorAccount2  string //上二级代理
	SuperiorAccount3  string //上三级代理
	SuperiorProfit1   int64  //上一级代理收益
	SuperiorProfit2   int64  //上一级代理收益
	SuperiorProfit3   int64  //上一级代理收益
	BelongAgent       string //属于推广
	TodayCharge       int64  //今日充值
	TodayDouDou       int64  //今日换豆豆
	TodayBet          int64  //今日投注
	TodayIncome       int64  //今日输赢
	TodayAgentBalance int64  //今日返佣
	TodayActivity     int64  //今日活动奖励
	VndBalance        int64  //余额
	AgentBalance      int64  //佣金余额
	InitBalance       int64  //初始余额
	UpdateAt          time.Time
	CreateAt          time.Time
}

func (s *AgentMemberData) TableName() string {
	return "agent_member_data"
}

func QueryAgentMemberData(date string, uid string) AgentMemberData {
	var agentMemberData AgentMemberData
	common.GetMysql().Model(&agentMemberData).FirstOrInit(&agentMemberData,
		&AgentMemberData{Uid: uid, Date: date})
	if agentMemberData.CreateAt.IsZero() {
		agentMemberData.CreateAt = utils.Now()
		user := userStorage.QueryUserId(utils.ConvertOID(uid))
		agentMemberData.NickName = user.NickName
	}
	if agentMemberData.UpdateAt.IsZero() {
		agentMemberData.UpdateAt = utils.Now()
	}
	return agentMemberData
}
func QueryAllAgentMemberData() []AgentMemberData { //查询昨天的数据
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	lastTime := thatTime.AddDate(0, 0, -1)
	date := utils.GetDateStr(lastTime)
	var agentMemberData []AgentMemberData
	common.GetMysql().Find(&agentMemberData, "date=?", date)
	return agentMemberData
}
func QueryHaveChargeByUid(uidHex primitive.ObjectID, thatTime time.Time) bool {
	payConf := payStorage.QueryPayConfByMethodType("giftCode")
	c := common.GetMongoDB().C("order")
	var order []payStorage.Order
	selector := bson.M{"Status": payStorage.StatusSuccess, "MethodId": bson.M{"$ne": payConf.Oid}, "UpdateAt": bson.M{"$lt": thatTime}, "UserId": uidHex}
	err := c.Find(selector).All(&order)
	if err != nil {
		return false
	}
	if len(order) > 0 {
		return true
	} else {
		return false
	}
}

func UpsertAgentMemberData(agentMemberData AgentMemberData) {
	uidHex := utils.ConvertOID(agentMemberData.Uid)
	user := userStorage.QueryUserId(uidHex)
	if user.Oid.IsZero() || user.Type != userStorage.TypeNormal {
		return
	}
	wallet := walletStorage.QueryWallet(uidHex)
	agentMemberData.VndBalance = wallet.VndBalance + wallet.SafeBalance
	agentMemberData.AgentBalance = wallet.AgentBalance
	if agentMemberData.ID == 0 {
		invite := QueryInvite(uidHex)
		if !invite.Oid.IsZero() && !invite.ParentOid.IsZero() {
			baba := userStorage.QueryUserId(invite.ParentOid)
			agentMemberData.SuperiorAccount1 = baba.Account
			if baba.Type == userStorage.TypeAgent {
				agentMemberData.BelongAgent = baba.Account
			} else {
				if !invite.ParentOid2.IsZero() {
					yeye := userStorage.QueryUserId(invite.ParentOid2)
					agentMemberData.SuperiorAccount2 = yeye.Account
					if yeye.Type == userStorage.TypeAgent {
						agentMemberData.BelongAgent = yeye.Account
					} else {
						if !invite.ParentOid3.IsZero() {
							zufu := userStorage.QueryUserId(invite.ParentOid3)
							agentMemberData.SuperiorAccount3 = zufu.Account
						}
					}
				}
			}
		}
		//thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
		//uInfo := userStorage.QueryUserInfo(uidHex)
		//if uInfo.HaveCharge != 0 {
		//	lastTime := thatTime.AddDate(0, 0, -1)
		//	date := utils.GetDateStr(lastTime)
		//	ret := QueryAgentMemberData(date, agentMemberData.Uid)
		//	agentMemberData.InitBalance = ret.VndBalance
		//} else {
		//	agentMemberData.InitBalance = 0
		//}
		u := userStorage.QueryUserId(uidHex)
		agentMemberData.Account = u.Account
		common.GetMysql().Create(&agentMemberData)
	} else {
		common.GetMysql().Save(&agentMemberData)
	}
}
