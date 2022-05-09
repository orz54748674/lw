package gameStorage

import (
	"encoding/json"
	"strconv"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/activity"
	"vn/storage/agentStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

var (
	cBetRecord = "BetRecord"
)

func InitBetRecord(incDataExpireDay time.Duration) {
	c := common.GetMongoDB().C(cBetRecord)
	key := bsonx.Doc{{Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(int32(incDataExpireDay/time.Second))); err != nil {
		log.Error("create BetRecord Index: %s", err)
	}
	key = bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}, {Key: "GameType", Value: bsonx.Int32(1)}, {Key: "CreateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create BetRecord Index: %s", err)
	}
	_ = common.GetMysql().AutoMigrate(&BetRecord{})
}

type BetRecordParam struct {
	Uid        string //用户ID
	GameType   game.Type
	Income     int64  //结算输赢
	BetAmount  int64  //下注金额
	CurBalance int64  //结算余额
	SysProfit  int64  //系统抽水
	BotProfit  int64  //暗抽金额
	BetDetails string //下注详情
	GameId     string //游戏ID
	GameNo     string //游戏期号 后台展示 必传
	GameResult string //开奖结果
	IsJackpot  bool   //有没有中大奖
	IsSettled  bool   //是否已结算 结算后会触发数据统计和佣金结算,如果未结算，记得结算时触发数据统计和佣金结算
	Status     int8   // 记录状态
}

func refundOnBetDataCollect(betRecord BetRecord) {
	refundAgentIncome(utils.ConvertOID(betRecord.Uid), betRecord.BetAmount, betRecord.GameNo, betRecord.GameType)
	user := userStorage.QueryUserId(utils.ConvertOID(betRecord.Uid))
	if user.Type == userStorage.TypeNormal {
		agentStorage.OnBetRecord(betRecord.Uid, -1*betRecord.Income, -1*betRecord.BetAmount) //数据统计
	}
	userStorage.IncUserBet(user.Oid, -1*betRecord.Income, -1*betRecord.BetAmount)
}
func OnBetDataCollect(user userStorage.User, params BetRecordParam) int64 {
	agentProfit := CheckoutAgentIncome(user.Oid, params.BetAmount,
		params.GameNo, params.GameType)
	lobbyStorage.Win2(user.Oid, user.NickName, params.Income, params.GameType, params.IsJackpot)
	//var agentProfit int64 = 0
	if user.Type == userStorage.TypeNormal {
		agentStorage.OnBetRecord(params.Uid, params.Income, params.BetAmount) //数据统计
	}
	userStorage.IncUserBet(utils.ConvertOID(params.Uid), params.Income, params.BetAmount)
	return agentProfit
}

func refundAgentIncome(uid primitive.ObjectID, amount int64, betId string, game game.Type) int64 {
	invite := agentStorage.QueryInvite(uid)
	user := userStorage.QueryUserId(uid)
	if user.Type != userStorage.TypeNormal { //陪玩号不计算
		return 0
	}
	if invite.Oid.IsZero() {
		return 0
	}
	totalProfit := int64(0)
	//前一级
	agentUid := invite.ParentOid

	agent := agentStorage.QueryAgent(agentUid) //下线会员首次下注设置为代理
	if agent == nil {
		return 0
	}

	profit := int64(0)

	level := 1
	agentConf := agentStorage.QueryAgentConf(level)
	profitPerThousand := agentConf.ProfitPerThousand
	profit = amount * int64(profitPerThousand) / 1000
	if profit == 0 {
		return 0
	}
	parentUser := userStorage.QueryUserId(agentUid)
	profit = -1 * profit
	if parentUser.Type == userStorage.TypeNormal {
		bill := walletStorage.NewBill(agentUid.Hex(),
			walletStorage.TypeExpenses, walletStorage.EventAgentIncomeRefund, betId, profit)
		walletStorage.OperateAgentBalance(*bill)
		walletStorage.NotifyUserWallet(agentUid.Hex())
		agentStorage.NewAgentIncome(invite.ParentOid, uid, profit, amount, game, level)
		agentStorage.OnAgentBalanceData(invite.ParentOid.Hex(), profit, level)
	}
	totalProfit += profit

	//前二级
	level = 2
	agentUid = invite.ParentOid2
	if agentUid.IsZero() {
		return totalProfit
	}
	user = userStorage.QueryUserId(invite.ParentOid2)
	if user.Type != userStorage.TypeNormal { //陪玩号不计算
		return totalProfit
	}
	agentConf = agentStorage.QueryAgentConf(level)
	profitPerThousand = agentConf.ProfitPerThousand
	profit = amount * int64(profitPerThousand) / 1000
	if profit == 0 {
		return totalProfit
	}
	parentUser = userStorage.QueryUserId(agentUid)
	profit = -1 * profit
	if parentUser.Type == userStorage.TypeNormal {
		bill := walletStorage.NewBill(agentUid.Hex(),
			walletStorage.TypeExpenses, walletStorage.EventAgentIncomeRefund, betId, profit)
		walletStorage.OperateAgentBalance(*bill)
		walletStorage.NotifyUserWallet(agentUid.Hex())
		agentStorage.NewAgentIncome(invite.ParentOid2, uid, profit, amount, game, level)
		agentStorage.OnAgentBalanceData(invite.ParentOid2.Hex(), profit, level)
	}
	totalProfit += profit

	//前三级
	level = 3
	agentUid = invite.ParentOid3
	if agentUid.IsZero() {
		return totalProfit
	}
	user = userStorage.QueryUserId(agentUid)
	if user.Type != userStorage.TypeNormal { //陪玩号不计算
		return totalProfit
	}
	agentConf = agentStorage.QueryAgentConf(level)
	profitPerThousand = agentConf.ProfitPerThousand
	profit = amount * int64(profitPerThousand) / 1000
	if profit == 0 {
		return totalProfit
	}
	parentUser = userStorage.QueryUserId(agentUid)
	profit = -1 * profit
	if parentUser.Type == userStorage.TypeNormal {
		bill := walletStorage.NewBill(agentUid.Hex(),
			walletStorage.TypeExpenses, walletStorage.EventAgentIncomeRefund, betId, profit)
		walletStorage.OperateAgentBalance(*bill)
		walletStorage.NotifyUserWallet(agentUid.Hex())
		agentStorage.NewAgentIncome(invite.ParentOid3, uid, profit, amount, game, level)
		agentStorage.OnAgentBalanceData(invite.ParentOid3.Hex(), profit, level)
	}
	totalProfit += profit

	return totalProfit
}
func CheckoutAgentIncome(uid primitive.ObjectID, amount int64, betId string, gameType game.Type) int64 {
	invite := agentStorage.QueryInvite(uid)
	user := userStorage.QueryUserId(uid)
	if user.Type != userStorage.TypeNormal || gameType == game.Lottery { //陪玩号不计算 彩票不计算
		return 0
	}
	if invite.Oid.IsZero() {
		return 0
	}
	totalProfit := int64(0)
	//前一级
	level := 1
	agentUid := invite.ParentOid

	agent := agentStorage.QueryAgent(agentUid) //下线会员首次下注设置为代理
	if agent == nil {
		agent = &agentStorage.Agent{
			Oid:      agentUid,
			Level:    1,
			Count:    1,
			UpdateAt: utils.Now(),
		}
		agentStorage.InsertAgent(agent)
	}

	profit := int64(0)

	agentConf := agentStorage.QueryAgentConf(level)
	profitPerThousand := agentConf.ProfitPerThousand
	profit = amount * int64(profitPerThousand) / 1000
	if profit == 0 {
		return 0
	}
	parentUser := userStorage.QueryUserId(agentUid)
	if parentUser.Type == userStorage.TypeNormal {
		bill := walletStorage.NewBill(agentUid.Hex(),
			walletStorage.TypeIncome, walletStorage.EventAgentIncome, betId, profit)
		walletStorage.OperateAgentBalance(*bill)
		walletStorage.NotifyUserWallet(agentUid.Hex())
		agentStorage.NewAgentIncome(invite.ParentOid, uid, profit, amount, gameType, level)
		agentStorage.OnAgentBalanceData(invite.ParentOid.Hex(), profit, level)
	}
	totalProfit += profit

	//前二级
	level = 2
	agentUid = invite.ParentOid2
	if agentUid.IsZero() {
		return totalProfit
	}
	user = userStorage.QueryUserId(agentUid)
	if user.Type != userStorage.TypeNormal { //陪玩号不计算
		return totalProfit
	}
	agentConf = agentStorage.QueryAgentConf(level)
	profitPerThousand = agentConf.ProfitPerThousand
	profit = amount * int64(profitPerThousand) / 1000
	if profit == 0 {
		return totalProfit
	}
	parentUser = userStorage.QueryUserId(agentUid)
	if parentUser.Type == userStorage.TypeNormal {
		bill := walletStorage.NewBill(agentUid.Hex(),
			walletStorage.TypeIncome, walletStorage.EventAgentIncome, betId, profit)
		walletStorage.OperateAgentBalance(*bill)
		walletStorage.NotifyUserWallet(agentUid.Hex())
		agentStorage.NewAgentIncome(invite.ParentOid2, uid, profit, amount, gameType, level)
		agentStorage.OnAgentBalanceData(invite.ParentOid2.Hex(), profit, level)
	}
	totalProfit += profit

	//前三级
	level = 3
	agentUid = invite.ParentOid3
	if agentUid.IsZero() {
		return totalProfit
	}
	user = userStorage.QueryUserId(agentUid)
	if user.Type != userStorage.TypeNormal { //陪玩号不计算
		return totalProfit
	}
	agentConf = agentStorage.QueryAgentConf(level)
	profitPerThousand = agentConf.ProfitPerThousand
	profit = amount * int64(profitPerThousand) / 1000
	if profit == 0 {
		return totalProfit
	}
	parentUser = userStorage.QueryUserId(agentUid)
	if parentUser.Type == userStorage.TypeNormal {
		bill := walletStorage.NewBill(agentUid.Hex(),
			walletStorage.TypeIncome, walletStorage.EventAgentIncome, betId, profit)
		walletStorage.OperateAgentBalance(*bill)
		walletStorage.NotifyUserWallet(agentUid.Hex())
		agentStorage.NewAgentIncome(invite.ParentOid3, uid, profit, amount, gameType, level)
		agentStorage.OnAgentBalanceData(invite.ParentOid3.Hex(), profit, level)
	}
	totalProfit += profit

	return totalProfit
}
func InsertBetRecord(params BetRecordParam) {
	user := userStorage.QueryUserId(utils.ConvertOID(params.Uid))
	var agentProfit int64 = 0
	if params.IsSettled { //已结算
		agentProfit = OnBetDataCollect(user, params)
	}

	ip := ""
	if token := userStorage.QueryTokenByUid(user.Oid); token != nil {
		ip = token.Ip
	}
	betRecord := BetRecord{
		Oid:         primitive.NewObjectID(),
		Uid:         params.Uid,
		UserType:    user.Type,
		Channel:     user.Channel,
		GameType:    params.GameType,
		GameId:      params.GameId,
		GameNo:      params.GameNo,
		GameResult:  params.GameResult,
		Income:      params.Income,
		BetAmount:   params.BetAmount,
		CurBalance:  params.CurBalance,
		CreateAt:    utils.Now(),
		SysProfit:   params.SysProfit,
		BotProfit:   params.BotProfit,
		AgentProfit: agentProfit,
		BetDetails:  params.BetDetails,
		Ip:          ip,
		UpdateAt:    utils.Now(),
		Status:      params.Status,
	}
	go activity.NotifyBetActivity(betRecord.Uid, betRecord.GameType, params.BetAmount)

	c := common.GetMongoDB().C(cBetRecord)
	if err := c.Insert(&betRecord); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			common.GetMysql().Create(&betRecord)
		})
	}
}

func ModifyBetNumber(gameId, gameNo string) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": gameId}
	update := bson.M{"GameNo": gameNo, "UpdateAt": utils.Now()}
	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"game_no": gameNo, "update_at": utils.Now()}
			common.GetMysql().Model(new(BetRecord)).Where("game_id=?", gameId).Updates(data)
		})
	}
}

/**
调用时机，结算时调用
uid: 用户ID
income : 输赢多少 ， betAmount 下注金额， curBalance： 结算后余额
sysProfit： 系统明抽，没有传 0
botProfit： 控盘暗抽 ，没有传0
betDetails： 下注详情 数据结构自定义，后台要做展示，可JSON,可逗号分割
gameId: 游戏期号，没有传空
gameResult： 开奖结果 数据结构自定义，后台要做展示
*/
//func InsertBetRecord(uid string, gameType game.Type, income, betAmount, curBalance, sysProfit, botProfit int64, betDetails, gameId, gameResult string) {
//	insertBetRecord(uid, gameType, income, betAmount, curBalance, sysProfit, botProfit, betDetails, gameId, gameResult)
//	user := userStorage.QueryUserId(utils.ConvertOID(uid))
//	if user.Type == userStorage.TypeNormal {
//		agentStorage.OnBetRecord(uid, income, betAmount)
//	}
//}

/**
调用时机，结算时调用
uid: 用户ID
income : 输赢多少 ， betAmount 下注金额， curBalance： 结算后余额
sysProfit： 系统明抽，没有传 0
botProfit： 控盘暗抽 ，没有传0
betDetails： 下注详情 数据结构自定义，后台要做展示，可JSON,可逗号分割
gameId: 游戏期号，没有传空
gameResult： 开奖结果 数据结构自定义，后台要做展示
*/
//func InsertLotteryBetRecord(uid string, gameType game.Type, income, betAmount, curBalance, sysProfit, botProfit int64, betDetails, gameId, gameResult string) {
//	insertBetRecord(uid, gameType, income, betAmount, curBalance, sysProfit, botProfit, betDetails, gameId, gameResult)
//}
func UpdateLotteryBetRecord(uid, gameId, gameNo, gameResult string, gameType game.Type, income, betAmount, curBalance, sysProfit, botProfit int64) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": gameId}
	update := bson.M{"GameResult": gameResult, "Income": income, "CurBalance": curBalance, "UpdateAt": utils.Now()}
	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"game_result": gameResult, "income": income, "cur_balance": curBalance, "update_at": utils.Now()}
			common.GetMysql().Model(new(BetRecord)).Where("game_id=?", gameId).Updates(data)
		})
	}
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	params := BetRecordParam{
		Uid:       uid,
		GameType:  gameType,
		Income:    income,
		BetAmount: betAmount,
		GameId:    gameId,
		GameNo:    gameNo,
		IsJackpot: false,
	}
	OnBetDataCollect(user, params)
	// agentStorage.OnBetRecord(uid, income, betAmount)
}

func CloseLotteryBetRecord(gameId string) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": gameId}
	update := bson.M{"BetAmount": 0, "UpdateAt": utils.Now()}
	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"bet_amount": 0, "update_at": utils.Now()}
			common.GetMysql().Model(new(BetRecord)).Where("game_id=?", gameId).Updates(data)
		})
	}
}

//actionType==1,取消注单
//actionType==2,正常结算
//actionType==3,已结算注单改成未结算
//actionType==4,取消取消注单
//actionType==5,系统调账
//actionType==6,拉取注单详情后修改betRecords
func UpdateApiCmdBetRecord(params BetRecordParam, actionType int) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"Uid": params.Uid, "GameNo": params.GameNo, "GameType": params.GameType}
	var betRecord BetRecord
	if err := c.Find(find).One(&betRecord); err != nil {
		log.Error(err.Error())
		return
	}

	if actionType != 6 {
		balanceMap := make(map[string]interface{})
		balanceMap["BalanceAfterCount"] = strconv.FormatInt(params.CurBalance, 10)
		tmpByte, _ := json.Marshal(balanceMap)
		betRecord.GameResult = string(tmpByte)
		betRecord.GameId = params.GameId
	}

	switch actionType {
	case 1:
		betRecord.BetAmount = 0
		betRecord.Income = 0
	case 2:
		var agentProfit int64 = 0
		params.BetAmount = betRecord.BetAmount
		params.Income = params.Income - betRecord.BetAmount
		betRecord.Income = params.Income
		user := userStorage.QueryUserId(utils.ConvertOID(params.Uid))
		agentProfit = OnBetDataCollect(user, params)
		betRecord.AgentProfit = agentProfit
	case 3:
		betRecord.Income = 0
		betRecord.AgentProfit = 0
		RefundCmdBetRecord(params.Uid, params.GameNo, params.GameType)
	case 4:
		betRecord.BetAmount = params.BetAmount
		betRecord.GameResult = ""
	case 5:
		betRecord.Income = betRecord.Income + params.Income
	case 6:
		unmarshalMap := make(map[string]interface{})
		err := json.Unmarshal([]byte(betRecord.GameResult), &unmarshalMap)
		if err == nil {
			if val, ok := unmarshalMap["BalanceAfterCount"]; ok {
				tmpMap := make(map[string]interface{})
				if err = json.Unmarshal([]byte(params.GameResult), &tmpMap); err == nil {
					tmpMap["BalanceAfterCount"] = val
					tmpByte, _ := json.Marshal(tmpMap)
					params.GameResult = string(tmpByte)
				}
			} else {
				tmpMap := make(map[string]interface{})
				if err = json.Unmarshal([]byte(params.GameResult), &tmpMap); err == nil {
					tmpMap["BalanceAfterCount"] = params.CurBalance
					if params.CurBalance == 0 {
						tmpMap["BalanceAfterCount"] = -1
					}
					tmpByte, _ := json.Marshal(tmpMap)
					params.GameResult = string(tmpByte)
				}
			}
		} else {
			tmpMap := make(map[string]interface{})
			if err = json.Unmarshal([]byte(params.GameResult), &tmpMap); err == nil {
				tmpMap["BalanceAfterCount"] = params.CurBalance
				if params.CurBalance == 0 {
					tmpMap["BalanceAfterCount"] = -1
				}
				tmpByte, _ := json.Marshal(tmpMap)
				params.GameResult = string(tmpByte)
			}
		}
		betRecord.GameResult = params.GameResult
		betRecord.BetDetails = params.BetDetails
		if params.Income != 0 {
			betRecord.Income = params.Income
		}
	}
	betRecord.UpdateAt = utils.Now()

	if err := c.Update(find, &betRecord); err != nil {
		log.Error(err.Error())
	} else {
		common.AddQueueByTag(params.Uid, func() {
			common.GetMysql().Model(new(BetRecord)).Where("uid=? and game_no=? and game_type=?", params.Uid, params.GameNo, params.GameType).Updates(&betRecord)
		})
	}
}

func RefundCmdBetRecord(uid, gameNo string, gameType game.Type) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"Uid": uid, "GameNo": gameNo, "GameType": gameType}
	//退统计
	var betRecord BetRecord
	if err := c.Find(find).One(&betRecord); err != nil {
		log.Error("RefundAwcBetRecord err: %v", err.Error())
		return
	}
	refundOnBetDataCollect(betRecord)
	//退统计 END
	update := bson.M{"Income": 0, "AgentProfit": 0, "UpdateAt": utils.Now()}
	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"income": 0, "agent_profit": 0, "update_at": utils.Now()}
			common.GetMysql().Debug().Model(new(BetRecord)).Where(
				"uid=? and game_no=? and game_type=?", uid, gameNo, gameType).Updates(&data)
		})
	}
}

func ApiCqUpsertBetRecord(params BetRecordParam) {
	user := userStorage.QueryUserId(utils.ConvertOID(params.Uid))
	var agentProfit int64 = 0
	if params.IsSettled { //已结算
		agentProfit = OnBetDataCollect(user, params)
	}

	ip := ""
	if token := userStorage.QueryTokenByUid(user.Oid); token != nil {
		ip = token.Ip
	}

	betRecord := BetRecord{
		Oid:         primitive.NewObjectID(),
		Uid:         params.Uid,
		Channel:     user.Channel,
		GameType:    params.GameType,
		GameId:      params.GameId,
		GameNo:      params.GameNo,
		GameResult:  params.GameResult,
		Income:      params.Income,
		BetAmount:   params.BetAmount,
		CurBalance:  params.CurBalance,
		CreateAt:    utils.Now(),
		SysProfit:   params.SysProfit,
		BotProfit:   params.BotProfit,
		AgentProfit: agentProfit,
		BetDetails:  params.BetDetails,
		Ip:          ip,
		UpdateAt:    utils.Now(),
	}
	c := common.GetMongoDB().C(cBetRecord)
	query := bson.M{"GameNo": params.GameNo, "GameType": params.GameType, "Uid": params.Uid}
	if _, err := c.Upsert(query, &betRecord); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			common.GetMysql().Create(&betRecord)
		})
	}
}

func RefundAwcBetRecord(oid, GameResult string, gameType game.Type) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": oid}
	//退统计
	var betRecord BetRecord
	if err := c.Find(find).One(&betRecord); err != nil {
		log.Error("RefundAwcBetRecord err: %v", err.Error())
		return
	}
	refundOnBetDataCollect(betRecord)
	//退统计 END
	update := bson.M{"GameResult": GameResult, "Income": 0, "BetAmount": 0, "AgentProfit": 0, "UpdateAt": utils.Now()}
	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"game_result": GameResult, "income": 0, "bet_amount": 0, "agent_profit": 0, "update_at": utils.Now()}
			common.GetMysql().Debug().Model(new(BetRecord)).Where(
				"game_id=?", oid).Updates(data)
		})
	}
}

func UpdateSboBetRecord(oid, uid, gameNo, GameResult string, gameType game.Type, income, curBalance, betAmount int64) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": oid}
	params := BetRecordParam{
		Uid:       uid,
		GameType:  gameType,
		Income:    income,
		BetAmount: betAmount,
		GameId:    oid,
		GameNo:    gameNo,
		IsJackpot: false,
	}
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	agentProfit := OnBetDataCollect(user, params)
	update := bson.M{"GameResult": GameResult, "Income": income, "BetAmount": betAmount, "AgentProfit": agentProfit, "CurBalance": curBalance, "UpdateAt": utils.Now()}

	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"agent_profit": agentProfit, "game_result": GameResult, "income": income, "bet_amount": betAmount, "cur_balance": curBalance, "update_at": utils.Now()}
			common.GetMysql().Model(new(BetRecord)).Where("game_id=?", oid).Updates(data)
		})
	}
}

func UpdateBetRecord(oid, uid, gameNo, GameResult string, gameType game.Type, income, curBalance, betAmount int64) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": oid}
	params := BetRecordParam{
		Uid:       uid,
		GameType:  gameType,
		Income:    income,
		BetAmount: betAmount,
		GameId:    oid,
		GameNo:    gameNo,
		IsJackpot: false,
	}
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	agentProfit := OnBetDataCollect(user, params)
	update := bson.M{"GameResult": GameResult, "Income": income, "BetAmount": betAmount, "AgentProfit": agentProfit, "CurBalance": curBalance, "UpdateAt": utils.Now()}

	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"agent_profit": agentProfit, "game_result": GameResult, "income": income, "bet_amount": betAmount, "cur_balance": curBalance, "update_at": utils.Now()}
			common.GetMysql().Model(new(BetRecord)).Where("game_id=?", oid).Updates(data)
		})
	}
}
func UpdateRecord(gameNo, GameResult string) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameNo": gameNo}

	update := bson.M{"$set": bson.M{"GameResult": GameResult, "BetDetails": GameResult, "UpdateAt": utils.Now()}}

	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"game_result": GameResult, "bet_details": GameResult, "update_at": utils.Now()}
			common.GetMysql().Model(new(BetRecord)).Where("game_no=?", gameNo).Updates(data)
		})
	}
}

func UpdateGameRes(oid, GameResult string) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": oid}

	update := bson.M{"$set": bson.M{"GameResult": GameResult, "UpdateAt": utils.Now()}}

	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"game_result": GameResult, "update_at": utils.Now()}
			common.GetMysql().Model(new(BetRecord)).Where("game_id=?", oid).Updates(data)
		})
	}
}

func RefundBetRecord(oid, GameResult string, gameType game.Type, income, betAmount, AgentProfit int64) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": oid}
	//退统计
	var betRecord BetRecord
	if err := c.Find(find).One(&betRecord); err != nil {
		log.Error("RefundSabaBetRecord err: %v", err.Error())
		return
	}
	refundOnBetDataCollect(betRecord)
	//退统计 END
	update := bson.M{"GameResult": GameResult, "Income": income, "BetAmount": betAmount, "AgentProfit": AgentProfit, "UpdateAt": utils.Now()}
	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"game_result": GameResult, "income": income, "bet_amount": betAmount, "agent_profit": AgentProfit, "update_at": utils.Now()}
			common.GetMysql().Debug().Model(new(BetRecord)).Where(
				"game_id=?", oid).Updates(data)
		})
	}
}

func RefundSboBetRecord(oid, GameResult string, gameType game.Type) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": oid}
	//退统计
	var betRecord BetRecord
	if err := c.Find(find).One(&betRecord); err != nil {
		log.Error("RefundSboBetRecord err: %v", err.Error())
		return
	}
	refundOnBetDataCollect(betRecord)
	//退统计 END
	update := bson.M{"GameResult": GameResult, "Income": 0, "BetAmount": 0, "AgentProfit": 0, "UpdateAt": utils.Now()}
	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"game_result": GameResult, "income": 0, "bet_amount": 0, "agent_profit": 0, "update_at": utils.Now()}
			common.GetMysql().Debug().Model(new(BetRecord)).Where(
				"game_id=?", oid).Updates(data)
		})
	}
}

func RefundWmBetRecord(betId string, income int64) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": betId}
	//退统计
	var betRecord BetRecord
	if err := c.Find(find).One(&betRecord); err != nil {
		log.Error("RefundWmBetRecord GameId:%v err: %v", betId, err.Error())
		return
	}
	refundOnBetDataCollect(betRecord)
	//退统计 END
	update := bson.M{"Income": income - betRecord.BetAmount, "UpdateAt": utils.Now()}
	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"income": income - betRecord.BetAmount, "update_at": utils.Now()}
			common.GetMysql().Debug().Model(new(BetRecord)).Where(
				"game_id=?", betId).Updates(data)
		})
	}
}

func RefundXgBetRecord(oid string, income, betAmount int64) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": oid}
	//退统计
	var betRecord BetRecord
	if err := c.Find(find).One(&betRecord); err != nil {
		log.Error("RefundXgBetRecord err: %v", err.Error())
		return
	}
	refundOnBetDataCollect(betRecord)
	//退统计 END
	update := bson.M{"Income": income, "BetAmount": betAmount, "UpdateAt": utils.Now()}
	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"income": income, "bet_amount": betAmount, "update_at": utils.Now()}
			common.GetMysql().Debug().Model(new(BetRecord)).Where(
				"game_id=?", oid).Updates(data)
		})
	}
}

func RefundXgBetRecordInfo(oid string, gameResult, betDetails string) {
	c := common.GetMongoDB().C(cBetRecord)
	find := bson.M{"GameId": oid}
	update := bson.M{"GameResult": gameResult, "BetDetails": betDetails, "UpdateAt": utils.Now()}
	if err := c.Update(find, update); err != nil {
		log.Error(err.Error())
	} else {
		common.ExecQueueFunc(func() {
			data := map[string]interface{}{"game_result": gameResult, "bet_details": betDetails, "update_at": utils.Now()}
			common.GetMysql().Debug().Model(new(BetRecord)).Where(
				"game_id=?", oid).Updates(data)
		})
	}
}

//func insertBetRecord(uid string, gameType game.Type, income, betAmount, curBalance, sysProfit, botProfit int64, betDetails, gameId, gameResult string) {
//	user := userStorage.QueryUserId(utils.ConvertOID(uid))
//	userStorage.IncUserBet(utils.ConvertOID(uid), income, betAmount)
//	invite := agentStorage.QueryInvite(utils.ConvertOID(uid))
//	var agentProfit int64 = 0
//	if invite != nil {
//		agent := agentStorage.QueryAgent(invite.ParentOid)
//		if agent != nil {
//			parentUser := userStorage.QueryUserId(invite.ParentOid) //
//			profitPerThousand := parentUser.ProfitPerThousand
//			if profitPerThousand < 0 {
//				agentConf := agentStorage.QueryAgentConf(agent.Level)
//				profitPerThousand = agentConf.ProfitPerThousand
//			}
//			agentProfit = betAmount * int64(profitPerThousand) / 1000
//		}
//	}
//	token := userStorage.QueryTokenByUid(user.Oid)
//	betRecord := BetRecord{
//		Oid:         primitive.NewObjectID(),
//		Uid:         uid,
//		Channel:     user.Channel,
//		GameType:    gameType,
//		GameId:      gameId,
//		GameResult:  gameResult,
//		Income:      income,
//		BetAmount:   betAmount,
//		CurBalance:  curBalance,
//		CreateAt:    utils.Now(),
//		SysProfit:   sysProfit,
//		BotProfit:   botProfit,
//		AgentProfit: agentProfit,
//		BetDetails:  betDetails,
//		Ip:          token.Ip,
//	}
//	c := common.GetMongoDB().C(cBetRecord)
//	if err := c.Insert(&betRecord); err != nil {
//		log.Error(err.Error())
//	} else {
//		common.ExecQueueFunc(func() {
//			common.GetMysql().Create(&betRecord)
//		})
//	}
//}

func QueryBetRecord(uid string, offset int, pageSize int, gameType string) []BetRecord {
	c := common.GetMongoDB().C(cBetRecord)
	if pageSize != 0 {
		if offset/pageSize > maxPageNum {
			return []BetRecord{}
		}
	}
	var selector map[string]interface{}
	if gameType == "All" {
		selector = bson.M{"Uid": uid}
	} else {
		selector = bson.M{"Uid": uid, "GameType": gameType}
	}
	var betRecord []BetRecord
	err := c.Find(selector).Sort("-CreateAt").Skip(offset).Limit(pageSize).All(&betRecord)
	if err != nil {
		return []BetRecord{}
	}
	if betRecord == nil {
		return []BetRecord{}
	}
	for k, v := range betRecord {
		betRecord[k].GameType = game.Type(common.I18str(string(v.GameType)))
		betRecord[k].CreateAt = betRecord[k].CreateAt.Local()
		betRecord[k].GameId = betRecord[k].GameNo
	}
	return betRecord
}
func QueryBetRecordTotal(uid string, gameType string) int {
	c := common.GetMongoDB().C(cBetRecord)
	var selector map[string]interface{}
	if gameType == "All" {
		selector = bson.M{"Uid": uid}
	} else {
		selector = bson.M{"Uid": uid, "GameType": gameType}
	}
	count, err := c.Find(selector).Count()
	if err != nil {
		return 0
	}
	return int(count)
}
func QueryBetRecordByUsers(uids []string, time time.Time) int {
	c := common.GetMongoDB().C(cBetRecord)
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"Uid": bson.M{"$in": uids},
			"CreateAt": bson.M{"$gt": time}},
		}},
		{{"$group", bson.M{"_id": "$Uid", "Count": bson.M{"$sum": 1}}}},
	}
	var res []map[string]interface{}
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	return len(res)
}
func QueryBetRecordByChannel(channel string, gameType game.Type, time time.Time) []map[string]interface{} {
	c := common.GetMongoDB().C(cBetRecord)
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"Channel": channel, "GameType": gameType, "CreateAt": bson.M{"$gt": time}}}},
		{{"$group", bson.M{"_id": "$Channel", "SysProfit": bson.M{"$sum": "$SysProfit"}, "BotProfit": bson.M{"$sum": "$BotProfit"}, "BetAmount": bson.M{"$sum": "$BetAmount"}, "AgentProfit": bson.M{"$sum": "$AgentProfit"}}}},
	}
	var res []map[string]interface{}
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	return res
}
func QueryBetRecordByChannelUsers(channel string, gameType game.Type, time time.Time) int {
	c := common.GetMongoDB().C(cBetRecord)
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"Channel": channel, "GameType": gameType, "CreateAt": bson.M{"$gt": time}}}},
		{{"$group", bson.M{"_id": "$Uid", "BetUsers": bson.M{"$sum": 1}}}},
	}
	var res []map[string]interface{}
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	return len(res)
}
func QueryTodayBetRecordTotal(uid string, gameType game.Type) int64 {
	c := common.GetMongoDB().C(cBetRecord)
	thatTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	pipe := mongo.Pipeline{
		{{"$match", bson.M{"Uid": uid, "GameType": gameType, "CreateAt": bson.M{"$gt": thatTime}}}},
		{{"$group", bson.M{"_id": "$Uid", "BetAmount": bson.M{"$sum": "$BetAmount"}}}},
	}
	var res []map[string]interface{}
	if err := c.Pipe(pipe).All(&res); err != nil {
		log.Error(err.Error())
	}
	if len(res) != 0 {
		return res[0]["BetAmount"].(int64)
	}
	return 0
}
